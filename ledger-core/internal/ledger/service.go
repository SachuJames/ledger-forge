package ledger

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Store is the persistence boundary. The service never touches SQL or
// locks directly; implementations do. Two ship with the project: an
// in-memory store for tests and local dev, and Postgres for production.
type Store interface {
	CreateAccount(ctx context.Context, a Account) error
	GetAccount(ctx context.Context, id string) (Account, error)
	AccountByCode(ctx context.Context, code string) (Account, error)
	// InsertEntry writes the entry and applies every balance delta in one
	// atomic unit. Implementations must roll everything back on error.
	InsertEntry(ctx context.Context, e Entry, deltas map[string]map[string]Money) error
	FindEntryByIdempotencyKey(ctx context.Context, key string) (Entry, bool, error)
	GetBalance(ctx context.Context, accountID, currency string) (Money, bool, error)
	ListEntries(ctx context.Context, limit, offset int) ([]Entry, error)
}

// Service is the accounting engine. It owns validation, id assignment, and
// the double-entry rules; the store owns durability.
type Service struct {
	store Store
	now   func() time.Time
}

// NewService wires a store to the engine.
func NewService(s Store) *Service {
	return &Service{store: s, now: time.Now}
}

// CreateAccount registers an account in the chart of accounts.
func (s *Service) CreateAccount(ctx context.Context, code, name string, typ AccountType) (Account, error) {
	a := Account{
		ID:        uuid.NewString(),
		Code:      code,
		Name:      name,
		Type:      typ,
		CreatedAt: s.now().UTC(),
	}
	if err := a.Validate(); err != nil {
		return Account{}, err
	}
	if _, err := s.store.AccountByCode(ctx, code); err == nil {
		return Account{}, errors.New("ledger: account code already exists: " + code)
	}
	if err := s.store.CreateAccount(ctx, a); err != nil {
		return Account{}, err
	}
	return a, nil
}

// PostEntry validates and journals an entry. Posting is idempotent: if an
// entry with the same idempotency key was already recorded, the original is
// returned and nothing is written twice. A missing key is treated as a
// fresh, non-idempotent post.
func (s *Service) PostEntry(ctx context.Context, e Entry) (Entry, bool, error) {
	if e.IdempotencyKey != "" {
		if prev, found, err := s.store.FindEntryByIdempotencyKey(ctx, e.IdempotencyKey); err != nil {
			return Entry{}, false, err
		} else if found {
			return prev, true, nil
		}
	}
	if err := e.Validate(); err != nil {
		return Entry{}, false, err
	}
	accounts := make(map[string]Account, len(e.Lines))
	for _, l := range e.Lines {
		if _, ok := accounts[l.AccountID]; ok {
			continue
		}
		a, err := s.store.GetAccount(ctx, l.AccountID)
		if err != nil {
			return Entry{}, false, errors.New("ledger: unknown account " + l.AccountID)
		}
		accounts[l.AccountID] = a
	}
	e.ID = uuid.NewString()
	e.CreatedAt = s.now().UTC()
	if err := s.store.InsertEntry(ctx, e, e.balanceDeltas(accounts)); err != nil {
		return Entry{}, false, err
	}
	return e, false, nil
}

// Balance returns the current balance of an account in the given currency.
// The bool is false when the account has never moved in that currency.
func (s *Service) Balance(ctx context.Context, accountID, currency string) (Money, bool, error) {
	cur, err := NormalizeCurrency(currency)
	if err != nil {
		return 0, false, err
	}
	if _, err := s.store.GetAccount(ctx, accountID); err != nil {
		return 0, false, errors.New("ledger: unknown account " + accountID)
	}
	return s.store.GetBalance(ctx, accountID, cur)
}

// Journal returns entries newest-first.
func (s *Service) Journal(ctx context.Context, limit, offset int) ([]Entry, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	return s.store.ListEntries(ctx, limit, offset)
}
