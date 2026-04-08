# Паттерны работы с PostgreSQL

## Обзор

Практические паттерны, которые встречаются в каждом production Go-сервисе: soft delete, optimistic locking, cursor pagination, repository pattern, prepared statements, connection pool tuning. Знание этих паттернов — must have для middle.

## Soft Delete

Вместо физического удаления строка помечается как удалённая. Позволяет восстановить данные и вести аудит.

### Базовая реализация

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    name TEXT NOT NULL,
    deleted_at TIMESTAMPTZ NULL, -- NULL = активная запись
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- "Удаление"
UPDATE users SET deleted_at = NOW() WHERE id = 42;

-- Выборка только активных
SELECT * FROM users WHERE deleted_at IS NULL;

-- Восстановление
UPDATE users SET deleted_at = NULL WHERE id = 42;
```

### Partial index для performance

Большинство запросов работают только с активными записями. Partial index исключает удалённые строки.

```sql
-- Индекс ТОЛЬКО по активным пользователям — меньше, быстрее
CREATE INDEX idx_users_email_active ON users(email) WHERE deleted_at IS NULL;

-- Этот запрос использует partial index:
SELECT * FROM users WHERE email = 'alice@example.com' AND deleted_at IS NULL;
```

### UNIQUE constraint с soft delete

Проблема: обычный UNIQUE не даст создать нового пользователя с email удалённого.

```sql
-- ПЛОХО: обычный unique — удалённый email блокирует регистрацию
CREATE UNIQUE INDEX idx_users_email_unique ON users(email);

-- ХОРОШО: unique только среди активных
CREATE UNIQUE INDEX idx_users_email_unique ON users(email) WHERE deleted_at IS NULL;

-- Теперь можно:
-- 1. alice@example.com (active)
-- 2. alice@example.com (deleted_at = '2024-01-01') — не конфликтует!
```

### Каскадное мягкое удаление

```sql
-- При soft delete пользователя нужно также "удалить" его заказы
CREATE OR REPLACE FUNCTION cascade_soft_delete_user()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
        UPDATE orders SET deleted_at = NEW.deleted_at WHERE user_id = NEW.id AND deleted_at IS NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_cascade_soft_delete_user
AFTER UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION cascade_soft_delete_user();
```

### Soft delete в Go

```go
type UserRepo struct {
    pool *pgxpool.Pool
}

func (r *UserRepo) Delete(ctx context.Context, id int64) error {
    tag, err := r.pool.Exec(ctx,
        "UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL", id)
    if err != nil {
        return err
    }
    if tag.RowsAffected() == 0 {
        return ErrNotFound
    }
    return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (User, error) {
    var u User
    err := r.pool.QueryRow(ctx,
        "SELECT id, email, name, created_at FROM users WHERE id = $1 AND deleted_at IS NULL", id,
    ).Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt)
    if errors.Is(err, pgx.ErrNoRows) {
        return User{}, ErrNotFound
    }
    return u, err
}
```

## Optimistic Locking

Защита от одновременного обновления одной записи несколькими клиентами. Не блокирует строки — проверяет версию при записи.

### Реализация через version

```sql
CREATE TABLE products (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price NUMERIC(10,2) NOT NULL,
    version INT NOT NULL DEFAULT 1
);

-- Чтение: запоминаем version
SELECT id, name, price, version FROM products WHERE id = 42;
-- Получили: id=42, name='Widget', price=9.99, version=3

-- Обновление: проверяем что version не изменилась
UPDATE products
SET price = 12.99, version = version + 1
WHERE id = 42 AND version = 3;
-- Если кто-то успел обновить раньше (version = 4), affected rows = 0
```

### Реализация в Go

```go
var ErrConflict = errors.New("optimistic lock conflict")

func (r *ProductRepo) UpdatePrice(ctx context.Context, id int64, newPrice float64, version int) error {
    tag, err := r.pool.Exec(ctx, `
        UPDATE products
        SET price = $1, version = version + 1
        WHERE id = $2 AND version = $3
    `, newPrice, id, version)
    if err != nil {
        return fmt.Errorf("update product: %w", err)
    }

    if tag.RowsAffected() == 0 {
        return ErrConflict // кто-то обновил раньше
    }
    return nil
}

