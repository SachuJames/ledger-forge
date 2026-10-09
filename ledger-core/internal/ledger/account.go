package ledger

import (
	"errors"
	"time"
)

// AccountType classifies an account in the chart of accounts. The type
// decides the sign convention when a balance is reported.
type AccountType string

const (
	Asset     AccountType = "asset"
	Liability AccountType = "liability"
	Equity    AccountType = "equity"
	Revenue   AccountType = "revenue"
	Expense   AccountType = "expense"
)

// ValidAccountType reports whether t is a known account type.
func ValidAccountType(t AccountType) bool {
	switch t {
	case Asset, Liability, Equity, Revenue, Expense:
		return true
	}
	return false
}

// Account is one node in the chart of accounts.
type Account struct {
	ID        string
	Code      string // human-friendly code, e.g. "1001"; unique
	Name      string
	Type      AccountType
	CreatedAt time.Time
}

// Validate checks the account fields that the service cannot fix itself.
func (a Account) Validate() error {
	if a.Code == "" {
		return errors.New("ledger: account code is required")
	}
	if a.Name == "" {
		return errors.New("ledger: account name is required")
	}
	if !ValidAccountType(a.Type) {
		return errors.New("ledger: unknown account type " + string(a.Type))
	}
	return nil
}

// normalBalanceSign returns +1 for account types whose balance grows with
// debits (assets, expenses) and -1 for the rest (credit-normal).
func normalBalanceSign(t AccountType) int64 {
	switch t {
	case Asset, Expense:
		return 1
	default:
		return -1
	}
}
