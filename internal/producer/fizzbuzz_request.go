package producer

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/houssemlou/fizz/internal/domain/fizzbuzz"
	"github.com/houssemlou/fizz/internal/kafka"
)

// FizzBuzzRequestsTopic is the Kafka topic for fizzbuzz request events.
const FizzBuzzRequestsTopic = "fizzbuzz-requests"

// FizzBuzzRequest is the Kafka message schema for a fizzbuzz request event.
type FizzBuzzRequest struct {
	RequestID    string          `json:"request_id"`    // unique per message (UUID v4)
	IdempotentID string          `json:"idempotent_id"` // deterministic hash of params (UUID v5)
	Timestamp    time.Time       `json:"timestamp"`
	Params       fizzbuzz.Params `json:"params"`
}

// FizzBuzzSender publishes FizzBuzzRequest events.
// It implements fizzbuzz.StatsRecorder.
type FizzBuzzSender struct {
	pub kafka.Publisher
}

func NewFizzBuzzSender(pub kafka.Publisher) *FizzBuzzSender {
	return &FizzBuzzSender{pub: pub}
}

func (s *FizzBuzzSender) Record(ctx context.Context, params fizzbuzz.Params) error {
	msg := FizzBuzzRequest{
		RequestID:    uuid.New().String(),
		IdempotentID: params.IdempotentID(),
		Timestamp:    time.Now().UTC(),
		Params:       params,
	}

	return s.pub.Publish(ctx, FizzBuzzRequestsTopic, msg.IdempotentID, msg, map[string]string{
		"request_id":     msg.RequestID,
		"idempotent_id":  msg.IdempotentID,
		"content-type":   "application/json",
		"schema-version": "1",
	})
}