// Использование с retry
func (s *ProductService) UpdatePrice(ctx context.Context, id int64, newPrice float64) error {
    const maxRetries = 3

    for attempt := 0; attempt < maxRetries; attempt++ {
        product, err := s.repo.GetByID(ctx, id)
        if err != nil {
            return err
        }

        err = s.repo.UpdatePrice(ctx, id, newPrice, product.Version)
        if err == nil {
            return nil // успешно обновили
        }
        if !errors.Is(err, ErrConflict) {
            return err // другая ошибка
        }
        // Конфликт — пробуем снова с актуальной версией
    }
    return fmt.Errorf("update failed after %d retries: %w", maxRetries, ErrConflict)
}
```

### Альтернатива: updated_at

```sql
UPDATE products
SET price = 12.99, updated_at = NOW()
WHERE id = 42 AND updated_at = '2024-01-15 10:30:00+00';
```

Минус `updated_at`: точность timestamp может быть проблемой при очень высокой конкурентности. `version INT` — надёжнее.

## Cursor Pagination vs Offset

### Проблема Offset

```sql
-- Offset: для страницы 100 PostgreSQL сканирует и отбрасывает 1000 строк
SELECT * FROM orders ORDER BY created_at DESC LIMIT 10 OFFSET 1000;
-- Чем дальше страница, тем медленнее!

-- Performance: O(offset + limit) — линейно растёт
-- Страница 1:    ~ 0.5 мс
-- Страница 100:  ~ 5 мс
-- Страница 10000: ~ 500 мс
```

Дополнительная проблема: если между запросами появились новые строки, элементы "сдвигаются" и могут дублироваться или пропускаться.

### Cursor Pagination

```sql
-- Cursor: всегда одинаково быстро, O(limit) при наличии индекса
-- Первая страница:
SELECT id, title, created_at
FROM orders
ORDER BY created_at DESC, id DESC
LIMIT 10;

-- Следующая страница (cursor = created_at и id последнего элемента):
SELECT id, title, created_at
FROM orders
WHERE (created_at, id) < ($1, $2)  -- row comparison
ORDER BY created_at DESC, id DESC
LIMIT 10;
```

**Важно**: сортировка по `created_at` одному может дать дубликаты (если несколько строк с одинаковым timestamp). Добавляем `id` как tiebreaker для стабильности.

### Кодирование курсора

```go
import "encoding/base64"
import "fmt"
import "strings"

// Cursor = base64(created_at + "|" + id)
func encodeCursor(createdAt time.Time, id int64) string {
    raw := fmt.Sprintf("%s|%d", createdAt.Format(time.RFC3339Nano), id)
    return base64.StdEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(cursor string) (time.Time, int64, error) {
    raw, err := base64.StdEncoding.DecodeString(cursor)
    if err != nil {
        return time.Time{}, 0, fmt.Errorf("invalid cursor: %w", err)
    }
    parts := strings.SplitN(string(raw), "|", 2)
    if len(parts) != 2 {
        return time.Time{}, 0, fmt.Errorf("invalid cursor format")
    }
    t, err := time.Parse(time.RFC3339Nano, parts[0])
    if err != nil {
        return time.Time{}, 0, err
    }
    id, err := strconv.ParseInt(parts[1], 10, 64)
    return t, id, err
}
```

### Полная реализация в Go

```go
type Page struct {
    Items      []Order `json:"items"`
    NextCursor string  `json:"next_cursor,omitempty"`
    HasMore    bool    `json:"has_more"`
}

func (r *OrderRepo) List(ctx context.Context, cursor string, limit int) (Page, error) {
    if limit <= 0 || limit > 100 {
        limit = 20
    }

    var (
        rows pgx.Rows
        err  error
    )

    // Запрашиваем limit+1, чтобы узнать есть ли ещё страницы
    fetchLimit := limit + 1

    if cursor == "" {
        // Первая страница
        rows, err = r.pool.Query(ctx, `
            SELECT id, title, total, created_at
            FROM orders
            ORDER BY created_at DESC, id DESC
            LIMIT $1
        `, fetchLimit)
    } else {
        cursorTime, cursorID, err := decodeCursor(cursor)
        if err != nil {
            return Page{}, fmt.Errorf("invalid cursor: %w", err)
        }
        rows, err = r.pool.Query(ctx, `
            SELECT id, title, total, created_at
            FROM orders
            WHERE (created_at, id) < ($1, $2)
            ORDER BY created_at DESC, id DESC
            LIMIT $3
        `, cursorTime, cursorID, fetchLimit)
    }
    if err != nil {
        return Page{}, err
    }
    defer rows.Close()

    var orders []Order
    for rows.Next() {
        var o Order
        if err := rows.Scan(&o.ID, &o.Title, &o.Total, &o.CreatedAt); err != nil {
            return Page{}, err
        }
        orders = append(orders, o)
    }

    page := Page{}
    if len(orders) > limit {
        page.HasMore = true
        orders = orders[:limit] // убираем лишний элемент
        last := orders[len(orders)-1]
        page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
    }
    page.Items = orders
    return page, nil
}
```

### Keyset pagination для составных ключей

```sql
-- Сортировка по нескольким полям: priority DESC, created_at ASC, id ASC
-- Cursor содержит все три значения

SELECT * FROM tasks
WHERE (priority, created_at, id) < ($1, $2, $3)  -- при DESC по priority
   OR (priority = $1 AND (created_at, id) > ($2, $3))  -- при ASC по created_at, id
ORDER BY priority DESC, created_at ASC, id ASC
LIMIT 20;

-- Упрощённый вариант с row comparison (если все направления одинаковые):
SELECT * FROM tasks
WHERE (created_at, id) > ($1, $2)
ORDER BY created_at ASC, id ASC
LIMIT 20;
```

## Repository Pattern

### Интерфейс

```go
type UserRepository interface {
    GetByID(ctx context.Context, id int64) (User, error)
    GetByEmail(ctx context.Context, email string) (User, error)
    List(ctx context.Context, filter UserFilter) ([]User, error)
    Create(ctx context.Context, user *User) error
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id int64) error
}

