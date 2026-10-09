package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SachuJames/ledger-forge/ledger-core/internal/ledger"
	"github.com/SachuJames/ledger-forge/ledger-core/internal/store"
)

func testServer() *Server {
	svc := ledger.NewService(store.NewMemory())
	return New(svc)
}

func doRequest(t *testing.T, srv *Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	rec := doRequest(t, testServer(), "GET", "/healthz", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestAccountLifecycle(t *testing.T) {
	srv := testServer()

	rec := doRequest(t, srv, "POST", "/v1/accounts", map[string]string{
		"code": "1001", "name": "Cash", "type": "asset",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create account: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var acct map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &acct); err != nil {
		t.Fatal(err)
	}
	id, _ := acct["id"].(string)
	if id == "" {
		t.Fatal("expected account id in response")
	}

	// bad type is rejected
	rec = doRequest(t, srv, "POST", "/v1/accounts", map[string]string{
		"code": "1002", "name": "Bogus", "type": "nonsense",
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("bad type: status = %d, want 422", rec.Code)
	}

	// balance before any movement
	rec = doRequest(t, srv, "GET", "/v1/accounts/"+id+"/balance?currency=INR", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("balance: status = %d", rec.Code)
	}
	var bal map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &bal)
	if bal["has_moved"] != false {
		t.Errorf("expected has_moved=false, got %v", bal)
	}
}

func TestEntryLifecycle(t *testing.T) {
	srv := testServer()

	var ids []string
	for _, a := range []map[string]string{
		{"code": "1001", "name": "Cash", "type": "asset"},
		{"code": "4001", "name": "Revenue", "type": "revenue"},
	} {
		rec := doRequest(t, srv, "POST", "/v1/accounts", a)
		var acct map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &acct)
		ids = append(ids, acct["id"].(string))
	}

	entry := map[string]any{
		"idempotency_key": "txn-1",
		"narration":       "test sale",
		"lines": []map[string]any{
			{"account_id": ids[0], "currency": "INR", "debit": 2500},
			{"account_id": ids[1], "currency": "INR", "credit": 2500},
		},
	}
	rec := doRequest(t, srv, "POST", "/v1/entries", entry)
	if rec.Code != http.StatusCreated {
		t.Fatalf("post entry: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Idempotent-Replayed") != "" {
		t.Error("first post must not be marked replayed")
	}

	// replay with the same key
	rec = doRequest(t, srv, "POST", "/v1/entries", entry)
	if rec.Header().Get("Idempotent-Replayed") != "true" {
		t.Error("expected Idempotent-Replayed header on duplicate key")
	}

	// balance reflects a single posting, not two
	rec = doRequest(t, srv, "GET", "/v1/accounts/"+ids[0]+"/balance?currency=INR", nil)
	var bal map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &bal)
	if bal["balance"] != float64(2500) {
		t.Errorf("balance = %v, want 2500", bal["balance"])
	}
	if bal["formatted"] != "25.00" {
		t.Errorf("formatted = %v, want 25.00", bal["formatted"])
	}

	// unbalanced entry is rejected
	bad := map[string]any{
		"lines": []map[string]any{
			{"account_id": ids[0], "currency": "INR", "debit": 100},
			{"account_id": ids[1], "currency": "INR", "credit": 90},
		},
	}
	rec = doRequest(t, srv, "POST", "/v1/entries", bad)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("unbalanced: status = %d, want 422", rec.Code)
	}

	// journal lists the entry
	rec = doRequest(t, srv, "GET", "/v1/entries?limit=10", nil)
	var journal map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &journal)
	if entries, ok := journal["entries"].([]any); !ok || len(entries) != 1 {
		t.Errorf("journal entries = %v, want exactly 1", journal["entries"])
	}
}
