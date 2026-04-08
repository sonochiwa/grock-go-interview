# MongoDB: Транзакции

## Когда нужны транзакции

MongoDB гарантирует атомарность на уровне **одного документа**. Если все связанные данные встроены в один документ (embedding), транзакции не нужны.

Транзакции нужны когда:
- Обновление нескольких документов должно быть атомарным
- Обновление нескольких коллекций должно быть атомарным
- Нужна консистентность чтения между коллекциями

```
Атомарность одного документа (ВСЕГДА):
├── InsertOne            — атомарно
├── UpdateOne            — атомарно
├── FindOneAndUpdate     — атомарно
└── Embedded subdocuments — обновление через $push/$pull атомарно

Multi-document транзакции (начиная с 4.0 для replica set, 4.2 для sharded):
├── Перевод денег между аккаунтами (два update)
├── Создание заказа + уменьшение остатков (две коллекции)
└── Любая операция, требующая all-or-nothing на нескольких документах
```

## Sessions

Все транзакции в MongoDB выполняются внутри **сессии**. Сессия отслеживает состояние транзакции на сервере.

```go
// Создание сессии
session, err := client.StartSession()
if err != nil {
    return fmt.Errorf("start session: %w", err)
}
defer session.EndSession(ctx)
```

## Базовая транзакция

### Ручное управление

```go
func transferMoney(ctx context.Context, client *mongo.Client, fromID, toID primitive.ObjectID, amount float64) error {
    session, err := client.StartSession()
    if err != nil {
        return err
    }
    defer session.EndSession(ctx)

    // Начало транзакции
    err = session.StartTransaction()
    if err != nil {
        return err
    }

    // Все операции внутри session context
    err = mongo.WithSession(ctx, session, func(sc context.Context) error {
        accounts := client.Database("bank").Collection("accounts")

        // Списание
        result, err := accounts.UpdateOne(sc,
            bson.M{"_id": fromID, "balance": bson.M{"$gte": amount}},
            bson.M{"$inc": bson.M{"balance": -amount}},
        )
        if err != nil {
            return err
        }
        if result.MatchedCount == 0 {
            return fmt.Errorf("insufficient funds or account not found")
        }

        // Зачисление
        _, err = accounts.UpdateOne(sc,
            bson.M{"_id": toID},
            bson.M{"$inc": bson.M{"balance": amount}},
        )
        if err != nil {
            return err
        }

        // Запись в лог транзакций
        txLog := client.Database("bank").Collection("transactions")
        _, err = txLog.InsertOne(sc, bson.M{
            "from":       fromID,
            "to":         toID,
            "amount":     amount,
            "created_at": time.Now(),
        })
        return err
    })

    if err != nil {
        // Rollback при ошибке
        _ = session.AbortTransaction(ctx)
        return fmt.Errorf("transaction failed: %w", err)
    }

    // Commit
    return session.CommitTransaction(ctx)
}
```

### Callback API (рекомендуемый способ)

Callback API автоматически обрабатывает commit, abort и retry при transient errors:

```go
func transferMoney(ctx context.Context, client *mongo.Client, fromID, toID primitive.ObjectID, amount float64) error {
    session, err := client.StartSession()
    if err != nil {
        return err
    }
    defer session.EndSession(ctx)

    _, err = session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
        accounts := client.Database("bank").Collection("accounts")

        // Списание
        result, err := accounts.UpdateOne(sc,
            bson.M{"_id": fromID, "balance": bson.M{"$gte": amount}},
            bson.M{"$inc": bson.M{"balance": -amount}},
        )
        if err != nil {
            return nil, err
        }
        if result.MatchedCount == 0 {
            return nil, fmt.Errorf("insufficient funds")
        }

        // Зачисление
        _, err = accounts.UpdateOne(sc,
            bson.M{"_id": toID},
            bson.M{"$inc": bson.M{"balance": amount}},
        )
        if err != nil {
            return nil, err
        }

        return nil, nil
    })

    return err
}
```

