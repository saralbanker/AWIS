package engine

// nonblocking_log.go — D-18: the ticker goroutine (Run's for-select loop,
// tick.go) — and anything Tick calls synchronously from it (processInstance,
// signalScan, signalTimeoutScan, emit, …) — must never block on log I/O. A
// blocked log writer (a full pipe, a stalled collector, a slow consumer on
// `awis start | ...`) would otherwise freeze workflow execution permanently,
// since the ticker goroutine is the only thing driving Tick.
//
// nonBlockingHandler wraps whatever slog.Handler is currently installed
// (JSON, text, or any custom Handler) so that EVERY engine log call reachable
// from Tick is covered uniformly, not just one hand-picked call site. This
// matters concretely: log/slog's built-in handlers (JSONHandler/TextHandler)
// hold a SHARED internal mutex around the underlying io.Writer for the
// lifetime of each Write call — so even relaying only the "risky" call (e.g.
// a per-instance failure) through a side channel is not sufficient, because
// any OTHER direct e.log().* call made from the same goroutine (e.g.
// signalTimeoutScan's Warn, or logEmit's Info) would still block trying to
// acquire that same mutex while a DIFFERENT goroutine's write to the same
// handler is stuck. Wrapping at the Handler level, once per Run(), is what
// actually closes that gap.
//
// Handle() never performs the wrapped handler's (possibly blocking) work
// itself: it clones the record and hands it to a single dedicated goroutine
// over a bounded channel, then returns immediately. Run() (tick.go) installs
// a nonBlockingHandler wrapping the engine's real logger for its own
// lifetime ONLY, via e.loggerPtr (an atomic.Pointer — Submit/Signal/Cancel
// can legitimately run concurrently with Run() in the daemon, so this is
// never a plain mutated field), and restores the original logger when Run
// returns. Tick() called directly (no Run() active, e.g. by tests) is
// unaffected: it logs synchronously via the original logger, exactly as
// before D-18 — which is why the existing tests that assert on synchronous
// log output (e.g. TestCancel_TerminalNoOpWarning) are untouched by this.
//
// Scoped to the engine's own logger only: nothing outside internal/engine is
// touched, and no other package's logging is restructured.

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
)

// nonBlockingHandlerChanCap bounds the relay channel: generous enough that a
// normal burst of operational logging (e.g. concurrent dispatch/settle lines
// under MaxParallelSteps > 1) is not dropped under healthy conditions, but
// FIXED — it can never grow without limit (D-18: "nothing may grow without
// bound").
const nonBlockingHandlerChanCap = 1024

// nonBlockingHandler implements slog.Handler by relaying every record to a
// single background goroutine (loop, started by newNonBlockingHandler).
// WithAttrs/WithGroup return a derived handler that shares the same channel,
// shutdown signal, and drop counter, so every handler derived from the same
// root still funnels through exactly one goroutine and one drop count.
type nonBlockingHandler struct {
	inner slog.Handler

	ch        chan slog.Record
	done      chan struct{}
	closeOnce *sync.Once
	dropped   *atomic.Int64

	// notify, if non-nil, receives a best-effort (non-blocking) signal after
	// every record the relay goroutine actually writes. Test-only
	// synchronization seam (nil in production; internal test — see
	// helpers_test.go's repo idiom of never using time.Sleep to synchronize).
	notify chan struct{}
}

// newNonBlockingHandler wraps inner and starts its single relay goroutine.
func newNonBlockingHandler(inner slog.Handler) *nonBlockingHandler {
	h := &nonBlockingHandler{
		inner:     inner,
		ch:        make(chan slog.Record, nonBlockingHandlerChanCap),
		done:      make(chan struct{}),
		closeOnce: &sync.Once{},
		dropped:   &atomic.Int64{},
	}
	go h.loop()
	return h
}

func (h *nonBlockingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle NEVER performs I/O and never blocks: it clones the record (required
// — slog's contract is that a Record passed to Handle, and its attribute
// storage, is only valid for the duration of that call) and attempts a
// non-blocking send. Under backpressure (the relay channel is full because
// the single consumer is itself stuck on a blocked write) the record is
// dropped — countable via h.dropped, folded into the next record that IS
// relayed (see loop), never silent.
func (h *nonBlockingHandler) Handle(_ context.Context, r slog.Record) error {
	rc := r.Clone()
	select {
	case h.ch <- rc:
	default:
		h.dropped.Add(1)
	}
	return nil
}

func (h *nonBlockingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &nonBlockingHandler{inner: h.inner.WithAttrs(attrs), ch: h.ch, done: h.done, closeOnce: h.closeOnce, dropped: h.dropped}
}

func (h *nonBlockingHandler) WithGroup(name string) slog.Handler {
	return &nonBlockingHandler{inner: h.inner.WithGroup(name), ch: h.ch, done: h.done, closeOnce: h.closeOnce, dropped: h.dropped}
}

// loop is the single goroutine that performs the actual (possibly blocking)
// handler work, so nothing else ever has to.
func (h *nonBlockingHandler) loop() {
	ctx := context.Background()
	for {
		select {
		case r := <-h.ch:
			if n := h.dropped.Swap(0); n > 0 {
				r.AddAttrs(slog.Int64("dropped_log_lines_under_backpressure", n))
			}
			_ = h.inner.Handle(ctx, r) // best-effort: nothing else to do with a logging failure here.
			if h.notify != nil {
				select {
				case h.notify <- struct{}{}:
				default:
				}
			}
		case <-h.done:
			return
		}
	}
}

// close stops the relay goroutine. Safe to call multiple times or on any
// handler derived via WithAttrs/WithGroup (they share closeOnce/done).
func (h *nonBlockingHandler) close() {
	h.closeOnce.Do(func() { close(h.done) })
}

// droppedCount is a test-observable snapshot of the pending (not yet folded
// into a relayed line) drop count.
func (h *nonBlockingHandler) droppedCount() int64 {
	return h.dropped.Load()
}
