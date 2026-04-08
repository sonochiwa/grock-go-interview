# PostgreSQL: Запросы и сканирование

## QueryRow / Query / Exec

```go
// QueryRow — один результат (SELECT ... WHERE id = $1)
// Возвращает *sql.Row, ошибка проверяется в Scan
var name string
err := pool.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", 42).Scan(&name)
if err == pgx.ErrNoRows {
    // не найдено — это НЕ ошибка сервера
}

// Query — несколько строк (SELECT ... WHERE active = true)
// Возвращает pgx.Rows, ОБЯЗАТЕЛЬНО закрыть
rows, err := pool.Query(ctx, "SELECT id, name, email FROM users WHERE active = $1", true)

// Exec — без результата (INSERT/UPDATE/DELETE)
// Возвращает количество затронутых строк
tag, err := pool.Exec(ctx, "DELETE FROM sessions WHERE expired_at < $1", time.Now())
fmt.Println(tag.RowsAffected()) // например: 15
```

## Scan — построчное сканирование

```go
rows, err := pool.Query(ctx, "SELECT id, name, email, age FROM users")
if err != nil {
    return fmt.Errorf("query users: %w", err)
}
defer rows.Close() // КРИТИЧЕСКИ ВАЖНО

var users []User
for rows.Next() {
    var u User
    err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Age)
    if err != nil {
        return fmt.Errorf("scan user: %w", err)
    }
    users = append(users, u)
}

// ОБЯЗАТЕЛЬНО проверить ошибки после итерации
if err := rows.Err(); err != nil {
    return fmt.Errorf("rows iteration: %w", err)
}
```

## Критический паттерн: rows.Close()

```go
// ПРОБЛЕМА: забыли rows.Close()
func getUsers(pool *pgxpool.Pool) ([]User, error) {
    rows, err := pool.Query(ctx, "SELECT * FROM users")
    if err != nil {
        return nil, err
    }
    // rows.Close() не вызван!
    // Соединение НЕ вернётся в пул
    // После MaxConns таких утечек — пул исчерпан, все запросы висят

    var users []User
    for rows.Next() {
        // ...
    }
    return users, nil
}

// ПРАВИЛЬНО: defer rows.Close() сразу после проверки ошибки
func getUsers(pool *pgxpool.Pool) ([]User, error) {
    rows, err := pool.Query(ctx, "SELECT * FROM users")
    if err != nil {
        return nil, err
    }
    defer rows.Close() // ВСЕГДА сразу после Query

    var users []User
    for rows.Next() {
        var u User
        if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
            return nil, err // rows.Close() вызовется через defer
        }
        users = append(users, u)
    }
    return users, rows.Err()
}
```

Что происходит при утечке `rows`:
1. Соединение не возвращается в пул
2. Пул создаёт новые соединения (до `MaxConns`)
3. После исчерпания пула `Query` блокируется (ждёт свободное соединение)
4. С контекстом с таймаутом: `context deadline exceeded`
5. Без таймаута: горутина зависает навсегда (goroutine leak)

## rows.Err() — проверка ошибок

```go
// rows.Next() возвращает false по двум причинам:
// 1. Строки закончились (нормально)
// 2. Произошла сетевая ошибка (плохо)
// Различить можно только через rows.Err()

for rows.Next() {
    // ...
}
// БЕЗ этой проверки сетевая ошибка будет молча проглочена
if err := rows.Err(); err != nil {
    return fmt.Errorf("iteration error: %w", err)
}
```

## Nullable-значения

### sql.NullString / sql.NullInt64

```go
// database/sql подход — специальные типы
var name sql.NullString
var age sql.NullInt64

err := db.QueryRowContext(ctx,
    "SELECT name, age FROM users WHERE id = $1", id,
).Scan(&name, &age)

if name.Valid {
    fmt.Println(name.String) // есть значение
} else {
    fmt.Println("NULL")
}
```

### Указатели (*string, *int64)

