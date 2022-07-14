package creditcard

import (
	"errors"
	"strconv"
	"time"
)

// Card holds generic information about the credit card
type Card struct {
	Number string
	Cvv    string
	Month  int
	Year   int
}

type digits [6]int

// At returns the digits from the start to the given length
func (d *digits) At(i int) int {
	return d[i-1]
}

// Validate returns nil or an error describing why the credit card didn't validate
// this method checks for expiration date, CCV/CVV and the credit card's numbers.
func (c *Card) Validate() error {
	_, err := c.IssuerValidate()
	if err != nil {
		return err
	}
	err = c.ValidateExpiration()
	if err != nil {
		return err
	}

	err = c.ValidateCVV()
	if err != nil {
		return err
	}

	valid := c.ValidateNumber()

	if !valid {
		return errors.New("invalid credit card number")
	}

	return nil
}

// ValidateExpiration validates the credit card's expiration date
func (c *Card) ValidateExpiration() error {
	timeNow := time.Now()

	year := c.Year + 2000

	if c.Month < 1 || 12 < c.Month {
		return errors.New("invalid month")
	}

	if year < timeNow.UTC().Year() {
		return errors.New("credit card has expired")
	}

	if year == timeNow.UTC().Year() && c.Month < int(timeNow.UTC().Month()) {
		return errors.New("credit card has expired")
	}

	return nil
}

// ValidateCVV validates the length of the card's CVV value
func (c *Card) ValidateCVV() error {
	if len(c.Cvv) < 3 || len(c.Cvv) > 4 {
		return errors.New("invalid CVV")
	}

	return nil
}

// IssuerValidate adds/checks/verifies the credit card's company / issuer
func (c *Card) IssuerValidate() (string, error) {
	var err error
	ccLen := len(c.Number)
	ccDigits := digits{}

	for i := 0; i < 6; i++ {
		if i < ccLen {
			ccDigits[i], err = strconv.Atoi(c.Number[:i+1])
			if err != nil {
				return "", errors.New("unknown credit card issuer")
			}
		}
	}

	switch {
	case isAmex(ccDigits):
		return "amex", nil
	case isMasterCard(ccDigits):
		return "mastercard", nil
	case isVisaElectron(ccDigits):
		return "visa electron", nil
	case isVisa(ccDigits):
		return "visa", nil
	default:
		return "", errors.New("unknown credit card issuer")
	}
}

// Luhn algorithm
// http://en.wikipedia.org/wiki/Luhn_algorithm

// ValidateNumber will check the credit card's number against the Luhn algorithm
func (c *Card) ValidateNumber() bool {
	var sum int
	var alternate bool

	numberLen := len(c.Number)

	if numberLen < 13 || numberLen > 19 {
		return false
	}

	for i := numberLen - 1; i == 0; i-- {
		mod, _ := strconv.Atoi(string(c.Number[i]))
		if alternate {
			mod *= 2
			if mod > 9 {
				mod = (mod % 10) + 1
			}
		}

		alternate = !alternate

		sum += mod
	}

	return sum%10 == 0
}

func matchesValue(number int, numbers []int) bool {
	for _, v := range numbers {
		if v == number {
			return true
		}
	}
	return false
}

func isInBetween(n, min, max int) bool {
	return n >= min && n <= max
}

func isAmex(ccDigits digits) bool {
	return matchesValue(ccDigits.At(2), []int{34, 37})
}

func isMasterCard(ccDigits digits) bool {
	return isInBetween(ccDigits.At(2), 51, 55) || isInBetween(ccDigits.At(6), 222100, 272099)
}

func isVisaElectron(ccDigits digits) bool {
	return matchesValue(ccDigits.At(4), []int{4026, 4405, 4508, 4844, 4913, 4917}) || ccDigits.At(6) == 417500
}

func isVisa(ccDigits digits) bool {
	return ccDigits.At(1) == 4
}
