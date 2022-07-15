package repositiory

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Repository interface {
	CreatePayment(ctx context.Context, payment *Payment) error
	GetPaymentByID(ctx context.Context, id uuid.UUID) (*Payment, error)
	UpdateStatus(ctx context.Context, payment *Payment) error
	Begin(ctx context.Context) (repo, error)
	Rollback(ctx context.Context) error
	Commit(ctx context.Context) error
}

type repo struct {
	db bun.IDB
}

func (r repo) Begin(ctx context.Context) (repo, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return repo{}, err
	}
	return repo{db: tx}, nil
}

func (r repo) Rollback(ctx context.Context) error {
	tx, ok := r.db.(bun.Tx)
	if !ok {
		return errors.New("trying to rollback a non transaction type")
	}
	return tx.Rollback()
}

func (r repo) Commit(ctx context.Context) error {
	tx, ok := r.db.(bun.Tx)
	if !ok {
		return errors.New("trying to commit a non transaction type")
	}
	return tx.Commit()
}

func NewRepository(db bun.IDB) Repository {
	return &repo{db: db}
}
