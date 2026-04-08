# PostgreSQL: Миграции

## Зачем нужны миграции

Миграции — это версионирование схемы базы данных. Аналог git для структуры таблиц.

```
Без миграций:
  "Добавь колонку email в таблицу users" — в Slack
  "А на staging добавили?" — "Не помню..."

С миграциями:
  migrations/
  ├── 001_create_users.sql
  ├── 002_add_email_to_users.sql
  └── 003_create_orders.sql

  Каждое окружение знает, какие миграции уже применены.
```

## goose

Один из самых популярных инструментов. Поддерживает SQL и Go миграции.

### Установка и CLI

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest

# Создать миграцию
goose -dir migrations create add_email_to_users sql
# → migrations/20240115120000_add_email_to_users.sql

# Применить все миграции
goose -dir migrations postgres "postgres://user:pass@localhost:5432/mydb?sslmode=disable" up

# Откатить последнюю
goose -dir migrations postgres "..." down

# Статус миграций
goose -dir migrations postgres "..." status
```

### SQL-миграции

```sql
-- migrations/20240115120000_add_email_to_users.sql

-- +goose Up
ALTER TABLE users ADD COLUMN email VARCHAR(255);
CREATE UNIQUE INDEX idx_users_email ON users (email);

-- +goose Down
DROP INDEX idx_users_email;
ALTER TABLE users DROP COLUMN email;
```

### Go-миграции

```go
// migrations/20240115130000_backfill_emails.go
package migrations

import (
    "context"
    "database/sql"

    "github.com/pressly/goose/v3"
)

func init() {
    goose.AddMigrationContext(upBackfillEmails, downBackfillEmails)
}

func upBackfillEmails(ctx context.Context, tx *sql.Tx) error {
    // Go-миграция для сложной логики (обращение к API, парсинг и т.д.)
    rows, err := tx.QueryContext(ctx,
        "SELECT id, username FROM users WHERE email IS NULL")
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var id int
        var username string
        if err := rows.Scan(&id, &username); err != nil {
            return err
        }
        email := username + "@legacy.example.com"
        _, err = tx.ExecContext(ctx,
            "UPDATE users SET email = $1 WHERE id = $2", email, id)
        if err != nil {
            return err
        }
    }
    return rows.Err()
}

func downBackfillEmails(ctx context.Context, tx *sql.Tx) error {
    _, err := tx.ExecContext(ctx,
        "UPDATE users SET email = NULL WHERE email LIKE '%@legacy.example.com'")
    return err
}
```

### Программный запуск с embed.FS

```go
package main

