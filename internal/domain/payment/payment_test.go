package payment_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/marioarizaj/payment-gateway/internal/acquiringbank"

	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/marioarizaj/payment-gateway"
	"github.com/marioarizaj/payment-gateway/internal/config"
	"github.com/marioarizaj/payment-gateway/internal/dependencies"
	"github.com/marioarizaj/payment-gateway/internal/domain/payment"
	"github.com/marioarizaj/payment-gateway/internal/repositiory"
	"github.com/marioarizaj/payment-gateway/kit/rediscache"
	"github.com/marioarizaj/payment-gateway/kit/responses"
	"github.com/stretchr/testify/assert"
)

var baseTestPayment = payment_gateway.Payment{
	ID:            uuid.Must(uuid.Parse("b5f9c307-5202-4c52-aba9-752167eef9bf")),
	MerchantID:    uuid.Must(uuid.Parse("6c5a19d0-f132-4a55-93d3-2c00db06d41b")),
	PaymentStatus: "processing",
	Amount: payment_gateway.Amount{
		AmountFractional: 2000,
		CurrencyCode:     "USD",
	},
	Description: "Payment test",
	CardInfo: payment_gateway.CardInfo{
		CardName:    "Mario Arizaj",
		CardNumber:  "378282246310005",
		ExpiryMonth: 10,
		ExpiryYear:  22,
		CVV:         "123",
	},
}

func TestDomain_CreatePayment(t *testing.T) {
	cfg, err := config.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	deps, err := dependencies.InitDependencies(cfg)
	if !assert.NoError(t, err) {
		return
	}

	cases := []struct {
		name                 string
		payment              func(domain payment.Domain) (payment_gateway.Payment, error)
		expectedError        error
		sleepTime            time.Duration
		mockConfig           config.MockBankConfig
		expectedStatus       string
		expectedFailedReason string
		shouldCreateRecord   bool
	}{
		{
			name:      "create_payment_success",
			sleepTime: time.Second,
			mockConfig: config.MockBankConfig{
				StatusCode:                  202,
				UpdateToStatus:              "succeeded",
				SleepIntervalInitialRequest: 10,
				SleepIntervalForCallback:    50,
				ShouldRunCallback:           true,
			},
			shouldCreateRecord: true,
			payment: func(domain payment.Domain) (payment_gateway.Payment, error) {
				p := baseTestPayment
				return p, nil
			},
			expectedStatus: "succeeded",
		},
		{
			name:               "create_payment_success_failing_acquiring_bank_async",
			sleepTime:          2 * time.Second,
			shouldCreateRecord: true,
			mockConfig: config.MockBankConfig{
				StatusCode:                  202,
				UpdateToStatus:              "failed",
				SleepIntervalInitialRequest: 10,
				SleepIntervalForCallback:    50,
				ShouldRunCallback:           true,
				FailedReason:                "no sufficient funds",
			},
			payment: func(domain payment.Domain) (payment_gateway.Payment, error) {
				p := baseTestPayment
				return p, nil
			},
			expectedStatus:       "failed",
			expectedFailedReason: "no sufficient funds",
		},
		{
			name:               "create_payment_success_failing_acquiring_bank_sync",
			sleepTime:          3 * time.Second,
			shouldCreateRecord: false,
			mockConfig: config.MockBankConfig{
				StatusCode:                  400,
				UpdateToStatus:              "failed",
				SleepIntervalInitialRequest: 10,
				SleepIntervalForCallback:    50,
				ShouldRunCallback:           false,
			},
			expectedError: responses.BadRequestError{Err: errors.New("payment failed to get created on acquring bank, status: 400")},
			payment: func(domain payment.Domain) (payment_gateway.Payment, error) {
				p := baseTestPayment
				return p, nil
			},
		},
		{
			name:               "create_payment_timeout_circuit_breaker",
			sleepTime:          3 * time.Second,
			shouldCreateRecord: false,
			mockConfig: config.MockBankConfig{
				UpdateToStatus:              "failed",
				SleepIntervalInitialRequest: 100000,
				ShouldRunCallback:           false,
			},
			expectedError: responses.InternalServerError{Err: errors.New("fallback failed with 'hystrix: timeout'. run error was 'hystrix: timeout'")},
			payment: func(domain payment.Domain) (payment_gateway.Payment, error) {
				p := baseTestPayment
				return p, nil
			},
			expectedStatus: "failed",
		},
		{
			name:               "create_payment_timeout_circuit_breaker_retry",
			sleepTime:          3 * time.Second,
			shouldCreateRecord: false,
			mockConfig: config.MockBankConfig{
				StatusCode:                  500,
				UpdateToStatus:              "failed",
				SleepIntervalInitialRequest: 10,
				ShouldRunCallback:           false,
			},
			expectedError: responses.InternalServerError{Err: errors.New("fallback failed with 'status was 500'. run error was 'status was 500'")},
			payment: func(domain payment.Domain) (payment_gateway.Payment, error) {
				p := baseTestPayment
				return p, nil
			},
			expectedStatus: "failed",
		},
		{
			name:               "create_payment_invalid_card",
			shouldCreateRecord: false,
			mockConfig: config.MockBankConfig{
				StatusCode:                  500,
				UpdateToStatus:              "failed",
				SleepIntervalInitialRequest: 10,
				ShouldRunCallback:           false,
			},
			payment: func(domain payment.Domain) (payment_gateway.Payment, error) {
				p := baseTestPayment
				p.CardInfo.ExpiryYear = 21
				return p, nil
			},
			expectedError: responses.BadRequestError{Err: errors.New("credit card has expired")},
		},
		{
			name:               "create_payment_duplicate_transaction",
			sleepTime:          2 * time.Second,
			shouldCreateRecord: true,
			mockConfig: config.MockBankConfig{
				StatusCode:                  202,
				UpdateToStatus:              "succeeded",
				SleepIntervalInitialRequest: 10,
				SleepIntervalForCallback:    50,
				ShouldRunCallback:           true,
			},
			payment: func(domain payment.Domain) (payment_gateway.Payment, error) {
				ctx := context.Background()
				p := baseTestPayment

				createdPayment, err := domain.CreatePayment(ctx, p)
				if err != nil {
					return payment_gateway.Payment{}, err
				}
				// Now let's change the amount, but keep the same id, so we pass redis validation, but fail on the db one
				createdPayment.Amount.AmountFractional = 3000
				createdPayment.CardInfo.CVV = p.CardInfo.CVV
				return createdPayment, nil
			},
			expectedError: responses.ConflictError{},
		},
		{
			name:               "create_payment_duplicate_transaction_caught_in_redis",
			mockConfig:         cfg.MockBankConfig,
			shouldCreateRecord: false,
			payment: func(domain payment.Domain) (payment_gateway.Payment, error) {
				ctx := context.Background()
				p := baseTestPayment
				createdPayment, err := domain.CreatePayment(ctx, p)
				if err != nil {
					return payment_gateway.Payment{}, err
				}
				// Now let's change the amount, but keep the same id, so we pass redis validation, but fail on the db one
				createdPayment.CardInfo.CVV = p.CardInfo.CVV
				createdPayment.ID = uuid.New()
				return createdPayment, nil
			},
			expectedError: responses.ConflictError{},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			deps.BankClient = acquiringbank.NewMockClient(c.mockConfig)
			d, cleanFn, err := getDomain(deps)
			if !assert.NoError(t, err) {
				return
			}
			defer cleanFn()
			p, err := c.payment(*d)
			if err != nil {
				t.Fatal(err)
				return
			}
			createdPayment, err := d.CreatePayment(context.Background(), p)
			if c.expectedError != nil {
				assert.Equal(t, c.expectedError, err)
				if !c.shouldCreateRecord {
					// Check that the record is not on database
					_, err = d.GetPayment(context.Background(), p.ID)
					assert.Equal(t, responses.NotFoundError{}, err)
				}
				return
			}
			// First verify that CVV is not returned
			assert.Empty(t, createdPayment.CardInfo.CVV)
			// Set the CVV to empty for comparison purposes
			p.CardInfo.CVV = ""
			// Force equal timestamps
			p.CreatedAt = createdPayment.CreatedAt
			p.UpdatedAt = createdPayment.UpdatedAt

			assert.Equal(t, p, createdPayment)
			time.Sleep(time.Second)
			// Let's wait a second, for the callback to update the database
			newPayment, err := d.GetPayment(context.Background(), p.ID)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, c.expectedStatus, newPayment.PaymentStatus)
			assert.Equal(t, c.expectedFailedReason, newPayment.FailedReason)
		})
	}
}

