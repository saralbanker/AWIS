package main

// nfr_plugin_bench_test.go — M18-C2r plugin NFR benchmarks: NFR-P-08 (plugin spawn
// time, Python cold start) and NFR-P-09 (plugin capability-call latency, excluding
// handler logic).
//
// Both benchmarks drive the shipped plugins/git-context-plugin through the production
// internal/plugin Manager path (NewManager -> Register -> Call -> Shutdown) and spawn
// the real `python3 -m git_context_plugin` process named by the shipped
// awis-plugin.yaml. Nothing is mocked and nothing is skipped: a missing python3 or
// git, or a plugin that cannot be spawned, FAILS the benchmark.
//
// Method (fixed by the M18-C2r card):
//
//   NFR-P-08 (target < 3s; verdict on the maximum cold start). A cold start is one
//   fresh Manager and one fresh plugin process per sample. The timed interval is the
//   first Manager.Call, which performs the lazy spawn and handshake (TDS-05 §7) and
//   then one immediate-reject RPC: empty inputs, so the handler fails before any git
//   subprocess. Handler work is therefore near zero, and the sample is spawn +
//   handshake + one minimal round trip, an upper bound on spawn time.
//
//   NFR-P-09 (target < 50ms; verdict on the maximum sample). BINDING: each sample is
//   one successful git.diff.fetch on a tiny temp repository (2 commits, a 1-line diff)
//   through a warm process. It includes the handler's own git subprocess work, so it
//   is a conservative upper bound on "excluding handler logic". INFORMATIONAL: each
//   sample is an immediate-reject call (empty inputs), the near-zero handler-logic
//   floor. The floor is printed on its own line and carries no verdict.
//
// Run through `make bench` (-benchtime=1x): sample counts are fixed by the harness and
// are not derived from b.N. Each benchmark prints its nfrbench result line(s). A MISS
// is a measured result for C4 and is never failed here.

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/nfrbench"
	"github.com/awis/awis/internal/plugin"
	"github.com/awis/awis/internal/storage"
)

const (
	// nfrPluginName and nfrPluginCapability identify the shipped git-context-plugin
	// and the capability both NFRs exercise (manifest capability git.diff.fetch).
	nfrPluginName       = "git-context-plugin"
	nfrPluginCapability = "git.diff.fetch"
	// nfrPluginRejectCode is the StepError code of an immediate-reject call: the
	// plugin's handler raised before any git subprocess and answered with a JSON-RPC
	// error, leaving the process alive.
	nfrPluginRejectCode = "handler_error"

	// nfrPluginSpawnSamples is the NFR-P-08 cold-start count: one fresh process each.
	nfrPluginSpawnSamples = 20
	// nfrPluginCallSamples is the NFR-P-09 binding sample count (successful calls).
	nfrPluginCallSamples = 100
	// nfrPluginFloorSamples is the NFR-P-09 informational sample count (rejects).
	nfrPluginFloorSamples = 100

	// nfrPluginHandshakeTimeout replaces the 5s production default, so a cold start
	// slower than the 3s target is measured and reported as a MISS rather than
	// aborted as a handshake error.
	nfrPluginHandshakeTimeout = 15 * time.Second
	// nfrPluginShutdownGrace bounds each Manager.Shutdown wait for a plugin exit.
	nfrPluginShutdownGrace = 500 * time.Millisecond
	// nfrPluginCallTimeout is a harness guard: no single Manager call or shutdown
	// may run past it, so a wedged plugin fails the benchmark instead of hanging it.
	nfrPluginCallTimeout = 60 * time.Second
)

// nfrRequireToolsWith reports the first of names that lookPath cannot resolve, as
// an error that wraps the lookup failure. It is the testable core of nfrRequireTools.
func nfrRequireToolsWith(lookPath func(string) (string, error), names ...string) error {
	for _, name := range names {
		if _, err := lookPath(name); err != nil {
			return fmt.Errorf("%s not found on PATH (%w); the plugin NFR benchmarks fail rather than skip without it", name, err)
		}
	}
	return nil
}

// nfrRequireTools fails tb unless python3 and git are on PATH.
func nfrRequireTools(tb testing.TB) {
	tb.Helper()
	if err := nfrRequireToolsWith(exec.LookPath, "python3", "git"); err != nil {
		tb.Fatal(err)
	}
}

