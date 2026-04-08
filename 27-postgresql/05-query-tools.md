# Инструменты для работы с PostgreSQL в Go

## Обзор

В Go-экосистеме существует спектр инструментов для работы с базой данных: от чистого SQL до полноценных ORM. Каждый подход имеет свои trade-offs. Понимание этого спектра и умение выбрать правильный инструмент — важный навык для middle Go-разработчика.

## Спектр инструментов

```
Raw SQL → sqlx → Query Builder → Codegen → ORM
(полный контроль)                      (максимальная абстракция)
```

## Raw SQL: database/sql и pgx

### database/sql

Стандартная библиотека. Минимальная абстракция, полный контроль.

```go
import "database/sql"
import _ "github.com/lib/pq"

db, err := sql.Open("postgres", "postgres://user:pass@localhost/mydb?sslmode=disable")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Query — несколько строк
rows, err := db.QueryContext(ctx, "SELECT id, name, email FROM users WHERE active = $1", true)
if err != nil {
    return err
}
defer rows.Close()

var users []User
for rows.Next() {
    var u User
    if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
        return err
    }
    users = append(users, u)
}
if err := rows.Err(); err != nil {
    return err
}

// QueryRow — одна строка
var u User
err = db.QueryRowContext(ctx, "SELECT id, name FROM users WHERE id = $1", 42).
    Scan(&u.ID, &u.Name)
if err == sql.ErrNoRows {
    // не найдено
}

// Exec — INSERT/UPDATE/DELETE
result, err := db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", 42)
affected, _ := result.RowsAffected()
```

**Проблемы database/sql**: ручное сканирование каждого поля, легко перепутать порядок `Scan`, нет проверки на этапе компиляции.

### pgx — нативный драйвер PostgreSQL

```go
import "github.com/jackc/pgx/v5/pgxpool"

pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost/mydb")
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// pgx поддерживает нативные типы PostgreSQL
var id int
var tags []string // PostgreSQL array → Go slice
var meta map[string]any // JSONB → map

err = pool.QueryRow(ctx,
    "SELECT id, tags, meta FROM items WHERE id = $1", 1,
).Scan(&id, &tags, &meta)

// Batch — отправка нескольких запросов за один round-trip
batch := &pgx.Batch{}
batch.Queue("SELECT name FROM users WHERE id = $1", 1)
batch.Queue("SELECT name FROM users WHERE id = $1", 2)

br := pool.SendBatch(ctx, batch)
defer br.Close()

var name1, name2 string
br.QueryRow().Scan(&name1)
br.QueryRow().Scan(&name2)
```

## sqlx — расширение database/sql

`sqlx` добавляет удобства поверх `database/sql`, сохраняя совместимость.

```go
import "github.com/jmoiron/sqlx"

db, err := sqlx.Connect("postgres", "postgres://user:pass@localhost/mydb")
if err != nil {
    log.Fatal(err)
}

type User struct {
    ID    int    `db:"id"`
    Name  string `db:"name"`
    Email string `db:"email"`
}

// Select — сканирование в слайс структур
var users []User
err = db.SelectContext(ctx, &users, "SELECT * FROM users WHERE active = $1", true)

// Get — сканирование одной строки в структуру
var user User
err = db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", 42)

// NamedExec — именованные параметры
_, err = db.NamedExecContext(ctx,
    "INSERT INTO users (name, email) VALUES (:name, :email)",
    User{Name: "Alice", Email: "alice@example.com"},
)

// NamedQuery — именованные параметры в SELECT
rows, err := db.NamedQueryContext(ctx,
    "SELECT * FROM users WHERE name = :name",
    map[string]any{"name": "Alice"},
)

// In — для IN-запросов (подстановка слайса)
ids := []int{1, 2, 3}
query, args, err := sqlx.In("SELECT * FROM users WHERE id IN (?)", ids)
query = db.Rebind(query) // ? → $1, $2, $3 для PostgreSQL

var users []User
err = db.SelectContext(ctx, &users, query, args...)

// StructScan — ручное сканирование строк
rows, err := db.QueryxContext(ctx, "SELECT * FROM users")
for rows.Next() {
    var u User
    err := rows.StructScan(&u)
    // ...
}
```

## squirrel — Query Builder

Программное построение SQL-запросов. Полезен когда запрос строится динамически.

