package engine

import (
	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/expr"
)

// assembleContext builds the StepContext for a step: it template-resolves the
// step's inputs against the instance environment (CONTRA-10: inputs are a
// template value-map, not a JSON schema) and wires the step-scoped logger.
//
// Returns a *core.StepError with code "template_error" if a template fails to
// PARSE at runtime (the validator should have caught this at Submit; the runtime
// guard stays typed, per the card). Resolution warnings never fail — they are
// logged at Warn (FR-WD-05).
func (e *Engine) assembleContext(inst core.WorkflowInstance, step core.Step, attempt int) (core.StepContext, *core.StepError) {
	env := buildEnv(inst)
	stepLog := e.log().With(
		"instance_id", string(inst.InstanceID),
		"step_id", step.ID,
	)
	warnSink := func(w expr.Warning) {
		stepLog.Warn("template resolution warning", "path", w.Path, "reason", w.Reason)
	}

	resolved, err := resolveValue(step.Inputs, env, warnSink)
	if err != nil {
		return core.StepContext{}, &core.StepError{
			Code:    "template_error",
			Message: err.Error(),
		}
	}
	inputs, _ := resolved.(map[string]any)
	if inputs == nil {
		inputs = map[string]any{}
	}

	return core.StepContext{
		InstanceID: string(inst.InstanceID),
		StepID:     step.ID,
		Attempt:    attempt,
		Inputs:     inputs,
		// Intelligence stays nil at C1 (no dispatcher wired; C2 supplies it).
		Logger: stepLog,
		// Deadline is set by the runner from step.Timeout (native runner).
	}, nil
}

// resolveValue walks v, template-resolving every string leaf (nested maps and
// slices are walked recursively). Non-string, non-composite leaves pass through
// unchanged. A ParseTemplate failure on any leaf aborts with an error.
func resolveValue(v any, env expr.Env, warnSink func(expr.Warning)) (any, error) {
	switch x := v.(type) {
	case string:
		tmpl, err := expr.ParseTemplate(x)
		if err != nil {
			return nil, err
		}
		out, warns := tmpl.Resolve(env)
		for _, w := range warns {
			warnSink(w)
		}
		return out, nil
	case map[string]any:
		m := make(map[string]any, len(x))
		for k, vv := range x {
			r, err := resolveValue(vv, env, warnSink)
			if err != nil {
				return nil, err
			}
			m[k] = r
		}
		return m, nil
	case []any:
		s := make([]any, len(x))
		for i, vv := range x {
			r, err := resolveValue(vv, env, warnSink)
			if err != nil {
				return nil, err
			}
			s[i] = r
		}
		return s, nil
	default:
		return v, nil
	}
}
