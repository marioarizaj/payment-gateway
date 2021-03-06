package repositiory_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/marioarizaj/payment-gateway/internal/config"
	"github.com/marioarizaj/payment-gateway/internal/dependencies"
	"github.com/marioarizaj/payment-gateway/internal/repositiory"
	"github.com/stretchr/testify/assert"
)

func TestRepo_CreatePayment(t *testing.T) {
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
		payment       *repositiory.Payment
		expectedError error
	}{
		{
			name: "insert_payment_success",
			payment: &repositiory.Payment{
				ID:              uuid.Must(uuid.Parse("b5f9c307-5202-4c52-aba9-752167eef9bf")),
				Amount:          2000,
				PaymentStatus:   "processing",
				MerchantID:      uuid.Must(uuid.Parse("6c5a19d0-f132-4a55-93d3-2c00db06d41b")),
				CurrencyCode:    "USD",
				Description:     "Payment test",
				CardName:        "Mario Arizaj",
				CardNumber:      "378282246310005",
				CardExpiryMonth: 10,
				CardExpiryYear:  22,
			},
		}, {
			name: "insert_payment_wrong_merchant_id",
			payment: &repositiory.Payment{
				ID:              uuid.Must(uuid.Parse("e2cf99cf-02a6-47f7-ad74-fa6245852176")),
				Amount:          2000,
				PaymentStatus:   "processing",
				MerchantID:      uuid.Must(uuid.Parse("f53718ed-cce8-4e4f-89e0-44626069e9cf")),
				CurrencyCode:    "USD",
				Description:     "Payment test",
				CardName:        "Mario Arizaj",
				CardNumber:      "378282246310005",
				CardExpiryMonth: 10,
				CardExpiryYear:  22,
			},
			expectedError: errors.New("ERROR: insert or update on table \"payments\" violates foreign key constraint \"payments_merchant_id_fkey\" (SQLSTATE=23503)"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			tx, err := deps.DB.BeginTx(ctx, &sql.TxOptions{})
			if !assert.NoError(t, err) {
				return
			}
			defer func() { _ = tx.Rollback() }()
			repo := repositiory.NewRepository(tx)
			err = repo.CreatePayment(ctx, c.payment)
			if c.expectedError != nil {
				assert.Equal(t, c.expectedError.Error(), err.Error())
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestRepo_GetPaymentByID(t *testing.T) {
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
		payment       *repositiory.Payment
		expectedError error
	}{
		{
			name: "get_payment_success",
			payment: &repositiory.Payment{
				ID:              uuid.Must(uuid.Parse("b5f9c307-5202-4c52-aba9-752167eef9bf")),
				Amount:          2000,
				MerchantID:      uuid.Must(uuid.Parse("6c5a19d0-f132-4a55-93d3-2c00db06d41b")),
				CurrencyCode:    "USD",
				PaymentStatus:   "processing",
				Description:     "Payment test",
				CardName:        "Mario Arizaj",
				CardNumber:      "378282246310005",
				CardExpiryMonth: 10,
				CardExpiryYear:  22,
			},
			idToSearch: uuid.Must(uuid.Parse("b5f9c307-5202-4c52-aba9-752167eef9bf")),
		}, {
			name: "get_payment_not_found",
			payment: &repositiory.Payment{
				ID:              uuid.Must(uuid.Parse("b5f9c307-5202-4c52-aba9-752167eef9bf")),
				Amount:          2000,
				MerchantID:      uuid.Must(uuid.Parse("6c5a19d0-f132-4a55-93d3-2c00db06d41b")),
				CurrencyCode:    "USD",
				PaymentStatus:   "processing",
				Description:     "Payment test",
				CardName:        "Mario Arizaj",
				CardNumber:      "378282246310005",
				CardExpiryMonth: 10,
				CardExpiryYear:  22,
			},
			idToSearch:    uuid.Must(uuid.Parse("c5693980-e5f1-4a20-8a2b-bd13ce9f460f")),
			expectedError: errors.New("sql: no rows in result set"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			tx, err := deps.DB.BeginTx(ctx, &sql.TxOptions{})
			if !assert.NoError(t, err) {
				return
			}
			defer func() { _ = tx.Rollback() }()
			repo := repositiory.NewRepository(tx)
			err = InsertTestPayment(repo, c.payment)
			if !assert.NoError(t, err) {
				return
			}
			payment, err := repo.GetPaymentByID(ctx, c.idToSearch)
			if c.expectedError != nil {
				assert.Equal(t, c.expectedError.Error(), err.Error())
				return
			}

			assert.NoError(t, err)
			// Force equal timestamps
			c.payment.CreatedAt = payment.CreatedAt
			c.payment.UpdatedAt = payment.UpdatedAt
			assert.Equal(t, c.payment, payment)
		})
	}
}

func TestRepo_UpdateStatus(t *testing.T) {
	cfg, err := config.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}
	deps, err := dependencies.InitDependencies(cfg)
	if !assert.NoError(t, err) {
		return
	}
	cases := []struct {
		name               string
		idToSearch         uuid.UUID
		newStatus          string
		newReason          string
		payment            *repositiory.Payment
		expectedError      error
		shouldUpdateStatus bool
	}{
		{
			name:               "update_payment_success",
			newStatus:          "succeeded",
			shouldUpdateStatus: true,
			payment: &repositiory.Payment{
				ID:              uuid.Must(uuid.Parse("b5f9c307-5202-4c52-aba9-752167eef9bf")),
				Amount:          2000,
				MerchantID:      uuid.Must(uuid.Parse("6c5a19d0-f132-4a55-93d3-2c00db06d41b")),
				CurrencyCode:    "USD",
				PaymentStatus:   "processing",
				Description:     "Payment test",
				CardName:        "Mario Arizaj",
				CardNumber:      "378282246310005",
				CardExpiryMonth: 10,
				CardExpiryYear:  22,
			},
			idToSearch: uuid.Must(uuid.Parse("b5f9c307-5202-4c52-aba9-752167eef9bf")),
		},
		{
			name:               "updates_payment_to_failed_with_status",
			newStatus:          "failed",
			newReason:          "failed...",
			shouldUpdateStatus: true,
			payment: &repositiory.Payment{
				ID:              uuid.Must(uuid.Parse("b5f9c307-5202-4c52-aba9-752167eef9bf")),
				Amount:          2000,
				MerchantID:      uuid.Must(uuid.Parse("6c5a19d0-f132-4a55-93d3-2c00db06d41b")),
				CurrencyCode:    "USD",
				PaymentStatus:   "processing",
				Description:     "Payment test",
				CardName:        "Mario Arizaj",
				CardNumber:      "378282246310005",
				CardExpiryMonth: 10,
				CardExpiryYear:  22,
			},
			idToSearch: uuid.Must(uuid.Parse("b5f9c307-5202-4c52-aba9-752167eef9bf")),
		},
		{
			name:               "does_not_update_payment_with_success_status",
			newStatus:          "failed",
			shouldUpdateStatus: false,
			payment: &repositiory.Payment{
				ID:              uuid.Must(uuid.Parse("b5f9c307-5202-4c52-aba9-752167eef9bf")),
				Amount:          2000,
				PaymentStatus:   "success",
				MerchantID:      uuid.Must(uuid.Parse("6c5a19d0-f132-4a55-93d3-2c00db06d41b")),
				CurrencyCode:    "USD",
				Description:     "Payment test",
				CardName:        "Mario Arizaj",
				CardNumber:      "378282246310005",
				CardExpiryMonth: 10,
				CardExpiryYear:  22,
			},
			idToSearch: uuid.Must(uuid.Parse("b5f9c307-5202-4c52-aba9-752167eef9bf")),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			tx, err := deps.DB.BeginTx(ctx, &sql.TxOptions{})
			if !assert.NoError(t, err) {
				return
			}
			defer func() { _ = tx.Rollback() }()
			repo := repositiory.NewRepository(tx)
			err = InsertTestPayment(repo, c.payment)
			if !assert.NoError(t, err) {
				return
			}
			updatedPayment := *c.payment
			updatedPayment.PaymentStatus = c.newStatus
			updatedPayment.FailedReason = c.newReason
			err = repo.UpdateStatus(ctx, &updatedPayment)
			if c.expectedError != nil {
				assert.Equal(t, c.expectedError.Error(), err.Error())
				return
			}
			payment, err := repo.GetPaymentByID(ctx, c.payment.ID)
			if !assert.NoError(t, err) {
				return
			}
			if c.shouldUpdateStatus {
				assert.Equal(t, c.newStatus, payment.PaymentStatus)
				assert.Equal(t, c.newReason, payment.FailedReason)
			} else {
				assert.Equal(t, c.payment.PaymentStatus, payment.PaymentStatus)
				assert.Equal(t, c.payment.FailedReason, payment.FailedReason)
			}
		})
	}
}

func InsertTestPayment(repo repositiory.Repository, payment *repositiory.Payment) error {
	return repo.CreatePayment(context.Background(), payment)
}