type User struct {
    ID        int64
    Email     string
    Name      string
    Version   int
    CreatedAt time.Time
}

type UserFilter struct {
    NameLike  string
    Active    *bool
    Limit     int
    Cursor    string
}
```

### Реализация с pgx

```go
type pgxUserRepo struct {
    pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
    return &pgxUserRepo{pool: pool}
}

func (r *pgxUserRepo) GetByID(ctx context.Context, id int64) (User, error) {
    var u User
    err := r.pool.QueryRow(ctx, `
        SELECT id, email, name, version, created_at
        FROM users
        WHERE id = $1 AND deleted_at IS NULL
    `, id).Scan(&u.ID, &u.Email, &u.Name, &u.Version, &u.CreatedAt)

    if errors.Is(err, pgx.ErrNoRows) {
        return User{}, ErrNotFound
    }
    return u, err
}

func (r *pgxUserRepo) Create(ctx context.Context, user *User) error {
    return r.pool.QueryRow(ctx, `
        INSERT INTO users (email, name)
        VALUES ($1, $2)
        RETURNING id, version, created_at
    `, user.Email, user.Name).Scan(&user.ID, &user.Version, &user.CreatedAt)
}

func (r *pgxUserRepo) Update(ctx context.Context, user *User) error {
    tag, err := r.pool.Exec(ctx, `
        UPDATE users
        SET email = $1, name = $2, version = version + 1
        WHERE id = $3 AND version = $4 AND deleted_at IS NULL
    `, user.Email, user.Name, user.ID, user.Version)
    if err != nil {
        return err
    }
    if tag.RowsAffected() == 0 {
        return ErrConflict
    }
    user.Version++
    return nil
}

func (r *pgxUserRepo) Delete(ctx context.Context, id int64) error {
    tag, err := r.pool.Exec(ctx,
        "UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL", id)
    if err != nil {
        return err
    }
    if tag.RowsAffected() == 0 {
        return ErrNotFound
    }
    return nil
}
```

### Тестирование с моками

```go
// Mock для unit-тестов
type mockUserRepo struct {
    users map[int64]User
    mu    sync.RWMutex
}

func newMockUserRepo() *mockUserRepo {
    return &mockUserRepo{users: make(map[int64]User)}
}

func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (User, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    u, ok := m.users[id]
    if !ok {
        return User{}, ErrNotFound
    }
    return u, nil
}

// В тестах service-уровня:
func TestUserService_Activate(t *testing.T) {
    repo := newMockUserRepo()
    repo.users[1] = User{ID: 1, Name: "Alice", Version: 1}

    svc := NewUserService(repo)
    err := svc.Activate(context.Background(), 1)
    require.NoError(t, err)
}
```

## Prepared Statements

Prepared statements парсятся и планируются один раз, а затем выполняются многократно с разными параметрами.

### Жизненный цикл в pgx

```go
// Ручной Prepare / Deallocate
conn, err := pool.Acquire(ctx)
defer conn.Release()

// Подготовка
sd, err := conn.Conn().Prepare(ctx, "get_user", "SELECT * FROM users WHERE id = $1")
// sd содержит описание полей и параметров

// Выполнение (многократно)
rows, err := conn.Query(ctx, "get_user", 42)
rows, err = conn.Query(ctx, "get_user", 43)

// Освобождение
conn.Conn().Deallocate(ctx, "get_user")
```

### Автоматические prepared statements в pgx

По умолчанию pgx автоматически готовит запросы. При первом выполнении запрос подготавливается, при повторных — используется кэш.

```go
// pgxpool.Config автоматически кэширует prepared statements
config, _ := pgxpool.ParseConfig(connString)

// Описанный режим (default) — pgx готовит запросы автоматически
// Каждый уникальный SQL-текст подготавливается один раз per connection

