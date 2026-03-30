package chat_service

import (
	"testing"
	"time"
)

func TestSendAndReceive(t *testing.T) {
	cs := NewChatServer()
	ch, unsub := cs.Subscribe("alice")
	defer unsub()

	msg := Message{From: "bob", Room: "general", Text: "hello", Timestamp: time.Now()}
	cs.Send(msg)

	select {
	case got := <-ch:
		if got.Text != "hello" {
			t.Errorf("got %q, want hello", got.Text)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	cs := NewChatServer()
	ch1, unsub1 := cs.Subscribe("alice")
	defer unsub1()
	ch2, unsub2 := cs.Subscribe("bob")
	defer unsub2()

	cs.Send(Message{From: "charlie", Text: "hi all"})

	for _, ch := range []<-chan Message{ch1, ch2} {
		select {
		case msg := <-ch:
			if msg.Text != "hi all" {
				t.Errorf("got %q", msg.Text)
			}
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestUnsubscribe(t *testing.T) {
	cs := NewChatServer()
	_, unsub := cs.Subscribe("alice")

	if cs.ActiveSubscribers() != 1 {
		t.Errorf("ActiveSubscribers = %d, want 1", cs.ActiveSubscribers())
	}

	unsub()

	if cs.ActiveSubscribers() != 0 {
		t.Errorf("ActiveSubscribers after unsub = %d, want 0", cs.ActiveSubscribers())
	}
}

func TestSendNoBlock(t *testing.T) {
	cs := NewChatServer()
	ch, unsub := cs.Subscribe("alice")
	defer unsub()

	// Fill buffer
	for range 200 {
		cs.Send(Message{Text: "spam"})
	}

	// Should not block
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count == 0 {
		t.Error("expected at least some messages")
	}
}
