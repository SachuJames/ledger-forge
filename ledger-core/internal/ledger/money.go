package ledger

import (
	"errors"
	"fmt"
	"strings"
)

// Money is an amount in minor units (cents, paise, etc.). Storing money as
// an integer avoids every class of float rounding bug; the currency tells
// you where the decimal point goes.
type Money int64

// currencies maps ISO 4217 codes to their minor-unit exponent. Only
// currencies in this table are accepted, which keeps bad input out early.
var currencies = map[string]int{
	"USD": 2, "EUR": 2, "INR": 2, "GBP": 2, "JPY": 0, "AED": 2,
	"SGD": 2, "AUD": 2, "CAD": 2, "CHF": 2, "CNY": 2, "BRL": 2,
	"KRW": 0, "ZAR": 2, "NGN": 2, "KES": 2, "SAR": 2, "QAR": 2,
	"KWD": 3, "BHD": 3, "OMR": 3,
}

// ValidCurrency reports whether code is a supported ISO 4217 currency.
func ValidCurrency(code string) bool {
	_, ok := currencies[strings.ToUpper(code)]
	return ok
}

// FormatMoney renders minor units as a human string, e.g. 1050 INR -> "10.50".
func FormatMoney(m Money, currency string) string {
	exp := currencies[strings.ToUpper(currency)]
	if exp == 0 {
		return fmt.Sprintf("%d", m)
	}
	div := int64(1)
	for i := 0; i < exp; i++ {
		div *= 10
	}
	neg := ""
	if m < 0 {
		neg, m = "-", -m
	}
	return fmt.Sprintf("%s%d.%0*d", neg, int64(m)/div, exp, int64(m)%div)
}

// NormalizeCurrency upper-cases the code and rejects anything unsupported.
func NormalizeCurrency(code string) (string, error) {
	c := strings.ToUpper(strings.TrimSpace(code))
	if !ValidCurrency(c) {
		return "", errors.New("ledger: unsupported currency " + code)
	}
	return c, nil
}