```go
// Подход с указателями — чище, но нужно проверять nil
var name *string
var age *int64

err := db.QueryRowContext(ctx,
    "SELECT name, age FROM users WHERE id = $1", id,
).Scan(&name, &age)

if name != nil {
    fmt.Println(*name)
}
```

### pgx native — pgtype

```go
// pgx нативный подход — типы с полным контролем
import "github.com/jackc/pgx/v5/pgtype"

var name pgtype.Text
var age pgtype.Int4

err := pool.QueryRow(ctx,
    "SELECT name, age FROM users WHERE id = $1", id,
).Scan(&name, &age)

if name.Valid {
    fmt.Println(name.String)
}
```

## Struct scanning с pgx

```go
type User struct {
    ID    int64  `db:"id"`
    Name  string `db:"name"`
    Email string `db:"email"`
    Age   int    `db:"age"`
}

// pgx.RowToStructByName — автоматический маппинг по имени колонки
rows, err := pool.Query(ctx, "SELECT id, name, email, age FROM users")
if err != nil {
    return nil, err
}
defer rows.Close()

// CollectRows + RowToStructByName — собрать все строки в слайс
users, err := pgx.CollectRows(rows, pgx.RowToStructByName[User])
if err != nil {
    return nil, fmt.Errorf("collect users: %w", err)
}

// Одна строка — RowToStructByName
row, err := pool.Query(ctx, "SELECT id, name, email, age FROM users WHERE id = $1", 42)
user, err := pgx.CollectOneRow(row, pgx.RowToStructByName[User])

// RowToAddrOfStructByName — возвращает указатель *User
users, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[User])
// users имеет тип []*User
```

## Struct scanning с sqlx

```go
import "github.com/jmoiron/sqlx"

db, err := sqlx.Connect("pgx", connString)

// Select — запрос в слайс структур
var users []User
err = db.SelectContext(ctx, &users,
    "SELECT id, name, email, age FROM users WHERE active = $1", true)

// Get — запрос в одну структуру
var user User
err = db.GetContext(ctx, &user,
    "SELECT id, name, email, age FROM users WHERE id = $1", 42)
if err == sql.ErrNoRows {
    // не найдено
}

// StructScan — ручное сканирование
rows, err := db.QueryxContext(ctx, "SELECT * FROM users")
defer rows.Close()
for rows.Next() {
    var u User
    err := rows.StructScan(&u)
    // ...
}
```

## Batch queries (pgx)

`pgx.Batch` отправляет несколько запросов за один round-trip к серверу. Это значительно уменьшает latency при выполнении множества запросов.

```go
batch := &pgx.Batch{}

// Добавляем запросы в batch
batch.Queue("SELECT name FROM users WHERE id = $1", 1)
batch.Queue("SELECT name FROM users WHERE id = $1", 2)
batch.Queue("INSERT INTO logs (message) VALUES ($1)", "batch executed")

// Отправляем все запросы за один round-trip
br := pool.SendBatch(ctx, batch)
defer br.Close() // ОБЯЗАТЕЛЬНО закрыть

// Читаем результаты в порядке добавления
var name1 string
err := br.QueryRow().Scan(&name1)

var name2 string
err = br.QueryRow().Scan(&name2)

// Для Exec (INSERT/UPDATE/DELETE)
tag, err := br.Exec()
fmt.Println(tag.RowsAffected())
```

Применение batch:
- Загрузка связанных данных (user + roles + permissions)
- Массовые проверки (EXISTS для списка ID)
- Сокращение round-trips при высоком latency до БД

## Named parameters и prepared statements

```go
// pgx — позиционные параметры ($1, $2, ...)
pool.QueryRow(ctx, "SELECT * FROM users WHERE name = $1 AND age > $2", "Alice", 30)

// sqlx — named параметры
db.NamedExecContext(ctx,
    "INSERT INTO users (name, email) VALUES (:name, :email)",
    User{Name: "Alice", Email: "alice@example.com"},
)

// sqlx — named query из map
db.NamedQueryContext(ctx,
    "SELECT * FROM users WHERE name = :name",
    map[string]interface{}{"name": "Alice"},
)
```

