# Notification Service Resilience Architecture

**Date:** 2026-05-16
**Status:** Implemented

## Decoupling Strategy

All messaging traffic flows through a small `MessageBroker` interface defined in
the `broker` package:

```go
type MessageBroker interface {
    Connect(ctx context.Context) error
    Consume(ctx context.Context, topics []string) error
    Close() error
}
```

`main.go` depends only on this interface and a factory (`broker.New`) that
inspects the `BROKER_TYPE` environment variable and returns one of two
implementations:

| Implementation   | When                       | Behavior                                         |
| ---------------- | -------------------------- | ------------------------------------------------ |
| `KafkaBroker`    | `BROKER_TYPE=kafka`        | Wraps `sarama.ConsumerGroup` with a retry loop.  |
| `MockLogBroker`  | anything else (incl. unset) | No network I/O; logs topics and a heartbeat.    |

The consumer-side business logic (`consumer.NotificationHandler`) is injected
into the Kafka broker through a `ConsumerFactory` callback, so swapping the
transport never touches event-dispatch code.

## Fallback Behavior

When `BROKER_TYPE` is unset or set to anything other than `kafka`, the factory
emits a warning log and returns a `MockLogBroker`. The mock broker:

1. Logs `[broker:mock] connected (no real Kafka — events will be logged only)`.
2. Logs the list of subscribed topics on `Consume`.
3. Emits a heartbeat line every 30 seconds so developers can confirm the loop
   is alive without spinning up Kafka or Docker.
4. Cleanly stops on `Close` (channel-based shutdown, safe to call twice).

This means the service boots and stays healthy on machines without Docker, and
the rest of the application (HTTP server, Eureka registration, email sender)
runs exactly as in production.

## Retry Mechanism

`KafkaBroker.Connect` is **non-blocking**. It spawns a background goroutine
(`connectLoop`) that:

1. Calls `sarama.NewConsumerGroup(brokers, groupID, config)`.
2. On success: stores the group, closes a `ready` channel, and starts an error
   drainer goroutine.
3. On failure: logs the attempt number and underlying error, then sleeps for
   **5 seconds** (`retryInterval`) before trying again. Cancellation of the
   parent `context.Context` is the only exit condition.

`KafkaBroker.Consume` is called immediately after `Connect`. It launches a
goroutine that **parks on the `ready` channel** until the connection lands,
then enters the standard sarama consume loop. This means a late-starting Kafka
container produces a clean burst of "connect attempt N failed" log lines
followed by "connected on attempt N" — never a panic and never a `log.Fatal`.

`main.go` no longer calls `log.Fatalf` on broker errors; it only logs and
continues, so the HTTP `/actuator/health` endpoint stays reachable while Kafka
is recovering.

## Verification Guide

### 1. Mock mode (no Docker required)

```bash
cd EventsLK-NotificationService
BROKER_TYPE=mock go run .
```

Expected log lines:

```
[main] broker selected: requested=mock active=mock
[broker:mock] connected (no real Kafka — events will be logged only)
[broker:mock] subscribed to topics: [eventslk.user.signup ...]
[broker:mock] heartbeat @ ...   (every 30s)
```

`curl http://localhost:8082/actuator/health` should return `200 OK`.

### 2. Kafka mode, broker offline (resilience check)

```bash
docker compose -f others/kafka-cluster/kafka-config.yml down
cd EventsLK-NotificationService
BROKER_TYPE=kafka go run .
```

Expected: the service starts, the HTTP listener binds, and the log shows
`[broker:kafka] connect attempt N failed: ... — retrying in 5s` repeating
every five seconds. **No panic, no exit.**

In another terminal, bring Kafka up:

```bash
docker compose -f others/kafka-cluster/kafka-config.yml up -d kafka kafka-ui
```

Within ~5 seconds the original process should print
`[broker:kafka] connected to [localhost:9092] on attempt N` followed by
`[broker:kafka] starting consumer loop on topics: [...]`.

### 3. Kafka mode, full happy path (VS Code)

Run the **Start All Services** task — it now depends on **Start Kafka
Infrastructure**, which executes `docker compose -f
others/kafka-cluster/kafka-config.yml up -d kafka kafka-ui` from the workspace
root before the Go service starts. Kafka UI is reachable at
<http://localhost:8085>.
