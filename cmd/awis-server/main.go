// Package main is the AWIS HTTP server binary (cmd/awis-server; E-G4-1).
//
// It embeds the AWIS engine runtime — via sdk.SQLiteStorage + sdk.NewRuntime,
// exactly as cmd/awis/start.go bootstraps the CLI daemon — and runs it
// alongside a stdlib http.Server. The engine pull loop and the HTTP listener
// run concurrently; SIGTERM/SIGINT triggers coordinated graceful shutdown of
// both, mirroring cmd/awis/start.go's signal-handling idiom
// (cmd/awis/start.go:264-289).
//
// Routing is delegated to internal/api.NewRouter (E-G4-3).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/awis/awis/internal/api"
	"github.com/awis/awis/sdk"
)

// defaultNamespace is the owning namespace for workflows run by this binary
// (mirrors cmd/awis/start.go's --namespace default).
const defaultNamespace = "default"

// defaultTick mirrors cmd/awis/start.go's --tick default (100ms).
const defaultTick = 100 * time.Millisecond

// defaultAddr is the loopback default for the HTTP listener. Port 8090 was
// checked against the rest of this repo (grep for 8090/8080/3000 and other
// literal port numbers under cmd/, internal/, docs/) and has no existing
// precedent or collision.
const defaultAddr = "127.0.0.1:8090"

// defaultDBPath is the --db flag default. It deliberately lives in the
// current working directory (rather than a subdirectory such as cmd/awis's
// ./.awis/) so the binary starts with zero directory-creation logic.
const defaultDBPath = "./awis-server.db"

func main() {
	var dbPath string
	var addr string
	var staticDir string
	fs := flag.NewFlagSet("awis-server", flag.ExitOnError)
	fs.StringVar(&dbPath, "db", defaultDBPath, "Path to the SQLite database file")
	fs.StringVar(&addr, "addr", defaultAddr, "HTTP listen address")
	fs.StringVar(&staticDir, "static-dir", "", "Serve frontend assets from this directory instead of the embedded build (dev mode; see web/README or GUI_PHASE1_ARCHITECTURE.md §2)")
	// flag.ExitOnError already calls os.Exit(2) on a parse error; no need to
	// check the return value here.
	_ = fs.Parse(os.Args[1:])

	if err := run(dbPath, addr, staticDir); err != nil {
		fmt.Fprintf(os.Stderr, "awis-server: %s\n", err)
		os.Exit(1)
	}
}

// run wires storage, the engine runtime, and the HTTP server together. It
// blocks until a shutdown signal (SIGTERM/SIGINT) is received and both the
// engine loop and the HTTP listener have stopped.
func run(dbPath, addr, staticDir string) error {
	startedAt := time.Now()

	// 1. Open storage (cmd/awis/start.go:99 OpenStorage → sdk.SQLiteStorage).
	store, err := sdk.SQLiteStorage(dbPath)
	if err != nil {
		return fmt.Errorf("cannot open storage at %q: %w", dbPath, err)
	}

	// 2. Construct the runtime (cmd/awis/start.go:155-163).
	rt, err := sdk.NewRuntime(sdk.Config{
		Namespace:    defaultNamespace,
		Storage:      store,
		TickInterval: defaultTick,
	})
	if err != nil {
		return fmt.Errorf("cannot create runtime: %w", err)
	}

	// 3. Install signal handlers: SIGTERM/SIGINT cancel a shared context
	// (cmd/awis/start.go:264-271 pattern, verbatim).
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		cancel()
	}()

	// 4. Router (E-G4-3, widened by E-G4-4/5/6 to accept rt/store), mounted
	// under /api/v1/ on a top-level mux alongside the GUI Phase 1 frontend's
	// static assets at "/" (static.go). Go's ServeMux prefers the more
	// specific pattern, so /api/v1/... always reaches the API handler even
	// though "/" also matches it.
	static, err := staticHandler(staticDir)
	if err != nil {
		return fmt.Errorf("cannot set up static asset handler: %w", err)
	}
	top := http.NewServeMux()
	top.Handle("/api/v1/", api.NewRouter(rt, store, startedAt))
	top.Handle("/", static)

	srv := &http.Server{
		Addr:    addr,
		Handler: top,
	}

	// 5. Run the engine pull loop off the main goroutine: sdk.NewRuntime
	// spawns no goroutines of its own, and rt.Start blocks
	// (internal/engine/tick.go), so it must run concurrently with the HTTP
	// server rather than after it (cmd/awis/start.go:285 runs it as the
	// final blocking call there; here the main goroutine is needed for the
	// HTTP server instead).
	engineErrCh := make(chan error, 1)
	go func() {
		engineErrCh <- rt.Start(ctx)
	}()

	// 6. Graceful shutdown: when ctx is cancelled (by the signal handler
	// above), stop accepting new connections and let in-flight ones finish
	// within a bounded window.
	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if serr := srv.Shutdown(shutdownCtx); serr != nil {
			log.Printf("awis-server: http shutdown: %s", serr)
		}
	}()

	// 7. Run the HTTP server on the main goroutine. ListenAndServe blocks
	// until Shutdown is called (step 6) or a listener error occurs.
	log.Printf("awis-server: listening on %s (db=%s)", addr, dbPath)
	srvErr := srv.ListenAndServe()
	if errors.Is(srvErr, http.ErrServerClosed) {
		srvErr = nil
	}
	// If the HTTP server exited for a reason other than a graceful shutdown
	// request (e.g. a bind failure), make sure the engine loop is told to
	// stop too rather than blocking forever — cancel is idempotent, so this
	// is a no-op on the normal signal-driven shutdown path.
	cancel()

	engineErr := <-engineErrCh
	// rt.Start (internal/engine/tick.go Run) returns ctx.Err() on cancellation
	// by design ("ctx cancellation returns ctx.Err()") — this is the expected
	// shutdown signal, not a failure. cmd/awis/start.go discards rt.Start's
	// return value entirely for the same reason; here the value is still
	// useful for surfacing a *different* engine error, so only
	// context.Canceled is filtered out.
	if errors.Is(engineErr, context.Canceled) {
		engineErr = nil
	}

	if srvErr != nil {
		return fmt.Errorf("http server: %w", srvErr)
	}
	if engineErr != nil {
		return fmt.Errorf("engine: %w", engineErr)
	}
	return nil
}
