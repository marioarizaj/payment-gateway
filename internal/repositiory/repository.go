package repositiory

import (
	"context"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Repository interface {
	CreatePayment(ctx context.Context, payment *Payment) error
	GetPaymentByID(ctx context.Context, id uuid.UUID) (*Payment, error)
	UpdatePayment(ctx context.Context, payment *Payment) error
}

type repo struct {
	db bun.IDB
}

func NewRepository(db bun.IDB) Repository {
	return &repo{db: db}
}
