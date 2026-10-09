// Command ledger-core runs the LedgerForge accounting engine.
//
// Configuration comes from the environment:
//
//	LEDGER_DSN   Postgres connection string. If empty, an in-memory store
//	             is used (fine for local dev, not for production).
//	LEDGER_ADDR  Listen address, default ":8081".
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SachuJames/ledger-forge/ledger-core/internal/httpapi"
	"github.com/SachuJames/ledger-forge/ledger-core/internal/ledger"
	"github.com/SachuJames/ledger-forge/ledger-core/internal/store"
)

func main() {
	addr := envOr("LEDGER_ADDR", ":8081")
	dsn := os.Getenv("LEDGER_DSN")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var st ledger.Store
	if dsn == "" {
		log.Println("LEDGER_DSN not set, using in-memory store")
		st = store.NewMemory()
	} else {
		pg, err := store.NewPostgres(ctx, dsn)
		if err != nil {
			log.Fatalf("postgres: %v", err)
		}
		defer pg.Close()
		st = pg
		log.Println("connected to postgres, migrations applied")
	}

	svc := ledger.NewService(st)
	srv := &http.Server{
		Addr:         addr,
		Handler:      httpapi.New(svc).Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("ledger-core listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
