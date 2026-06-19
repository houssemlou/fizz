package fizzbuzz

import (
	"context"
	"log/slog"
	"strconv"
)

// EventRecorder is notified after each successful generation — the domain doesn't know who listens.
type EventRecorder interface {
	Record(ctx context.Context, p Params) error
}

type Repository interface {
	Record(ctx context.Context, p Params, requestID string) error
	Top(ctx context.Context) (*TopResult, error)
}

type Service struct {
	recorder EventRecorder
	repo     Repository
}

func NewService(recorder EventRecorder, repo Repository) *Service {
	return &Service{recorder: recorder, repo: repo}
}

func (s *Service) Generate(ctx context.Context, p Params) ([]string, error) {
	result, err := s.generate(p)
	if err != nil {
		return nil, err
	}
	if err := s.recorder.Record(ctx, p); err != nil {
		slog.Error("event record failed", "error", err)
	}
	return result, nil
}

func (s *Service) Top(ctx context.Context) (*TopResult, error) {
	return s.repo.Top(ctx)
}

func (s *Service) generate(p Params) ([]string, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	result := make([]string, p.Limit)
	for i := 1; i <= p.Limit; i++ {
		switch {
		case i%p.Int1 == 0 && i%p.Int2 == 0:
			result[i-1] = p.Str1 + p.Str2
		case i%p.Int1 == 0:
			result[i-1] = p.Str1
		case i%p.Int2 == 0:
			result[i-1] = p.Str2
		default:
			result[i-1] = strconv.Itoa(i)
		}
	}
	return result, nil
}
