# PostgreSQL: Транзакции

## Начало транзакции

```go
// database/sql
tx, err := db.BeginTx(ctx, &sql.TxOptions{
    Isolation: sql.LevelSerializable,
    ReadOnly:  false,
})
if err != nil {
    return err
}
defer tx.Rollback() // безопасно вызвать после Commit — вернёт sql.ErrTxDone

// ... выполнение запросов внутри tx ...

if err := tx.Commit(); err != nil {
    return fmt.Errorf("commit: %w", err)
}

// pgx native
tx, err := pool.BeginTx(ctx, pgx.TxOptions{
    IsoLevel:   pgx.Serializable,
    AccessMode: pgx.ReadWrite,
})
if err != nil {
    return err
}
defer tx.Rollback(ctx)

_, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance - $1 WHERE id = $2", amount, fromID)
if err != nil {
    return err
}
_, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance + $1 WHERE id = $2", amount, toID)
if err != nil {
    return err
}

return tx.Commit(ctx)
```

## pgx.BeginTxFunc — автоматический commit/rollback

```go
// Если функция вернёт nil — Commit
// Если функция вернёт error — Rollback
err := pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
    _, err := tx.Exec(ctx,
        "UPDATE accounts SET balance = balance - $1 WHERE id = $2", amount, fromID)
    if err != nil {
        return err // автоматический Rollback
    }

    _, err = tx.Exec(ctx,
        "UPDATE accounts SET balance = balance + $1 WHERE id = $2", amount, toID)
    if err != nil {
        return err // автоматический Rollback
    }

    return nil // автоматический Commit
})
```

## Уровни изоляции PostgreSQL

```
| Уровень | Dirty Read | Non-Repeatable Read | Phantom Read | Serialization Anomaly |
|---|---|---|---|---|
| Read Committed | Нет | Возможно | Возможно | Возможно |
| Repeatable Read | Нет | Нет | Нет* | Возможно |
| Serializable | Нет | Нет | Нет | Нет |
```

*PostgreSQL реализует Repeatable Read через MVCC (snapshot isolation), что также предотвращает phantom reads, в отличие от стандарта SQL.

### Read Committed (default)

Каждый **запрос** внутри транзакции видит данные, зафиксированные до **начала этого запроса**.

```go
// Сценарий: параллельные обновления баланса
// Tx1: SELECT balance FROM accounts WHERE id = 1  → 100
// Tx2: UPDATE accounts SET balance = 50 WHERE id = 1; COMMIT;
// Tx1: SELECT balance FROM accounts WHERE id = 1  → 50 (видит коммит Tx2!)

// Проблема: non-repeatable read
// Два SELECT в одной транзакции возвращают разные значения

tx, _ := db.BeginTx(ctx, &sql.TxOptions{
    Isolation: sql.LevelReadCommitted, // default, можно не указывать
})
```

Когда использовать: подходит для большинства CRUD-операций, где не требуется консистентность между запросами внутри одной транзакции.

### Repeatable Read

Транзакция видит **snapshot** данных на момент **первого запроса** в транзакции. Все последующие запросы видят те же данные.

```go
// Сценарий:
// Tx1 (RR): SELECT balance FROM accounts WHERE id = 1  → 100
// Tx2:      UPDATE accounts SET balance = 50 WHERE id = 1; COMMIT;
// Tx1 (RR): SELECT balance FROM accounts WHERE id = 1  → 100 (snapshot!)

// Но при UPDATE возникает конфликт:
// Tx1 (RR): UPDATE accounts SET balance = balance + 10 WHERE id = 1
// ERROR: could not serialize access due to concurrent update

tx, _ := db.BeginTx(ctx, &sql.TxOptions{
    Isolation: sql.LevelRepeatableRead,
})
```

Когда использовать: отчёты, где все SELECT должны видеть консистентный snapshot; операции чтения нескольких таблиц, которые должны быть согласованы.

### Serializable

Полная изоляция — результат выполнения параллельных транзакций эквивалентен их последовательному выполнению. PostgreSQL использует SSI (Serializable Snapshot Isolation).

```go
// Сценарий: проверка инварианта "сумма балансов = 1000"
// Tx1: SELECT SUM(balance) FROM accounts → 1000
// Tx1: INSERT INTO audit (total) VALUES (1000)
// Tx2: UPDATE accounts SET balance = balance + 100 WHERE id = 1
// Tx2: UPDATE accounts SET balance = balance - 100 WHERE id = 2
// Tx2: COMMIT → OK
// Tx1: COMMIT → ERROR: could not serialize access

tx, _ := db.BeginTx(ctx, &sql.TxOptions{
    Isolation: sql.LevelSerializable,
})
```

Когда использовать: финансовые операции, бронирование, любые случаи, где нужна полная консистентность.

## Обработка serialization failures

