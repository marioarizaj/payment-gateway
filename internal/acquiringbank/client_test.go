package acquiringbank_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/marioarizaj/payment-gateway"
	"github.com/marioarizaj/payment-gateway/internal/acquiringbank"
	"github.com/marioarizaj/payment-gateway/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMockClient_CreatePayment(t *testing.T) {
	signalChan := make(chan payment_gateway.Payment)
	cases := []struct {
		name         string
		inputPayment payment_gateway.Payment
		callback     func(payment payment_gateway.Payment)
		bankConfig   config.MockBankConfig
	}{
		{
			name: "create_payment_success",
			inputPayment: payment_gateway.Payment{
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
			},
			callback: func(payment payment_gateway.Payment) {
				signalChan <- payment
			},
			bankConfig: config.MockBankConfig{
				StatusCode:                  202,
				UpdateToStatus:              "succeeded",
				SleepIntervalInitialRequest: 1,
				SleepIntervalForCallback:    10,
				ShouldRunCallback:           true,
			},
		},
		{
			name: "create_payment_failed",
			inputPayment: payment_gateway.Payment{
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
			},
			callback: func(payment payment_gateway.Payment) {
				signalChan <- payment
			},
			bankConfig: config.MockBankConfig{
				StatusCode:                  400,
				UpdateToStatus:              "failed",
				SleepIntervalInitialRequest: 1,
				SleepIntervalForCallback:    10,
				ShouldRunCallback:           false,
			},
		},
		{
			name: "create_payment_no_callback",
			inputPayment: payment_gateway.Payment{
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
			},
			callback: func(payment payment_gateway.Payment) {
				signalChan <- payment
			},
			bankConfig: config.MockBankConfig{
				StatusCode:                  400,
				UpdateToStatus:              "failed",
				SleepIntervalInitialRequest: 1,
				SleepIntervalForCallback:    10,
				ShouldRunCallback:           true,
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mockBank := acquiringbank.NewMockClient(c.bankConfig)
			res := mockBank.CreatePayment(c.inputPayment, c.callback)
			assert.Equal(t, c.bankConfig.StatusCode, res.StatusCode)
			if c.bankConfig.ShouldRunCallback {
				returnedPayment := <-signalChan
				assert.Equal(t, c.bankConfig.UpdateToStatus, returnedPayment.PaymentStatus)
			} else {
				select {
				case <-signalChan:
					t.Error("signal was sent to channel when callback disabled")
				case <-time.After(50 * time.Millisecond):
					// Here means success
					return
				}
			}
		})
	}
	close(signalChan)
}
