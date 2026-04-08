# Event Processor

Реализуй in-memory event processor (имитация Kafka consumer паттерна):

- `Event{Topic, Key, Value string, Timestamp time.Time}`
- `NewBroker() *Broker`
- `Publish(topic string, key, value string)` — опубликовать событие
- `Subscribe(topic string, handler func(Event) error) (cancel func())` — подписка с обработчиком
- Обработка в отдельной горутине для каждого subscriber
- At-least-once delivery: если handler вернул error, событие retry (max 3 раза)
- Graceful shutdown через `Close()` — ждёт завершения обработки

Goroutine-safe!