При Repeatable Read и Serializable PostgreSQL может отклонить транзакцию с ошибкой `40001` (serialization_failure). Приложение **обязано** повторить всю транзакцию.

```go
import "github.com/jackc/pgx/v5/pgconn"

func ExecuteWithRetry(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
    const maxRetries = 5

    for attempt := 0; attempt < maxRetries; attempt++ {
        err := pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{
            IsoLevel: pgx.Serializable,
        }, fn)

        if err == nil {
            return nil
        }

        // Проверяем: это serialization failure?
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "40001" {
            // Экспоненциальный backoff
            backoff := time.Duration(1<<attempt) * 10 * time.Millisecond
            jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
            sleep := backoff + jitter

            select {
            case <-time.After(sleep):
                continue // повторяем всю транзакцию
            case <-ctx.Done():
                return ctx.Err()
            }
        }

        return err // другая ошибка — не повторяем
    }

    return fmt.Errorf("transaction failed after %d retries", maxRetries)
}

// Использование
err := ExecuteWithRetry(ctx, pool, func(tx pgx.Tx) error {
    var balance int
    err := tx.QueryRow(ctx,
        "SELECT balance FROM accounts WHERE id = $1", accountID,
    ).Scan(&balance)
    if err != nil {
        return err
    }

    if balance < amount {
        return fmt.Errorf("insufficient funds")
    }

    _, err = tx.Exec(ctx,
        "UPDATE accounts SET balance = balance - $1 WHERE id = $2", amount, accountID)
    return err
})
```

## SELECT FOR UPDATE / SELECT FOR SHARE

Пессимистичные блокировки — блокируют строки до конца транзакции.

```go
// SELECT FOR UPDATE — эксклюзивная блокировка
// Другие транзакции с FOR UPDATE на эти же строки будут ЖДАТЬ
tx, _ := pool.Begin(ctx)
defer tx.Rollback(ctx)

var balance int
err := tx.QueryRow(ctx,
    "SELECT balance FROM accounts WHERE id = $1 FOR UPDATE", accountID,
).Scan(&balance)
// Строка заблокирована! Другие FOR UPDATE ждут.

if balance < amount {
    return fmt.Errorf("insufficient funds")
}

_, err = tx.Exec(ctx,
    "UPDATE accounts SET balance = balance - $1 WHERE id = $2", amount, accountID)
tx.Commit(ctx) // блокировка снята

// SELECT FOR SHARE — разделяемая блокировка
// Другие FOR SHARE — ОК, но FOR UPDATE ждёт
tx.QueryRow(ctx, "SELECT * FROM products WHERE id = $1 FOR SHARE", productID)
// Гарантия: строка не будет изменена, пока транзакция активна
// Но другие транзакции могут её читать с FOR SHARE
```

### SELECT FOR UPDATE SKIP LOCKED — очередь задач

```go
// Паттерн: job queue на PostgreSQL
// Несколько воркеров берут задачи без блокировки друг друга
func pickJob(ctx context.Context, pool *pgxpool.Pool) (*Job, error) {
    tx, err := pool.Begin(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback(ctx)

    var job Job
    err = tx.QueryRow(ctx, `
        SELECT id, payload, created_at
        FROM jobs
        WHERE status = 'pending'
        ORDER BY created_at
        LIMIT 1
        FOR UPDATE SKIP LOCKED
    `).Scan(&job.ID, &job.Payload, &job.CreatedAt)

    if err == pgx.ErrNoRows {
        return nil, nil // нет задач
    }
    if err != nil {
        return nil, err
    }

    _, err = tx.Exec(ctx,
        "UPDATE jobs SET status = 'processing', worker_id = $1 WHERE id = $2",
        workerID, job.ID)
    if err != nil {
        return nil, err
    }

    return &job, tx.Commit(ctx)
}
```

`SKIP LOCKED` пропускает уже заблокированные строки вместо ожидания. Идеально для конкурентных воркеров.

## Savepoints

Savepoints позволяют откатить часть транзакции, не отменяя её целиком.

```go
tx, _ := pool.Begin(ctx)
defer tx.Rollback(ctx)

// Основная операция
_, err := tx.Exec(ctx, "INSERT INTO orders (user_id, total) VALUES ($1, $2)", userID, total)
if err != nil {
    return err
}

// Savepoint перед необязательной операцией
_, err = tx.Exec(ctx, "SAVEPOINT send_notification")

_, err = tx.Exec(ctx,
    "INSERT INTO notifications (user_id, message) VALUES ($1, $2)",
    userID, "Order created")
if err != nil {
    // Откат только уведомления, заказ сохраняется
    tx.Exec(ctx, "ROLLBACK TO SAVEPOINT send_notification")
    log.Warn("notification failed, continuing", "err", err)
}

return tx.Commit(ctx) // заказ будет создан даже если уведомление упало

// pgx также поддерживает вложенные транзакции через savepoints:
tx, _ := pool.Begin(ctx)
nestedTx, _ := tx.Begin(ctx) // автоматически создаёт SAVEPOINT
// ... операции ...
nestedTx.Rollback(ctx) // ROLLBACK TO SAVEPOINT
// tx всё ещё активна
tx.Commit(ctx)
```

