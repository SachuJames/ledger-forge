package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SachuJames/ledger-forge/ledger-core/internal/ledger"
	"github.com/SachuJames/ledger-forge/ledger-core/migrations"
)

// Postgres is the production Store backed by PostgreSQL.
type Postgres struct {
	pool *pgxpool.Pool
}

// NewPostgres connects and runs pending migrations.
func NewPostgres(ctx context.Context, dsn string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("store: connect: %w", err)
	}
	p := &Postgres{pool: pool}
	if err := p.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return p, nil
}

func (p *Postgres) migrate(ctx context.Context) error {
	sql, err := migrations.FS.ReadFile("001_init.sql")
	if err != nil {
		return fmt.Errorf("store: read migration: %w", err)
	}
	if _, err := p.pool.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("store: migrate: %w", err)
	}
	return nil
}

// Close releases the connection pool.
func (p *Postgres) Close() { p.pool.Close() }

func (p *Postgres) CreateAccount(ctx context.Context, a ledger.Account) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO accounts (id, code, name, type, created_at) VALUES ($1,$2,$3,$4,$5)`,
		a.ID, a.Code, a.Name, string(a.Type), a.CreatedAt)
	return err
}

func (p *Postgres) GetAccount(ctx context.Context, id string) (ledger.Account, error) {
	var a ledger.Account
	var typ string
	err := p.pool.QueryRow(ctx,
		`SELECT id, code, name, type, created_at FROM accounts WHERE id = $1`, id).
		Scan(&a.ID, &a.Code, &a.Name, &typ, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ledger.Account{}, errNotFound
	}
	a.Type = ledger.AccountType(typ)
	return a, err
}

func (p *Postgres) AccountByCode(ctx context.Context, code string) (ledger.Account, error) {
	var a ledger.Account
	var typ string
	err := p.pool.QueryRow(ctx,
		`SELECT id, code, name, type, created_at FROM accounts WHERE code = $1`, code).
		Scan(&a.ID, &a.Code, &a.Name, &typ, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ledger.Account{}, errNotFound
	}
	a.Type = ledger.AccountType(typ)
	return a, err
}

// InsertEntry writes the entry, its lines, and the balance deltas in a
// single transaction. Any failure rolls everything back.
func (p *Postgres) InsertEntry(ctx context.Context, e ledger.Entry, deltas map[string]map[string]ledger.Money) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	meta, _ := json.Marshal(e.Metadata)
	var key *string
	if e.IdempotencyKey != "" {
		key = &e.IdempotencyKey
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO entries (id, idempotency_key, narration, metadata, created_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		e.ID, key, e.Narration, meta, e.CreatedAt); err != nil {
		return err
	}
	for _, l := range e.Lines {
		if _, err := tx.Exec(ctx,
			`INSERT INTO entry_lines (entry_id, account_id, currency, debit, credit, memo)
			 VALUES ($1,$2,$3,$4,$5,$6)`,
			e.ID, l.AccountID, l.Currency, int64(l.Debit), int64(l.Credit), l.Memo); err != nil {
			return err
		}
	}
	for acct, perCur := range deltas {
		for cur, d := range perCur {
			if _, err := tx.Exec(ctx,
				`INSERT INTO balances (account_id, currency, balance) VALUES ($1,$2,$3)
				 ON CONFLICT (account_id, currency)
				 DO UPDATE SET balance = balances.balance + EXCLUDED.balance`,
				acct, cur, int64(d)); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func (p *Postgres) FindEntryByIdempotencyKey(ctx context.Context, key string) (ledger.Entry, bool, error) {
	var e ledger.Entry
	var meta []byte
	err := p.pool.QueryRow(ctx,
		`SELECT id, idempotency_key, narration, metadata, created_at
		 FROM entries WHERE idempotency_key = $1`, key).
		Scan(&e.ID, &e.IdempotencyKey, &e.Narration, &meta, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ledger.Entry{}, false, nil
	}
	if err != nil {
		return ledger.Entry{}, false, err
	}
	_ = json.Unmarshal(meta, &e.Metadata)
	lines, err := p.linesFor(ctx, p.pool, e.ID)
	if err != nil {
		return ledger.Entry{}, false, err
	}
	e.Lines = lines
	return e, true, nil
}

func (p *Postgres) GetBalance(ctx context.Context, accountID, currency string) (ledger.Money, bool, error) {
	var bal int64
	err := p.pool.QueryRow(ctx,
		`SELECT balance FROM balances WHERE account_id = $1 AND currency = $2`,
		accountID, currency).Scan(&bal)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	return ledger.Money(bal), err == nil, err
}

func (p *Postgres) ListEntries(ctx context.Context, limit, offset int) ([]ledger.Entry, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT id, idempotency_key, narration, metadata, created_at
		 FROM entries ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ledger.Entry
	for rows.Next() {
		var e ledger.Entry
		var meta []byte
		var key *string
		if err := rows.Scan(&e.ID, &key, &e.Narration, &meta, &e.CreatedAt); err != nil {
			return nil, err
		}
		if key != nil {
			e.IdempotencyKey = *key
		}
		_ = json.Unmarshal(meta, &e.Metadata)
		lines, err := p.linesFor(ctx, p.pool, e.ID)
		if err != nil {
			return nil, err
		}
		e.Lines = lines
		out = append(out, e)
	}
	return out, rows.Err()
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (p *Postgres) linesFor(ctx context.Context, q querier, entryID string) ([]ledger.EntryLine, error) {
	rows, err := q.Query(ctx,
		`SELECT account_id, currency, debit, credit, memo
		 FROM entry_lines WHERE entry_id = $1 ORDER BY id`, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ledger.EntryLine
	for rows.Next() {
		var l ledger.EntryLine
		var debit, credit int64
		if err := rows.Scan(&l.AccountID, &l.Currency, &debit, &credit, &l.Memo); err != nil {
			return nil, err
		}
		l.Debit, l.Credit = ledger.Money(debit), ledger.Money(credit)
		out = append(out, l)
	}
	return out, rows.Err()
}
