package signal

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/storage"
)

// fakeStore is a scriptable Store: it returns a fixed undelivered list, a fixed
// DeliverSignal outcome, and records audit appends.
type fakeStore struct {
	sigs         []storage.Signal
	listErr      error
	outcome      storage.DeliverOutcome
	deliverErr   error
	deliverCalls []storage.DeliverInput
	audits       []storage.AuditEntry
	auditErr     error
}

func (f *fakeStore) ListUndeliveredSignals(_ context.Context, _ core.InstanceID) ([]storage.Signal, error) {
	return f.sigs, f.listErr
}

func (f *fakeStore) DeliverSignal(_ context.Context, in storage.DeliverInput) (storage.DeliverOutcome, error) {
	f.deliverCalls = append(f.deliverCalls, in)
	return f.outcome, f.deliverErr
}

func (f *fakeStore) AppendAudit(_ context.Context, e storage.AuditEntry) error {
	f.audits = append(f.audits, e)
	return f.auditErr
}

// fakeWaits is a scriptable Waits.
type fakeWaits struct {
	resolveOK bool
	stepID    string
	version   int
	delivered []string // signal names for which OnDelivered fired
}

func (f *fakeWaits) Resolve(_ context.Context, _ core.InstanceID, _ string) (string, int, bool) {
	return f.stepID, f.version, f.resolveOK
}

func (f *fakeWaits) OnDelivered(_ core.InstanceID, signalName string) {
	f.delivered = append(f.delivered, signalName)
}

func (f *fakeWaits) CompleteStep(_ context.Context, _ core.InstanceID, _ string, _ map[string]any) error {
	return nil // no-op stub: unit tests assert delivery behaviour, not step completion
}

func testScanner(store Store, waits Waits) *Scanner {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewScanner(store, waits, func() time.Time { return time.Unix(0, 0).UTC() },
		func() string { return "ev-fixed" }, "worker-1", logger)
}

var oneSig = []storage.Signal{{SignalID: "s1", InstanceID: "i1", SignalName: "go", Payload: map[string]any{"k": "v"}}}

// TestScan_DeliveredNotifiesAndAudits: Delivered ⇒ OnDelivered fires and one
// SignalDelivered audit is appended at the delivery site.
func TestScan_DeliveredNotifiesAndAudits(t *testing.T) {
	store := &fakeStore{sigs: oneSig, outcome: storage.Delivered}
	waits := &fakeWaits{resolveOK: true, stepID: "w", version: 7}
	if err := testScanner(store, waits).Scan(context.Background()); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(store.deliverCalls) != 1 {
		t.Fatalf("want 1 DeliverSignal call, got %d", len(store.deliverCalls))
	}
	if got := store.deliverCalls[0].ExpectedVersion; got != 7 {
		t.Fatalf("expected_version = %d, want 7 (from Resolve)", got)
	}
	if len(waits.delivered) != 1 || waits.delivered[0] != "go" {
		t.Fatalf("OnDelivered not fired for delivered signal: %v", waits.delivered)
	}
	if len(store.audits) != 1 || store.audits[0].EventType != "SignalDelivered" {
		t.Fatalf("want 1 SignalDelivered audit, got %+v", store.audits)
	}
	if store.audits[0].Actor != "worker-1" {
		t.Fatalf("audit actor = %q, want worker-1", store.audits[0].Actor)
	}
}

// TestScan_NoWaitSkips: Resolve ok=false ⇒ no delivery, no audit (a signal never
// resumes an instance with no live wait).
func TestScan_NoWaitSkips(t *testing.T) {
	store := &fakeStore{sigs: oneSig, outcome: storage.Delivered}
	waits := &fakeWaits{resolveOK: false}
	if err := testScanner(store, waits).Scan(context.Background()); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(store.deliverCalls) != 0 {
		t.Fatalf("want no DeliverSignal call when no wait, got %d", len(store.deliverCalls))
	}
	if len(store.audits) != 0 || len(waits.delivered) != 0 {
		t.Fatalf("no side effects expected when no wait")
	}
}

// TestScan_LockConflictNoAudit: LockConflict ⇒ no OnDelivered, no audit; the scan
// still returns nil (retry next tick).
func TestScan_LockConflictNoAudit(t *testing.T) {
	store := &fakeStore{sigs: oneSig, outcome: storage.DeliverLockConflict}
	waits := &fakeWaits{resolveOK: true, stepID: "w", version: 3}
	if err := testScanner(store, waits).Scan(context.Background()); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(waits.delivered) != 0 || len(store.audits) != 0 {
		t.Fatalf("lock conflict must not notify or audit: delivered=%v audits=%v", waits.delivered, store.audits)
	}
}

// TestScan_NoOpNoAudit: DeliverNoOp ⇒ no OnDelivered, no audit.
func TestScan_NoOpNoAudit(t *testing.T) {
	store := &fakeStore{sigs: oneSig, outcome: storage.DeliverNoOp}
	waits := &fakeWaits{resolveOK: true, stepID: "w", version: 3}
	if err := testScanner(store, waits).Scan(context.Background()); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(waits.delivered) != 0 || len(store.audits) != 0 {
		t.Fatalf("no-op must not notify or audit")
	}
}

// TestScan_DeliverErrorAborts: a real store fault on DeliverSignal aborts the pass.
func TestScan_DeliverErrorAborts(t *testing.T) {
	store := &fakeStore{sigs: oneSig, deliverErr: errors.New("db down")}
	waits := &fakeWaits{resolveOK: true, stepID: "w", version: 1}
	if err := testScanner(store, waits).Scan(context.Background()); err == nil {
		t.Fatalf("want error when DeliverSignal faults")
	}
}

// TestScan_ListErrorAborts: a fault listing undelivered signals aborts the pass.
func TestScan_ListErrorAborts(t *testing.T) {
	store := &fakeStore{listErr: errors.New("db down")}
	if err := testScanner(store, &fakeWaits{}).Scan(context.Background()); err == nil {
		t.Fatalf("want error when ListUndeliveredSignals faults")
	}
}

// TestScan_AuditFailureIsNonFatal: an audit-append fault after a committed
// delivery is logged, not returned (delivery already durable).
func TestScan_AuditFailureIsNonFatal(t *testing.T) {
	store := &fakeStore{sigs: oneSig, outcome: storage.Delivered, auditErr: errors.New("audit down")}
	waits := &fakeWaits{resolveOK: true, stepID: "w", version: 2}
	if err := testScanner(store, waits).Scan(context.Background()); err != nil {
		t.Fatalf("audit failure must not fail the scan: %v", err)
	}
	if len(waits.delivered) != 1 {
		t.Fatalf("delivery must still be notified despite audit failure")
	}
}
