package infrastructure

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/iqbaljlldn/nexus/apps/api/internal/platform/txcontext"
)

type PostgresTransactionManager struct {
	db *sql.DB
}

func NewPostgresTransactionManager(db *sql.DB) *PostgresTransactionManager {
	return &PostgresTransactionManager{db: db}
}

// WithinTransaction executes fn within a database transaction. The *sql.Tx is
// propagated via context so that any repository using txcontext.ExtractDBTX
// will automatically participate in the same transaction.
func (m *PostgresTransactionManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	txCtx := txcontext.WithTx(ctx, tx)

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %w (original: %v)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
