package ledger

import "testing"

func TestFormatMoney(t *testing.T) {
	cases := []struct {
		money    Money
		currency string
		want     string
	}{
		{1050, "INR", "10.50"},
		{1050, "inr", "10.50"},
		{5, "INR", "0.05"},
		{0, "USD", "0.00"},
		{-250, "USD", "-2.50"},
		{1000, "JPY", "1000"}, // zero-exponent currency
		{123456, "KWD", "123.456"},
	}
	for _, c := range cases {
		if got := FormatMoney(c.money, c.currency); got != c.want {
			t.Errorf("FormatMoney(%d, %q) = %q, want %q", c.money, c.currency, got, c.want)
		}
	}
}

func TestNormalizeCurrency(t *testing.T) {
	if c, err := NormalizeCurrency("inr"); err != nil || c != "INR" {
		t.Errorf("NormalizeCurrency(inr) = %q, %v", c, err)
	}
	if _, err := NormalizeCurrency("XXX"); err == nil {
		t.Error("expected error for unknown currency")
	}
}