// nfrToolVersion returns the first line of `<name> --version`, for fixture
// descriptions. It fails tb if the tool cannot report its version.
func nfrToolVersion(tb testing.TB, name string) string {
	tb.Helper()
	out, err := exec.Command(name, "--version").Output()
	if err != nil {
		tb.Fatalf("%s --version: %v", name, err)
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	return line
}

// nfrPluginManifest returns the absolute path of the shipped git-context-plugin
// manifest. Its relative runtime paths resolve against its own directory (B-21), so
// the plugin runs from any working directory.
func nfrPluginManifest(tb testing.TB) string {
	tb.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "plugins", "git-context-plugin"))
	if err != nil {
		tb.Fatalf("resolve plugins/git-context-plugin: %v", err)
	}
	manifest := filepath.Join(dir, "awis-plugin.yaml")
	if _, err := os.Stat(manifest); err != nil {
		tb.Fatalf("shipped plugin manifest missing: %v", err)
	}
	return manifest
}

// nfrPluginStore opens an on-disk runtime store for one plugin benchmark. The store
// backs the Manager's capability lookup and plugin registration.
func nfrPluginStore(tb testing.TB) *storage.SQLiteStorage {
	tb.Helper()
	db, err := storage.Open(filepath.Join(tb.TempDir(), "plugin-bench.db"), nil)
	if err != nil {
		tb.Fatalf("open plugin benchmark store: %v", err)
	}
	tb.Cleanup(func() {
		if err := db.Close(); err != nil {
			tb.Logf("close plugin benchmark store: %v", err)
		}
	})
	return storage.NewSQLiteStorage(db, nil)
}

// nfrNewPluginManager constructs a production Manager over store and registers the
// shipped manifest. It spawns nothing: the first Call performs the lazy spawn.
func nfrNewPluginManager(tb testing.TB, store storage.PluginStore, manifest string) *plugin.Manager {
	tb.Helper()
	mgr := plugin.NewManager(store, plugin.ManagerConfig{
		HandshakeTimeout: nfrPluginHandshakeTimeout,
		ShutdownGrace:    nfrPluginShutdownGrace,
	})
	ctx, cancel := context.WithTimeout(context.Background(), nfrPluginCallTimeout)
	defer cancel()
	if err := mgr.Register(ctx, manifest); err != nil {
		tb.Fatalf("Manager.Register(%s): %v", manifest, err)
	}
	return mgr
}

// nfrShutdown stops every plugin process the Manager owns. Manager.Shutdown is
// idempotent, so calling it again from cleanup is safe.
func nfrShutdown(mgr *plugin.Manager) {
	ctx, cancel := context.WithTimeout(context.Background(), nfrPluginCallTimeout)
	defer cancel()
	mgr.Shutdown(ctx)
}

// nfrShutdownAfter registers nfrShutdown as tb cleanup, so no plugin process
// outlives the benchmark or test, whether it passes or fails.
func nfrShutdownAfter(tb testing.TB, mgr *plugin.Manager) {
	tb.Cleanup(func() { nfrShutdown(mgr) })
}

// nfrSilencePluginLog discards the Manager's D-11 warning ("plugins_user is not
// configured") for the length of a benchmark, so each cold start does not print it
// again. The fixture is the production default, and the warning is not a measurement
// input. The returned function restores the previous log output.
func nfrSilencePluginLog() func() {
	prev := log.Writer()
	log.SetOutput(io.Discard)
	return func() { log.SetOutput(prev) }
}

// nfrCall makes one Manager.Call with no step timeout (the capability's timeout_ms
// applies) and returns its outputs, its error, and the wall-clock of the call alone.
func nfrCall(mgr *plugin.Manager, stepID string, inputs map[string]any) (map[string]any, *core.StepError, time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), nfrPluginCallTimeout)
	defer cancel()
	t0 := time.Now()
	outputs, serr := mgr.Call(ctx, nfrPluginCapability, stepID, inputs, "")
	return outputs, serr, time.Since(t0)
}

// nfrDiffInputs are the git.diff.fetch inputs for the tiny fixture: HEAD~1 .. HEAD.
func nfrDiffInputs(repo string) map[string]any {
	return map[string]any{"repo_path": repo, "from_ref": "HEAD~1", "to_ref": "HEAD"}
}

