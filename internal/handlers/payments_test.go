package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/marioarizaj/payment_gateway"
	"github.com/marioarizaj/payment_gateway/internal/config"
	"github.com/marioarizaj/payment_gateway/internal/dependencies"
	"github.com/marioarizaj/payment_gateway/internal/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
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

func TestHandler_CreatePayment(t *testing.T) {
	cfg, err := config.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}

	cases := []struct {
		name                 string
		payload              func() []byte
		expectedCode         int
		expectedErrorMessage string
		username             string
		password             string
	}{
		{
			name: "create_payment_success",
			payload: func() []byte {
				bts, _ := json.Marshal(baseTestPayment)
				return bts
			},
			expectedCode: http.StatusCreated,
			username:     "6c5a19d0-f132-4a55-93d3-2c00db06d41b",
			password:     "a7898e515691064b49a15a01e69503f83cd918594e643cc3e949adef273b309f",
		},
		{
			name: "create_payment_invalid_card",
			payload: func() []byte {
				p := baseTestPayment
				p.CardInfo.ExpiryYear = 21
				bts, _ := json.Marshal(p)
				return bts
			},
			expectedCode:         http.StatusBadRequest,
			expectedErrorMessage: "credit card has expired",
			username:             "6c5a19d0-f132-4a55-93d3-2c00db06d41b",
			password:             "a7898e515691064b49a15a01e69503f83cd918594e643cc3e949adef273b309f",
		},
		{
			name: "create_payment_dummy_payload",
			payload: func() []byte {
				bts, _ := json.Marshal(baseTestPayment)
				return bts
			},
			expectedCode:         http.StatusUnauthorized,
			expectedErrorMessage: "unauthorized",
			username:             "test-user",
			password:             "test-pass",
		},
		{
			name: "create_payment_unauthorized",
			payload: func() []byte {
				bts, _ := json.Marshal(`dummy: "true"`)
				return bts
			},
			expectedCode:         http.StatusBadRequest,
			expectedErrorMessage: "could not decode request body",
			username:             "6c5a19d0-f132-4a55-93d3-2c00db06d41b",
			password:             "a7898e515691064b49a15a01e69503f83cd918594e643cc3e949adef273b309f",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			deps, err := dependencies.InitDependencies(cfg)
			if !assert.NoError(t, err) {
				return
			}
			deps.DB, err = deps.DB.BeginTx(context.Background(), &sql.TxOptions{})
			if !assert.NoError(t, err) {
				return
			}
			defer func() { _ = cleanupFunc(deps.DB.(bun.Tx), deps.Redis) }()
			r := handlers.NewRouter(cfg, deps, zap.NewNop())
			req, err := http.NewRequest(http.MethodPost, "/v1/payments", bytes.NewBuffer(c.payload()))
			if !assert.NoError(t, err) {
				return
			}
			req.Header.Add("Authorization", fmt.Sprintf("Basic %s", basicAuth(c.username, c.password)))
			res := executeRequest(r, req)
			assert.Equal(t, c.expectedCode, res.Code)
			var resBody map[string]interface{}
			err = json.NewDecoder(res.Body).Decode(&resBody)
			if !assert.NoError(t, err) {
				return
			}
			if res.Code > 300 {
				errMsg := resBody["error"].(string)
				assert.Equal(t, c.expectedErrorMessage, errMsg)
				return
			}
		})
	}
}

func TestHandler_GetPayment(t *testing.T) {
	cfg, err := config.LoadConfig()
	if !assert.NoError(t, err) {
		return
	}

	cases := []struct {
		name                 string
		expectedCode         int
		testPayment          payment_gateway.Payment
		expectedErrorMessage string
		username             string
		password             string
		idToSearch           string
	}{
		{
			name:         "get_payment_success",
			testPayment:  baseTestPayment,
			idToSearch:   "b5f9c307-5202-4c52-aba9-752167eef9bf",
			expectedCode: http.StatusOK,
			username:     "6c5a19d0-f132-4a55-93d3-2c00db06d41b",
			password:     "a7898e515691064b49a15a01e69503f83cd918594e643cc3e949adef273b309f",
		},
		{
			name:                 "get_payment_wrong_id_format",
			testPayment:          baseTestPayment,
			idToSearch:           "1234",
			expectedCode:         http.StatusBadRequest,
			expectedErrorMessage: "id format not accurate",
			username:             "6c5a19d0-f132-4a55-93d3-2c00db06d41b",
			password:             "a7898e515691064b49a15a01e69503f83cd918594e643cc3e949adef273b309f",
		},
		{
			name:                 "get_payment_unauthorized",
			testPayment:          baseTestPayment,
			idToSearch:           "b5f9c307-5202-4c52-aba9-752167eef9bf",
			expectedCode:         http.StatusUnauthorized,
			expectedErrorMessage: "unauthorized",
			username:             "test-username",
			password:             "test-password",
		},
		{
			name:                 "get_payment_not_found",
			testPayment:          baseTestPayment,
			idToSearch:           "b5f9c307-5202-4c52-aba9-752167eef8bf",
			expectedCode:         http.StatusNotFound,
			expectedErrorMessage: "not found",
			username:             "6c5a19d0-f132-4a55-93d3-2c00db06d41b",
			password:             "a7898e515691064b49a15a01e69503f83cd918594e643cc3e949adef273b309f",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			deps, err := dependencies.InitDependencies(cfg)
			if !assert.NoError(t, err) {
				return
			}
			deps.DB, err = deps.DB.BeginTx(context.Background(), &sql.TxOptions{})
			if !assert.NoError(t, err) {
				return
			}
			defer func() { _ = cleanupFunc(deps.DB.(bun.Tx), deps.Redis) }()
			err = InsertTestPayment(deps.DB, c.testPayment)
			if !assert.NoError(t, err) {
				return
			}
			r := handlers.NewRouter(cfg, deps, zap.NewNop())
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/v1/payments/%s", c.idToSearch), nil)
			if !assert.NoError(t, err) {
				return
			}
			req.Header.Add("Authorization", fmt.Sprintf("Basic %s", basicAuth(c.username, c.password)))
			res := executeRequest(r, req)
			assert.Equal(t, c.expectedCode, res.Code)
			fmt.Println()
			fmt.Println()
			fmt.Println(res.Body.String())
			fmt.Println()
			fmt.Println()
			if res.Code > 300 {
				var resBody map[string]interface{}
				err = json.NewDecoder(res.Body).Decode(&resBody)
				if !assert.NoError(t, err) {
					return
				}
				errMsg := resBody["error"].(string)
				assert.Equal(t, c.expectedErrorMessage, errMsg)
				return
			}
			var actual payment_gateway.Payment
			err = json.NewDecoder(res.Body).Decode(&actual)
			if !assert.NoError(t, err) {
				return
			}
			compareResults(t, c.testPayment, actual)
		})
	}
}

func compareResults(t *testing.T, expected payment_gateway.Payment, actual payment_gateway.Payment) {
	// We force the equalisation of these values since they are variable
	expected.CreatedAt = actual.CreatedAt
	expected.UpdatedAt = actual.UpdatedAt
	expected.CardInfo.CVV = ""

	assert.Equal(t, expected, actual)
}

func cleanupFunc(tx bun.Tx, redis *redis.Client) error {
	redis.FlushAll(context.Background())
	return tx.Rollback()
}

func executeRequest(r *mux.Router, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func InsertTestPayment(db bun.IDB, payment payment_gateway.Payment) error {
	_, err := db.NewInsert().Model(payment.GetStoragePayment()).Exec(context.Background())
	return err
}