### Prepared statements

```go
// database/sql — prepared statement кешируется на соединении
stmt, err := db.PrepareContext(ctx, "SELECT name FROM users WHERE id = $1")
defer stmt.Close()

// Переиспользуем для разных параметров
var name string
err = stmt.QueryRowContext(ctx, 1).Scan(&name)
err = stmt.QueryRowContext(ctx, 2).Scan(&name)

// pgx — автоматический кеш prepared statements
// По умолчанию pgx кеширует prepared statements на каждом соединении.
// Первый вызов: Parse + Bind + Execute
// Последующие: Bind + Execute (быстрее)
pool.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", 42)
```

## Типичные ошибки

```go
// 1. Забыли rows.Close() — утечка соединений
rows, _ := pool.Query(ctx, "SELECT * FROM users")
for rows.Next() { /* ... */ }
// rows.Close() не вызван!

// 2. Не проверили rows.Err() — пропущена сетевая ошибка
rows, _ := pool.Query(ctx, "SELECT * FROM users")
defer rows.Close()
for rows.Next() { /* ... */ }
// rows.Err() не проверен!

// 3. Scan в неправильный тип — runtime panic/error
var age string // колонка age — integer!
rows.Scan(&age) // ошибка: cannot scan int into *string

// 4. SQL-инъекция через конкатенацию строк
name := "Alice'; DROP TABLE users;--"
query := "SELECT * FROM users WHERE name = '" + name + "'" // ОПАСНО!

// ПРАВИЛЬНО: параметризованные запросы
pool.Query(ctx, "SELECT * FROM users WHERE name = $1", name) // безопасно

// 5. SELECT * в production — хрупкий код
// При добавлении колонки в таблицу Scan сломается
rows.Scan(&u.ID, &u.Name) // было 2 колонки, стало 3 — ошибка
// ПРАВИЛЬНО: явно перечислять колонки
pool.Query(ctx, "SELECT id, name FROM users")
```

---

## Вопросы на собеседовании

1. **Что произойдёт, если не вызвать `rows.Close()`?**
   Соединение не вернётся в пул. После исчерпания `MaxConns` все новые запросы будут блокироваться. С контекстом — `context deadline exceeded`, без — goroutine leak.

2. **Зачем проверять `rows.Err()` после цикла `rows.Next()`?**
   `rows.Next()` возвращает `false` и при штатном окончании данных, и при сетевой ошибке. Без `rows.Err()` ошибка будет молча проглочена, и функция вернёт неполные данные.

3. **Чем `sql.NullString` отличается от `*string`?**
   Оба решают проблему NULL. `sql.NullString` — явный тип с полем `Valid`, `*string` — указатель (`nil` = NULL). Указатели проще в коде, `sql.Null*` совместимы с JSON-маршалингом без кастомизации (хотя маршалят как `{"String":"...", "Valid":true}`).

4. **Что такое `pgx.Batch` и когда его использовать?**
   `pgx.Batch` группирует несколько запросов и отправляет их за один сетевой round-trip. Полезен при загрузке связанных данных или массовых операциях, особенно при высоком latency до БД.

5. **Почему конкатенация строк в SQL-запросах опасна?**
   Это открывает путь для SQL-инъекций. Пользовательский ввод может содержать SQL-код, который будет выполнен. Всегда использовать параметризованные запросы (`$1`, `$2`).

6. **В чём разница между `Query` и `Exec`?**
   `Query` возвращает `Rows` (для SELECT). `Exec` возвращает `CommandTag` с количеством затронутых строк (для INSERT/UPDATE/DELETE). Использование `Query` для INSERT без чтения результата приведёт к утечке соединения, если не вызвать `rows.Close()`.

7. **Как `pgx.CollectRows` упрощает код по сравнению с ручным `Scan`?**
   Убирает boilerplate: не нужен цикл `rows.Next()`, `defer rows.Close()`, проверка `rows.Err()`. Всё это делает `CollectRows` внутри, возвращая готовый слайс структур.
