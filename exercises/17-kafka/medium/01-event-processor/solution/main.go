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

func (b *Broker) Publish(topic, key, value string) {
	evt := Event{
		Topic:     topic,
		Key:       key,
		Value:     value,
		Timestamp: time.Now(),
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, sub := range b.subscribers[topic] {
		select {
		case sub.ch <- evt:
		default: // drop if slow
		}
	}
}

func (b *Broker) Subscribe(topic string, handler func(Event) error) func() {
	sub := &subscriber{
		ch:      make(chan Event, 100),
		handler: handler,
		done:    make(chan struct{}),
	}

	b.mu.Lock()
	b.subscribers[topic] = append(b.subscribers[topic], sub)
	b.mu.Unlock()

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		for evt := range sub.ch {
			for attempt := range 3 {
				if err := handler(evt); err == nil {
					break
				}
				_ = attempt
			}
		}
	}()

	return func() {
		b.mu.Lock()
		subs := b.subscribers[topic]
		for i, s := range subs {
			if s == sub {
				b.subscribers[topic] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		b.mu.Unlock()
		close(sub.ch)
	}
}

func (b *Broker) Close() {
	b.mu.Lock()
	b.closed = true
	for topic, subs := range b.subscribers {
		for _, sub := range subs {
			close(sub.ch)
		}
		delete(b.subscribers, topic)
	}
	b.mu.Unlock()
	b.wg.Wait()
}
