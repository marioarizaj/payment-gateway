package creditcard_test

import (
	"errors"
	"testing"

	"github.com/marioarizaj/payment_gateway/internal/creditcard"
	"github.com/stretchr/testify/assert"
)

func TestCard_IssuerValidate(t *testing.T) {
	cases := []struct {
		name           string
		inputCard      *creditcard.Card
		expectedError  error
		expectedIssuer string
	}{
		{
			name: "validate_issuer_amex",
			inputCard: &creditcard.Card{
				Number: "378282246310005",
			},
			expectedIssuer: "amex",
		}, {
			name: "validate_issuer_mastercard",
			inputCard: &creditcard.Card{
				Number: "5555555555554444",
			},
			expectedIssuer: "mastercard",
		}, {
			name: "validate_issuer_visa_electron",
			inputCard: &creditcard.Card{
				Number: "4026111111111115",
			},
			expectedIssuer: "visa electron",
		}, {
			name: "validate_issuer_visa",
			inputCard: &creditcard.Card{
				Number: "4111111111111111",
			},
			expectedIssuer: "visa",
		}, {
			name: "validate_issuer_unsupported",
			inputCard: &creditcard.Card{
				Number: "6011111111111117",
			},
			expectedError: errors.New("unknown credit card issuer"),
		}, {
			name: "validate_issuer_unknown",
			inputCard: &creditcard.Card{
				Number: "a011111111111117",
			},
			expectedError: errors.New("unknown credit card issuer"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			issuer, err := c.inputCard.IssuerValidate()
			assert.Equal(t, c.expectedError, err)
			assert.Equal(t, c.expectedIssuer, issuer)
		})
	}
}

func TestCard_ValidateNumber(t *testing.T) {
	cases := []struct {
		name      string
		inputCard *creditcard.Card
		isValid   bool
	}{
		{
			name: "card_valid",
			inputCard: &creditcard.Card{
				Number: "378282246310005",
			},
			isValid: true,
		}, {
			name: "card_invalid",
			inputCard: &creditcard.Card{
				Number: "49927398717",
			},
			isValid: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			isValid := c.inputCard.ValidateNumber()
			assert.Equal(t, c.isValid, isValid)
		})
	}
}

func TestCard_ValidateExpiration(t *testing.T) {
	cases := []struct {
		name          string
		inputCard     *creditcard.Card
		expectedError error
	}{
		{
			name: "card_expiry_success",
			inputCard: &creditcard.Card{
				Month: 10,
				Year:  22,
			},
		}, {
			name: "card_expiry_expired_year",
			inputCard: &creditcard.Card{
				Month: 10,
				Year:  21,
			},
			expectedError: errors.New("credit card has expired"),
		}, {
			name: "card_expiry_expired_month",
			inputCard: &creditcard.Card{
				Month: 6,
				Year:  22,
			},
			expectedError: errors.New("credit card has expired"),
		}, {
			name: "card_expiry_invalid_month",
			inputCard: &creditcard.Card{
				Month: 13,
				Year:  22,
			},
			expectedError: errors.New("invalid month"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.inputCard.ValidateExpiration()
			assert.Equal(t, c.expectedError, err)
		})
	}
}

func TestCard_ValidateCVV(t *testing.T) {
	cases := []struct {
		name          string
		inputCard     *creditcard.Card
		expectedError error
	}{
		{
			name: "card_cvv_success",
			inputCard: &creditcard.Card{
				Cvv: "123",
			},
		}, {
			name: "card_cvv_success_4_digits",
			inputCard: &creditcard.Card{
				Cvv: "1234",
			},
		}, {
			name: "card_invalid_cvv",
			inputCard: &creditcard.Card{
				Cvv: "12",
			},
			expectedError: errors.New("invalid CVV"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.inputCard.ValidateCVV()
			assert.Equal(t, c.expectedError, err)
		})
	}
}

func TestCard_Validate(t *testing.T) {
	cases := []struct {
		name          string
		inputCard     *creditcard.Card
		expectedError error
	}{
		{
			name: "test_card_valid",
			inputCard: &creditcard.Card{
				Number: "378282246310005",
				Cvv:    "1234",
				Month:  10,
				Year:   22,
			},
		}, {
			name: "validate_unknown_issuer",
			inputCard: &creditcard.Card{
				Number: "6011111111111117",
				Cvv:    "1234",
				Month:  10,
				Year:   22,
			},
			expectedError: errors.New("unknown credit card issuer"),
		}, {
			name: "validate_card_expired",
			inputCard: &creditcard.Card{
				Number: "378282246310005",
				Cvv:    "123",
				Month:  10,
				Year:   21,
			},
			expectedError: errors.New("credit card has expired"),
		}, {
			name: "validate_invalid_cvv",
			inputCard: &creditcard.Card{
				Number: "378282246310005",
				Cvv:    "12",
				Month:  10,
				Year:   22,
			},
			expectedError: errors.New("invalid CVV"),
		}, {
			name: "validate_invalid_numbers",
			inputCard: &creditcard.Card{
				Number: "49927398717",
				Cvv:    "123",
				Month:  10,
				Year:   22,
			},
			expectedError: errors.New("invalid credit card number"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.inputCard.Validate()
			assert.Equal(t, c.expectedError, err)
		})
	}
}
