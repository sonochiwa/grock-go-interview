# Kafka: Producer паттерны

## Sync vs Async Producer

```go
// Sync — простой, гарантированная доставка, медленнее
producer, _ := sarama.NewSyncProducer(brokers, config)
partition, offset, err := producer.SendMessage(msg)
// err != nil → точно знаем что не доставлено

// Async — быстрее, нужно обрабатывать Errors channel
producer, _ := sarama.NewAsyncProducer(brokers, config)

// ОБЯЗАТЕЛЬНО читать оба канала (иначе deadlock)
go func() {
    for err := range producer.Errors() {
        log.Error("send failed", "topic", err.Msg.Topic, "err", err.Err)
        // Retry? DLQ? Метрика?
    }
}()
go func() {
    for msg := range producer.Successes() {
        log.Debug("sent", "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset)
    }
}()

producer.Input() <- msg // non-blocking
```

## Partitioning

```go
// По key (hash) — сообщения с одним key в одну partition
msg := &sarama.ProducerMessage{
    Topic: "orders",
    Key:   sarama.StringEncoder(orderID), // hash(orderID) % partitions
    Value: sarama.ByteEncoder(data),
}

// Round-robin (нет key) — равномерное распределение
msg := &sarama.ProducerMessage{
    Topic: "orders",
    Value: sarama.ByteEncoder(data),
    // Key не указан → round-robin
}

// Custom partitioner
config.Producer.Partitioner = func(topic string) sarama.Partitioner {
    return &priorityPartitioner{} // своя логика
}

// Явное указание partition
msg.Partition = 0 // конкретная partition
```

## Exactly-Once Producer

```go
config := sarama.NewConfig()
config.Producer.Idempotent = true          // дедупликация на стороне broker
config.Producer.RequiredAcks = sarama.WaitForAll
config.Net.MaxOpenRequests = 1             // обязательно для idempotent

// Transactional producer (для Kafka Transactions)
config.Producer.Transaction.ID = "my-service-tx" // уникальный ID

producer, _ := sarama.NewAsyncProducer(brokers, config)

// Транзакция
err := producer.BeginTxn()
producer.Input() <- msg1
producer.Input() <- msg2
err = producer.CommitTxn()  // атомарно: либо оба, либо ни один
// или
err = producer.AbortTxn()   // откатить
```

## Сериализация

```go
// JSON (простой, но медленный)
data, _ := json.Marshal(order)
msg.Value = sarama.ByteEncoder(data)

// Protobuf (рекомендуется для production)
data, _ := proto.Marshal(orderProto)
msg.Value = sarama.ByteEncoder(data)

// Avro + Schema Registry (enterprise)
// Используется с confluent-kafka-go + schema registry client

// Headers для metadata
msg.Headers = []sarama.RecordHeader{
    {Key: []byte("event-type"), Value: []byte("OrderCreated")},
    {Key: []byte("schema-version"), Value: []byte("1")},
    {Key: []byte("content-type"), Value: []byte("application/protobuf")},
}
```

## Best Practices

```
1. Всегда задавай Key для ordering:
   key = entity_id (order_id, user_id)
   → все события одной сущности в одной partition → порядок гарантирован

2. Acks:
   acks=all для критичных данных (orders, payments)
   acks=1 для логов, метрик (допустима потеря)

3. Batching:
   batch.size + linger.ms — баланс latency vs throughput
   Больше batch → выше throughput, выше latency

4. Compression:
   lz4 — лучший баланс скорость/степень сжатия
   snappy — быстрее, меньше сжатие
   zstd — лучшее сжатие, медленнее

5. Graceful shutdown:
   producer.Close() — дожидается отправки всех сообщений
   Установи таймаут: config.Producer.Flush.MaxMessages

6. Мониторинг:
   - producer errors rate
   - send latency (p50, p99)
   - batch size
   - compression ratio
```
