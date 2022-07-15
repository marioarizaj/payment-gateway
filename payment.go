package payment_gateway

import (
	"time"

	"github.com/google/uuid"
	"github.com/marioarizaj/payment_gateway/internal/repositiory"
)

// Payment represents a transaction request object received from a Merchant
type Payment struct {
	// ID is the unique Payment ID that globally identifies this Payment, this will be our idempotency key
	ID         uuid.UUID `json:"id"`
	MerchantID uuid.UUID `json:"merchant_id"`
	// Amount is the amount that we need to charge to the given card fo this transaction.
	PaymentStatus string `json:"payment_status"`
	Amount        Amount `json:"amount"`
	// Description describes the reason why we are charging this given card
	Description string     `json:"description"`
	CardInfo    CardInfo   `json:"card_info"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

type Amount struct {
	// AmountFractional is the amount we need to charge to the card in fractional form.
	// For example, if we want to charge $10.00 to the card, we need to set AmountFractional to 1000
	AmountFractional int64  `json:"amount_fractional"`
	CurrencyCode     string `json:"currency_code"`
}

// CardInfo contains the credit/debit card info for a specific transaction.
type CardInfo struct {
	// CardName represents the name displayed on the card
	CardName string `json:"card_name"`
	// CardNumber represents the credit/debit card number
	CardNumber string `json:"card_number"`
	// ExpiryMonth is the month that the card expires
	ExpiryMonth int `json:"expiry_month"`
	// ExpiryYear is the year that the card expires
	ExpiryYear int `json:"expiry_year"`
	// CVV is an abbreviation of Card Verification Value. This consists of a 3-digit number on VISA, MASTERCARD
	// and DISCOVER credit cards. It is a 4-digit number on American Express.
	CVV string `json:"cvv,omitempty"`
}

func (p Payment) GetStoragePayment() *repositiory.Payment {
	return &repositiory.Payment{
		ID:              p.ID,
		Amount:          p.Amount.AmountFractional,
		MerchantID:      p.MerchantID,
		PaymentStatus:   &p.PaymentStatus,
		CurrencyCode:    p.Amount.CurrencyCode,
		Description:     p.Description,
		CardName:        p.CardInfo.CardName,
		CardNumber:      p.CardInfo.CardNumber,
		CardExpiryMonth: p.CardInfo.ExpiryMonth,
		CardExpiryYear:  p.CardInfo.ExpiryYear,
	}
}

func GetPaymentFromStoredPayment(p *repositiory.Payment) Payment {
	return Payment{
		ID:            p.ID,
		PaymentStatus: *p.PaymentStatus,
		Amount: Amount{
			AmountFractional: p.Amount,
			CurrencyCode:     p.CurrencyCode,
		},
		MerchantID:  p.MerchantID,
		Description: p.Description,
		CardInfo: CardInfo{
			CardName:    p.CardName,
			CardNumber:  p.CardNumber,
			ExpiryMonth: p.CardExpiryMonth,
			ExpiryYear:  p.CardExpiryYear,
		},
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
