package main

// errors_test.go — tests for the shared CLI helpers in errors.go:
//  - permuteArgs / mustParse (defect B-6: flags after a positional argument
//    were silently discarded by stdlib's flag.Parse).
//  - emitJSON (defect B-8: discarded JSON encode errors on --json output
//    paths; also the fix vehicle for B-7's silent empty output).

import (
	"bytes"
	"flag"
	"io"
	"reflect"
	"strings"
	"testing"
)

// newPermuteTestFlagSet builds a flag.FlagSet registering the flag shapes
// exercised by real subcommands: a boolean flag ("force", like init/cancel),
// a single-value string flag ("name"), and a repeatable flag ("input", the
// same multiStringFlag type submit.go and signal.go use for --input/--payload).
func newPermuteTestFlagSet() (fs *flag.FlagSet, force *bool, name *string, inputs *multiStringFlag) {
	fs = flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(new(bytes.Buffer)) // silence usage/error output in tests
	force = fs.Bool("force", false, "")
	name = fs.String("name", "", "")
	inputs = &multiStringFlag{}
	fs.Var(inputs, "input", "")
	return fs, force, name, inputs
}

// TestPermuteArgs is table-driven over the exact scenarios named in the
// B-6 task card: flag-before-positional, flag-after-positional,
// --flag=value form, boolean flag after positional, repeated --input flags
// after positional, "--" terminator, and unknown flag.
func TestPermuteArgs(t *testing.T) {
	cases := []struct {
		name           string
		args           []string
		wantPositional []string
		wantName       string
		wantForce      bool
		wantInputs     []string
	}{
		{
			name:           "flag-before-positional",
			args:           []string{"--name", "value", "myworkflow"},
			wantPositional: []string{"myworkflow"},
			wantName:       "value",
		},
		{
			name:           "flag-after-positional",
			args:           []string{"myworkflow", "--name", "value"},
			wantPositional: []string{"myworkflow"},
			wantName:       "value",
		},
		{
			name:           "flag-equals-value-form-after-positional",
			args:           []string{"myworkflow", "--name=value"},
			wantPositional: []string{"myworkflow"},
			wantName:       "value",
		},
		{
			name:           "boolean-flag-after-positional",
			args:           []string{"myworkflow", "--force"},
			wantPositional: []string{"myworkflow"},
			wantForce:      true,
		},
		{
			name:           "repeated-input-flags-after-positional",
			args:           []string{"myworkflow", "--input", "name=World", "--input", "foo=bar"},
			wantPositional: []string{"myworkflow"},
			wantInputs:     []string{"name=World", "foo=bar"},
		},
		{
			name:           "double-dash-terminator",
			args:           []string{"--name", "value", "--", "--not-a-flag", "myworkflow"},
			wantPositional: []string{"--not-a-flag", "myworkflow"},
			wantName:       "value",
		},
		{
			name:           "multiple-positionals-preserve-relative-order",
			args:           []string{"pos1", "--name", "value", "pos2"},
			wantPositional: []string{"pos1", "pos2"},
			wantName:       "value",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs, force, name, inputs := newPermuteTestFlagSet()
			permuted := permuteArgs(fs, tc.args)
			if err := fs.Parse(permuted); err != nil {
				t.Fatalf("fs.Parse(%v) (permuted from %v) failed: %v", permuted, tc.args, err)
			}

			if got := fs.Args(); !reflect.DeepEqual(got, tc.wantPositional) {
				t.Errorf("fs.Args() = %#v, want %#v (permuted: %v)", got, tc.wantPositional, permuted)
			}
			if *name != tc.wantName {
				t.Errorf("--name = %q, want %q", *name, tc.wantName)
			}
			if *force != tc.wantForce {
				t.Errorf("--force = %v, want %v", *force, tc.wantForce)
			}
			if tc.wantInputs != nil {
				if !reflect.DeepEqual([]string(*inputs), tc.wantInputs) {
					t.Errorf("--input = %#v, want %#v", []string(*inputs), tc.wantInputs)
				}
			}
		})
	}
}

// TestPermuteArgsUnknownFlag proves an unknown flag still produces the
// existing usage error (fs.Parse returns a non-nil error) rather than being
// silently treated as a positional argument — even after permutation, and
// even when the unknown flag appears after a positional argument.
func TestPermuteArgsUnknownFlag(t *testing.T) {
	fs, _, _, _ := newPermuteTestFlagSet()
	args := []string{"myworkflow", "--bogus", "value"}

	permuted := permuteArgs(fs, args)
	err := fs.Parse(permuted)
	if err == nil {
		t.Fatalf("fs.Parse(%v) = nil error, want an error for unknown flag --bogus", permuted)
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("fs.Parse error = %q, want it to mention the unknown flag %q", err.Error(), "bogus")
	}
}

