package broker

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

const retryInterval = 5 * time.Second

// ConsumerFactory builds the sarama.ConsumerGroupHandler that processes
// messages once the Kafka connection is healthy. Keeping this as a callback
// decouples the broker from the concrete handler implementation.
type ConsumerFactory func() sarama.ConsumerGroupHandler

// KafkaBroker wraps a sarama ConsumerGroup with a non-blocking, retrying
// connect loop so a missing or late-starting Kafka broker never crashes the
// service at boot.
type KafkaBroker struct {
	brokers []string
	groupID string
	config  *sarama.Config
	factory ConsumerFactory

	mu    sync.Mutex
	group sarama.ConsumerGroup
	ready chan struct{}
}

func NewKafkaBroker(brokerCSV, groupID string, factory ConsumerFactory) *KafkaBroker {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_0_0_0
	cfg.Consumer.Return.Errors = true
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	return &KafkaBroker{
		brokers: splitTrim(brokerCSV, ","),
		groupID: groupID,
		config:  cfg,
		factory: factory,
		ready:   make(chan struct{}),
	}
}

// Connect spawns a background goroutine that retries the Kafka handshake every
// 5 seconds. It returns immediately so a missing broker never blocks startup.
func (k *KafkaBroker) Connect(ctx context.Context) error {
	go k.connectLoop(ctx)
	return nil
}

func (k *KafkaBroker) connectLoop(ctx context.Context) {
	attempt := 0
	for {
		attempt++
		group, err := sarama.NewConsumerGroup(k.brokers, k.groupID, k.config)
		if err == nil {
			k.mu.Lock()
			k.group = group
			k.mu.Unlock()

			log.Printf("[broker:kafka] connected to %v on attempt %d", k.brokers, attempt)
			close(k.ready)

			go k.drainErrors(group)
			return
		}

		log.Printf("[broker:kafka] connect attempt %d failed: %v — retrying in %s",
			attempt, err, retryInterval)

		select {
		case <-ctx.Done():
			log.Printf("[broker:kafka] giving up — context cancelled")
			return
		case <-time.After(retryInterval):
		}
	}
}

func (k *KafkaBroker) drainErrors(group sarama.ConsumerGroup) {
	for err := range group.Errors() {
		log.Printf("[broker:kafka] runtime error: %v", err)
	}
}

// Consume waits (in a background goroutine) for Connect to succeed, then
// enters the standard sarama consume loop. Calling this before Connect
// completes is safe — it simply parks until the connection is ready.
func (k *KafkaBroker) Consume(ctx context.Context, topics []string) error {
	handler := k.factory()

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-k.ready:
		}

		k.mu.Lock()
		group := k.group
		k.mu.Unlock()

		log.Printf("[broker:kafka] starting consumer loop on topics: %v", topics)
		for {
			if err := group.Consume(ctx, topics, handler); err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[broker:kafka] consume error: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	return nil
}

func (k *KafkaBroker) Close() error {
	k.mu.Lock()
	group := k.group
	k.mu.Unlock()

	if group == nil {
		return nil
	}
	return group.Close()
}

func splitTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
