package repositiory

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	// ID is the unique Payment ID that globally identifies this Payment, this will be our idempotency key
	ID uuid.UUID `json:"id"`
	// Amount is the amount that we need to charge to the given card fo this transaction.
	Amount        int64     `json:"amount"`
	MerchantID    uuid.UUID `json:"merchant_id"`
	PaymentStatus *string   `json:"payment_status"`
	CurrencyCode  string    `json:"currency_code"`
	// Description describes the reason why we are charging this given card
	Description string `json:"description"`
	// CardName represents the name displayed on the card
	CardName string `json:"card_name"`
	// CardNumber represents the credit/debit card number
	CardNumber string `json:"card_number"`
	// CardExpiryMonth is the month that the card expires
	CardExpiryMonth int `json:"expiry_month"`
	// CardExpiryYear is the year that the card expires
	CardExpiryYear int        `json:"expiry_year"`
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

func (r *repo) CreatePayment(ctx context.Context, payment *Payment) error {
	_, err := r.db.NewInsert().Model(payment).Exec(ctx)
	return err
}

func (r *repo) GetPaymentByID(ctx context.Context, id uuid.UUID) (*Payment, error) {
	var payment Payment
	_, err := r.db.NewSelect().Model(&payment).Where("id = ?", id).Exec(ctx, &payment)
	return &payment, err
}