func TestDomain_GetPayment(t *testing.T) {
	cfg, err := config.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	deps, err := dependencies.InitDependencies(cfg)
	if !assert.NoError(t, err) {
		return
	}
	cases := []struct {
		name          string
		idToSearch    uuid.UUID
		payment       payment_gateway.Payment
		expectedError error
	}{
		{
			name:       "get_payment_success",
			payment:    baseTestPayment,
			idToSearch: uuid.Must(uuid.Parse("b5f9c307-5202-4c52-aba9-752167eef9bf")),
		}, {
			name:          "get_payment_not_found",
			payment:       baseTestPayment,
			idToSearch:    uuid.Must(uuid.Parse("c5693980-e5f1-4a20-8a2b-bd13ce9f460f")),
			expectedError: responses.NotFoundError{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, cleanFn, err := getDomain(deps)
			if !assert.NoError(t, err) {
				return
			}
			defer cleanFn()
			err = InsertTestPayment(d, c.payment)
			if !assert.NoError(t, err) {
				return
			}
			p, err := d.GetPayment(context.Background(), c.idToSearch)
			if c.expectedError != nil {
				assert.Equal(t, c.expectedError, err)
				return
			}
			assert.NoError(t, err)
			// Force equal timestamps
			c.payment.CreatedAt = p.CreatedAt
			c.payment.UpdatedAt = p.UpdatedAt
			p.CardInfo.CVV = c.payment.CardInfo.CVV
			assert.Equal(t, c.payment, p)
			// Verify that when we search for a second time, and cache returns the result,
			// it returns the correct result
			p, err = d.GetPayment(context.Background(), c.idToSearch)
			if c.expectedError != nil {
				assert.Equal(t, c.expectedError, err)
				return
			}
			assert.NoError(t, err)
			// Force equal timestamps
			c.payment.CreatedAt = p.CreatedAt
			c.payment.UpdatedAt = p.UpdatedAt
			p.CardInfo.CVV = c.payment.CardInfo.CVV
			assert.Equal(t, c.payment, p)
		})
	}
}

func getDomain(deps dependencies.Dependencies) (*payment.Domain, func(), error) {
	ctx := context.Background()
	tx, err := deps.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, nil, err
	}
	repo := repositiory.NewRepository(tx)
	redisCache := rediscache.NewRedisClient(deps.Redis)
	d := payment.NewDomain(repo, redisCache, zap.NewNop(), deps.BankClient)
	return d, func() {
		_ = tx.Rollback()
		deps.Redis.FlushAll(ctx)
	}, nil
}

func InsertTestPayment(domain *payment.Domain, payment payment_gateway.Payment) error {
	_, err := domain.CreatePayment(context.Background(), payment)
	return err
}
