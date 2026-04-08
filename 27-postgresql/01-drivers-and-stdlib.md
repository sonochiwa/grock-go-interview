# PostgreSQL: Драйверы и database/sql

## Архитектура database/sql

Стандартная библиотека Go предоставляет пакет `database/sql` — универсальный интерфейс для работы с реляционными БД. Ключевые компоненты:

```
database/sql (stdlib)
├── sql.DB          — пул соединений (НЕ одно соединение)
├── sql.Conn        — одно конкретное соединение из пула
├── sql.Tx          — транзакция
├── sql.Rows        — результат запроса (курсор)
├── sql.Row         — одна строка
└── driver.Driver   — интерфейс, который реализует драйвер
```

`sql.DB` — это **пул соединений**, а не одно соединение. Он потокобезопасен и должен создаваться один раз на приложение.

```go
// sql.DB управляет пулом автоматически:
// - открывает новые соединения по мере необходимости
// - переиспользует idle соединения
// - закрывает просроченные соединения
db, err := sql.Open("postgres", connString)

// Настройка пула
db.SetMaxOpenConns(25)              // макс. открытых соединений
db.SetMaxIdleConns(10)              // макс. idle соединений
db.SetConnMaxLifetime(5 * time.Minute) // макс. время жизни соединения
db.SetConnMaxIdleTime(1 * time.Minute) // макс. время простоя
```

## Сравнение драйверов

```
| | lib/pq | pgx (native) | pgx/stdlib |
|---|---|---|---|
| Статус | Maintenance mode | Активная разработка | Обёртка над pgx |
| Протокол | Через database/sql | Нативный PG протокол | Через database/sql |
| Производительность | Базовая | Лучшая | Средняя |
| COPY | Нет | Да | Нет |
| LISTEN/NOTIFY | Ограничено | Полная поддержка | Нет |
| Custom types | Нет | Да (pgtype) | Частично |
| Batch queries | Нет | Да | Нет |
| Prepared stmts | Автоматически | Контролируемо | Автоматически |
| Рекомендация | Не для новых проектов | PostgreSQL-only | Портируемость |
```

**lib/pq** — исторически первый драйвер. Сейчас в maintenance mode, авторы рекомендуют переходить на pgx.

**pgx** — современный драйвер с нативным PostgreSQL протоколом. Два режима работы:
- **Нативный**: `pgx.Conn`, `pgxpool.Pool` — максимальная производительность и функциональность
- **stdlib-совместимый**: `pgx/v5/stdlib` — через интерфейс `database/sql`, для портируемости

## Connection string

```go
// URL формат (рекомендуемый)
connString := "postgres://user:password@localhost:5432/mydb?sslmode=disable"

// DSN формат (keyword=value)
connString := "host=localhost port=5432 user=myuser password=secret dbname=mydb sslmode=disable"

// С дополнительными параметрами
connString := "postgres://user:pass@host:5432/db?sslmode=verify-full&sslrootcert=/path/ca.crt&connect_timeout=5"
```

## Подключение через pgxpool (рекомендуемый способ)

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    ctx := context.Background()

    // Конфигурация через URL + ParseConfig
    config, err := pgxpool.ParseConfig("postgres://user:pass@localhost:5432/mydb?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }

    // Настройка пула
    config.MaxConns = 25                       // максимум соединений (default: 4)
    config.MinConns = 5                        // минимум живых соединений
    config.MaxConnLifetime = 30 * time.Minute  // пересоздавать соединения каждые 30 мин
    config.MaxConnIdleTime = 5 * time.Minute   // закрывать idle через 5 мин
    config.HealthCheckPeriod = 1 * time.Minute // проверять здоровье каждую минуту

    // Callback после установки соединения
    config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
        // Регистрация custom types, установка search_path и т.д.
        _, err := conn.Exec(ctx, "SET search_path TO myschema, public")
        return err
    }

    pool, err := pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close() // graceful shutdown

    // Проверка подключения
    if err := pool.Ping(ctx); err != nil {
        log.Fatal("cannot connect to database:", err)
    }

    // Использование
    var greeting string
    err = pool.QueryRow(ctx, "SELECT 'Hello, PostgreSQL!'").Scan(&greeting)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(greeting)

    // Статистика пула
    stat := pool.Stat()
    fmt.Printf("total: %d, idle: %d, in-use: %d\n",
        stat.TotalConns(), stat.IdleConns(), stat.AcquiredConns())
}
```

## Подключение через database/sql + pgx/stdlib

Этот вариант нужен, если:
- Код должен работать с разными БД (PostgreSQL, MySQL, SQLite)
- Используются библиотеки, требующие `*sql.DB` (например, sqlx, goose)
- Нужен постепенный переход с lib/pq

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "time"

    _ "github.com/jackc/pgx/v5/stdlib" // регистрирует драйвер "pgx"
)

func main() {
    connString := "postgres://user:pass@localhost:5432/mydb?sslmode=disable"

    db, err := sql.Open("pgx", connString)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Настройка пула database/sql
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(10)
    db.SetConnMaxLifetime(5 * time.Minute)
    db.SetConnMaxIdleTime(1 * time.Minute)

    // Проверка подключения
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        log.Fatal("cannot connect:", err)
    }

    var version string
    err = db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(version)
}
```

