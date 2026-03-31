package event_processor

import (
	"sync"
	"time"
)

type Event struct {
	Topic     string
	Key       string
	Value     string
	Timestamp time.Time
}

type subscriber struct {
	ch      chan Event
	handler func(Event) error
	done    chan struct{}
}

type Broker struct {
	mu          sync.RWMutex
	subscribers map[string][]*subscriber
	closed      bool
	wg          sync.WaitGroup
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string][]*subscriber),
	}
}

// TODO: отправь event всем подписчикам топика (non-blocking)
func (b *Broker) Publish(topic, key, value string) {
}

// TODO: подпишись на топик
// - Создай канал для subscriber
// - Запусти горутину для обработки событий
// - Retry до 3 раз если handler возвращает error
// - Верни cancel func для отписки
func (b *Broker) Subscribe(topic string, handler func(Event) error) func() {
	return func() {}
}

// TODO: закрой все каналы, дождись завершения
func (b *Broker) Close() {
}
