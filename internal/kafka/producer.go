package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	ckafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Publisher is the generic interface for sending a message to a Kafka topic.
// value is marshalled to JSON by the underlying implementation.
// key controls partition routing — pass "" for round-robin distribution.
type Publisher interface {
	Publish(ctx context.Context, topic string, key string, value any, headers map[string]string) error
	Close()
}

// Producer is the concrete Kafka publisher.
type Producer struct {
	p *ckafka.Producer
}

func NewProducer(broker string) (*Producer, error) {
	p, err := ckafka.NewProducer(&ckafka.ConfigMap{
		"bootstrap.servers": broker,
	})
	if err != nil {
		return nil, err
	}
	prod := &Producer{p: p}
	go prod.handleDeliveries()
	return prod, nil
}

func (p *Producer) Publish(_ context.Context, topic string, key string, value any, headers map[string]string) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	hdrs := make([]ckafka.Header, 0, len(headers))
	for k, v := range headers {
		hdrs = append(hdrs, ckafka.Header{Key: k, Value: []byte(v)})
	}

	// nil key → round-robin; non-empty string → consistent partition routing
	var keyBytes []byte
	if key != "" {
		keyBytes = []byte(key)
	}

	return p.p.Produce(&ckafka.Message{
		TopicPartition: ckafka.TopicPartition{
			Topic:     &topic,
			Partition: ckafka.PartitionAny,
		},
		Key:     keyBytes,
		Headers: hdrs,
		Value:   data,
	}, nil)
}

func (p *Producer) Close() {
	p.p.Flush(5_000)
	p.p.Close()
}

func (p *Producer) handleDeliveries() {
	for e := range p.p.Events() {
		msg, ok := e.(*ckafka.Message)
		if !ok {
			continue
		}
		if msg.TopicPartition.Error != nil {
			slog.Error("kafka delivery failed",
				"topic", *msg.TopicPartition.Topic,
				"error", msg.TopicPartition.Error,
			)
		} else {
			slog.Debug("kafka delivery ok", "offset", msg.TopicPartition.Offset)
		}
	}
}
