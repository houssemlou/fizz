package kafka

import (
	"context"
	"errors"
	"log/slog"
	"time"

	ckafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Message is a received Kafka message passed to a Handler.
// It abstracts away the underlying Kafka client so handlers
// don't need to import confluent-kafka-go directly.
type Message struct {
	Topic   string
	Offset  int64
	Value   []byte
	Headers map[string]string
}

// Handler processes a single Kafka message.
// Return nil to commit the offset (success or intentional skip).
// Return an error to leave the offset uncommitted so the message is retried.
type Handler interface {
	Handle(ctx context.Context, msg *Message) error
}

// Consumer is a generic Kafka consumer that delegates message processing to a Handler.
type Consumer struct {
	c       *ckafka.Consumer
	handler Handler
}

func NewConsumer(broker, topic, groupID string, handler Handler) (*Consumer, error) {
	c, err := ckafka.NewConsumer(&ckafka.ConfigMap{
		"bootstrap.servers":  broker,
		"group.id":           groupID,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
	})
	if err != nil {
		return nil, err
	}
	if err := c.Subscribe(topic, nil); err != nil {
		c.Close()
		return nil, err
	}
	return &Consumer{c: c, handler: handler}, nil
}

// Run polls for messages and delegates to the Handler until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		raw, err := c.c.ReadMessage(100 * time.Millisecond)
		if err != nil {
			var kafkaErr ckafka.Error
			if errors.As(err, &kafkaErr) && kafkaErr.Code() == ckafka.ErrTimedOut {
				continue
			}
			slog.Error("kafka read failed", "error", err)
			continue
		}

		msg := &Message{
			Topic:   *raw.TopicPartition.Topic,
			Offset:  int64(raw.TopicPartition.Offset),
			Value:   raw.Value,
			Headers: headersMap(raw.Headers),
		}

		if err := c.handler.Handle(ctx, msg); err != nil {
			slog.Error("handler failed, skipping commit", "error", err, "offset", msg.Offset)
			continue
		}

		if _, err := c.c.CommitMessage(raw); err != nil {
			slog.Error("kafka commit failed", "error", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.c.Close()
}

func headersMap(headers []ckafka.Header) map[string]string {
	m := make(map[string]string, len(headers))
	for _, h := range headers {
		m[h.Key] = string(h.Value)
	}
	return m
}
