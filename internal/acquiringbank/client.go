package acquiringbank

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/marioarizaj/payment_gateway/internal/config"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/marioarizaj/payment_gateway"
)

var notFoundError = errors.New("not found")

type paymentsStore struct {
	cache map[string]payment_gateway.Payment
	lock  *sync.Mutex
}

func (p *paymentsStore) set(key string, value payment_gateway.Payment) {
	p.lock.Lock()
	p.cache[key] = value
	p.lock.Unlock()
}

func (p *paymentsStore) get(key string) (payment_gateway.Payment, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if value, found := p.cache[key]; found {
		return value, nil
	}
	return payment_gateway.Payment{}, notFoundError
}

type MockClient struct {
	paymentsStore               paymentsStore
	StatusCode                  int
	FailedReason                string
	NewStatus                   string
	SleepIntervalInitialRequest time.Duration
	SleepIntervalForCallback    time.Duration
	ShouldRunCallback           bool
}

func NewMockClient(cfg config.MockBankConfig) *MockClient {
	return &MockClient{
		paymentsStore: paymentsStore{
			cache: map[string]payment_gateway.Payment{},
			lock:  &sync.Mutex{},
		},
		StatusCode:                  cfg.StatusCode,
		NewStatus:                   cfg.UpdateToStatus,
		SleepIntervalInitialRequest: time.Duration(cfg.SleepIntervalInitialRequest) * time.Millisecond,
		SleepIntervalForCallback:    time.Duration(cfg.SleepIntervalForCallback) * time.Millisecond,
		ShouldRunCallback:           cfg.ShouldRunCallback,
		FailedReason:                cfg.FailedReason,
	}
}

func (c *MockClient) CreatePayment(payment payment_gateway.Payment, callBack func(payment payment_gateway.Payment)) http.Response {
	time.Sleep(c.SleepIntervalInitialRequest)
	go func() {
		time.Sleep(c.SleepIntervalForCallback)
		if c.ShouldRunCallback {
			payment.PaymentStatus = c.NewStatus
			if payment.PaymentStatus == "" {
				payment.PaymentStatus = "success"
			}
			if c.FailedReason != "" {
				payment.FailedReason = c.FailedReason
			}
			c.paymentsStore.set(payment.ID.String(), payment)
			callBack(payment)
		}
	}()
	return http.Response{
		StatusCode: c.StatusCode,
		Body:       io.NopCloser(bytes.NewBuffer([]byte(fmt.Sprintf(`{"status": "%d"}`, c.StatusCode)))),
	}
}
