package repositiory

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	// ID is the unique Payment ID that globally identifies this Payment, this will be our idempotency key
	ID uuid.UUID
	// Amount is the amount that we need to charge to the given card fo this transaction.
	Amount        int64
	MerchantID    uuid.UUID
	PaymentStatus string

	FailedReason string
	CurrencyCode string
	// Description describes the reason why we are charging this given card
	Description string
	// CardName represents the name displayed on the card
	CardName string
	// CardNumber represents the credit/debit card number
	CardNumber string
	// CardExpiryMonth is the month that the card expires
	CardExpiryMonth int
	// CardExpiryYear is the year that the card expires
	CardExpiryYear int
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}

func (r *repo) CreatePayment(ctx context.Context, payment *Payment) error {
	_, err := r.db.NewInsert().Model(payment).Exec(ctx)
	return err
}

func (r *repo) UpdateStatus(ctx context.Context, payment *Payment) error {
	now := time.Now()
	payment.UpdatedAt = &now
	_, err := r.db.NewUpdate().Model(payment).Where("id = ? AND payment_status = ?", payment.ID, "processing").Column("payment_status", "updated_at", "failed_reason").Exec(ctx)
	return err
}

func (r *repo) GetPaymentByID(ctx context.Context, id uuid.UUID) (*Payment, error) {
	var payment Payment
	_, err := r.db.NewSelect().Model(&payment).Where("id = ?", id).Exec(ctx, &payment)
	return &payment, err
}
