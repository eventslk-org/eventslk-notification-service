package broker

import (
	"log"
	"strings"
)

// BrokerSelection captures the broker type the caller asked for vs the one
// actually built, so main can log the fallback clearly.
type BrokerSelection struct {
	Requested string
	Active    string
	Broker    MessageBroker
}

// New picks an implementation based on brokerType. Any value other than
// "kafka" (case-insensitive) returns the MockLogBroker with a warning log so
// developers know real Kafka is not in play.
func New(brokerType, kafkaBrokers, kafkaGroupID string, factory ConsumerFactory) BrokerSelection {
	requested := strings.ToLower(strings.TrimSpace(brokerType))

	if requested == "kafka" {
		return BrokerSelection{
			Requested: requested,
			Active:    "kafka",
			Broker:    NewKafkaBroker(kafkaBrokers, kafkaGroupID, factory),
		}
	}

	if requested == "" {
		log.Println("[broker] WARNING: BROKER_TYPE not set — falling back to MockLogBroker")
		requested = "(unset)"
	} else {
		log.Printf("[broker] WARNING: BROKER_TYPE=%q is not 'kafka' — falling back to MockLogBroker", requested)
	}

	return BrokerSelection{
		Requested: requested,
		Active:    "mock",
		Broker:    NewMockLogBroker(),
	}
}