```go
import sq "github.com/Masterminds/squirrel"

// Используем PostgreSQL placeholder ($1, $2 вместо ?)
psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// SELECT
sql, args, err := psql.
    Select("id", "name", "email").
    From("users").
    Where(sq.Eq{"active": true}).
    Where(sq.Gt{"age": 18}).
    OrderBy("name ASC").
    Limit(10).
    ToSql()
// sql:  "SELECT id, name, email FROM users WHERE active = $1 AND age > $2 ORDER BY name ASC LIMIT 10"
// args: [true, 18]

// Динамическая фильтрация
func buildUserQuery(filters UserFilters) (string, []any, error) {
    q := psql.Select("*").From("users")

    if filters.Name != "" {
        q = q.Where(sq.Like{"name": "%" + filters.Name + "%"})
    }
    if filters.MinAge > 0 {
        q = q.Where(sq.GtOrEq{"age": filters.MinAge})
    }
    if filters.Active != nil {
        q = q.Where(sq.Eq{"active": *filters.Active})
    }

    return q.ToSql()
}

// INSERT
sql, args, err := psql.
    Insert("users").
    Columns("name", "email").
    Values("Alice", "alice@example.com").
    Values("Bob", "bob@example.com").
    Suffix("RETURNING id").
    ToSql()

// UPDATE
sql, args, err := psql.
    Update("users").
    Set("name", "Alice Updated").
    Set("updated_at", sq.Expr("NOW()")).
    Where(sq.Eq{"id": 1}).
    ToSql()
```

## sqlc — Codegen из SQL

`sqlc` генерирует типобезопасный Go-код из SQL-запросов. Ты пишешь SQL, а sqlc генерирует Go-структуры и методы.

### Конфигурация sqlc.yaml

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "query/"
    schema: "schema/"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
```

### SQL-схема (schema/001_users.sql)

```sql
CREATE TABLE users (
    id    BIGSERIAL PRIMARY KEY,
    name  TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### SQL-запросы (query/users.sql)

```sql
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: ListActiveUsers :many
SELECT * FROM users WHERE active = true ORDER BY name;

-- name: CreateUser :one
INSERT INTO users (name, email) VALUES ($1, $2) RETURNING *;

-- name: UpdateUserName :exec
UPDATE users SET name = $2 WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: ListUsersByIDs :many
SELECT * FROM users WHERE id = ANY($1::bigint[]);
```

### Сгенерированный код (internal/db/users.sql.go)

```go
// Code generated by sqlc. DO NOT EDIT.

type User struct {
    ID        int64     `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    Active    bool      `json:"active"`
    CreatedAt time.Time `json:"created_at"`
}

func (q *Queries) GetUser(ctx context.Context, id int64) (User, error) {
    row := q.db.QueryRow(ctx, getUserSQL, id)
    var i User
    err := row.Scan(&i.ID, &i.Name, &i.Email, &i.Active, &i.CreatedAt)
    return i, err
}

func (q *Queries) ListActiveUsers(ctx context.Context) ([]User, error) {
    rows, err := q.db.Query(ctx, listActiveUsersSQL)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var items []User
    for rows.Next() {
        var i User
        if err := rows.Scan(&i.ID, &i.Name, &i.Email, &i.Active, &i.CreatedAt); err != nil {
            return nil, err
        }
        items = append(items, i)
    }
    return items, nil
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
    // ...
}
```

### Использование

```go
pool, _ := pgxpool.New(ctx, connString)
queries := db.New(pool)

user, err := queries.GetUser(ctx, 42)
users, err := queries.ListActiveUsers(ctx)
newUser, err := queries.CreateUser(ctx, db.CreateUserParams{
    Name:  "Alice",
    Email: "alice@example.com",
})
```

### Плюсы sqlc

- **Compile-time safety**: ошибка в SQL — ошибка на этапе генерации, а не в рантайме
- **IDE support**: автодополнение, переход к определению
- **Нет рефлексии**: сканирование полей — прямой код, без reflect
- **Производительность**: сгенерированный код так же быстр, как ручной

### Минусы sqlc

- Поддерживает только PostgreSQL, MySQL, SQLite
- Сложные динамические запросы (фильтры) требуют workaround-ов (`sqlc.arg`, `CASE WHEN`)
- Необходимость перегенерации при изменении схемы или запросов
- Learning curve для сложных PostgreSQL-фич (CTE, оконные функции)

## GORM — ORM

Полноценная ORM для Go. Максимальная абстракция, минимум SQL.

```go
import "gorm.io/gorm"
import "gorm.io/driver/postgres"

db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

type User struct {
    ID        uint   `gorm:"primaryKey"`
    Name      string
    Email     string `gorm:"uniqueIndex"`
    Orders    []Order
    CreatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"` // soft delete
}

type Order struct {
    ID     uint
    UserID uint
    Total  float64
}

// Автомиграция
db.AutoMigrate(&User{}, &Order{})

// CRUD
db.Create(&User{Name: "Alice", Email: "alice@example.com"})

var user User
db.First(&user, 1)                           // по primary key
db.Where("email = ?", "alice@example.com").First(&user)

db.Model(&user).Update("name", "Alice Updated")

db.Delete(&user) // soft delete если есть DeletedAt