// nfrCheckDiffOutput checks that a successful git.diff.fetch result carries exactly
// the fixture's one added line ("two"). A timed sample is therefore real git work,
// never an empty success.
func nfrCheckDiffOutput(outputs map[string]any) error {
	raw, ok := outputs["diff"]
	if !ok {
		return fmt.Errorf("outputs has no %q key (got %v)", "diff", outputs)
	}
	diff, ok := raw.(string)
	if !ok {
		return fmt.Errorf("outputs[%q] is %T, want string", "diff", raw)
	}
	added := 0
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			added++
			if line != "+two" {
				return fmt.Errorf("unexpected added line %q in diff %q", line, diff)
			}
		}
	}
	if added != 1 {
		return fmt.Errorf("diff has %d added lines, want 1: %q", added, diff)
	}
	return nil
}

// nfrDiffCall makes one successful git.diff.fetch Call on the warm process and
// returns its wall-clock. It fails tb if the call errors or its diff is not the
// fixture's added line.
func nfrDiffCall(tb testing.TB, mgr *plugin.Manager, repo, stepID string) time.Duration {
	tb.Helper()
	outputs, serr, elapsed := nfrCall(mgr, stepID, nfrDiffInputs(repo))
	if serr != nil {
		tb.Fatalf("git.diff.fetch (%s) failed: %s: %s", stepID, serr.Code, serr.Message)
	}
	if err := nfrCheckDiffOutput(outputs); err != nil {
		tb.Fatalf("git.diff.fetch (%s): %v", stepID, err)
	}
	return elapsed
}

// nfrPluginPIDAfterWarmUp makes one untimed successful git.diff.fetch Call, so the
// timed samples start on a live plugin process. It returns that process's PID.
func nfrPluginPIDAfterWarmUp(tb testing.TB, mgr *plugin.Manager, repo string) int {
	tb.Helper()
	_ = nfrDiffCall(tb, mgr, repo, "nfr-p09-warmup")
	pid := mgr.PluginPID(nfrPluginName)
	if pid == 0 {
		tb.Fatalf("no live %s process after the warm-up call", nfrPluginName)
	}
	return pid
}

// nfrRequireSamePID fails tb unless the Manager still runs the process it had at
// want. The warm-process method requires one process for every timed sample.
func nfrRequireSamePID(tb testing.TB, mgr *plugin.Manager, want int, phase string) {
	tb.Helper()
	if got := mgr.PluginPID(nfrPluginName); got != want {
		tb.Fatalf("NFR-P-09 %s: plugin process changed (pid %d -> %d); the warm-process method needs one process", phase, want, got)
	}
}

// nfrTinyGitRepo builds the NFR-P-09 fixture: a git repository with exactly two
// commits. The first adds change.txt with one line, and the second adds one line to
// it. The fixture checks its own shape (2 commits, one added line) before returning.
// Git runs with no global or system configuration, so a developer's signing or
// alias settings cannot change the fixture.
func nfrTinyGitRepo(tb testing.TB) string {
	tb.Helper()
	dir := tb.TempDir()
	nfrGit(tb, dir, "init", "-q")
	file := filepath.Join(dir, "change.txt")
	if err := os.WriteFile(file, []byte("one\n"), 0o600); err != nil {
		tb.Fatalf("write fixture file: %v", err)
	}
	nfrGit(tb, dir, "add", "--", "change.txt")
	nfrGit(tb, dir, "commit", "-q", "-m", "first")
	if err := os.WriteFile(file, []byte("one\ntwo\n"), 0o600); err != nil {
		tb.Fatalf("write fixture file: %v", err)
	}
	nfrGit(tb, dir, "add", "--", "change.txt")
	nfrGit(tb, dir, "commit", "-q", "-m", "second")
	if got := strings.TrimSpace(nfrGit(tb, dir, "rev-list", "--count", "HEAD")); got != "2" {
		tb.Fatalf("fixture %s has %s commits, want 2", dir, got)
	}
	if got := nfrGit(tb, dir, "diff", "--numstat", "HEAD~1", "HEAD"); got != "1\t0\tchange.txt\n" {
		tb.Fatalf("fixture %s: diff HEAD~1..HEAD numstat = %q, want one added line in change.txt", dir, got)
	}
	return dir
}

