package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
)

type pinger struct{ pool *pgxpool.Pool }

// NewPinger adapts pgxpool to the repository.Pinger interface.
func NewPinger(pool *pgxpool.Pool) repository.Pinger { return &pinger{pool: pool} }

func (p *pinger) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }
