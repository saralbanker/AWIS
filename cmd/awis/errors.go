package main

import (
	"flag"
	"fmt"
	"os"
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

// mustParse parses args with fs; on error ExitOnError handles os.Exit(2).
func mustParse(fs *flag.FlagSet, args []string) {
	_ = fs.Parse(args) // ExitOnError handles parse failures.
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