// TestPermuteArgsNoPositionalsNoTrailingTerminator proves permuteArgs does
// not append a spurious "--" when there are no positional arguments (would
// otherwise be harmless but is unnecessary noise / a needless behavior change
// for the common all-flags case).
func TestPermuteArgsNoPositionalsNoTrailingTerminator(t *testing.T) {
	fs, _, _, _ := newPermuteTestFlagSet()
	args := []string{"--name", "value", "--force"}
	got := permuteArgs(fs, args)
	want := []string{"--name", "value", "--force"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("permuteArgs(%v) = %v, want %v", args, got, want)
	}
}

// TestMustParseSubmitRepro is a direct regression test for the PROVEN LIVE
// repro in the B-6 task card: `awis submit hello-world --input name=World`
// must record inputs, not `{}`. It builds a flag.FlagSet exactly as
// runSubmit does (same flag names/types) and drives it through mustParse.
func TestMustParseSubmitRepro(t *testing.T) {
	fs := newFlagSet("submit")
	var inputFlags multiStringFlag
	var wait bool
	var timeoutStr string
	fs.Var(&inputFlags, "input", "Input key=value or @file.json (repeatable)")
	fs.BoolVar(&wait, "wait", false, "Block until terminal status")
	fs.StringVar(&timeoutStr, "timeout", "1h", "Maximum wait duration (requires --wait)")

	mustParse(fs, []string{"hello-world", "--input", "name=World"})

	rest := fs.Args()
	if len(rest) != 1 || rest[0] != "hello-world" {
		t.Fatalf("fs.Args() = %#v, want [\"hello-world\"]", rest)
	}
	inputs, err := parseInputFlags(inputFlags)
	if err != nil {
		t.Fatalf("parseInputFlags: %v", err)
	}
	if got, want := inputs["name"], "World"; got != want {
		t.Errorf("inputs[%q] = %v, want %q (defect B-6: flag after positional was silently discarded, "+
			"producing inputs == {})", "name", got, want)
	}
}

// TestEmitJSON proves emitJSON writes exactly one line of valid,
// non-HTML-escaped JSON for a normal value (the converged success path used
// by every --json output site, including the former main.go runVersion
// duplicate-error-check code this replaces).
func TestEmitJSON(t *testing.T) {
	type payload struct {
		Message string `json:"message"`
	}
	got := captureOutput(func() {
		emitJSON(payload{Message: "a<b>c&d"})
	})
	// SetEscapeHTML(false) leaves '<', '>' and '&' unescaped.
	want := "{\"message\":\"a<b>c&d\"}\n"
	if string(got) != want {
		t.Errorf("emitJSON output = %q, want %q", string(got), want)
	}
}

// TestPermuteArgsDanglingFlagStillErrors: a non-boolean flag written last with
// no value is a user error, and flag.Parse must be allowed to say so.
//
// permuteArgs used to append its "--" positional terminator unconditionally.
// When the last token was a value-less flag, flag.Parse consumed that "--" as
// the flag's VALUE — turning a clean exit-2 "flag needs an argument" into a
// silent success with namespace="--", which then surfaced as a baffling
// "workflow not found". Reproduced against the real binary as:
//
//	awis workflow show myworkflow --namespace
func TestPermuteArgsDanglingFlagStillErrors(t *testing.T) {
	newFS := func() *flag.FlagSet {
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		fs.String("namespace", "", "ns")
		fs.Bool("full", false, "full")
		return fs
	}

	t.Run("dangling flag after a positional errors", func(t *testing.T) {
		fs := newFS()
		got := permuteArgs(fs, []string{"myworkflow", "--namespace"})
		if err := fs.Parse(got); err == nil {
			t.Fatalf("permuteArgs(%v) = %v; Parse succeeded with namespace=%q, want an error",
				[]string{"myworkflow", "--namespace"}, got, fs.Lookup("namespace").Value)
		}
	})

	t.Run("dangling flag with no positional errors", func(t *testing.T) {
		fs := newFS()
		if err := fs.Parse(permuteArgs(fs, []string{"--namespace"})); err == nil {
			t.Error("Parse succeeded on a value-less flag, want an error")
		}
	})

	t.Run("a dangling BOOLEAN flag is fine", func(t *testing.T) {
		fs := newFS()
		if err := fs.Parse(permuteArgs(fs, []string{"myworkflow", "--full"})); err != nil {
			t.Fatalf("Parse: %v", err)
		}
		if fs.Lookup("full").Value.String() != "true" {
			t.Error("--full was not set")
		}
		if len(fs.Args()) != 1 || fs.Args()[0] != "myworkflow" {
			t.Errorf("positionals = %v, want [myworkflow]", fs.Args())
		}
	})

	t.Run("the normal case still works", func(t *testing.T) {
		fs := newFS()
		if err := fs.Parse(permuteArgs(fs, []string{"myworkflow", "--namespace", "examples"})); err != nil {
			t.Fatalf("Parse: %v", err)
		}
		if got := fs.Lookup("namespace").Value.String(); got != "examples" {
			t.Errorf("namespace = %q, want examples", got)
		}
		if len(fs.Args()) != 1 || fs.Args()[0] != "myworkflow" {
			t.Errorf("positionals = %v, want [myworkflow]", fs.Args())
		}
	})
}
