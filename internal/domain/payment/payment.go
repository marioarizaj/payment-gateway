package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/eapache/go-resiliency/retrier"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/marioarizaj/payment-gateway"
	"github.com/marioarizaj/payment-gateway/internal/creditcard"
	"github.com/marioarizaj/payment-gateway/internal/repositiory"
	"github.com/marioarizaj/payment-gateway/kit/responses"
	"github.com/uptrace/bun/driver/pgdriver"
)

const (
	deduplicationCacheKey = "deduplication"
	paymentCacheKey       = "payment"

	createPaymentAcquiringBank = "create_payment_in_acquiring_bank"

	retries = 3
)

type BankClient interface {
	CreatePayment(payment payment_gateway.Payment, callBack func(payment payment_gateway.Payment)) http.Response
}

type Cache interface {
	SetValue(ctx context.Context, k string, v interface{}, expiration time.Duration) error
	GetValue(ctx context.Context, k string, dest interface{}) error
	DeleteKey(ctx context.Context, k string) error
}

type Domain struct {
	repo       repositiory.Repository
	cache      Cache
	logger     *zap.Logger
	bankClient BankClient
}

func NewDomain(repo repositiory.Repository, cache Cache, l *zap.Logger, bankClient BankClient) *Domain {
	return &Domain{
		repo:       repo,
		cache:      cache,
		logger:     l,
		bankClient: bankClient,
	}
}

func (d *Domain) callbackFromAcquiringBank(payment payment_gateway.Payment) {
	// First let's invalidate the cache
	err := d.cache.DeleteKey(context.Background(), fmt.Sprintf("%s_%s", paymentCacheKey, payment.ID))
	if err != nil {
		d.logger.Error("Unexpected redis error", zap.Error(err))
	}
	err = d.repo.UpdateStatus(context.Background(), payment.GetStoragePayment())
	if err != nil {
		d.logger.Error("Unexpected internal database error", zap.Error(err))
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

	payment.PaymentStatus = "processing"
	txRepo, err := d.repo.Begin(ctx)
	if err != nil {
		d.logger.Error("Error initialising transaction")
		return payment_gateway.Payment{}, responses.InternalServerError{Err: err}
	}
	// Create the payment on our internal system, with a status of processing
	err = txRepo.CreatePayment(ctx, payment.GetStoragePayment())
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
	err = d.CreatePaymentOnAcquiringBank(payment)
	if err != nil {
		internalErr := txRepo.Rollback(ctx)
		if internalErr != nil {
			d.logger.Error("Internal database unexpected error", zap.Error(err))
		}
		return payment_gateway.Payment{}, err
	}
	err = txRepo.Commit(ctx)
	if err != nil {
		d.logger.Error("Could not commit transaction", zap.Error(err))
		return payment_gateway.Payment{}, err
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

func (d *Domain) CreatePaymentOnAcquiringBank(payment payment_gateway.Payment) error {
	// Use a circuit breaking library in cases when the acquiring bank is offline
	res, err := d.CreatePaymentUsingCircuitBreaker(payment, d.callbackFromAcquiringBank)
	if err != nil {
		return responses.GetErrorResponseFromStatusCode(res.StatusCode, err)
	}
	_ = res.Body.Close()
	return nil
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

func (d *Domain) CreatePaymentUsingCircuitBreaker(payment payment_gateway.Payment, callBackFn func(payment_gateway.Payment)) (http.Response, error) {
	output := make(chan http.Response, 1)          // Declare the channel where the hystrix goroutine will put success responses.
	errs := hystrix.Go(createPaymentAcquiringBank, // Pass the name of the circuit breaker as first parameter.

		// 2nd parameter, the inlined func to run inside the breaker.
		func() error {
			// For hystrix, forward the err from the retrier. It's nil if successful.
			return d.callBankWithRetries(payment, callBackFn, output)
		},

		// 3rd parameter, the fallback func. In this case, we just do a bit of logging and return the error.
		func(err error) error {
			d.logger.Error("In fallback function for breaker", zap.String("breaker_name", createPaymentAcquiringBank), zap.Error(err))
			circuit, _, _ := hystrix.GetCircuit(createPaymentAcquiringBank)
			d.logger.Info("Circuit state is", zap.Bool("is_open", circuit.IsOpen()))
			return err
		})

	// Response and error handling. If the call was successful, the output channel gets the response. Otherwise,
	// the errors channel gives us the error.
	select {
	case out := <-output:
		d.logger.Info("Call in breaker successful", zap.String("breaker name", createPaymentAcquiringBank))
		if out.StatusCode < 299 {
			return out, nil
		}
		return out, fmt.Errorf("payment failed to get created on acquring bank, status: %d", out.StatusCode)
	case err := <-errs:
		return http.Response{}, err
	}
}

func (d *Domain) callBankWithRetries(payment payment_gateway.Payment, callBackFn func(payment_gateway.Payment), output chan http.Response) error {
	// Create a retrier with constant backoff, RETRIES number of attempts (3) with a 100ms sleep between retries.
	r := retrier.New(retrier.ConstantBackoff(retries, 100*time.Millisecond), nil)

	// This counter is just for getting some logging for showcasing, remove in production code.
	attempt := 0

	// Retrier works similar to hystrix, we pass the actual work (doing the HTTP request) in a func.
	err := r.Run(func() error {
		attempt++
		var err error
		// Do the mock request and handle response. If successful, pass resp over output channel,
		// otherwise, do a bit of error logging and return to err.
		resp := d.bankClient.CreatePayment(payment, callBackFn)
		// Retry only for 500 codes as it is not almost impossible to recover from a 4xx
		if resp.StatusCode < 299 || (resp.StatusCode > 399 && resp.StatusCode < 500) {
			output <- resp
			return nil
		} else {
			err = fmt.Errorf("status was %v", resp.StatusCode)
		}

		d.logger.Error(fmt.Sprintf("retrier failed, attempt %v", attempt))
		return err
	})
	return err
}
