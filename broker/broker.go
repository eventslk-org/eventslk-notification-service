package broker

import "context"

// MessageBroker abstracts the underlying messaging infrastructure so that the
// notification service can switch between a real Kafka connection and a local
// development fallback without touching call sites.
type MessageBroker interface {
	// Connect establishes the connection to the underlying broker. For
	// implementations that need to retry on failure, this method must be
	// non-blocking and return nil immediately while the retry runs in the
	// background.
	Connect(ctx context.Context) error

	// Consume subscribes to the given topics and begins dispatching messages
	// once the underlying connection is healthy.
	Consume(ctx context.Context, topics []string) error

	// Close releases all resources held by the broker.
	Close() error
}
