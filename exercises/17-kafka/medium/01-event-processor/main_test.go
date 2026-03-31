package event_processor

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestPublishSubscribe(t *testing.T) {
	b := NewBroker()
	defer b.Close()

	var received atomic.Int32
	b.Subscribe("orders", func(e Event) error {
		received.Add(1)
		return nil
	})

	b.Publish("orders", "order-1", `{"amount": 100}`)

	time.Sleep(100 * time.Millisecond)
	if r := received.Load(); r != 1 {
		t.Errorf("received = %d, want 1", r)
	}
}

func TestMultipleSubscribers(t *testing.T) {
	b := NewBroker()
	defer b.Close()

	var count atomic.Int32
	b.Subscribe("topic", func(e Event) error { count.Add(1); return nil })
	b.Subscribe("topic", func(e Event) error { count.Add(1); return nil })

	b.Publish("topic", "k", "v")
	time.Sleep(100 * time.Millisecond)

	if c := count.Load(); c != 2 {
		t.Errorf("count = %d, want 2", c)
	}
}

func TestRetry(t *testing.T) {
	b := NewBroker()
	defer b.Close()

	var attempts atomic.Int32
	b.Subscribe("topic", func(e Event) error {
		n := attempts.Add(1)
		if n < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	b.Publish("topic", "k", "v")
	time.Sleep(200 * time.Millisecond)

	if a := attempts.Load(); a != 3 {
		t.Errorf("attempts = %d, want 3", a)
	}
}

func TestCancel(t *testing.T) {
	b := NewBroker()
	defer b.Close()

	var count atomic.Int32
	cancel := b.Subscribe("topic", func(e Event) error {
		count.Add(1)
		return nil
	})

	cancel()
	b.Publish("topic", "k", "v")
	time.Sleep(100 * time.Millisecond)

	if c := count.Load(); c != 0 {
		t.Errorf("count = %d after cancel, want 0", c)
	}
}

func TestDifferentTopics(t *testing.T) {
	b := NewBroker()
	defer b.Close()

	var ordersCount, paymentsCount atomic.Int32
	b.Subscribe("orders", func(e Event) error { ordersCount.Add(1); return nil })
	b.Subscribe("payments", func(e Event) error { paymentsCount.Add(1); return nil })

	b.Publish("orders", "k", "v")
	time.Sleep(100 * time.Millisecond)

	if o := ordersCount.Load(); o != 1 {
		t.Errorf("orders = %d, want 1", o)
	}
	if p := paymentsCount.Load(); p != 0 {
		t.Errorf("payments = %d, want 0", p)
	}
}
