package main

// signal.go — 'awis signal' command (TDS-07 §4; M14-C3 T7).
//
// Delivers a signal to a waiting instance using the M07 signal path
// (direct storage write under WAL+busy-timeout; engine picks up on next tick ≤100ms).

import (
	"context"
	"fmt"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/sdk"
)

func init() {
	commands["signal"] = command{fn: runSignal, summary: "Deliver a signal to a waiting instance"}
}

// signalOutput is the JSON schema for 'awis signal --json' (TDS-07 §4).
type signalOutput struct {
	InstanceID string  `json:"instance_id"`
	SignalName string  `json:"signal_name"`
	Delivered  bool    `json:"delivered"`
	NextStep   *string `json:"next_step"`
}

func runSignal(args []string) {
	fs := newFlagSet("signal")
	var payloadFlags multiStringFlag
	fs.Var(&payloadFlags, "payload", "Payload key=value or @file.json (repeatable)")
	mustParse(fs, args)

	rest := fs.Args()
	if len(rest) < 2 {
		fail(2, "signal requires <instance-id> and <signal-name> arguments", "", "awis signal <instance-id> <signal-name>")
	}
	instanceID := core.InstanceID(rest[0])
	signalName := rest[1]

	payload, err := parseInputFlags(payloadFlags)
	if err != nil {
		fail(1, fmt.Sprintf("signal: payload parse error: %s", err), "", "use --payload key=value or --payload @file.json")
	}

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("signal: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	// Check instance exists and is waiting.
	ctx := context.Background()
	inst, serr := store.GetInstance(ctx, instanceID)
	if serr != nil {
		fail(1, fmt.Sprintf("instance %q not found", instanceID),
			"--data-dir "+globalDataDir,
			"awis status")
	}
	if inst.Status != core.InstanceStatusWaiting {
		fail(1, fmt.Sprintf("instance %q is not in waiting state (status: %s)", instanceID, inst.Status),
			string(instanceID),
			"awis status  # to see current instance state")
	}

	// Deliver signal via runtime (M07 path).
	rt, rerr := sdk.NewRuntime(sdk.Config{
		Namespace: string(inst.Namespace),
		Storage:   store,
	})
	if rerr != nil {
		fail(1, fmt.Sprintf("signal: cannot create runtime: %s", rerr), "", "")
	}

	if err := rt.Signal(ctx, instanceID, signalName, payload); err != nil {
		fail(1, fmt.Sprintf("signal: delivery failed: %s", err),
			string(instanceID),
			"awis status "+string(instanceID))
	}

	// Determine the next step, if the instance has actually advanced yet.
	//
	// Delivering a signal only RECORDS the delivery; the engine's next tick is
	// what resumes the instance. Reading current_steps straight afterwards
	// therefore usually still shows the step that was WAITING, and reporting
	// that as the "next step" named the step the signal had just satisfied —
	// while also claiming the instance had resumed when it had not.
	//
	// So next_step is reported only when it genuinely differs from the step
	// that was parked. Otherwise it stays nil and the human output says the
	// runtime will resume the instance on its next tick, which is the truth.
	// Compare the whole SET of parked steps, not just index 0. An instance can
	// hold several concurrent waits (a parallel join of signal steps — see
	// status.go's lookupWait), and the signal just delivered may resolve a
	// branch that is not at index 0. Comparing only CurrentSteps[0] would then
	// see an unchanged value and report "not resumed yet" even though a
	// different branch genuinely advanced.
	wasParked := make(map[string]bool, len(inst.CurrentSteps))
	for _, s := range inst.CurrentSteps {
		wasParked[s] = true
	}
	var nextStep *string
	if updated, serr := store.GetInstance(ctx, instanceID); serr == nil {
		for _, s := range updated.CurrentSteps {
			if !wasParked[s] {
				step := s
				nextStep = &step
				break
			}
		}
	}

	if globalJSON {
		out := signalOutput{
			InstanceID: string(instanceID),
			SignalName: signalName,
			Delivered:  true,
			NextStep:   nextStep,
		}
		emitJSON(out)
		return
	}

	// TDS-07 §4 human output:
	// Signal delivered: entry_confirmed → i-d4e5f6
	// Instance resumed; next step: append-to-record
	fmt.Printf("Signal delivered: %s → %s\n", signalName, instanceID)
	if nextStep != nil {
		fmt.Printf("Instance resumed; next step: %s\n", *nextStep)
	} else {
		fmt.Printf("The runtime will resume the instance on its next tick.\n")
		fmt.Printf("  What now: awis status %s\n", instanceID)
	}
}