## Когда что использовать

```
database/sql + pgx/stdlib
├── Портируемость между PostgreSQL / MySQL / SQLite
├── Используются sqlx, goose, gorm (требуют *sql.DB)
└── Миграция с lib/pq (drop-in replacement)

pgxpool (нативный pgx)
├── Проект работает только с PostgreSQL
├── Нужны PG-specific фичи:
│   ├── COPY (массовая загрузка данных)
│   ├── LISTEN/NOTIFY (pub/sub через БД)
│   ├── pgtype (PostGIS, hstore, composite types)
│   └── Batch (несколько запросов за один round-trip)
├── Максимальная производительность
└── Нужен контроль над prepared statements
```

## Graceful shutdown

```go
func main() {
    pool, err := pgxpool.New(ctx, connString)
    if err != nil {
        log.Fatal(err)
    }

    // Обработка сигналов завершения
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigCh
        log.Println("shutting down...")

        // pool.Close() ждёт завершения всех текущих запросов
        // и закрывает все соединения
        pool.Close()
    }()

    // ... запуск сервера ...
}
```

Для `database/sql`:

```go
// db.Close() закрывает все idle соединения
// Активные соединения закроются после завершения операций
defer db.Close()
```

## Типичные ошибки

```go
// ОШИБКА: создание нового пула на каждый запрос
func handler(w http.ResponseWriter, r *http.Request) {
    db, _ := sql.Open("pgx", connString) // ПЛОХО! Новый пул каждый раз
    defer db.Close()
    // ...
}

// ПРАВИЛЬНО: один пул на всё приложение
var db *sql.DB

func main() {
    db, _ = sql.Open("pgx", connString)
    defer db.Close()
    http.HandleFunc("/", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {
    db.QueryRowContext(r.Context(), "SELECT ...") // используем общий пул
}
```

```go
// ОШИБКА: не вызывать Ping после Open
db, _ := sql.Open("pgx", connString)
// sql.Open НЕ устанавливает соединение! Только валидирует DSN.
// Ошибка подключения обнаружится только при первом запросе.

// ПРАВИЛЬНО: проверить соединение сразу
db, _ := sql.Open("pgx", connString)
if err := db.PingContext(ctx); err != nil {
    log.Fatal("db unreachable:", err)
}
```

---

## Вопросы на собеседовании

1. **Чем `sql.DB` отличается от одного соединения?**
   `sql.DB` — это пул соединений. Он потокобезопасен, управляет жизненным циклом соединений, переиспользует idle-соединения и создаёт новые по мере необходимости.

2. **Почему `sql.Open` не возвращает ошибку подключения?**
   `sql.Open` только парсит DSN и инициализирует пул. Реальное соединение устанавливается лениво при первом запросе. Для проверки нужен `db.Ping()`.

3. **В чём разница между `lib/pq` и `pgx`?**
   `lib/pq` в maintenance mode, работает только через `database/sql`. `pgx` активно развивается, поддерживает нативный протокол PG, COPY, LISTEN/NOTIFY, batch queries и custom types.

4. **Когда использовать `database/sql`, а когда нативный `pgx`?**
   `database/sql` — для портируемости между БД и совместимости с библиотеками (sqlx, goose). Нативный `pgx` — когда проект привязан к PostgreSQL и нужны PG-specific фичи или максимальная производительность.

5. **Какие параметры пула соединений важно настраивать?**
   `MaxConns` (ограничивает нагрузку на БД), `MinConns` (уменьшает latency холодного старта), `MaxConnLifetime` (обновление соединений для балансировки после failover), `MaxConnIdleTime` (освобождение ресурсов).

6. **Что произойдёт, если не вызвать `pool.Close()` при завершении?**
   Соединения закроются принудительно ОС, но на стороне PostgreSQL останутся «мёртвые» backend-процессы до `tcp_keepalives_idle` таймаута. Это может исчерпать `max_connections`.
