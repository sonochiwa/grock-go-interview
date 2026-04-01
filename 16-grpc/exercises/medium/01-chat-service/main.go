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

// TODO: отправь сообщение всем подписчикам (non-blocking)
func (cs *ChatServer) Send(msg Message) error {
	return nil
}

// TODO: подпишись на сообщения, верни канал и функцию отписки
func (cs *ChatServer) Subscribe(userID string) (<-chan Message, func()) {
	return nil, func() {}
}

// TODO: количество активных подписчиков
func (cs *ChatServer) ActiveSubscribers() int {
	return 0
}
