package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/SachuJames/ledger-forge/ledger-core/internal/ledger"
)

// Server exposes the ledger engine over HTTP. It is the internal API that
// the NestJS gateway (and operators) talk to.
type Server struct {
	svc *ledger.Service
	mux *http.ServeMux
}

// New wires routes to the service.
func New(svc *ledger.Service) *Server {
	s := &Server{svc: svc, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("POST /v1/accounts", s.createAccount)
	s.mux.HandleFunc("POST /v1/entries", s.postEntry)
	s.mux.HandleFunc("GET /v1/accounts/{id}/balance", s.balance)
	s.mux.HandleFunc("GET /v1/entries", s.journal)
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type createAccountReq struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var req createAccountReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	a, err := s.svc.CreateAccount(r.Context(), req.Code, req.Name, ledger.AccountType(req.Type))
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, accountJSON(a))
}

type entryLineReq struct {
	AccountID string `json:"account_id"`
	Currency  string `json:"currency"`
	Debit     int64  `json:"debit"`
	Credit    int64  `json:"credit"`
	Memo      string `json:"memo"`
}

type postEntryReq struct {
	IdempotencyKey string            `json:"idempotency_key"`
	Narration      string            `json:"narration"`
	Metadata       map[string]string `json:"metadata"`
	Lines          []entryLineReq    `json:"lines"`
}

func (s *Server) postEntry(w http.ResponseWriter, r *http.Request) {
	var req postEntryReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	e := ledger.Entry{
		IdempotencyKey: req.IdempotencyKey,
		Narration:      req.Narration,
		Metadata:       req.Metadata,
	}
	for _, l := range req.Lines {
		e.Lines = append(e.Lines, ledger.EntryLine{
			AccountID: l.AccountID,
			Currency:  l.Currency,
			Debit:     ledger.Money(l.Debit),
			Credit:    ledger.Money(l.Credit),
			Memo:      l.Memo,
		})
	}
	e, replayed, err := s.svc.PostEntry(r.Context(), e)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if replayed {
		w.Header().Set("Idempotent-Replayed", "true")
	}
	writeJSON(w, http.StatusCreated, entryJSON(e))
}

func (s *Server) balance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	currency := r.URL.Query().Get("currency")
	if currency == "" {
		currency = "INR"
	}
	bal, found, err := s.svc.Balance(r.Context(), id, currency)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	cur, _ := ledger.NormalizeCurrency(currency)
	writeJSON(w, http.StatusOK, map[string]any{
		"account_id": id,
		"currency":   cur,
		"balance":    int64(bal),
		"formatted":  ledger.FormatMoney(bal, cur),
		"has_moved":  found,
	})
}

func (s *Server) journal(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	entries, err := s.svc.Journal(r.Context(), limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]any, 0, len(entries))
	for _, e := range entries {
		out = append(out, entryJSON(e))
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": out})
}

func accountJSON(a ledger.Account) map[string]any {
	return map[string]any{
		"id":         a.ID,
		"code":       a.Code,
		"name":       a.Name,
		"type":       string(a.Type),
		"created_at": a.CreatedAt.Format(time.RFC3339),
	}
}

func entryJSON(e ledger.Entry) map[string]any {
	lines := make([]any, 0, len(e.Lines))
	for _, l := range e.Lines {
		lines = append(lines, map[string]any{
			"account_id": l.AccountID,
			"currency":   l.Currency,
			"debit":      int64(l.Debit),
			"credit":     int64(l.Credit),
			"memo":       l.Memo,
		})
	}
	return map[string]any{
		"id":              e.ID,
		"idempotency_key": e.IdempotencyKey,
		"narration":       e.Narration,
		"metadata":        e.Metadata,
		"lines":           lines,
		"created_at":      e.CreatedAt.Format(time.RFC3339),
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
