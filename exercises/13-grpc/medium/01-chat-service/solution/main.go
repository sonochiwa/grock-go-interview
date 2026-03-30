package chat_service

import (
	"sync"
	"time"
)

type Message struct {
	From      string
	Room      string
	Text      string
	Timestamp time.Time
}

type ChatServer struct {
	mu          sync.RWMutex
	subscribers map[string]chan Message
}

func NewChatServer() *ChatServer {
	return &ChatServer{
		subscribers: make(map[string]chan Message),
	}
}

func (cs *ChatServer) Send(msg Message) error {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	for _, ch := range cs.subscribers {
		select {
		case ch <- msg:
		default: // drop if subscriber is slow
		}
	}
	return nil
}

func (cs *ChatServer) Subscribe(userID string) (<-chan Message, func()) {
	ch := make(chan Message, 100)
	cs.mu.Lock()
	cs.subscribers[userID] = ch
	cs.mu.Unlock()

	return ch, func() {
		cs.mu.Lock()
		delete(cs.subscribers, userID)
		close(ch)
		cs.mu.Unlock()
	}
}

func (cs *ChatServer) ActiveSubscribers() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return len(cs.subscribers)
}
