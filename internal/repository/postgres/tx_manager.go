package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
)

// q is a minimal query executor implemented by both pgxpool.Pool and pgx.Tx.
type q interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txKey struct{}

func withTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func getQ(ctx context.Context, pool *pgxpool.Pool) q {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok && tx != nil {
		return tx
	}
	return pool
}

const defaultPageLimit = 50

func sanitizeLimitOffset(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

type txManager struct{ pool *pgxpool.Pool }

func NewTxManager(pool *pgxpool.Pool) repository.TxManager { return &txManager{pool: pool} }

func (m *txManager) WithinTx(ctx context.Context, fn repository.TxFunc) error {
	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return repository.MapPgError(err)
	}
	defer func() {
		// If not committed yet, rollback; ignore rollback errors if context canceled.
		_ = tx.Rollback(context.Background())
	}()

	ctx = withTx(ctx, tx)
	if err := fn(ctx); err != nil {
		// Rollback and return mapped error from fn
		_ = tx.Rollback(context.Background())
		return repository.MapPgError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return repository.MapPgError(err)
	}
	return nil
}

// ensure interfaces are satisfied at compile time
var _ repository.TxManager = (*txManager)(nil)

// helper to assert we didn't accidentally nil the pool
func ensurePool(pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("pgx pool is nil")
	}
	return nil
}
