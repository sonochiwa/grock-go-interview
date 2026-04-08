# Kafka в Go: Библиотеки

## Сравнение

```
| | sarama | kafka-go | confluent-kafka-go |
|---|---|---|---|
| Автор | IBM (ex-Shopify) | segment.io | Confluent |
| Обёртка | Pure Go | Pure Go | CGo (librdkafka) |
| Популярность | ★★★★★ | ★★★★ | ★★★ |
| Performance | Хорошая | Хорошая | Лучшая (C) |
| API | Сложнее | Проще | Средняя |
| Cross-compile | ✅ | ✅ | ❌ (CGo) |
| Schema Registry | Нет | Нет | Встроено |
| Рекомендация | Production, гибкость | Простые случаи | Max performance |
```

## kafka-go (простой API)

```go
import "github.com/segmentio/kafka-go"

// Writer (producer)
writer := &kafka.Writer{
    Addr:         kafka.TCP("localhost:9092"),
    Topic:        "orders",
    Balancer:     &kafka.Hash{},  // partition by key
    BatchSize:    100,
    BatchTimeout: 10 * time.Millisecond,
    RequiredAcks: kafka.RequireAll,
    Compression:  kafka.Snappy,
}
defer writer.Close()

err := writer.WriteMessages(ctx,
    kafka.Message{
        Key:   []byte("order-123"),
        Value: jsonBytes,
        Headers: []kafka.Header{
            {Key: "event-type", Value: []byte("OrderCreated")},
        },
    },
)

// Reader (consumer)
reader := kafka.NewReader(kafka.ReaderConfig{
    Brokers:  []string{"localhost:9092"},
    GroupID:  "order-service",
    Topic:    "orders",
    MinBytes: 1,
    MaxBytes: 10e6,
})
defer reader.Close()

for {
    msg, err := reader.ReadMessage(ctx) // auto-commit
    if err != nil {
        break
    }
    fmt.Printf("offset=%d key=%s value=%s\n", msg.Offset, msg.Key, msg.Value)
}

// Manual commit
for {
    msg, err := reader.FetchMessage(ctx) // НЕ коммитит
    if err != nil {
        break
    }
    process(msg)
    reader.CommitMessages(ctx, msg) // explicit commit
}
```

## sarama (production)

```go
import "github.com/IBM/sarama"

// Producer
config := sarama.NewConfig()
config.Producer.RequiredAcks = sarama.WaitForAll
config.Producer.Retry.Max = 3
config.Producer.Return.Successes = true
config.Producer.Idempotent = true
config.Net.MaxOpenRequests = 1

producer, _ := sarama.NewSyncProducer([]string{"localhost:9092"}, config)
defer producer.Close()

msg := &sarama.ProducerMessage{
    Topic: "orders",
    Key:   sarama.StringEncoder("order-123"),
    Value: sarama.ByteEncoder(jsonBytes),
    Headers: []sarama.RecordHeader{
        {Key: []byte("event-type"), Value: []byte("OrderCreated")},
    },
}
partition, offset, err := producer.SendMessage(msg)

// Async producer (higher throughput)
asyncProducer, _ := sarama.NewAsyncProducer(brokers, config)
go func() {
    for err := range asyncProducer.Errors() {
        log.Error("producer error", "err", err)
    }
}()
go func() {
    for msg := range asyncProducer.Successes() {
        log.Debug("message sent", "offset", msg.Offset)
    }
}()
asyncProducer.Input() <- msg

// Consumer Group
consumer := &consumerGroupHandler{}
group, _ := sarama.NewConsumerGroup(brokers, "order-service", config)
for {
    err := group.Consume(ctx, []string{"orders"}, consumer)
    if err != nil {
        log.Error("consumer error", "err", err)
    }
    if ctx.Err() != nil {
        return
    }
}
```

## Рекомендация

```
Новый проект, простой случай → kafka-go
  + Простой API, быстрый старт
  + Достаточно для большинства случаев

Production, сложные сценарии → sarama
  + Больше контроля (async producer, manual partition assignment)
  + Больше community, examples

Максимальная производительность → confluent-kafka-go
  + Обёртка над librdkafka (C)
  - CGo, сложнее cross-compile и Docker builds
```
