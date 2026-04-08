# MongoDB: Драйвер и подключение

## Архитектура драйвера

Официальный Go-драйвер `go.mongodb.org/mongo-driver/v2` — единственный рекомендованный способ работы с MongoDB из Go. Ключевые компоненты:

```
go.mongodb.org/mongo-driver/v2
├── mongo.Client     — точка входа, управляет connection pool
├── mongo.Database   — ссылка на конкретную БД
├── mongo.Collection — ссылка на коллекцию (аналог таблицы)
├── bson             — сериализация/десериализация документов
├── options          — конфигурация клиента, операций
└── readpref         — настройка read preference для replica set
```

`mongo.Client` — это **пул соединений**, а не одно соединение. Он потокобезопасен и создаётся один раз на приложение.

## Установка

```bash
go get go.mongodb.org/mongo-driver/v2
```

## Подключение

### Базовое подключение

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // mongo.Connect создаёт клиент и устанавливает соединение
    client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        log.Fatal(err)
    }
    defer func() {
        if err := client.Disconnect(context.Background()); err != nil {
            log.Fatal(err)
        }
    }()

    // Ping проверяет реальное соединение с сервером
    if err := client.Ping(ctx, nil); err != nil {
        log.Fatal("cannot connect to MongoDB:", err)
    }

    fmt.Println("Connected to MongoDB")

    // Получение ссылки на БД и коллекцию
    db := client.Database("myapp")
    users := db.Collection("users")

    _ = users
}
```

## Connection String (URI)

### Формат URI

```
mongodb://[username:password@]host1[:port1][,...hostN[:portN]][/database][?options]
```

### Примеры

```go
// Локальный сервер
uri := "mongodb://localhost:27017"

// С авторизацией
uri := "mongodb://admin:secret@localhost:27017"

// Replica set
uri := "mongodb://host1:27017,host2:27017,host3:27017/?replicaSet=myrs"

// С указанием БД для авторизации
uri := "mongodb://admin:secret@localhost:27017/mydb?authSource=admin"

// MongoDB Atlas (SRV record)
uri := "mongodb+srv://user:pass@cluster0.abc123.mongodb.net/mydb?retryWrites=true&w=majority"

// С TLS
uri := "mongodb://localhost:27017/?tls=true&tlsCAFile=/path/to/ca.pem"
```

### Основные параметры URI

| Параметр | Описание | Пример |
|----------|----------|--------|
| `replicaSet` | Имя replica set | `replicaSet=myrs` |
| `authSource` | БД для авторизации | `authSource=admin` |
| `retryWrites` | Автоповтор записи при сетевых ошибках | `retryWrites=true` |
| `w` | Write concern | `w=majority` |
| `readPreference` | Предпочтение чтения | `readPreference=secondaryPreferred` |
| `maxPoolSize` | Макс. соединений в пуле | `maxPoolSize=100` |
| `minPoolSize` | Мин. соединений в пуле | `minPoolSize=5` |
| `maxIdleTimeMS` | Макс. время простоя соединения | `maxIdleTimeMS=60000` |
| `connectTimeoutMS` | Таймаут подключения | `connectTimeoutMS=5000` |
| `serverSelectionTimeoutMS` | Таймаут выбора сервера | `serverSelectionTimeoutMS=5000` |
| `compressors` | Сжатие трафика | `compressors=zstd,snappy` |

## Конфигурация через options

```go
clientOpts := options.Client().
    ApplyURI("mongodb://localhost:27017").
    SetMaxPoolSize(50).                              // default: 100
    SetMinPoolSize(5).                               // default: 0
    SetMaxConnIdleTime(5 * time.Minute).             // default: 0 (no limit)
    SetConnectTimeout(5 * time.Second).              // default: 30s
    SetServerSelectionTimeout(5 * time.Second).      // default: 30s
    SetRetryWrites(true).                            // default: true
    SetRetryReads(true).                             // default: true
    SetCompressors([]string{"zstd", "snappy"}).      // compression
    SetAppName("my-go-service")                      // visible in db.currentOp()

client, err := mongo.Connect(clientOpts)
```

## Connection Pool

### Как работает пул

```
                 ┌─────────────────────────────────────────┐
                 │            mongo.Client                  │
                 │                                          │
   goroutine 1 ──►  ┌──────────────────────────────────┐  │
   goroutine 2 ──►  │     Connection Pool               │  │──── MongoDB Server
   goroutine 3 ──►  │  [conn1] [conn2] [conn3] [idle]  │  │
   goroutine N ──►  └──────────────────────────────────┘  │
                 │                                          │
                 └─────────────────────────────────────────┘
```

- Пул создаётся **per server** (для replica set — пул на каждый узел)
- Соединения создаются лениво по мере необходимости
- Idle-соединения переиспользуются
- При превышении `maxPoolSize` горутины ждут освобождения соединения

### Параметры пула

| Параметр | Default | Описание |
|----------|---------|----------|
| `maxPoolSize` | 100 | Макс. соединений к одному серверу |
| `minPoolSize` | 0 | Драйвер поддерживает минимум "тёплых" соединений |
| `maxIdleTimeMS` | 0 (no limit) | Закрывать idle-соединения после этого времени |
| `maxConnecting` | 2 | Макс. одновременно устанавливаемых соединений |
| `waitQueueTimeoutMS` | 0 (no limit) | Таймаут ожидания свободного соединения |

### Мониторинг пула

```go
// Pool events через event monitoring
poolMonitor := &event.PoolMonitor{
    Event: func(evt *event.PoolEvent) {
        switch evt.Type {
        case event.ConnectionCreated:
            log.Printf("connection created: %d", evt.ConnectionID)
        case event.ConnectionClosed:
            log.Printf("connection closed: %d, reason: %s", evt.ConnectionID, evt.Reason)
        case event.PoolCleared:
            log.Printf("pool cleared for: %s", evt.Address)
        }
    },
}