`WithTransaction` автоматически:
1. Вызывает `StartTransaction`
2. Выполняет callback
3. При успехе — `CommitTransaction`
4. При ошибке — `AbortTransaction`
5. При transient error — **повторяет всю транзакцию**
6. При unknown commit result — **повторяет commit**

## Write Concern

Write concern определяет уровень гарантий подтверждения записи.

```
| Write Concern | Описание | Скорость | Надёжность |
|---------------|----------|----------|------------|
| w: 0 | Fire and forget — не ждать подтверждения | Максимальная | Нет гарантий |
| w: 1 | Подтверждение от primary (default) | Быстро | Потеря при failover |
| w: "majority" | Подтверждение от большинства replica set | Средне | Высокая |
| w: N | Подтверждение от N узлов | Зависит от N | Зависит от N |
| j: true | Запись в journal (WAL) на primary | Медленнее | Выживает restart |
```

```go
// Write concern на уровне клиента
opts := options.Client().
    ApplyURI(uri).
    SetWriteConcern(writeconcern.Majority())

// Write concern на уровне коллекции
wc := writeconcern.Majority()
coll := db.Collection("orders", options.Collection().SetWriteConcern(wc))

// Write concern для транзакции
txOpts := options.Transaction().
    SetWriteConcern(writeconcern.Majority())

session.WithTransaction(ctx, callback, txOpts)
```

### w: "majority" для транзакций

```
Replica Set: Primary + Secondary1 + Secondary2

w: 1 (default):
1. Client → Primary: INSERT
2. Primary → Client: OK (записано на primary)
3. Primary → Secondary1, Secondary2: replication (async)
Проблема: если Primary упал ДО репликации → данные потеряны

w: "majority":
1. Client → Primary: INSERT
2. Primary → Secondary1: replicate
3. Primary ← Secondary1: ACK
4. Primary → Client: OK (записано на 2 из 3 — majority)
Гарантия: данные выживут при failover
```

## Read Concern

Read concern определяет, какие данные видны при чтении.

```
| Read Concern | Описание | Для транзакций |
|--------------|----------|----------------|
| local | Текущие данные primary (может быть не реплицировано) | Default |
| available | Как local, но для sharded — может быть orphaned docs | Нет |
| majority | Только данные, подтверждённые majority | Да |
| snapshot | Snapshot isolation для транзакций | Да (default для transactions) |
| linearizable | Linearizable read — самые свежие committed данные | Нет |
```

```go
// Read concern на уровне клиента
opts := options.Client().
    ApplyURI(uri).
    SetReadConcern(readconcern.Majority())

// Read concern для транзакции
txOpts := options.Transaction().
    SetReadConcern(readconcern.Snapshot())
```

### snapshot для транзакций

```
Транзакция с read concern "snapshot":
1. При первом чтении фиксируется snapshot
2. Все последующие чтения в транзакции видят тот же snapshot
3. Гарантия: repeatable reads внутри транзакции
4. Write-write конфликты обнаруживаются при commit

Аналог Repeatable Read / Serializable в PostgreSQL
```

## Read Preference

Read preference определяет, с какого узла replica set читать данные.

```
| Read Preference | Описание | Когда использовать |
|-----------------|----------|--------------------|
| primary | Всегда с primary (default) | Актуальные данные |
| primaryPreferred | С primary, fallback на secondary | Актуальность + доступность |
| secondary | Только с secondary | Разгрузить primary |
| secondaryPreferred | С secondary, fallback на primary | Аналитика, отчёты |
| nearest | С ближайшего по latency | Минимальная задержка |
```

```go
import "go.mongodb.org/mongo-driver/v2/mongo/readpref"

// Read preference на уровне клиента
opts := options.Client().
    ApplyURI(uri).
    SetReadPreference(readpref.SecondaryPreferred())

// Read preference для конкретной коллекции
rp := readpref.SecondaryPreferred()
coll := db.Collection("analytics", options.Collection().SetReadPreference(rp))

// ВАЖНО: внутри транзакции read preference фиксируется на primary
// Нельзя читать с secondary внутри multi-document транзакции
```

