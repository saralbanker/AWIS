// Package main is the AWIS command-line interface.
// It provides the development-loop commands described in TDS-07 (docs/CLI_CONTRACT.md).
// Commands are dispatched via a stdlib flag mux; no third-party CLI framework is used
// (dependency policy: IMP §16, SPEC Conventions).
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/awis/awis/internal/buildinfo"
)

// Global flags — parsed before the subcommand name.
var (
	globalDataDir   string
	globalJSON      bool
	globalNamespace string
)

// command is a registered subcommand.
type command struct {
	fn      func(args []string)
	summary string
}

// commands maps subcommand names to their implementations.
// Commands not yet implemented (C2–C4) are absent from this map;
// unknown commands trigger a usage error (exit 2).
var commands = map[string]command{
	"version": {fn: runVersion, summary: "Show version and build info"},
}

func main() {
	// Global flags parsed before subcommand name.
	flag.StringVar(&globalDataDir, "data-dir", "./.awis/", "Data directory (runtime.db, awis.pid, logs)")
	flag.BoolVar(&globalJSON, "json", false, "Output machine-readable JSON")
	flag.StringVar(&globalNamespace, "namespace", "default", "Namespace to target (default: \"default\")")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usageText())
	}

	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usageText())
		os.Exit(2)
	}

	name := args[0]
	rest := args[1:]

	cmd, ok := commands[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "awis: unknown command %q\n\n", name)
		fmt.Fprint(os.Stderr, usageText())
		os.Exit(2)
	}

	cmd.fn(rest)
}

// versionOutput is the JSON schema for 'awis version --json' (TDS-07 §4).
type versionOutput struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
}

// runVersion implements 'awis version'.
// Human format:  awis version <version>  go <go_version>
// JSON format:   {"version":"…","go_version":"…"}
func runVersion(args []string) {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: awis version\n\nShow version and build info.\n")
	}
	_ = fs.Parse(args)

	goVer := runtime.Version()

	if globalJSON {
		out := versionOutput{
			Version:   buildinfo.Version,
			GoVersion: goVer,
		}
		emitJSON(out)
		return
	}

	fmt.Printf("awis version %s  go %s\n", buildinfo.Version, goVer)
}
