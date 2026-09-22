package engine

// D-18 proof: no engine goroutine — in particular Run's ticker goroutine,
// and anything Tick calls synchronously from it — can block on log I/O, the
// first occurrence of a log record is never lost, nothing grows without
// bound, and any dropped output is countable, never silent.
//
// blockingWriter is a TEST-ONLY io.Writer (this file) that, once armed,
// blocks inside Write() until the test releases it, deterministically
// simulating a stalled log consumer (a full pipe, a stuck collector) WITHOUT
// real OS pipes or wall-clock timing races. Before arm() is called, Write
// passes through immediately — mirroring the repo's existing arm/disarm
// fault-injection idiom (claim_release_test.go's faultInjectingStorage).
// entered is closed the first time an ARMED Write is called, so a test can
// synchronize on "the write is now blocked" precisely rather than polling —
// this file follows the repo idiom (see hydrate_test.go) of never using
// time.Sleep as synchronisation.

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/runner/native"
)

type blockingWriter struct {
	unblock chan struct{}
	entered chan struct{}
	once    sync.Once

	mu    sync.Mutex
	armed bool
	buf   bytes.Buffer
}

func newBlockingWriter() *blockingWriter {
	return &blockingWriter{unblock: make(chan struct{}), entered: make(chan struct{})}
}

// arm makes subsequent Write calls block until release(). See the type
// comment: before arm(), Write passes through immediately.
func (w *blockingWriter) arm() {
	w.mu.Lock()
	w.armed = true
	w.mu.Unlock()
}

func (w *blockingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	armed := w.armed
	w.mu.Unlock()
	if armed {
		w.once.Do(func() { close(w.entered) })
		<-w.unblock
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *blockingWriter) release() { close(w.unblock) }

func (w *blockingWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// assertNonBlocking fails the test if fn does not return within a generous
// deadline. The deadline is a deadlock guard, not the mechanism under test.
func assertNonBlocking(t *testing.T, name string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { fn(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s: blocked; an engine goroutine must never block on log I/O (D-18)", name)
	}
}

// TestNonBlockingHandler_HandleNeverBlocksUnderBackpressure is the D-18
// proof at the Handler level: Handle() (what every e.logger.* call reaches)
// returns immediately even once the relay channel itself is full because the
// single consumer goroutine is stuck inside the wrapped handler's write —
// and the resulting drop count is folded into the next record that DOES get
// relayed, never silently lost.
func TestNonBlockingHandler_HandleNeverBlocksUnderBackpressure(t *testing.T) {
	bw := newBlockingWriter()
	bw.arm() // every write blocks from the start.
	inner := slog.NewTextHandler(bw, nil)
	h := newNonBlockingHandler(inner)
	defer h.close()
	notify := make(chan struct{}, nonBlockingHandlerChanCap+16)
	h.notify = notify

	logger := slog.New(h)

	// The first record: relayed into the (empty) channel, picked up by the
	// consumer goroutine immediately, which then blocks inside bw.Write.
	assertNonBlocking(t, "log(first)", func() { logger.Error("first") })
	<-bw.entered

	// Fill the bounded channel with records the consumer cannot possibly
	// drain (it is stuck on the first write): capacity+1 more calls, all of
	// which MUST return immediately regardless.
	for i := 0; i < nonBlockingHandlerChanCap+1; i++ {
		assertNonBlocking(t, "log(fill)", func() { logger.Info("fill") })
	}

	if got := h.droppedCount(); got == 0 {
		t.Fatal("droppedCount() = 0, want > 0 (the channel was overfilled while the consumer was stuck; drops must be countable, not silent)")
	}

	// Release the stalled writer: everything queued flushes (nonBlockingHandlerChanCap+1
	// records, the first of which carries the folded-in drop count).
	// Deterministic synchronization via h.notify — never time.Sleep.
	bw.release()
	for i := 0; i < nonBlockingHandlerChanCap+1; i++ {
		select {
		case <-notify:
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for the relay goroutine to drain (got %d of %d)", i, nonBlockingHandlerChanCap+1)
		}
	}

	out := bw.String()
	if !bytesContainsAll(out, "first") {
		t.Fatalf("logged output missing the first record: %q", out)
	}
	if !bytesContainsAll(out, "dropped_log_lines_under_backpressure") {
		t.Fatalf("logged output never carried the folded-in drop count (recoverable, not silent): %q", out)
	}
}

func bytesContainsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !bytes.Contains([]byte(s), []byte(sub)) {
			return false
		}
	}
	return true
}

// TestRun_TickerNeverFreezesWhenLogWriterStalls is the end-to-end D-18 proof
// at the level the card actually cares about: Run()'s ticker goroutine keeps
// firing Tick — and therefore keeps making progress (D-17) — even when the
// engine's logger is wired to a writer that never returns. Before D-18, any
// of several direct, synchronous e.logger.* calls reachable from Tick (the
// D-17 per-instance failure log, signalTimeoutScan's Warn, logEmit's Info,
// …) would have frozen the ticker goroutine on the first one whose write
// stalled — proven by this exact test hanging against the pre-fix code.
func TestRun_TickerNeverFreezesWhenLogWriterStalls(t *testing.T) {
	def := core.WorkflowDefinition{
		SchemaVersion: 1, ID: "t.d18run", Version: "1.0.0", Namespace: "t", Name: "d18run",
		Triggers:    []core.Trigger{{Type: core.TriggerTypeManual, Config: map[string]any{}}},
		Steps:       []core.Step{nativeStep("greet", "hgreet")},
		InitialStep: "greet", FinalSteps: []string{"greet"}, Metadata: map[string]any{},
	}
	hgreet := &stepHandler{id: "hgreet", outputs: map[string]any{"msg": "hi"}}

	real := openStorage(t)
	if err := real.RegisterWorkflow(context.Background(), def); err != nil {
		t.Fatalf("RegisterWorkflow: %v", err)
	}
	fs := &hydrateFaultingStorage{SQLiteStorage: real}

	nr := native.New()
	nr.Register(hgreet)

	bw := newBlockingWriter()
	logger := slog.New(slog.NewTextHandler(bw, nil))

	e := New(fs, map[core.StepType]Runner{core.StepTypeNative: nr},
		Config{MaxParallelSteps: 1, TickInterval: 5 * time.Millisecond}, logger)

	ctx := context.Background()
	iid, err := e.Submit(ctx, "t.d18run", "1.0.0", nil)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	// Arm only now: Submit's own WorkflowStarted emission logs synchronously
	// (it runs before Run() swaps in the non-blocking relay), so it must not
	// stall — only the tick-loop failure logging this test targets should.
	bw.arm()
	fs.setTarget(iid) // fails hydrate every tick, forever, so Tick logs a per-instance failure every tick (D-17).

	runCtx, cancel := context.WithCancel(ctx)
	runErrCh := make(chan error, 1)
	go func() { runErrCh <- e.Run(runCtx) }()

	// The relay's single consumer goroutine stalls inside the wrapped
	// handler's write; if D-18 were unfixed (or incompletely fixed — only
	// covering one call site rather than the handler as a whole), this would
	// freeze the ticker goroutine itself, since Tick's logging calls happen
	// in it. Wait for the stall, then prove the ticker goroutine is NOT
	// frozen by cancelling and requiring Run() to return promptly.
	<-bw.entered

	cancel()
	select {
	case err := <-runErrCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run() returned %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return after ctx cancellation; the ticker goroutine is frozen behind the stalled log writer (D-18 regression)")
	}

	bw.release() // let the stuck write complete so the relay goroutine can exit cleanly.
}