// nfrGit runs one git command in dir with a fixed identity and returns its stdout.
// It fails tb, with git's stderr, on a non-zero exit.
func nfrGit(tb testing.TB, dir string, args ...string) string {
	tb.Helper()
	full := append([]string{
		"-c", "user.name=AWIS Bench",
		"-c", "user.email=bench@awis.invalid",
		"-c", "commit.gpgsign=false",
		"-C", dir,
	}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		tb.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String()
}

// nfrPluginColdSample is one NFR-P-08 sample: a fresh Manager and one timed
// immediate-reject Call, which performs the lazy spawn and handshake. The Manager is
// shut down before the function returns, so the next sample starts with no process.
// It fails tb unless the call reached a live plugin process.
func nfrPluginColdSample(tb testing.TB, store storage.PluginStore, manifest string) time.Duration {
	tb.Helper()
	mgr := nfrNewPluginManager(tb, store, manifest)
	defer nfrShutdown(mgr)
	outputs, serr, elapsed := nfrCall(mgr, "nfr-p08", map[string]any{})
	if serr == nil || serr.Code != nfrPluginRejectCode {
		tb.Fatalf("NFR-P-08 cold start: want an immediate %s reject from a live plugin, got outputs=%v error=%+v",
			nfrPluginRejectCode, outputs, serr)
	}
	if mgr.PluginPID(nfrPluginName) == 0 {
		tb.Fatalf("NFR-P-08 cold start: no live %s process after the first call", nfrPluginName)
	}
	return elapsed
}

// nfrFmtDur renders a duration as nfrbench's result line does: seconds at or above
// one second, milliseconds otherwise, always with three decimals. nfrbench keeps its
// formatter unexported, so this mirrors it for the informational line.
func nfrFmtDur(d time.Duration) string {
	if d >= time.Second {
		return fmt.Sprintf("%.3fs", d.Seconds())
	}
	return fmt.Sprintf("%.3fms", float64(d)/float64(time.Millisecond))
}

// nfrInformationalLine renders an informational sub-measure in the field layout of
// nfrbench.Result.Line. It ends in INFORMATIONAL instead of a RESULT token:
// nfrbench.NewResult requires a binding bound, and an informational measure has none,
// so it cannot take that path.
func nfrInformationalLine(nfr, name, fixture string, s nfrbench.Summary) string {
	return fmt.Sprintf("%s | %s | fixture: %s | measured N=%d: P50=%s P95=%s P99=%s max=%s | INFORMATIONAL (no verdict; the NFR target applies to the binding line only)",
		nfr, name, fixture, s.N,
		nfrFmtDur(s.P50), nfrFmtDur(s.P95), nfrFmtDur(s.P99), nfrFmtDur(s.Max))
}

// BenchmarkNFRP08PluginSpawn measures NFR-P-08: plugin spawn time, Python cold start
// (PRD §11: "< 3s"). Each sample is a fresh process. The verdict is on the maximum
// cold start, the strictest reading of a bound stated without a percentile.
func BenchmarkNFRP08PluginSpawn(b *testing.B) {
	nfrRequireTools(b)
	restore := nfrSilencePluginLog()
	defer restore()
	manifest := nfrPluginManifest(b)
	store := nfrPluginStore(b)

	samples := make([]time.Duration, 0, nfrPluginSpawnSamples)
	for i := 0; i < nfrPluginSpawnSamples; i++ {
		samples = append(samples, nfrPluginColdSample(b, store, manifest))
	}
	sum, err := nfrbench.Summarize(samples)
	if err != nil {
		b.Fatalf("NFR-P-08 summarize: %v", err)
	}
	res, err := nfrbench.NewResult("NFR-P-08", "plugin spawn time (Python, cold start)",
		fmt.Sprintf("plugins/git-context-plugin shipped awis-plugin.yaml (runtime: python3 -m git_context_plugin; %s; %s); cold start = fresh Manager and fresh plugin process per sample; timed = first Manager.Call (lazy spawn + handshake, handshake timeout %s) with empty inputs, which the handler rejects before any git subprocess (%s), so handler work is near zero and the sample is an upper bound on spawn; %d cold starts; on-disk store; plugins_user unset (D-11 default)",
			nfrToolVersion(b, "python3"), nfrToolVersion(b, "git"), nfrPluginHandshakeTimeout, nfrPluginRejectCode, nfrPluginSpawnSamples),
		sum,
		nfrbench.Bound{Label: "NFR-P-08 target", Limit: 3 * time.Second, Strict: true,
			Basis: nfrbench.BasisMax, Binding: true},
	)
	if err != nil {
		b.Fatalf("NFR-P-08 verdict: %v", err)
	}
	nfrbench.Report(b, res)
}

// BenchmarkNFRP09PluginCall measures NFR-P-09: plugin capability call latency,
// excluding handler logic (PRD §11: "< 50ms"). The binding line times successful
// git.diff.fetch calls on a warm process and carries the verdict. The informational
// line times immediate-reject calls, the handler-logic floor, and carries none.
func BenchmarkNFRP09PluginCall(b *testing.B) {
	nfrRequireTools(b)
	restore := nfrSilencePluginLog()
	defer restore()
	manifest := nfrPluginManifest(b)
	repo := nfrTinyGitRepo(b)
	mgr := nfrNewPluginManager(b, nfrPluginStore(b), manifest)
	nfrShutdownAfter(b, mgr)
	pid := nfrPluginPIDAfterWarmUp(b, mgr, repo)

	binding := make([]time.Duration, 0, nfrPluginCallSamples)
	for i := 0; i < nfrPluginCallSamples; i++ {
		binding = append(binding, nfrDiffCall(b, mgr, repo, "nfr-p09"))
	}
	nfrRequireSamePID(b, mgr, pid, "binding samples")

	floor := make([]time.Duration, 0, nfrPluginFloorSamples)
	for i := 0; i < nfrPluginFloorSamples; i++ {
		_, serr, elapsed := nfrCall(mgr, "nfr-p09-floor", map[string]any{})
		if serr == nil || serr.Code != nfrPluginRejectCode {
			b.Fatalf("NFR-P-09 informational sample %d: want an immediate %s reject, got error=%+v", i, nfrPluginRejectCode, serr)
		}
		floor = append(floor, elapsed)
	}
	nfrRequireSamePID(b, mgr, pid, "informational samples")

	sumBinding, err := nfrbench.Summarize(binding)
	if err != nil {
		b.Fatalf("NFR-P-09 binding summarize: %v", err)
	}
	sumFloor, err := nfrbench.Summarize(floor)
	if err != nil {
		b.Fatalf("NFR-P-09 informational summarize: %v", err)
	}
	fixture := fmt.Sprintf("plugins/git-context-plugin shipped awis-plugin.yaml (runtime: python3 -m git_context_plugin; %s; %s); warm process (pid %d, unchanged across samples, one untimed warm-up call first); on-disk store; plugins_user unset (D-11 default); ",
		nfrToolVersion(b, "python3"), nfrToolVersion(b, "git"), pid)
	res, err := nfrbench.NewResult("NFR-P-09", "plugin capability call latency, excluding handler logic (binding: successful git.diff.fetch)",
		fixture+fmt.Sprintf("tiny temp git repo, 2 commits, 1-line diff; each sample is one successful Manager.Call of git.diff.fetch HEAD~1..HEAD, output checked for the added line; includes the handler's git subprocess work, so this is a conservative upper bound on the handler-excluded figure; %d samples",
			nfrPluginCallSamples),
		sumBinding,
		nfrbench.Bound{Label: "NFR-P-09 target", Limit: 50 * time.Millisecond, Strict: true,
			Basis: nfrbench.BasisMax, Binding: true},
	)
	if err != nil {
		b.Fatalf("NFR-P-09 verdict: %v", err)
	}
	nfrbench.Report(b, res)

	floorName := "informational floor: immediate-reject call (empty inputs; handler error before any git subprocess)"
	_, _ = fmt.Fprintln(os.Stdout, nfrInformationalLine("NFR-P-09", floorName,
		fixture+fmt.Sprintf("each sample is one Manager.Call with empty inputs, rejected by the handler; %d samples", nfrPluginFloorSamples),
		sumFloor))
	ms := func(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
	b.ReportMetric(ms(sumFloor.P50), "floor-p50-ms")
	b.ReportMetric(ms(sumFloor.Max), "floor-max-ms")
}
