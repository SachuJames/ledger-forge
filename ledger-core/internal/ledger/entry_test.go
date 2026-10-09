package ledger

import "testing"

func line(acct, cur string, debit, credit Money) EntryLine {
	return EntryLine{AccountID: acct, Currency: cur, Debit: debit, Credit: credit}
}

func TestEntryValidate(t *testing.T) {
	cases := []struct {
		name    string
		lines   []EntryLine
		wantErr bool
	}{
		{"balanced", []EntryLine{
			line("a", "INR", 1000, 0),
			line("b", "INR", 0, 1000),
		}, false},
		{"multi-currency balanced", []EntryLine{
			line("a", "INR", 1000, 0),
			line("b", "INR", 0, 1000),
			line("a", "USD", 500, 0),
			line("b", "USD", 0, 500),
		}, false},
		{"three legs", []EntryLine{
			line("a", "INR", 700, 0),
			line("b", "INR", 300, 0),
			line("c", "INR", 0, 1000),
		}, false},
		{"unbalanced", []EntryLine{
			line("a", "INR", 1000, 0),
			line("b", "INR", 0, 999),
		}, true},
		{"single line", []EntryLine{
			line("a", "INR", 1000, 0),
		}, true},
		{"no lines", nil, true},
		{"mixed debit and credit on one line", []EntryLine{
			line("a", "INR", 1000, 500),
			line("b", "INR", 0, 500),
		}, true},
		{"zero amount line", []EntryLine{
			line("a", "INR", 0, 0),
			line("b", "INR", 0, 0),
		}, true},
		{"negative amount", []EntryLine{
			line("a", "INR", -100, 0),
			line("b", "INR", 0, 100),
		}, true},
		{"unknown currency", []EntryLine{
			line("a", "XXX", 100, 0),
			line("b", "XXX", 0, 100),
		}, true},
		{"missing account", []EntryLine{
			line("", "INR", 100, 0),
			line("b", "INR", 0, 100),
		}, true},
		{"unbalanced in second currency", []EntryLine{
			line("a", "INR", 100, 0),
			line("b", "INR", 0, 100),
			line("a", "USD", 50, 0),
			line("b", "USD", 0, 40),
		}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Entry{Lines: c.lines}.Validate()
			if c.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBalanceDeltas(t *testing.T) {
	accounts := map[string]Account{
		"cash": {ID: "cash", Type: Asset},
		"loan": {ID: "loan", Type: Liability},
	}
	e := Entry{Lines: []EntryLine{
		line("cash", "INR", 1000, 0), // asset debit: balance +1000
		line("loan", "INR", 0, 1000), // liability credit: balance +1000
	}}
	d := e.balanceDeltas(accounts)
	if d["cash"]["INR"] != 1000 {
		t.Errorf("cash delta = %d, want 1000", d["cash"]["INR"])
	}
	if d["loan"]["INR"] != 1000 {
		t.Errorf("loan delta = %d, want 1000", d["loan"]["INR"])
	}

	// Paying down the loan: debit on a liability shrinks it.
	e2 := Entry{Lines: []EntryLine{
		line("loan", "INR", 400, 0),
		line("cash", "INR", 0, 400),
	}}
	d2 := e2.balanceDeltas(accounts)
	if d2["loan"]["INR"] != -400 {
		t.Errorf("loan delta = %d, want -400", d2["loan"]["INR"])
	}
	if d2["cash"]["INR"] != -400 {
		t.Errorf("cash delta = %d, want -400", d2["cash"]["INR"])
	}
}
