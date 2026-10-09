package ledger

import (
	"errors"
	"fmt"
	"time"
)

// EntryLine is one leg of a journal entry. Exactly one of Debit or Credit
// must be positive; the other must be zero. Mixed debit+credit on a single
// line is rejected so every leg reads unambiguously.
type EntryLine struct {
	AccountID string
	Currency  string
	Debit     Money
	Credit    Money
	Memo      string
}

// Entry is a balanced set of lines posted to the journal atomically. Either
// all lines land or none do; there is no partial posting.
type Entry struct {
	ID             string
	IdempotencyKey string
	Lines          []EntryLine
	CreatedAt      time.Time
	Narration      string
	Metadata       map[string]string
}

// Validate enforces the double-entry invariants:
//
//  1. at least two lines,
//  2. every line touches exactly one side,
//  3. per currency, total debits equal total credits and are positive,
//  4. a single entry never mixes currencies on one side imbalance.
func (e Entry) Validate() error {
	if len(e.Lines) < 2 {
		return errors.New("ledger: entry needs at least two lines")
	}
	debits := map[string]Money{}
	credits := map[string]Money{}
	for i, l := range e.Lines {
		if l.AccountID == "" {
			return fmt.Errorf("ledger: line %d has no account", i)
		}
		cur, err := NormalizeCurrency(l.Currency)
		if err != nil {
			return fmt.Errorf("ledger: line %d: %w", i, err)
		}
		e.Lines[i].Currency = cur
		if l.Debit < 0 || l.Credit < 0 {
			return fmt.Errorf("ledger: line %d has a negative amount", i)
		}
		switch {
		case l.Debit > 0 && l.Credit == 0:
			debits[cur] += l.Debit
		case l.Credit > 0 && l.Debit == 0:
			credits[cur] += l.Credit
		case l.Debit == 0 && l.Credit == 0:
			return fmt.Errorf("ledger: line %d has a zero amount", i)
		default:
			return fmt.Errorf("ledger: line %d mixes debit and credit", i)
		}
	}
	seen := map[string]bool{}
	for cur := range debits {
		seen[cur] = true
	}
	for cur := range credits {
		seen[cur] = true
	}
	for cur := range seen {
		if debits[cur] != credits[cur] {
			return fmt.Errorf("ledger: unbalanced in %s: debits %d != credits %d",
				cur, debits[cur], credits[cur])
		}
		if debits[cur] <= 0 {
			return fmt.Errorf("ledger: no movement in %s", cur)
		}
	}
	return nil
}

// balanceDeltas converts validated lines into signed per-account deltas in
// normal-balance terms: positive means the balance grew.
func (e Entry) balanceDeltas(accounts map[string]Account) map[string]map[string]Money {
	out := map[string]map[string]Money{}
	for _, l := range e.Lines {
		sign := normalBalanceSign(accounts[l.AccountID].Type)
		delta := sign * int64(l.Debit-l.Credit)
		if out[l.AccountID] == nil {
			out[l.AccountID] = map[string]Money{}
		}
		out[l.AccountID][l.Currency] += Money(delta)
	}
	return out
}
