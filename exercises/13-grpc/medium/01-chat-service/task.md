# Chat Service

Реализуй in-memory chat service (имитация gRPC паттернов без protobuf):

- `NewChatServer() *ChatServer`
- `Send(msg Message) error` — отправить сообщение в комнату
- `Subscribe(userID string) (<-chan Message, func())` — подписка на сообщения, возвращает канал и unsubscribe func
- `ActiveSubscribers() int`

Message: `{From, Room, Text string, Timestamp time.Time}`

Goroutine-safe! Unsubscribe должен закрывать канал подписчика.