clientOpts := options.Client().
    ApplyURI(uri).
    SetPoolMonitor(poolMonitor)
```

## Context

Контекст в mongo-driver используется для:
1. **Таймауты** — ограничение времени операции
2. **Отмена** — отмена операции при отмене HTTP-запроса
3. **Deadline propagation** — пробрасывание дедлайна от HTTP-хендлера до MongoDB

```go
// Таймаут на конкретную операцию
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

_, err := collection.InsertOne(ctx, doc)
// Если операция не завершилась за 5 секунд — context.DeadlineExceeded

// В HTTP-хендлере — используем context запроса
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
    // r.Context() отменяется, если клиент закрыл соединение
    user, err := h.users.FindOne(r.Context(), bson.M{"_id": id})
    // ...
}
```

### Timeout на уровне клиента (v2)

```go
// В mongo-driver v2 можно задать timeout на уровне клиента
clientOpts := options.Client().
    ApplyURI(uri).
    SetTimeout(30 * time.Second) // default timeout для всех операций

// Timeout на уровне конкретной операции переопределяет клиентский
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Используется min(client timeout, context timeout) = 5s
collection.FindOne(ctx, filter)
```

## Graceful Shutdown

```go
func main() {
    client, err := mongo.Connect(options.Client().ApplyURI(uri))
    if err != nil {
        log.Fatal(err)
    }

    // Обработка сигналов завершения
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigCh
        log.Println("shutting down...")

        // Disconnect закрывает все соединения в пуле
        // и ждёт завершения текущих операций
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        if err := client.Disconnect(ctx); err != nil {
            log.Printf("disconnect error: %v", err)
        }
    }()

    // ... запуск сервера ...
}
```

## Topology Discovery

Драйвер автоматически обнаруживает топологию кластера:

```
| Режим | Описание | Когда |
|-------|----------|-------|
| Single | Прямое соединение | Один сервер, directConnection=true |
| ReplicaSet | Primary + Secondaries | replicaSet указан в URI |
| Sharded | Mongos-роутеры | Подключение к mongos |
```

```go
// Принудительное прямое соединение к одному серверу (без discovery)
opts := options.Client().
    ApplyURI("mongodb://localhost:27017").
    SetDirect(true)
```

## Типичные ошибки

```go
// ОШИБКА: создание клиента на каждый запрос
func handler(w http.ResponseWriter, r *http.Request) {
    client, _ := mongo.Connect(options.Client().ApplyURI(uri)) // ПЛОХО!
    defer client.Disconnect(r.Context())
    // Новый пул каждый раз — нет переиспользования соединений
}

// ПРАВИЛЬНО: один клиент на всё приложение
var client *mongo.Client

func main() {
    client, _ = mongo.Connect(options.Client().ApplyURI(uri))
    defer client.Disconnect(context.Background())
    http.HandleFunc("/", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {
    coll := client.Database("myapp").Collection("users")
    coll.FindOne(r.Context(), bson.M{"_id": id}) // используем общий клиент
}
```

```go
// ОШИБКА: не проверять соединение при старте
client, _ := mongo.Connect(options.Client().ApplyURI(uri))
// Если MongoDB недоступен, ошибка обнаружится только при первой операции

// ПРАВИЛЬНО: Ping при старте
client, _ := mongo.Connect(options.Client().ApplyURI(uri))
if err := client.Ping(ctx, nil); err != nil {
    log.Fatal("MongoDB unreachable:", err)
}
```

```go
// ОШИБКА: забыли Disconnect — соединения не закрываются при завершении
func main() {
    client, _ := mongo.Connect(options.Client().ApplyURI(uri))
    // defer client.Disconnect(ctx) — ЗАБЫЛИ!
    // При завершении процесса соединения обрываются принудительно
    // MongoDB будет ждать timeout перед очисткой серверных ресурсов
}
```

---

## Вопросы на собеседовании

1. **Чем `mongo.Client` отличается от одного соединения?**
   `mongo.Client` управляет пулом соединений. Он потокобезопасен и должен создаваться один раз на приложение. Внутри пул создаёт соединения лениво и переиспользует их между горутинами.

2. **Что делает `client.Ping()` и зачем его вызывать?**
   `Ping` отправляет команду `ping` на сервер для проверки реального соединения. `mongo.Connect` может не обнаружить проблемы сети сразу — реальная ошибка всплывёт только при первой операции. `Ping` при старте позволяет fail fast.

3. **Какие параметры пула соединений важно настраивать?**
   `maxPoolSize` (ограничивает нагрузку на MongoDB, default 100), `minPoolSize` (уменьшает latency холодного старта), `maxIdleTimeMS` (освобождение ресурсов). Для production также важны `serverSelectionTimeoutMS` и `connectTimeoutMS`.

4. **Как работает retry в mongo-driver?**
   `retryWrites=true` (default) — драйвер автоматически повторяет idempotent write-операции (InsertOne, UpdateOne, DeleteOne и др.) при transient network errors. `retryReads=true` — аналогично для чтения. Retry происходит один раз на другом сервере или том же самом.

5. **Что произойдёт, если горутин больше, чем `maxPoolSize`?**
   Горутины, которые не смогли получить соединение из пула, будут ждать в очереди. Если задан `waitQueueTimeoutMS`, при превышении таймаута вернётся ошибка. Без таймаута — ожидание бесконечное (goroutine leak при проблемах с БД).

6. **Зачем указывать `appName` в опциях клиента?**
   `appName` отображается в `db.currentOp()` и логах MongoDB. Помогает идентифицировать, какой сервис создаёт нагрузку на базу при дебаге проблем в production.
