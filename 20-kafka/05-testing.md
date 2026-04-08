# Kafka: Testing

## Unit тесты с mock

```go
// sarama предоставляет mocks из коробки
import "github.com/IBM/sarama/mocks"

func TestProducer(t *testing.T) {
    config := sarama.NewConfig()
    config.Producer.Return.Successes = true

    producer := mocks.NewSyncProducer(t, config)
    // Ожидаем один успешный send
    producer.ExpectSendMessageAndSucceed()

    partition, offset, err := producer.SendMessage(&sarama.ProducerMessage{
        Topic: "orders",
        Value: sarama.StringEncoder(`{"id":"123"}`),
    })

    assert.NoError(t, err)
    assert.Equal(t, int32(0), partition)
    assert.Equal(t, int64(0), offset)
}

func TestProducerError(t *testing.T) {
    producer := mocks.NewSyncProducer(t, nil)
    producer.ExpectSendMessageAndFail(sarama.ErrNotLeaderForPartition)

    _, _, err := producer.SendMessage(&sarama.ProducerMessage{
        Topic: "orders",
        Value: sarama.StringEncoder("data"),
    })

    assert.Error(t, err)
}
```

## Integration тесты с testcontainers

```go
import (
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/kafka"
)

func TestKafkaIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()

    // Запуск Kafka в Docker
    kafkaContainer, err := kafka.Run(ctx,
        "confluentinc/confluent-local:7.5.0",
        kafka.WithClusterID("test-cluster"),
    )
    require.NoError(t, err)
    defer kafkaContainer.Terminate(ctx)

    brokers, err := kafkaContainer.Brokers(ctx)
    require.NoError(t, err)

    // Создать topic
    admin, err := sarama.NewClusterAdmin(brokers, sarama.NewConfig())
    require.NoError(t, err)
    defer admin.Close()

    err = admin.CreateTopic("test-orders", &sarama.TopicDetail{
        NumPartitions:     3,
        ReplicationFactor: 1,
    }, false)
    require.NoError(t, err)

    // Тест producer + consumer
    t.Run("produce and consume", func(t *testing.T) {
        // Produce
        config := sarama.NewConfig()
        config.Producer.Return.Successes = true
        producer, err := sarama.NewSyncProducer(brokers, config)
        require.NoError(t, err)
        defer producer.Close()

        _, _, err = producer.SendMessage(&sarama.ProducerMessage{
            Topic: "test-orders",
            Key:   sarama.StringEncoder("order-1"),
            Value: sarama.StringEncoder(`{"id":"order-1","amount":100}`),
        })
        require.NoError(t, err)

        // Consume
        consumer, err := sarama.NewConsumer(brokers, sarama.NewConfig())
        require.NoError(t, err)
        defer consumer.Close()

        pc, err := consumer.ConsumePartition("test-orders", 0, sarama.OffsetOldest)
        require.NoError(t, err)
        defer pc.Close()

        select {
        case msg := <-pc.Messages():
            assert.Equal(t, "order-1", string(msg.Key))
            assert.Contains(t, string(msg.Value), "order-1")
        case <-time.After(10 * time.Second):
            t.Fatal("timeout waiting for message")
        }
    })
}
```

## Тестирование consumer handler

```go
// Тестировать бизнес-логику отдельно от Kafka
type OrderProcessor interface {
    Process(ctx context.Context, event OrderEvent) error
}

// В consumer:
func (c *orderConsumer) ConsumeClaim(session sarama.ConsumerGroupSession,
    claim sarama.ConsumerGroupClaim) error {
    for msg := range claim.Messages() {
        var event OrderEvent
        if err := json.Unmarshal(msg.Value, &event); err != nil {
            c.sendToDLQ(msg, err)
            session.MarkMessage(msg, "")
            continue
        }
        if err := c.processor.Process(session.Context(), event); err != nil {
            // handle error...
            continue
        }
        session.MarkMessage(msg, "")
    }
    return nil
}

// Тест бизнес-логики (без Kafka):
func TestOrderProcessor(t *testing.T) {
    db := setupTestDB(t)
    processor := NewOrderProcessor(db)

    err := processor.Process(context.Background(), OrderEvent{
        ID:     "order-1",
        Amount: 100,
        Action: "create",
    })
    assert.NoError(t, err)

    // Проверить результат в БД
    order, err := db.GetOrder("order-1")
    assert.NoError(t, err)
    assert.Equal(t, 100, order.Amount)
}

// Тест idempotency:
func TestOrderProcessorIdempotent(t *testing.T) {
    db := setupTestDB(t)
    processor := NewOrderProcessor(db)

    event := OrderEvent{ID: "order-1", Amount: 100, Action: "create"}

    // Обработать дважды
    err := processor.Process(context.Background(), event)
    assert.NoError(t, err)
    err = processor.Process(context.Background(), event)
    assert.NoError(t, err) // не должно быть ошибки

    // Только один заказ в БД
    count, _ := db.CountOrders()
    assert.Equal(t, 1, count)
}
```

## Рекомендации

```
1. Unit тесты:
   - Тестируй бизнес-логику отдельно от Kafka (interface + mock)
   - sarama/mocks для producer/consumer mock
   - Тестируй error handling, DLQ routing, idempotency

2. Integration тесты:
   - testcontainers-go для Kafka в Docker
   - -short flag для skip в CI (если медленно)
   - Тестируй полный flow: produce → consume → verify

3. End-to-end:
   - Docker Compose: Kafka + Zookeeper + приложение
   - Проверять consumer lag, DLQ, метрики
```
