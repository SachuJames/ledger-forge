package store

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/SachuJames/ledger-forge/ledger-core/internal/ledger"
)

// Memory is a thread-safe in-memory Store. It exists for tests and local
// development; production uses Postgres.
type Memory struct {
	mu       sync.Mutex
	accounts map[string]ledger.Account
	byCode   map[string]string
	entries  []ledger.Entry
	byKey    map[string]string // idempotency key -> entry ID
	balances map[string]map[string]ledger.Money
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory {
	return &Memory{
		accounts: map[string]ledger.Account{},
		byCode:   map[string]string{},
		byKey:    map[string]string{},
		balances: map[string]map[string]ledger.Money{},
	}
}

var errNotFound = errors.New("store: not found")

func (m *Memory) CreateAccount(_ context.Context, a ledger.Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, dup := m.byCode[a.Code]; dup {
		return errors.New("store: duplicate account code")
	}
	m.accounts[a.ID] = a
	m.byCode[a.Code] = a.ID
	return nil
}

func (m *Memory) GetAccount(_ context.Context, id string) (ledger.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.accounts[id]
	if !ok {
		return ledger.Account{}, errNotFound
	}
	return a, nil
}

func (m *Memory) AccountByCode(_ context.Context, code string) (ledger.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byCode[code]
	if !ok {
		return ledger.Account{}, errNotFound
	}
	return m.accounts[id], nil
}

func (m *Memory) InsertEntry(_ context.Context, e ledger.Entry, deltas map[string]map[string]ledger.Money) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e.IdempotencyKey != "" {
		if _, dup := m.byKey[e.IdempotencyKey]; dup {
			return errors.New("store: duplicate idempotency key")
		}
	}
	for acct, perCur := range deltas {
		if m.balances[acct] == nil {
			m.balances[acct] = map[string]ledger.Money{}
		}
		for cur, d := range perCur {
			m.balances[acct][cur] += d
		}
	}
	m.entries = append(m.entries, e)
	if e.IdempotencyKey != "" {
		m.byKey[e.IdempotencyKey] = e.ID
	}
	return nil
}

func (m *Memory) FindEntryByIdempotencyKey(_ context.Context, key string) (ledger.Entry, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byKey[key]
	if !ok {
		return ledger.Entry{}, false, nil
	}
	for _, e := range m.entries {
		if e.ID == id {
			return e, true, nil
		}
	}
	return ledger.Entry{}, false, nil
}

func (m *Memory) GetBalance(_ context.Context, accountID, currency string) (ledger.Money, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	bal, ok := m.balances[accountID][currency]
	return bal, ok, nil
}

func (m *Memory) ListEntries(_ context.Context, limit, offset int) ([]ledger.Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]ledger.Entry, len(m.entries))
	copy(out, m.entries)
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if offset >= len(out) {
		return []ledger.Entry{}, nil
	}
	out = out[offset:]
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
