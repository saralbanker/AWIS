package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

// cliErr prints a structured error in the what/where/what-now format
// (PRD §26 / TDS-07 §5) to stderr and returns an exit code. Callers should
// call os.Exit with the returned code.
//
// what    — what happened (specific, not generic)
// where   — instance ID, step ID, file, config key, line number
// whatNow — a concrete next command or action
func cliErr(what, where, whatNow string) {
	fmt.Fprintf(os.Stderr, "awis: %s\n", what)
	if where != "" {
		fmt.Fprintf(os.Stderr, "  Where:    %s\n", where)
	}
	if whatNow != "" {
		fmt.Fprintf(os.Stderr, "  What now: %s\n", whatNow)
	}
}

// fail prints a structured error and exits with the given code. Convenience
// wrapper for cliErr + os.Exit.
//
// Exit codes (TDS-07 §2):
//
//	0  ok
//	1  operational error
//	2  usage error
//	3  validation failure
func fail(code int, what, where, whatNow string) {
	cliErr(what, where, whatNow)
	os.Exit(code)
}

// newFlagSet creates a flag.FlagSet for a subcommand using ExitOnError.
func newFlagSet(name string) *flag.FlagSet {
	return flag.NewFlagSet(name, flag.ExitOnError)
}

// mustParse permutes args so that every flag (and, for a non-boolean flag,
// its separately-supplied value) precedes every positional argument, then
// parses the permuted list with fs. On error ExitOnError handles os.Exit(2).
//
// Without the permutation, stdlib's flag.Parse stops scanning for flags at
// the first non-flag (positional) argument, so any flag written after a
// positional argument is silently discarded — e.g. `awis submit hello-world
// --input name=World` would record `{"inputs":{}}` (defect B-6).
func mustParse(fs *flag.FlagSet, args []string) {
	_ = fs.Parse(permuteArgs(fs, args)) // ExitOnError handles parse failures.
}

// permuteArgs reorders args so every flag token — and, for a registered
// non-boolean flag written without "=", the argument that supplies its
// value — comes before every positional argument, while preserving the
// original relative order within each group. When at least one positional
// argument is present, a literal "--" terminator is inserted immediately
// ahead of the (reordered) positionals, so fs.Parse does not try to
// interpret a positional that happens to start with "-" as a flag.
//
// A literal "--" in the input terminates flag scanning immediately;
// everything after it is appended to the positional group verbatim, exactly
// as stdlib's flag package treats it.
//
// Unknown flags are left in the flag group without consuming a following
// argument as a value: fs.Parse rejects them with the existing usage error
// (exit 2) before value-consumption would ever matter.
func permuteArgs(fs *flag.FlagSet, args []string) []string {
	flagArgs := make([]string, 0, len(args))
	positional := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		a := args[i]

		if a == "--" {
			// "--" terminates flag scanning; everything after is positional.
			positional = append(positional, args[i+1:]...)
			break
		}

		if len(a) < 2 || a[0] != '-' {
			// Not flag syntax (includes a bare "-").
			positional = append(positional, a)
			continue
		}

		flagArgs = append(flagArgs, a)

		// Determine the flag name to look up (strip leading "-" or "--").
		name := a[1:]
		if len(name) > 0 && name[0] == '-' {
			name = name[1:]
		}
		if idx := strings.IndexByte(name, '='); idx >= 0 {
			// "-flag=value" / "--flag=value": value is embedded, nothing to consume.
			continue
		}

		fl := fs.Lookup(name)
		if fl == nil {
			// Unknown flag: don't guess whether it takes a value; fs.Parse
			// will reject it with the usual usage error.
			continue
		}
		if bv, ok := fl.Value.(interface{ IsBoolFlag() bool }); ok && bv.IsBoolFlag() {
			// Boolean flags don't consume the next argument (unless written
			// as "--flag=value", handled above).
			continue
		}

		// Non-boolean flag: the next argument, if any, is its value.
		if i+1 < len(args) {
			i++
			flagArgs = append(flagArgs, args[i])
		}
	}

	if len(positional) == 0 {
		return flagArgs
	}
	result := make([]string, 0, len(flagArgs)+len(positional)+1)
	result = append(result, flagArgs...)
	result = append(result, "--")
	result = append(result, positional...)
	return result
}

// emitJSON writes v as a single JSON line to stdout. An encode failure is a
// hard error: a --json consumer must never receive empty output and exit 0
// (defect B-8; this also fixes B-7, where an invalid json.RawMessage caused
// enc.Encode to fail silently and 'trace --json' printed nothing).
func emitJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		fail(1, fmt.Sprintf("JSON encode failed: %s", err), "", "this is a bug in awis; please report it")
	}
}

// usageText returns the global usage string printed on usage errors (exit 2).
func usageText() string {
	return `awis — AWIS workflow runtime CLI

Usage:
  awis [--data-dir <path>] [--json] [--namespace <ns>] <command> [flags]

Global flags:
  --data-dir <path>   Data directory (default: ./.awis/)
  --json              Output machine-readable JSON
  --namespace <ns>    Namespace to target for submit/signal/cancel/status/trace (default: "default")

Commands (M14 — live):
  version             Show version and build info
  start               Start the runtime (foreground)
  stop                Gracefully stop the runtime
  status              Live status: active + recent instances
  submit              Submit a workflow instance
  signal              Deliver a signal to a waiting instance
  cancel              Cancel a running or waiting instance
  trace               Full execution trace for one instance
  workflow            Workflow sub-commands (validate, list, show)
  plugin              Plugin sub-commands (install, list)

Commands (M17-C1 — live):
  history             Recent completed instances
  logs                Structured log stream
  metrics             Aggregate execution statistics
  recall              Query execution history (FTS)
  replay              Re-run completed instance (dry-run)
  audit               View audit log entries
  export              Export execution history to JSON
  prune-events        Prune EventLog (dry-run only in V1)

Commands (M17-C2 — live):
  config              Configuration sub-commands (show, set, validate, edit)
  rebuild-state       Rebuild workflow_instances projection from EventLog

Commands (M17-C3 — live):
  init                Initialize project structure and config (FR-RM-01)

Run 'awis <command> --help' for per-command usage.
`
}
