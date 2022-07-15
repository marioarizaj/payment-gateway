package acquiringbank

import (
	"errors"
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

func (p *paymentsStore) Set(key string, value payment_gateway.Payment) {
	p.lock.Lock()
	p.cache[key] = value
	p.lock.Unlock()
}

func (p *paymentsStore) Get(key string) (payment_gateway.Payment, error) {
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
	newStatus                   string
	SleepIntervalInitialRequest time.Duration
	SleepIntervalForCallback    time.Duration
	ShouldRunCallback           bool
}

func (c *MockClient) CreatePayment(payment payment_gateway.Payment, callBack func(payment payment_gateway.Payment)) http.Response {
	time.Sleep(c.SleepIntervalInitialRequest)
	go func() {
		time.Sleep(c.SleepIntervalForCallback)
		if c.ShouldRunCallback {
			if c.newStatus == "" {
				payment.PaymentStatus = "success"
			}
			c.paymentsStore.Set(payment.ID.String(), payment)
			callBack(payment)
		}
	}()
	return http.Response{
		StatusCode: c.StatusCode,
	}
}
