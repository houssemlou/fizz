package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/houssemlou/fizz/internal/domain/fizzbuzz"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// WHERE clause skips the increment when last_request_id already matches — Kafka dedup.
func (r *Repository) Record(ctx context.Context, p fizzbuzz.Params, requestID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO fizz_requests (idempotent_id, last_request_id, int1, int2, lim, str1, str2)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (idempotent_id) DO UPDATE
			SET hits            = fizz_requests.hits + 1,
			    last_request_id = EXCLUDED.last_request_id,
			    updated_at      = NOW()
		WHERE fizz_requests.last_request_id != EXCLUDED.last_request_id
	`, p.IdempotentID(), requestID, p.Int1, p.Int2, p.Limit, p.Str1, p.Str2)
	return err
}

func (r *Repository) Top(ctx context.Context) (*fizzbuzz.TopResult, error) {
	var e fizzbuzz.TopResult
	err := r.pool.QueryRow(ctx, `
		SELECT int1, int2, lim, str1, str2, hits
		FROM fizz_requests
		ORDER BY hits DESC
		LIMIT 1
	`).Scan(&e.Params.Int1, &e.Params.Int2, &e.Params.Limit, &e.Params.Str1, &e.Params.Str2, &e.Hits)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &e, err
}
