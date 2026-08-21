package main

// init.go — 'awis init' command (FR-RM-01; M17-C3).
//
// Behaviour (verbatim from SPEC §35 + FR-RM-01):
//  1. Determine target directory (first positional arg, or cwd if omitted).
//  2. If target is non-empty and --force is not set, exit 1 with what/where/what-now error.
//  3. Write scaffold files from the embedded FS (go:embed all:scaffold — the "all:" prefix is
//     required so dot-files such as scaffold/.gitignore are not excluded by the default
//     embed rule that drops files/dirs starting with "." or "_"):
//       config.yaml
//       .gitignore
//       workflows/hello-world.yaml      — byte-identical to examples/workflows/
//       workflows/with-signal.yaml      — byte-identical to examples/workflows/
//       workflows/with-intelligence.yaml — byte-identical to examples/workflows/
//       handlers/example_handler.go
//       README_AWIS.md
//  4. Print the list of created files (human) or JSON on --json.

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:scaffold
var scaffoldFS embed.FS

func init() {
	commands["init"] = command{fn: runInit, summary: "Initialize project structure and config (FR-RM-01)"}
}

// initOutputJSON is the JSON schema for 'awis init --json' (TDS-07 §4 init).
type initOutputJSON struct {
	Target string   `json:"target"`
	Files  []string `json:"files"`
}

// scaffoldFiles enumerates the files embedded under scaffold/ in the order
// they must be written to the target directory.
// The "scaffold/" prefix is stripped when writing.
var scaffoldFiles = []string{
	"scaffold/config.yaml",
	"scaffold/.gitignore",
	"scaffold/workflows/hello-world.yaml",
	"scaffold/workflows/with-signal.yaml",
	"scaffold/workflows/with-intelligence.yaml",
	"scaffold/handlers/example_handler.go",
	"scaffold/README_AWIS.md",
}

func runInit(args []string) {
	var force bool
	fs2 := newFlagSet("init")
	fs2.BoolVar(&force, "force", false, "Overwrite files in a non-empty target directory")
	fs2.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: awis init [--force] [directory]\n\nInitialize AWIS project structure in [directory] (default: current directory).\n")
	}
	mustParse(fs2, args)

	// Determine target directory.
	target := "."
	if fs2.NArg() > 0 {
		target = fs2.Arg(0)
	}

	// Resolve to absolute path.
	absTarget, err := filepath.Abs(target)
	if err != nil {
		fail(1, fmt.Sprintf("init: cannot resolve target path: %s", err), target, "")
	}

	// Create target directory if it does not exist.
	if err := os.MkdirAll(absTarget, 0o755); err != nil {
		fail(1, fmt.Sprintf("init: cannot create target directory: %s", err), absTarget, "check file permissions")
	}

	// Guard: refuse non-empty target unless --force.
	if !force {
		entries, err := os.ReadDir(absTarget)
		if err != nil {
			fail(1, fmt.Sprintf("init: cannot read target directory: %s", err), absTarget, "check file permissions")
		}
		if len(entries) > 0 {
			fail(1,
				fmt.Sprintf("init: target directory is not empty (%d entries)", len(entries)),
				absTarget,
				"use --force to overwrite",
			)
		}
	}

	// Write scaffold files.
	written := make([]string, 0, len(scaffoldFiles))
	for _, embPath := range scaffoldFiles {
		data, err := fs.ReadFile(scaffoldFS, embPath)
		if err != nil {
			fail(1, fmt.Sprintf("init: cannot read embedded file %q: %s", embPath, err), embPath, "this is a build error; reinstall awis")
		}
		// Strip "scaffold/" prefix.
		relPath := embPath[len("scaffold/"):]
		destPath := filepath.Join(absTarget, filepath.FromSlash(relPath))

		// Create parent directory.
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			fail(1, fmt.Sprintf("init: cannot create directory for %q: %s", relPath, err), filepath.Dir(destPath), "check file permissions")
		}

		// Write file (overwrite if --force).
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			fail(1, fmt.Sprintf("init: cannot write %q: %s", relPath, err), destPath, "check file permissions")
		}
		written = append(written, relPath)
	}

	// Output.
	if globalJSON {
		out := initOutputJSON{
			Target: absTarget,
			Files:  written,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(out); err != nil {
			fail(1, "init: JSON encode failed", "", "")
		}
		return
	}

	// Human output.
	fmt.Printf("Initialized AWIS project in %s\n\n", absTarget)
	for _, f := range written {
		fmt.Printf("  created  %s\n", f)
	}
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  awis start\n")
	fmt.Printf("  awis submit hello-world --input name=World\n")
	fmt.Printf("  awis status\n")
}
