package acquiringbank

import (
	"net/http"
	"time"

	"github.com/marioarizaj/payment_gateway"
)

type MockClient struct {
	// paymentsCache               map[string]payment_gateway.Payment
	StatusCode                  int
	SleepIntervalInitialRequest time.Duration
	SleepIntervalForCallback    time.Duration
	ShouldRunCallback           bool
}

func (c *MockClient) CreatePayment(payment payment_gateway.Payment, callBack func()) http.Response {
	time.Sleep(c.SleepIntervalInitialRequest)
	go func() {
		time.Sleep(c.SleepIntervalForCallback)
		if c.ShouldRunCallback {
			callBack()
		}
	}()
	return http.Response{
		StatusCode: c.StatusCode,
	}
}
