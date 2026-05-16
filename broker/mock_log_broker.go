package broker

import (
	"context"
	"log"
	"sync"
	"time"
)

// MockLogBroker is a no-op broker used during local development when Kafka is
// unavailable. It logs subscribed topics and periodically emits a synthetic
// heartbeat so developers can verify the wiring without running Kafka.
type MockLogBroker struct {
	mu     sync.Mutex
	closed bool
	stopCh chan struct{}
}

func NewMockLogBroker() *MockLogBroker {
	return &MockLogBroker{stopCh: make(chan struct{})}
}

func (m *MockLogBroker) Connect(_ context.Context) error {
	log.Println("[broker:mock] connected (no real Kafka — events will be logged only)")
	return nil
}

func (m *MockLogBroker) Consume(ctx context.Context, topics []string) error {
	log.Printf("[broker:mock] subscribed to topics: %v", topics)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-m.stopCh:
				return
			case t := <-ticker.C:
				log.Printf("[broker:mock] heartbeat @ %s — would be polling %d topic(s)",
					t.Format(time.RFC3339), len(topics))
			}
		}
	}()
	return nil
}

func (m *MockLogBroker) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	close(m.stopCh)
	log.Println("[broker:mock] closed")
	return nil
}