## Deadlocks

Deadlock возникает, когда две транзакции ждут блокировки друг друга.

```go
// Tx1: UPDATE accounts SET balance = 100 WHERE id = 1  (блокирует строку 1)
// Tx2: UPDATE accounts SET balance = 200 WHERE id = 2  (блокирует строку 2)
// Tx1: UPDATE accounts SET balance = 300 WHERE id = 2  (ждёт строку 2 → ждёт Tx2)
// Tx2: UPDATE accounts SET balance = 400 WHERE id = 1  (ждёт строку 1 → ждёт Tx1)
// DEADLOCK! PostgreSQL обнаруживает и убивает одну из транзакций.
```

Предотвращение:

```go
// 1. Единый порядок блокировок — всегда блокировать строки в одном порядке
ids := []int{fromID, toID}
sort.Ints(ids) // всегда блокируем меньший ID первым

for _, id := range ids {
    tx.QueryRow(ctx, "SELECT balance FROM accounts WHERE id = $1 FOR UPDATE", id)
}

// 2. lock_timeout — не ждать бесконечно
tx.Exec(ctx, "SET LOCAL lock_timeout = '5s'")
// Если блокировка не получена за 5 сек — ошибка вместо зависания

// 3. statement_timeout — ограничение времени запроса
tx.Exec(ctx, "SET LOCAL statement_timeout = '30s'")
```

Обработка deadlock (error code `40P01`):

```go
var pgErr *pgconn.PgError
if errors.As(err, &pgErr) {
    switch pgErr.Code {
    case "40001": // serialization_failure
        // retry всей транзакции
    case "40P01": // deadlock_detected
        // retry всей транзакции
    case "55P03": // lock_not_available (lock_timeout)
        // retry или вернуть ошибку пользователю
    }
}
```

## Типичные ошибки

```go
// 1. Долгие транзакции — блокируют строки, мешают autovacuum
tx, _ := pool.Begin(ctx)
// ... HTTP-запрос к внешнему сервису (5+ секунд) ...
// ПЛОХО! Транзакция держит блокировки всё это время
tx.Commit(ctx)

// ПРАВИЛЬНО: делать IO вне транзакции
data, err := fetchExternalAPI(ctx) // сначала получить данные
tx, _ := pool.Begin(ctx)
tx.Exec(ctx, "INSERT INTO ...", data) // потом записать
tx.Commit(ctx) // транзакция минимальна

// 2. Забыли defer tx.Rollback() — при панике транзакция зависнет
tx, _ := pool.Begin(ctx)
// panic("oops") → транзакция не закрыта!

// 3. Использование соединения после Rollback/Commit
tx.Commit(ctx)
tx.Exec(ctx, "INSERT ...") // ОШИБКА: tx уже закрыта
```

---

## Вопросы на собеседовании

1. **Какие уровни изоляции поддерживает PostgreSQL? В чём разница?**
   Read Committed (default): каждый запрос видит последние committed данные. Repeatable Read: snapshot на момент первого запроса. Serializable: полная изоляция, эквивалент последовательного выполнения.

2. **Что такое serialization failure и как его обрабатывать?**
   Ошибка `40001` — PostgreSQL обнаружил конфликт при Repeatable Read / Serializable. Приложение обязано повторить всю транзакцию (не только последний запрос) с экспоненциальным backoff.

3. **Чем `SELECT FOR UPDATE` отличается от `SELECT FOR SHARE`?**
   `FOR UPDATE` — эксклюзивная блокировка: блокирует и чтение с `FOR UPDATE/SHARE`, и запись. `FOR SHARE` — разделяемая: блокирует только запись и `FOR UPDATE`, но несколько `FOR SHARE` совместимы.

4. **Как реализовать очередь задач на PostgreSQL?**
   `SELECT ... FOR UPDATE SKIP LOCKED LIMIT 1`. `SKIP LOCKED` пропускает строки, заблокированные другими транзакциями, вместо ожидания. Каждый воркер получает свою задачу без конкуренции.

5. **Как предотвратить deadlock?**
   Единый порядок блокировки строк (например, по возрастанию ID), `lock_timeout` для ограничения ожидания, минимальные транзакции (без внешних вызовов внутри).

6. **Зачем нужны savepoints?**
   Позволяют откатить часть транзакции без отмены всей. Полезны, когда часть операций необязательна (уведомления, логирование), и их ошибка не должна отменять основную операцию.

7. **Почему долгие транзакции — это проблема?**
   Держат блокировки (другие транзакции ждут), мешают autovacuum (растёт bloat таблиц), занимают соединение из пула, увеличивают вероятность deadlock.
