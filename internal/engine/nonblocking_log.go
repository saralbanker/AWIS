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
	"time"
)

// nonBlockingHandlerChanCap bounds the relay channel: generous enough that a
// normal burst of operational logging (e.g. concurrent dispatch/settle lines
// under MaxParallelSteps > 1) is not dropped under healthy conditions, but
// FIXED — it can never grow without limit (D-18: "nothing may grow without
// bound").
const nonBlockingHandlerChanCap = 1024

// nonBlockingHandlerDrainDeadline bounds how long close() waits for records
// still queued in h.ch to be delivered to the inner handler before giving up
// (D-18 gap 1: close() must drain, not silently discard, but must also never
// block shutdown indefinitely). 200ms is short enough that Run() returning
// stays prompt — well under the 2s deadlock guard the D-18 tests already use
// elsewhere in this package — while being generous for flushing up to
// nonBlockingHandlerChanCap queued records through a healthy inner handler,
// which is the only case this deadline is meant to accommodate; a genuinely
// stalled writer will exceed it every time; see loop's GAP 2 comment.
const nonBlockingHandlerDrainDeadline = 200 * time.Millisecond

// nonBlockingHandler implements slog.Handler by relaying every record to a
// single background goroutine (loop, started by newNonBlockingHandler).
// WithAttrs/WithGroup return a derived handler that shares the same channel,
// shutdown signal, drop counter, and drained signal, so every handler
// derived from the same root still funnels through exactly one goroutine and
// one drop count.
type nonBlockingHandler struct {
	inner slog.Handler

	ch        chan slog.Record
	done      chan struct{}
	closeOnce *sync.Once
	dropped   *atomic.Int64

	// drained is closed by loop() once it has fully drained h.ch after done
	// fires (see loop's drain step). close() waits on it, bounded by
	// nonBlockingHandlerDrainDeadline — see close()'s doc comment and GAP 2.
	drained chan struct{}

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
		drained:   make(chan struct{}),
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
	return &nonBlockingHandler{inner: h.inner.WithAttrs(attrs), ch: h.ch, done: h.done, closeOnce: h.closeOnce, dropped: h.dropped, drained: h.drained}
}

func (h *nonBlockingHandler) WithGroup(name string) slog.Handler {
	return &nonBlockingHandler{inner: h.inner.WithGroup(name), ch: h.ch, done: h.done, closeOnce: h.closeOnce, dropped: h.dropped, drained: h.drained}
}

// loop is the single goroutine that performs the actual (possibly blocking)
// handler work, so nothing else ever has to.
//
// GAP 2 (documented, accepted limitation — not fixed): if h.inner.Handle is
// itself permanently stalled (e.g. a full pipe or a stuck collector that
// never unblocks), this goroutine is blocked in-flight inside deliver's call
// to h.inner.Handle below and cannot be interrupted — closing h.done does not
// preempt an in-progress call. The goroutine then leaks for the life of the
// process, pinning up to nonBlockingHandlerChanCap queued records. This is a
// genuine, pathological Go limitation (a blocking call in progress cannot be
// cancelled from outside), not something this fix addresses; close() bounds
// how long IT waits (nonBlockingHandlerDrainDeadline) so shutdown itself is
// never held hostage by this leaked goroutine — see close()'s doc comment.
func (h *nonBlockingHandler) loop() {
	ctx := context.Background()
	for {
		select {
		case r := <-h.ch:
			h.deliver(ctx, r)
		case <-h.done:
			h.drain(ctx)
			return
		}
	}
}

// deliver hands r to the inner handler and pings notify (if set). Shared by
// loop's normal path and drain so every record is handled identically
// regardless of which path delivers it.
func (h *nonBlockingHandler) deliver(ctx context.Context, r slog.Record) {
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
}

// drain delivers every record still queued in h.ch to the inner handler
// before the relay goroutine exits, so a normal shutdown does not silently
// discard queued log lines (D-18 gap 1: "the first occurrence of an error
// must never be lost", "dropped output must be countable, not silent").
// It has no deadline of its own — close() is what bounds how long the
// CALLER waits for this to finish (see close()'s doc comment and GAP 2
// above); drain itself just keeps delivering until h.ch is observed empty,
// then signals h.drained so a waiting close() can stop waiting.
func (h *nonBlockingHandler) drain(ctx context.Context) {
	for {
		select {
		case r := <-h.ch:
			h.deliver(ctx, r)
		default:
			close(h.drained)
			return
		}
	}
}

// close stops the relay goroutine, first waiting (bounded by
// nonBlockingHandlerDrainDeadline) for it to drain and deliver every record
// still queued in h.ch — a normal shutdown (writer healthy, channel empty or
// lightly loaded) drains essentially immediately and returns well within the
// deadline. It returns the number of queued records that were still
// undelivered when the deadline was reached (0 in the normal case): if the
// inner handler is itself stuck mid-write when close is called, the relay
// goroutine cannot be interrupted (GAP 2, documented on loop/drain above) and
// the drain signal will never arrive, so close() gives up after the deadline
// rather than hanging shutdown, and reports whatever is still sitting in
// h.ch at that point as undelivered. Safe to call multiple times or on any
// handler derived via WithAttrs/WithGroup (they share closeOnce/done/
// drained); only the first call's return value is meaningful, matching
// closeOnce's existing once-only semantics.
func (h *nonBlockingHandler) close() int64 {
	var undelivered int64
	h.closeOnce.Do(func() {
		close(h.done)
		select {
		case <-h.drained:
		case <-time.After(nonBlockingHandlerDrainDeadline):
			undelivered = int64(len(h.ch))
		}
	})
	return undelivered
}

// droppedCount is a test-observable snapshot of the pending (not yet folded
// into a relayed line) drop count.
func (h *nonBlockingHandler) droppedCount() int64 {
	return h.dropped.Load()
}