## Causal Consistency

Гарантирует порядок операций: чтение после записи увидит свою запись, даже если читаем с secondary.

```go
// Session с causal consistency
sessionOpts := options.Session().
    SetCausalConsistency(true)

session, err := client.StartSession(sessionOpts)
if err != nil {
    return err
}
defer session.EndSession(ctx)

err = mongo.WithSession(ctx, session, func(sc context.Context) error {
    // Запись на primary
    _, err := coll.InsertOne(sc, bson.M{"name": "Alice"})
    if err != nil {
        return err
    }

    // Чтение с secondary — гарантированно увидит запись выше
    // (без causal consistency secondary мог отдать данные до репликации)
    rp := readpref.Secondary()
    secondaryColl := db.Collection("users", options.Collection().SetReadPreference(rp))
    var user bson.M
    return secondaryColl.FindOne(sc, bson.M{"name": "Alice"}).Decode(&user)
})
```

## Ограничения транзакций

```
1. Требуется replica set (или sharded cluster с 4.2+)
   Standalone MongoDB НЕ поддерживает транзакции

2. Максимальное время — 60 секунд (по умолчанию)
   transactionLifetimeLimitSeconds на сервере

3. Размер oplog entry — 16 MB
   Все операции транзакции должны поместиться в один oplog entry

4. Нельзя создавать/удалять коллекции внутри транзакции

5. Нельзя использовать capped collections

6. Read preference фиксируется на primary

7. Не рекомендуются для > 1000 документов
   Длинные транзакции блокируют ресурсы
```

## Обработка ошибок

### TransientTransactionError

Temporary network error или конфликт записи. Безопасно повторить **всю транзакцию**.

```go
func executeWithRetry(ctx context.Context, client *mongo.Client, fn func(ctx context.Context) error) error {
    session, err := client.StartSession()
    if err != nil {
        return err
    }
    defer session.EndSession(ctx)

    for {
        _, err = session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
            return nil, fn(sc)
        })

        if err == nil {
            return nil
        }

        // WithTransaction уже обрабатывает TransientTransactionError
        // и UnknownTransactionCommitResult автоматически
        // Если ошибка дошла сюда — это permanent error
        return err
    }
}
```

### UnknownTransactionCommitResult

Commit отправлен, но результат неизвестен (сетевая ошибка). Безопасно повторить **commit**.

```go
// WithTransaction обрабатывает это автоматически
// При ручном управлении:
err := session.CommitTransaction(ctx)
if err != nil {
    cmdErr, ok := err.(mongo.CommandError)
    if ok && cmdErr.HasErrorLabel("UnknownTransactionCommitResult") {
        // Retry commit
        err = session.CommitTransaction(ctx)
    }
}
```

### Write Conflict

Два клиента пытаются обновить один документ в транзакции. Один из них получит ошибку.

```go
// WithTransaction автоматически повторяет при write conflict
// При ручном управлении нужно повторять всю транзакцию
```

## Паттерн: создание заказа

```go
func (s *OrderService) CreateOrder(ctx context.Context, userID primitive.ObjectID, items []OrderItem) (*Order, error) {
    session, err := s.client.StartSession()
    if err != nil {
        return nil, err
    }
    defer session.EndSession(ctx)

    var order *Order

    _, err = session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
        products := s.client.Database("shop").Collection("products")
        orders := s.client.Database("shop").Collection("orders")

        // 1. Проверить и уменьшить остатки для каждого товара
        var total float64
        for _, item := range items {
            result, err := products.UpdateOne(sc,
                bson.M{
                    "_id":   item.ProductID,
                    "stock": bson.M{"$gte": item.Quantity},
                },
                bson.M{"$inc": bson.M{"stock": -item.Quantity}},
            )
            if err != nil {
                return nil, fmt.Errorf("update stock: %w", err)
            }
            if result.MatchedCount == 0 {
                return nil, fmt.Errorf("product %s: out of stock", item.ProductID.Hex())
            }
            total += item.Price * float64(item.Quantity)
        }

        // 2. Создать заказ
        order = &Order{
            UserID:    userID,
            Items:     items,
            Total:     total,
            Status:    "created",
            CreatedAt: time.Now(),
        }

        result, err := orders.InsertOne(sc, order)
        if err != nil {
            return nil, fmt.Errorf("insert order: %w", err)
        }
        order.ID = result.InsertedID.(primitive.ObjectID)

        return nil, nil
    })

    if err != nil {
        return nil, err
    }
    return order, nil
}
```

