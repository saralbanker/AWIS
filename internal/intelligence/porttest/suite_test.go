package porttest_test

import (
	"testing"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/intelligence/adapters/null"
	"github.com/awis/awis/internal/intelligence/porttest"
)

// TestNullAdapterContract runs the full IntelligencePort contract suite against
// the NullAdapter, verifying AC-4: "IntelligencePort contract suite exists and
// NullAdapter passes it."
func TestNullAdapterContract(t *testing.T) {
	porttest.Run(t, func(t *testing.T) core.IntelligencePort {
		return null.New()
	})
}
