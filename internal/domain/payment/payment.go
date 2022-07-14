package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/marioarizaj/payment_gateway"
	"github.com/marioarizaj/payment_gateway/internal/creditcard"
	"github.com/marioarizaj/payment_gateway/internal/repositiory"
	"github.com/marioarizaj/payment_gateway/kit/rediscache"
	"github.com/marioarizaj/payment_gateway/kit/responses"
	"github.com/uptrace/bun/driver/pgdriver"
)

const (
	deduplicationCacheKey = "deduplication"
	paymentCacheKey       = "payment"
)

type Domain struct {
	repo   repositiory.Repository
	cache  *rediscache.Client
	logger *zap.Logger
}

func NewDomain(repo repositiory.Repository, cache *rediscache.Client, l *zap.Logger) *Domain {
	return &Domain{
		repo:   repo,
		cache:  cache,
		logger: l,
	}
}

func (d *Domain) CreatePayment(ctx context.Context, payment payment_gateway.Payment) (payment_gateway.Payment, error) {
	err := validateCard(payment.CardInfo)
	if err != nil {
		return payment_gateway.Payment{}, responses.BadRequestError{
			Err: err,
		}
	}
	cacheDedupKey := fmt.Sprintf("%s_%s_%d_%s", deduplicationCacheKey, payment.CardInfo.CardNumber, payment.Amount.AmountFractional, payment.Amount.CurrencyCode)
	err = d.isPaymentValid(ctx, cacheDedupKey)
	if err != nil {
		d.logger.Error("Bad Request", zap.Error(err))
		return payment_gateway.Payment{}, err
	}
	// Here we need to persist this
	err = d.repo.CreatePayment(ctx, payment.GetStoragePayment())
	if err != nil {
		var pgErr pgdriver.Error
		if errors.As(err, &pgErr) {
			if pgErr.IntegrityViolation() {
				d.logger.Error("Pg Integrity violation", zap.Error(err))
				return payment_gateway.Payment{}, responses.ConflictError{}
			}
		}
		d.logger.Error("Database unexpected error", zap.Error(err))
		return payment_gateway.Payment{}, responses.InternalServerError{Err: err}
	}

	err = d.cache.SetValue(ctx, cacheDedupKey, true, 5*time.Minute)
	if err != nil {
		d.logger.Error("Redis unexpected error", zap.Error(err))
		return payment_gateway.Payment{}, responses.InternalServerError{Err: err}
	}
	storedPayment, err := d.repo.GetPaymentByID(ctx, payment.ID)
	if err != nil {
		d.logger.Error("Database Unexpected error", zap.Error(err))
		return payment_gateway.Payment{}, responses.InternalServerError{Err: err}
	}
	p := payment_gateway.GetPaymentFromStoredPayment(storedPayment)
	return p, nil
}

func (d *Domain) GetPayment(ctx context.Context, id uuid.UUID) (payment_gateway.Payment, error) {
	var payment payment_gateway.Payment
	key := fmt.Sprintf("%s_%s", paymentCacheKey, id)
	err := d.cache.GetValue(ctx, key, &payment)
	if err != nil && err != redis.Nil {
		// Here we need to log that cache is down/ maybe even alert but don't stop serving customers
		d.logger.Error("Redis unexpected error", zap.Error(err))
	}
	if err != redis.Nil {
		return payment, nil
	}

	storedPayment, err := d.repo.GetPaymentByID(ctx, id)
	if err != nil && err != sql.ErrNoRows {
		d.logger.Error("Database unexpected error", zap.Error(err))
		return payment_gateway.Payment{}, responses.InternalServerError{}
	}
	if err == sql.ErrNoRows {
		d.logger.Info("Payment not found", zap.String("id", id.String()))
		return payment_gateway.Payment{}, responses.NotFoundError{}
	}
	payment = payment_gateway.GetPaymentFromStoredPayment(storedPayment)
	err = d.cache.SetValue(ctx, key, payment, 24*time.Hour)
	if err != nil {
		// Here we should not stop execution, but rather log the error
		d.logger.Error("Redis unexpected error", zap.Error(err))
	}
	return payment, nil
}

func (d *Domain) isPaymentValid(ctx context.Context, cacheDedupKey string) error {
	// Check cache if this payment was attempted a few minutes ago
	var exists bool
	err := d.cache.GetValue(ctx, cacheDedupKey, &exists)
	if err != nil && err != redis.Nil {
		d.logger.Error("Redis unexpected error", zap.Error(err))
		return responses.InternalServerError{
			Err: err,
		}
	}
	if exists {
		d.logger.Warn("Duplicate payment attempt within 5 minutes")
		return responses.ConflictError{}
	}
	return nil
}

func validateCard(cardInfo payment_gateway.CardInfo) error {
	card := creditcard.Card{
		Number: cardInfo.CardNumber,
		Cvv:    cardInfo.CVV,
		Month:  cardInfo.ExpiryMonth,
		Year:   cardInfo.ExpiryYear,
	}
	return card.Validate()
}