// Preload ассоциаций
var users []User
db.Preload("Orders").Find(&users)

// Hooks
func (u *User) BeforeCreate(tx *gorm.DB) error {
    u.Email = strings.ToLower(u.Email)
    return nil
}
```

### Плюсы GORM

- Быстрый старт, мало boilerplate
- Автомиграции для прототипирования
- Hooks (BeforeCreate, AfterUpdate и т.д.)
- Встроенные ассоциации, soft delete
- Большое комьюнити, много плагинов

### Минусы GORM

- **Скрытые запросы**: непонятно какой SQL генерируется
- **N+1 проблема**: без явного Preload каждый доступ к ассоциации = отдельный запрос
- **Сложная отладка**: ошибки часто всплывают в рантайме
- **Performance overhead**: рефлексия, аллокации
- **Ограниченные PostgreSQL-фичи**: CTE, оконные функции, JSONB — через Raw

### Anti-patterns

```go
// ПЛОХО: full table scan — загружает ВСЮ таблицу
var users []User
db.Find(&users) // SELECT * FROM users — без WHERE, без LIMIT!

// ПЛОХО: N+1 — каждый user = отдельный запрос для orders
var users []User
db.Find(&users)
for _, u := range users {
    db.Where("user_id = ?", u.ID).Find(&u.Orders) // N запросов!
}

// ХОРОШО: Preload решает N+1
db.Preload("Orders").Find(&users) // 2 запроса вместо N+1
```

## Сравнительная таблица

| Критерий | Raw SQL (pgx) | sqlx | squirrel | sqlc | GORM |
|----------|---------------|------|----------|------|------|
| Performance | Максимальный | Высокий | Высокий | Максимальный | Средний (рефлексия) |
| Type safety | Нет (рантайм) | Нет (рантайм) | Нет (рантайм) | Да (compile-time) | Нет (рантайм) |
| Learning curve | Низкий | Низкий | Средний | Средний | Средний |
| Flexibility | Полный SQL | Полный SQL | Ограничено API | Полный SQL | Ограничено ORM |
| PG features | Все | Все | Базовые | Почти все | Базовые + Raw |
| Динамические запросы | Ручная конкатенация | Ручная | Удобно | Сложно | Удобно |
| Boilerplate | Много | Средне | Средне | Мало | Мало |

## Когда что использовать

| Ситуация | Рекомендация |
|----------|-------------|
| Максимальный перфоманс, нативные PG-фичи | pgx |
| Много ручного SQL, хочется удобства | sqlx |
| Динамические фильтры, search endpoints | squirrel + sqlx |
| Стабильная схема, CRUD-сервисы | sqlc |
| Быстрый прототип, CRUD без сложных запросов | GORM |
| Микросервис с 5-10 запросами | sqlc или pgx |
| Легаси-проект с database/sql | sqlx (drop-in совместимость) |

## Комбинирование инструментов

В реальных проектах часто используют комбинацию:

```go
// sqlc для стандартных CRUD-запросов
user, err := queries.GetUser(ctx, id)

// squirrel для динамических фильтров
sql, args, _ := psql.Select("*").From("users").Where(filters).ToSql()

// pgx напрямую для bulk operations
_, err = pool.CopyFrom(ctx, pgx.Identifier{"users"}, columns, pgx.CopyFromRows(rows))
```

## Вопросы для собеседования

1. **Почему многие Go-команды избегают ORM?**
   Go-философия — явность и простота. ORM скрывает генерируемый SQL, затрудняет отладку, добавляет overhead через рефлексию, и плохо поддерживает продвинутые фичи PostgreSQL (CTE, оконные функции, JSONB). В Go-проектах предпочитают инструменты ближе к SQL: pgx, sqlx, sqlc.

2. **Что такое sqlc и почему он популярен?**
   sqlc — кодогенератор, который из SQL-запросов создаёт типобезопасный Go-код. Популярен потому что даёт compile-time проверку запросов, нет рефлексии в рантайме, IDE-поддержку и производительность на уровне ручного кода.

3. **В чём разница между sqlx и database/sql?**
   sqlx — надстройка над database/sql с обратной совместимостью. Добавляет: `Select`/`Get` для сканирования в структуры, `NamedExec` для именованных параметров, `In` для IN-запросов. Не меняет подход, а убирает boilerplate.

4. **Когда query builder лучше ORM?**
   Когда нужны динамические запросы (фильтры, сортировка) с полным контролем над SQL. Query builder генерирует предсказуемый SQL, не скрывает сложность, и не создаёт overhead рефлексии.

5. **Что такое N+1 проблема и как её решить?**
   N+1 — когда для загрузки связанных сущностей делается 1 запрос для основной коллекции и N запросов для каждого элемента. Решения: JOIN в SQL, batch-загрузка (WHERE id IN (...)), или Preload в ORM.
