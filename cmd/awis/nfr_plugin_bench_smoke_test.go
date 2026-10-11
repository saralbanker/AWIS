package main

// nfr_plugin_bench_smoke_test.go — ordinary (non-benchmark) tests for the M18-C2r
// plugin NFR harness in nfr_plugin_bench_test.go. They run under plain `go test ./...`
// and drive the same production path as BenchmarkNFRP08PluginSpawn and
// BenchmarkNFRP09PluginCall against the real shipped plugin, so a broken fixture or
// plugin fails the ordinary test run, not only `make bench`.

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/awis/awis/internal/nfrbench"
)

// TestNFRRequireToolsNamesTheMissingTool pins the fail-loud rule: a missing python3
// or git is an error that names the tool and wraps the lookup failure, never a skip.
func TestNFRRequireToolsNamesTheMissingTool(t *testing.T) {
	lookupErr := errors.New("exec: not found")
	lookPath := func(name string) (string, error) {
		if name == "git" {
			return "", lookupErr
		}
		return "/usr/bin/" + name, nil
	}
	err := nfrRequireToolsWith(lookPath, "python3", "git")
	if err == nil {
		t.Fatal("missing git: want an error, got nil")
	}
	if !strings.Contains(err.Error(), "git") || !errors.Is(err, lookupErr) {
		t.Fatalf("missing git: error %q must name git and wrap the lookup failure", err)
	}
	if err := nfrRequireToolsWith(lookPath, "python3"); err != nil {
		t.Fatalf("python3 present: unexpected error %v", err)
	}
}

// TestNFRCheckDiffOutputAcceptsOnlyTheFixtureLine pins the binding-sample check: a
// timed git.diff.fetch sample counts only when its diff is exactly the fixture's one
// added line. Missing, non-string, empty, and different diffs all fail.
func TestNFRCheckDiffOutputAcceptsOnlyTheFixtureLine(t *testing.T) {
	const oneLine = "diff --git a/change.txt b/change.txt\n" +
		"index 1111111..2222222 100644\n" +
		"--- a/change.txt\n" +
		"+++ b/change.txt\n" +
		"@@ -1 +1,2 @@\n" +
		" one\n" +
		"+two\n"
	cases := []struct {
		name    string
		outputs map[string]any
		wantErr bool
	}{
		{name: "fixture diff", outputs: map[string]any{"diff": oneLine}},
		{name: "missing diff key", outputs: map[string]any{}, wantErr: true},
		{name: "diff not a string", outputs: map[string]any{"diff": 42}, wantErr: true},
		{name: "empty diff", outputs: map[string]any{"diff": ""}, wantErr: true},
		{name: "second added line", outputs: map[string]any{"diff": oneLine + "+three\n"}, wantErr: true},
		{name: "different added line", outputs: map[string]any{"diff": strings.Replace(oneLine, "+two", "+other", 1)}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := nfrCheckDiffOutput(tc.outputs)
			if (err != nil) != tc.wantErr {
				t.Fatalf("nfrCheckDiffOutput: err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

// TestNFRInformationalLineHasNoVerdict pins the informational rendering: the line
// carries the NFR id, the fixture, and the measured percentiles, but no RESULT token,
// because an informational measure must never read as a PASS or MISS.
func TestNFRInformationalLineHasNoVerdict(t *testing.T) {
	s, err := nfrbench.Summarize([]time.Duration{time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond})
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	line := nfrInformationalLine("NFR-P-09", "floor", "fixture-x", s)
	for _, want := range []string{"NFR-P-09 | floor | fixture: fixture-x | measured N=3:", "max=3.000ms", "INFORMATIONAL (no verdict"} {
		if !strings.Contains(line, want) {
			t.Errorf("informational line %q missing %q", line, want)
		}
	}
	if strings.Contains(line, "RESULT:") {
		t.Errorf("informational line must not carry a RESULT token: %q", line)
	}
}

// TestNFRFmtDurUnits pins the unit rule nfrbench uses for result lines: milliseconds
// below one second, seconds at or above it, always three decimals.
func TestNFRFmtDurUnits(t *testing.T) {
	if got := nfrFmtDur(1500 * time.Microsecond); got != "1.500ms" {
		t.Errorf("nfrFmtDur(1.5ms) = %q, want 1.500ms", got)
	}
	if got := nfrFmtDur(2500 * time.Millisecond); got != "2.500s" {
		t.Errorf("nfrFmtDur(2.5s) = %q, want 2.500s", got)
	}
}

// TestNFRPluginPathSmoke drives the real shipped plugin through the benchmark path:
// one cold start, a warm successful git.diff.fetch series on one unchanged process,
// and an immediate reject that leaves the process alive. It also checks that the
// fixture repository is the expected two-commit, one-line-diff shape.
func TestNFRPluginPathSmoke(t *testing.T) {
	nfrRequireTools(t)
	manifest := nfrPluginManifest(t)
	repo := nfrTinyGitRepo(t)

	if d := nfrPluginColdSample(t, nfrPluginStore(t), manifest); d <= 0 {
		t.Fatalf("cold start sample = %v, want a positive duration", d)
	}

	mgr := nfrNewPluginManager(t, nfrPluginStore(t), manifest)
	nfrShutdownAfter(t, mgr)
	pid := nfrPluginPIDAfterWarmUp(t, mgr, repo)
	for i := 0; i < 3; i++ {
		if d := nfrDiffCall(t, mgr, repo, "smoke"); d <= 0 {
			t.Fatalf("warm call %d duration = %v, want a positive duration", i, d)
		}
	}
	nfrRequireSamePID(t, mgr, pid, "smoke warm calls")

	_, serr, _ := nfrCall(mgr, "smoke-reject", map[string]any{})
	if serr == nil || serr.Code != nfrPluginRejectCode {
		t.Fatalf("empty inputs: want immediate %s reject, got error=%+v", nfrPluginRejectCode, serr)
	}
	nfrRequireSamePID(t, mgr, pid, "after the immediate reject")
}
