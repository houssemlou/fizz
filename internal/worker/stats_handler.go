package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/houssemlou/fizz/internal/domain/fizzbuzz"
	"github.com/houssemlou/fizz/internal/kafka"
	"github.com/houssemlou/fizz/internal/producer"
)

type StatsHandler struct {
	repo fizzbuzz.Repository
}

func NewStatsHandler(repo fizzbuzz.Repository) *StatsHandler {
	return &StatsHandler{repo: repo}
}

func (h *StatsHandler) Handle(ctx context.Context, msg *kafka.Message) error {
	var req producer.FizzBuzzRequest
	if err := json.Unmarshal(msg.Value, &req); err != nil {
		slog.Error("unmarshal failed — skipping", "error", err, "offset", msg.Offset)
		return nil
	}

	slog.Debug("processing fizzbuzz request",
		"request_id", req.RequestID,
		"idempotent_id", req.IdempotentID,
		"lag", time.Since(req.Timestamp).String(),
	)

	return h.repo.Record(ctx, req.Params, req.RequestID)
}
