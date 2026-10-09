package ledger_test

import (
	"context"
	"testing"

	"github.com/SachuJames/ledger-forge/ledger-core/internal/ledger"
	"github.com/SachuJames/ledger-forge/ledger-core/internal/store"
)

func testService() *ledger.Service {
	return ledger.NewService(store.NewMemory())
}

func mustAccount(t *testing.T, svc *ledger.Service, code, name string, typ ledger.AccountType) ledger.Account {
	t.Helper()
	a, err := svc.CreateAccount(context.Background(), code, name, typ)
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	return a
}

func TestCreateAccount(t *testing.T) {
	svc := testService()
	a := mustAccount(t, svc, "1001", "Cash", ledger.Asset)
	if a.ID == "" {
		t.Error("expected an ID to be assigned")
	}

	if _, err := svc.CreateAccount(context.Background(), "1001", "Dupe", ledger.Asset); err == nil {
		t.Error("expected duplicate code to fail")
	}
	if _, err := svc.CreateAccount(context.Background(), "", "No code", ledger.Asset); err == nil {
		t.Error("expected empty code to fail")
	}
	if _, err := svc.CreateAccount(context.Background(), "1002", "Weird", "nonsense"); err == nil {
		t.Error("expected bad account type to fail")
	}
}

func TestPostEntryAndBalances(t *testing.T) {
	ctx := context.Background()
	svc := testService()
	cash := mustAccount(t, svc, "1001", "Cash", ledger.Asset)
	rev := mustAccount(t, svc, "4001", "Revenue", ledger.Revenue)

	e, replayed, err := svc.PostEntry(ctx, ledger.Entry{
		Narration: "sale",
		Lines: []ledger.EntryLine{
			{AccountID: cash.ID, Currency: "INR", Debit: 5000},
			{AccountID: rev.ID, Currency: "INR", Credit: 5000},
		},
	})
	if err != nil {
		t.Fatalf("PostEntry: %v", err)
	}
	if replayed {
		t.Error("first post should not be a replay")
	}
	if e.ID == "" {
		t.Error("expected entry ID to be assigned")
	}

	bal, found, err := svc.Balance(ctx, cash.ID, "INR")
	if err != nil || !found || bal != 5000 {
		t.Errorf("cash balance = %d, found=%v, err=%v; want 5000", bal, found, err)
	}
	bal, _, _ = svc.Balance(ctx, rev.ID, "inr") // lowercase is fine
	if bal != 5000 {
		t.Errorf("revenue balance = %d, want 5000", bal)
	}
}

func TestPostEntryIdempotent(t *testing.T) {
	ctx := context.Background()
	svc := testService()
	cash := mustAccount(t, svc, "1001", "Cash", ledger.Asset)
	rev := mustAccount(t, svc, "4001", "Revenue", ledger.Revenue)

	post := func() (ledger.Entry, bool, error) {
		return svc.PostEntry(ctx, ledger.Entry{
			IdempotencyKey: "order-123",
			Lines: []ledger.EntryLine{
				{AccountID: cash.ID, Currency: "INR", Debit: 2000},
				{AccountID: rev.ID, Currency: "INR", Credit: 2000},
			},
		})
	}
	first, replayed, err := post()
	if err != nil || replayed {
		t.Fatalf("first post: %v, replayed=%v", err, replayed)
	}
	second, replayed, err := post()
	if err != nil || !replayed {
		t.Fatalf("second post: %v, replayed=%v", err, replayed)
	}
	if first.ID != second.ID {
		t.Error("replay returned a different entry")
	}
	bal, _, _ := svc.Balance(ctx, cash.ID, "INR")
	if bal != 2000 {
		t.Errorf("balance = %d after replay, want 2000 (no double post)", bal)
	}
}

func TestPostEntryRejects(t *testing.T) {
	ctx := context.Background()
	svc := testService()
	cash := mustAccount(t, svc, "1001", "Cash", ledger.Asset)

	// unbalanced
	if _, _, err := svc.PostEntry(ctx, ledger.Entry{Lines: []ledger.EntryLine{
		{AccountID: cash.ID, Currency: "INR", Debit: 100},
		{AccountID: cash.ID, Currency: "INR", Credit: 99},
	}}); err == nil {
		t.Error("expected unbalanced entry to fail")
	}
	// unknown account
	if _, _, err := svc.PostEntry(ctx, ledger.Entry{Lines: []ledger.EntryLine{
		{AccountID: "nope", Currency: "INR", Debit: 100},
		{AccountID: cash.ID, Currency: "INR", Credit: 100},
	}}); err == nil {
		t.Error("expected unknown account to fail")
	}
}

func TestJournalOrdering(t *testing.T) {
	ctx := context.Background()
	svc := testService()
	cash := mustAccount(t, svc, "1001", "Cash", ledger.Asset)
	rev := mustAccount(t, svc, "4001", "Revenue", ledger.Revenue)

	for i := 0; i < 3; i++ {
		_, _, err := svc.PostEntry(ctx, ledger.Entry{Lines: []ledger.EntryLine{
			{AccountID: cash.ID, Currency: "INR", Debit: 100},
			{AccountID: rev.ID, Currency: "INR", Credit: 100},
		}})
		if err != nil {
			t.Fatal(err)
		}
	}
	got, err := svc.Journal(ctx, 2, 0)
	if err != nil || len(got) != 2 {
		t.Fatalf("journal: %v, len=%d", err, len(got))
	}
	if got[0].CreatedAt.Before(got[1].CreatedAt) {
		t.Error("expected newest-first ordering")
	}
}