## Типичные ошибки

```go
// 1. Транзакция на standalone MongoDB — не работает
// Нужен replica set (хотя бы single-node replica set для dev)
// Для dev: rs.initiate() в mongosh

// 2. Слишком длинные транзакции
// ПЛОХО: HTTP-запрос внутри транзакции
session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
    fetchExternalAPI(sc) // 5+ секунд — транзакция держит locks!
    coll.InsertOne(sc, doc)
    return nil, nil
})
// ПРАВИЛЬНО: внешние вызовы ВНЕ транзакции
data := fetchExternalAPI(ctx)
session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
    coll.InsertOne(sc, data)
    return nil, nil
})

// 3. Забыли EndSession — утечка ресурсов на сервере
session, _ := client.StartSession()
// defer session.EndSession(ctx) — ЗАБЫЛИ!

// 4. Используют обычный ctx вместо session context
session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
    coll.InsertOne(ctx, doc) // ПЛОХО! Нужен sc, не ctx
    coll.InsertOne(sc, doc)  // ПРАВИЛЬНО
    return nil, nil
})

// 5. Транзакция вместо атомарного обновления одного документа
// Если данные в одном документе — используй $set, $inc, $push
// Транзакции — overhead, используй только когда действительно нужно
```

---

## Вопросы на собеседовании

1. **Когда нужны multi-document транзакции в MongoDB?**
   Когда нужна атомарность операций над несколькими документами или коллекциями: перевод денег (два update), создание заказа + уменьшение остатков. Если данные в одном документе — транзакции не нужны, MongoDB гарантирует атомарность на уровне одного документа.

2. **Что такое write concern "majority" и зачем он нужен?**
   Запись подтверждается только когда данные реплицированы на большинство узлов replica set. Гарантирует, что данные не потеряются при failover primary. `w: 1` (default) подтверждает только запись на primary — при падении primary до репликации данные могут быть потеряны.

3. **Чем read concern "snapshot" отличается от "majority"?**
   `snapshot` фиксирует snapshot данных на момент начала транзакции — все чтения видят одни и те же данные (repeatable read). `majority` гарантирует, что данные подтверждены большинством, но два чтения внутри транзакции могут увидеть разные данные.

4. **Что такое causal consistency?**
   Гарантирует, что чтение после записи увидит результат записи, даже если читаем с secondary. Без causal consistency чтение с secondary может вернуть данные до репликации. Включается на уровне сессии.

5. **Что такое TransientTransactionError?**
   Временная ошибка (network error, write conflict). Вся транзакция может быть безопасно повторена. `WithTransaction` callback API обрабатывает это автоматически, повторяя транзакцию.

6. **Какие ограничения у транзакций MongoDB?**
   Требуется replica set (4.0+) или sharded cluster (4.2+). Максимум 60 секунд. Oplog entry до 16 MB. Нельзя создавать/удалять коллекции. Чтение только с primary. Не рекомендуются для тысяч документов.

7. **Почему `WithTransaction` лучше ручного управления?**
   `WithTransaction` автоматически обрабатывает `TransientTransactionError` (повтор транзакции) и `UnknownTransactionCommitResult` (повтор commit). При ручном управлении нужно писать retry-логику самостоятельно, что error-prone.
