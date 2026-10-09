package ledger_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/SachuJames/ledger-forge/ledger-core/internal/ledger"
	"github.com/SachuJames/ledger-forge/ledger-core/internal/store"
)

// BenchmarkPostEntry measures the hot path: validate + journal + balances,
// using the in-memory store so the number reflects engine cost, not I/O.
func BenchmarkPostEntry(b *testing.B) {
	ctx := context.Background()
	svc := ledger.NewService(store.NewMemory())
	cash, err := svc.CreateAccount(ctx, "1001", "Cash", ledger.Asset)
	if err != nil {
		b.Fatal(err)
	}
	rev, err := svc.CreateAccount(ctx, "4001", "Revenue", ledger.Revenue)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := svc.PostEntry(ctx, ledger.Entry{
			IdempotencyKey: fmt.Sprintf("bench-%d", i),
			Lines: []ledger.EntryLine{
				{AccountID: cash.ID, Currency: "INR", Debit: 100},
				{AccountID: rev.ID, Currency: "INR", Credit: 100},
			},
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
