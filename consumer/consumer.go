package consumer

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/IBM/sarama"
	"eventslk-notification-service/email"
)

var topics = []string{
	"eventslk.user.signup",
	"eventslk.user.otp",
	"eventslk.booking.confirmed",
	"eventslk.booking.cancelled",
	"eventslk.promotion",
}

// StartConsumers creates a Kafka ConsumerGroup and begins consuming all notification topics.
// The returned ConsumerGroup must be closed by the caller on shutdown.
func StartConsumers(ctx context.Context, brokers, groupID string, sender *email.EmailSender) (sarama.ConsumerGroup, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_0_0_0
	cfg.Consumer.Return.Errors = true
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	brokerList := splitTrim(brokers, ",")

	group, err := sarama.NewConsumerGroup(brokerList, groupID, cfg)
	if err != nil {
		return nil, fmt.Errorf("create consumer group: %w", err)
	}

	handler := NewNotificationHandler(sender)

	// Drain Kafka-level errors; required when Consumer.Return.Errors is true.
	go func() {
		for err := range group.Errors() {
			log.Printf("[consumer] kafka error: %v", err)
		}
	}()

	// Main consume loop — re-enters after every rebalance.
	go func() {
		for {
			if err := group.Consume(ctx, topics, handler); err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[consumer] consume error: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	log.Printf("[consumer] started, listening on topics: %v", topics)
	return group, nil
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