// Режимы подготовки:
// 1. pgx.QueryExecModeDescribeExec (default) — Prepare + Describe + Exec
// 2. pgx.QueryExecModeExec — без Prepare, параметры как текст
// 3. pgx.QueryExecModeSimpleProtocol — простой протокол, без prepared statements

// Для PgBouncer в transaction mode нужно отключить prepared statements:
config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
```

**Важно**: при использовании PgBouncer в transaction pooling mode prepared statements не работают (соединение переключается между транзакциями). Используйте `QueryExecModeExec` или `QueryExecModeSimpleProtocol`.

## Connection Pool Tuning

Подробнее о пуле соединений: [22-performance/03-io-networking.md](../22-performance/03-io-networking.md).

### Формула для MaxConns

```
MaxConns = CPU_cores * 2 + effective_spindle_count
```

Для SSD: `effective_spindle_count` обычно приравнивается к количеству ядер CPU. Для типичного сервера с 4 ядрами: `4 * 2 + 4 = 12` максимальных соединений.

**Важно**: больше соединений -- не значит быстрее. Слишком много соединений приводит к contention и деградации.

### Настройка pgxpool

```go
config, err := pgxpool.ParseConfig(connString)
if err != nil {
    log.Fatal(err)
}

// Основные параметры
config.MaxConns = 12                          // Максимум соединений
config.MinConns = 4                           // Минимум "тёплых" соединений
config.MaxConnLifetime = 1 * time.Hour        // Макс. время жизни соединения
config.MaxConnIdleTime = 30 * time.Minute     // Макс. время простоя
config.HealthCheckPeriod = 1 * time.Minute    // Проверка живости

// Для connection pool важно мониторить:
pool, err := pgxpool.NewWithConfig(ctx, config)

stat := pool.Stat()
// stat.TotalConns()       — всего соединений
// stat.IdleConns()        — простаивающих
// stat.AcquiredConns()    — занятых
// stat.AcquireCount()     — сколько раз запрашивали
// stat.AcquireDuration()  — суммарное время ожидания соединения
```

### Мониторинг пула в Go

```go
// Экспорт метрик пула (например, в Prometheus)
func collectPoolMetrics(pool *pgxpool.Pool) {
    stat := pool.Stat()

    poolTotalConns.Set(float64(stat.TotalConns()))
    poolIdleConns.Set(float64(stat.IdleConns()))
    poolAcquiredConns.Set(float64(stat.AcquiredConns()))
    poolAcquireCount.Add(float64(stat.AcquireCount()))
}
```

## Вопросы для собеседования

1. **Как реализовать soft delete и какие проблемы он создаёт?**
   Soft delete — пометка `deleted_at` вместо физического удаления. Проблемы: (1) все SELECT-ы должны фильтровать `WHERE deleted_at IS NULL` — легко забыть; (2) UNIQUE constraint нужен как partial index; (3) каскадное удаление требует триггеров; (4) таблица растёт бесконечно — нужна стратегия архивации.

2. **Что такое optimistic locking и когда его использовать?**
   Optimistic locking проверяет версию записи при обновлении: `WHERE version = $1`. Если кто-то обновил запись раньше, affected rows = 0 — конфликт. Использовать когда конфликты редки (read-heavy workload). Не подходит когда конфликты часты — лучше pessimistic locking (SELECT FOR UPDATE).

3. **Почему cursor pagination лучше offset?**
   Offset сканирует и отбрасывает N строк — O(offset + limit), деградирует на больших страницах. Cursor использует WHERE-условие с индексом — всегда O(limit). Дополнительно: offset нестабилен при вставке новых строк (дубликаты/пропуски), cursor стабилен.

4. **Как правильно настроить пул соединений PostgreSQL?**
   Формула: `CPU_cores * 2 + spindle_count`. Для SSD ~2-3x CPU cores. Больше соединений = больше contention (context switching, lock contention). Важно мониторить: время ожидания соединения, соотношение idle/acquired, количество отказов.

5. **Зачем нужен repository pattern в Go?**
   Repository инкапсулирует доступ к данным за интерфейсом. Позволяет: (1) тестировать бизнес-логику без базы (mock repository); (2) менять реализацию (pgx -> sqlc) без изменения сервисного слоя; (3) централизовать SQL-запросы. В Go интерфейс репозитория обычно определяется на стороне потребителя (dependency inversion).

6. **Что происходит с prepared statements при использовании PgBouncer?**
   В transaction pooling mode PgBouncer переключает серверное соединение между транзакциями. Prepared statement привязан к серверному соединению — после переключения он не существует. Решение: использовать `QueryExecModeExec` в pgx (отправлять параметры инлайн) или настроить PgBouncer в session mode.