import (
    "database/sql"
    "embed"
    "log"

    "github.com/pressly/goose/v3"
    _ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {
    db, err := sql.Open("pgx", "postgres://user:pass@localhost:5432/mydb")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    goose.SetBaseFS(embedMigrations)

    if err := goose.SetDialect("postgres"); err != nil {
        log.Fatal(err)
    }

    if err := goose.Up(db, "migrations"); err != nil {
        log.Fatal(err)
    }

    log.Println("migrations applied successfully")
}
```

## golang-migrate

Альтернативный инструмент с более широким набором источников и драйверов.

### Установка и CLI

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Создать миграцию (создаёт пару up/down файлов)
migrate create -ext sql -dir migrations -seq add_email_to_users
# → migrations/000002_add_email_to_users.up.sql
# → migrations/000002_add_email_to_users.down.sql

# Применить
migrate -path migrations -database "postgres://user:pass@localhost:5432/mydb?sslmode=disable" up

# Откатить последнюю
migrate -path migrations -database "..." down 1

# Принудительно установить версию (после ручного исправления)
migrate -path migrations -database "..." force 2
```

### SQL-миграции

```sql
-- migrations/000002_add_email_to_users.up.sql
ALTER TABLE users ADD COLUMN email VARCHAR(255);
CREATE UNIQUE INDEX idx_users_email ON users (email);

-- migrations/000002_add_email_to_users.down.sql
DROP INDEX idx_users_email;
ALTER TABLE users DROP COLUMN email;
```

### Программный запуск с embed.FS

```go
package main

import (
    "embed"
    "log"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    "github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var fs embed.FS

func main() {
    source, err := iofs.New(fs, "migrations")
    if err != nil {
        log.Fatal(err)
    }

    m, err := migrate.NewWithSourceInstance("iofs", source,
        "postgres://user:pass@localhost:5432/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        log.Fatal(err)
    }

    log.Println("migrations applied successfully")
}
```

## Сравнение: goose vs golang-migrate

```
| | goose | golang-migrate |
|---|---|---|
| Go-миграции | Да (полноценные) | Нет (только SQL) |
| Формат файлов | timestamp_name.sql | seq_name.up/down.sql |
| embed.FS | Да | Да (через iofs) |
| Источники | Filesystem, embed | Filesystem, embed, S3, GitHub |
| БД | PG, MySQL, SQLite, и др. | PG, MySQL, SQLite, Mongo, и др. |
| Транзакции | Каждая миграция в tx | Нет (вручную) |
| CLI | Встроенный | Отдельный бинарь |
| Рекомендация | Go-миграции, простота | Только SQL, много источников |
```

## Embedding миграций

```go
// Встраивание SQL-файлов в бинарь — миграции едут вместе с приложением

//go:embed migrations/*.sql
var migrations embed.FS

// Преимущества:
// - Один бинарь, не нужно копировать файлы миграций
// - Атомарный деплой: бинарь + миграции всегда в sync
// - Нет проблем с путями в контейнерах

// Типичный паттерн запуска
func runMigrations(db *sql.DB) error {
    goose.SetBaseFS(migrations)
    if err := goose.SetDialect("postgres"); err != nil {
        return err
    }
    return goose.Up(db, "migrations")
}

func main() {
    db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))

    if err := runMigrations(db); err != nil {
        log.Fatal("migrations failed:", err)
    }

    // ... запуск сервера ...
}
```

## Паттерны миграций

### Idempotent миграции

```sql
-- Безопасно запускать повторно
-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at);

-- Добавление колонки с проверкой
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'email'
    ) THEN
        ALTER TABLE users ADD COLUMN email VARCHAR(255);
    END IF;
END $$;
```

### Data migrations

```sql
-- +goose Up
-- Шаг 1: Добавить колонку
ALTER TABLE users ADD COLUMN full_name VARCHAR(255);

-- Шаг 2: Заполнить данными
UPDATE users SET full_name = first_name || ' ' || last_name;

-- Шаг 3: Сделать NOT NULL после заполнения
ALTER TABLE users ALTER COLUMN full_name SET NOT NULL;

-- +goose Down
ALTER TABLE users DROP COLUMN full_name;
```

## Zero-downtime миграции (expand/contract)

Проблема: изменение схемы БД во время работы приложения. Старые инстансы могут не понимать новую схему.

### Expand/Contract паттерн

```
Фаза 1 (Expand) — добавить новое, не ломая старое:
├── Добавить новый столбец (nullable или с default)
├── Начать писать в оба столбца (старый и новый)
├── Backfill: заполнить новый столбец из старого
└── Деплой нового кода, который читает из нового столбца

Фаза 2 (Contract) — убрать старое:
├── Убедиться, что все инстансы используют новый столбец
├── Удалить старый столбец
└── Убрать код, который писал в старый столбец
```

Пример: переименование `name` в `full_name`.

```sql
-- Миграция 1 (Expand): добавить новый столбец
-- +goose Up
ALTER TABLE users ADD COLUMN full_name VARCHAR(255);
-- Backfill
UPDATE users SET full_name = name WHERE full_name IS NULL;

-- Код v2: пишет в оба (name И full_name), читает из full_name
-- Деплой v2...
-- Ждём пока все инстансы v1 остановлены...

-- Миграция 2 (Contract): удалить старый столбец
-- +goose Up
ALTER TABLE users DROP COLUMN name;
```

### Опасные операции

```sql
-- ОПАСНО: до PostgreSQL 11, ADD COLUMN ... DEFAULT блокирует таблицу
-- PostgreSQL 11+: безопасно (default хранится в metadata)
ALTER TABLE users ADD COLUMN status VARCHAR(20) DEFAULT 'active';

-- ОПАСНО: CREATE INDEX блокирует запись в таблицу
CREATE INDEX idx_users_email ON users (email);

-- БЕЗОПАСНО: CONCURRENTLY не блокирует запись (но медленнее)
CREATE INDEX CONCURRENTLY idx_users_email ON users (email);
-- ВАЖНО: CONCURRENTLY нельзя в транзакции!

-- ОПАСНО: NOT NULL на существующей колонке сканирует всю таблицу
ALTER TABLE users ALTER COLUMN email SET NOT NULL;

-- БЕЗОПАСНО: CHECK constraint с NOT VALID
ALTER TABLE users ADD CONSTRAINT users_email_not_null
    CHECK (email IS NOT NULL) NOT VALID;
-- Затем в отдельной миграции (валидация не блокирует запись):
ALTER TABLE users VALIDATE CONSTRAINT users_email_not_null;

-- ОПАСНО: изменение типа колонки перезаписывает всю таблицу
ALTER TABLE users ALTER COLUMN id TYPE BIGINT;
-- Для больших таблиц: создать новую колонку, backfill, переключить
```

## Запуск миграций в CI/CD

```yaml
# Пример для GitHub Actions
- name: Run migrations
  run: |
    goose -dir migrations postgres "$DATABASE_URL" up
  env:
    DATABASE_URL: ${{ secrets.DATABASE_URL }}

# Или через приложение (embed.FS)
- name: Deploy
  run: |
    ./myapp --migrate-and-exit  # запустить миграции и выйти
    ./myapp serve               # запустить сервер
```

```go
// Флаг для запуска миграций отдельно
func main() {
    migrateOnly := flag.Bool("migrate-and-exit", false, "run migrations and exit")
    flag.Parse()

    db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))

    if err := runMigrations(db); err != nil {
        log.Fatal("migrations failed:", err)
    }

    if *migrateOnly {
        log.Println("migrations complete, exiting")
        return
    }

    // ... запуск сервера ...
}
```

## Rollback стратегии

```
Стратегия 1: Down-миграции
├── Каждая миграция имеет up и down
├── goose down / migrate down 1
├── Минус: down-миграции часто не тестируются и могут потерять данные
└── Подходит для development

Стратегия 2: Forward-only (рекомендуется для production)
├── Не откатывать, а писать новую миграцию, исправляющую проблему
├── "Откат" — это новый деплой с fix-миграцией
├── Плюс: не теряем данные, история изменений линейна
└── Подходит для production

Стратегия 3: Версионирование с feature flags
├── Новая схема за feature flag
├── Если проблема — выключить flag, написать fix
├── Деплой с fix, включить flag обратно
└── Подходит для крупных изменений
```

```go
// Forward-only: вместо отката пишем новую миграцию
// migrations/003_add_status.sql — добавили колонку status VARCHAR(10)
// Проблема: 10 символов мало!
// Не откатываем 003, а пишем 004:

// migrations/004_fix_status_length.sql
// +goose Up
ALTER TABLE users ALTER COLUMN status TYPE VARCHAR(50);

// +goose Down
ALTER TABLE users ALTER COLUMN status TYPE VARCHAR(10);
```

---

## Вопросы на собеседовании

1. **Зачем нужны миграции, если можно менять схему вручную?**
   Миграции обеспечивают версионирование схемы, воспроизводимость на всех окружениях (dev, staging, prod), откат изменений, ревью через pull request и атомарный деплой вместе с кодом.

2. **Как выполнить миграцию без downtime?**
   Expand/contract паттерн: сначала добавить новую структуру (не ломая старый код), мигрировать данные, переключить код на новую структуру, затем удалить старую. Для индексов использовать `CREATE INDEX CONCURRENTLY`.

3. **Чем goose отличается от golang-migrate?**
   goose поддерживает Go-миграции (сложная логика), оборачивает каждую миграцию в транзакцию. golang-migrate — только SQL, но поддерживает больше источников (S3, GitHub). Для большинства проектов оба подходят.

4. **Что такое `embed.FS` и зачем встраивать миграции?**
   `embed.FS` встраивает SQL-файлы в Go-бинарь. Один бинарь содержит и код, и миграции — не нужно копировать файлы в контейнер, бинарь и миграции всегда синхронизированы.

5. **Почему `CREATE INDEX` может быть опасен на production?**
   Обычный `CREATE INDEX` берёт `ShareLock` на таблицу, блокируя INSERT/UPDATE/DELETE. Для больших таблиц это могут быть минуты. Решение: `CREATE INDEX CONCURRENTLY` — не блокирует запись, но работает медленнее и не может быть в транзакции.

6. **Какую стратегию rollback использовать в production?**
   Forward-only: вместо отката пишем новую миграцию, исправляющую проблему. Down-миграции часто не тестируются, могут потерять данные и не учитывают изменения, произошедшие между версиями.
