package engine

import (
	"time"

	"github.com/awis/awis/internal/core"
)

// Retry mechanics (ADJ-6, EDR-011 §3). A step's FIRST activation claims the step
// (ClaimStep); retries re-dispatch under the EXISTING claim with `attempt`
// incremented (retries do NOT re-claim — a retrying step stays in current_steps
// because StepFailed{retrying:true} does not remove it, EDR-011 §3). Backoff is
// tick-quantized: a scheduled retry becomes due at the first tick whose clock is
// at or past nextAttemptAt (not sleep-exact, EDR-011 §3).

// retryDefaults holds ADJ-6's absent-field defaults (initial 1s, max 30s).
const (
	retryDefaultInitial = 1 * time.Second
	retryDefaultMax     = 30 * time.Second
)

// backoffDelay computes the delay before an attempt-numbered retry per ADJ-6 /
// EDR-011 §3:
//
//	immediate   ⇒ 0
//	linear      ⇒ initial × attempt
//	exponential ⇒ initial × 2^(attempt−1)
//
// capped at max. `attempt` is the ordinal of the attempt that just FAILED (the
// delay precedes the next attempt). Absent InitialDelay/MaxDelay default to
// 1s/30s (ADJ-6). Durations parse via time.ParseDuration (the same M10-deferred
// idiom as native.parseTimeout); an unparseable value falls back to the default.
//
// semantics-bearing: backoff schedule (ADJ-6 / EDR-011 §3).
func backoffDelay(p core.RetryPolicy, attempt int) time.Duration {
	initial := parseDurationOr(p.InitialDelay, retryDefaultInitial)
	maxDelay := parseDurationOr(p.MaxDelay, retryDefaultMax)

	var d time.Duration
	switch p.Backoff {
	case "immediate":
		d = 0
	case "linear":
		d = initial * time.Duration(attempt)
	case "exponential":
		// initial × 2^(attempt−1); shift is safe for the small attempt counts a
		// RetryPolicy expresses.
		d = initial * time.Duration(int64(1)<<(attempt-1))
	default:
		// The grammar freezes backoff ∈ {immediate,linear,exponential} and the
		// validator now rejects any other value at submit/validate time
		// (internal/validate CodeRetryBackoff). This branch is a defensive
		// backstop for RetryPolicy values persisted BEFORE that validation
		// existed: an unknown value here is treated as immediate (0) so a
		// malformed policy never blocks forever (judgment call, C2r).
		d = 0
	}
	if d > maxDelay {
		d = maxDelay
	}
	if d < 0 {
		d = 0
	}
	return d
}

// parseDurationOr parses a core.Duration, returning def on empty/unparseable
// input (M10 owns the final serialization form; native.parseTimeout uses the
// same stdlib reading).
func parseDurationOr(d core.Duration, def time.Duration) time.Duration {
	s := string(d)
	if s == "" {
		return def
	}
	parsed, err := time.ParseDuration(s)
	if err != nil || parsed < 0 {
		return def
	}
	return parsed
}

// retryEligible reports whether a just-failed attempt may be retried under policy
// (ADJ-6): a policy must exist, `attempt` must be below Attempts, and the error
// code must be retryable (RetryableErrors empty ⇒ all codes retryable). No policy
// ⇒ single attempt (never eligible).
func retryEligible(policy *core.RetryPolicy, attempt int, code string) bool {
	if policy == nil {
		return false
	}
	if attempt >= policy.Attempts {
		return false
	}
	return codeRetryable(policy, code)
}

// codeRetryable reports whether code is in the policy's RetryableErrors list; an
// empty list means every code is retryable (ADJ-6).
func codeRetryable(policy *core.RetryPolicy, code string) bool {
	if policy == nil || len(policy.RetryableErrors) == 0 {
		return true
	}
	for _, c := range policy.RetryableErrors {
		if c == code {
			return true
		}
	}
	return false
}

// scheduleRetry records that (iid, step) should re-dispatch at attempt with a
// backoff-derived earliest time. Called from the serial settle path.
func (e *Engine) scheduleRetry(iid core.InstanceID, step string, nextAttempt int, delay time.Duration) {
	e.mu.Lock()
	e.retries[retryKey{iid: iid, step: step}] = retrySched{
		nextAttempt:   nextAttempt,
		nextAttemptAt: e.now().Add(delay),
	}
	e.mu.Unlock()
}

// clearRetry drops any scheduled retry for (iid, step).
func (e *Engine) clearRetry(iid core.InstanceID, step string) {
	e.mu.Lock()
	delete(e.retries, retryKey{iid: iid, step: step})
	e.mu.Unlock()
}

// dueRetries returns, in current_steps order, the steps whose scheduled retry is
// due (Clock() ≥ nextAttemptAt). It reads the clock ONLY when the instance has at
// least one scheduled retry, so the common (no-retry) path never perturbs the
// engine clock — this keeps the C1 duration fixtures exact.
func (e *Engine) dueRetries(iid core.InstanceID, currentSteps []string) []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	hasAny := false
	for _, s := range currentSteps {
		if _, ok := e.retries[retryKey{iid: iid, step: s}]; ok {
			hasAny = true
			break
		}
	}
	if !hasAny {
		return nil
	}
	now := e.now()
	var out []string
	for _, s := range currentSteps {
		if rs, ok := e.retries[retryKey{iid: iid, step: s}]; ok && !now.Before(rs.nextAttemptAt) {
			out = append(out, s)
		}
	}
	return out
}

// retryAttempt returns the attempt ordinal to run for a due retry of (iid, step).
func (e *Engine) retryAttempt(iid core.InstanceID, step string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	if rs, ok := e.retries[retryKey{iid: iid, step: step}]; ok {
		return rs.nextAttempt
	}
	return 1
}

// scheduledRetrySteps returns every current step that has a pending retry
// schedule (regardless of whether it is due yet) — used by the cancellation path
// to terminally fail retry-waiting steps (Finalization B4-adjacent, EDR-011 §8).
func (e *Engine) scheduledRetrySteps(iid core.InstanceID, currentSteps []string) []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []string
	for _, s := range currentSteps {
		if _, ok := e.retries[retryKey{iid: iid, step: s}]; ok {
			out = append(out, s)
		}
	}
	return out
}
