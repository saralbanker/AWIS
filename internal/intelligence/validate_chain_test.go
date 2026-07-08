package intelligence

// ValidateChain unit coverage (M04-V1 advisory, validate_chain.go). The two
// failure modes (unregistered entry; null not last) and the three accepting
// shapes (null last; no null; empty chain).

import (
	"strings"
	"testing"
)

func chainReg(name string) Registration {
	return Registration{Adapter: &fakeAdapter{name: name}}
}

func TestValidateChain_UnregisteredEntry(t *testing.T) {
	err := ValidateChain(nil, []string{"ghost"})
	if err == nil {
		t.Fatalf("chain entry with no registration must error")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Fatalf("error must name the offending entry; got %v", err)
	}
}

func TestValidateChain_NullNotLast(t *testing.T) {
	regs := []Registration{chainReg("null"), chainReg("real")}
	err := ValidateChain(regs, []string{"null", "real"})
	if err == nil {
		t.Fatalf("null not last must error")
	}
	if !strings.Contains(err.Error(), "position 0") {
		t.Fatalf("error must name the offending position; got %v", err)
	}
}

func TestValidateChain_NullLastOK(t *testing.T) {
	regs := []Registration{chainReg("real"), chainReg("null")}
	if err := ValidateChain(regs, []string{"real", "null"}); err != nil {
		t.Fatalf("null last must be accepted; got %v", err)
	}
}

func TestValidateChain_NoNullOK(t *testing.T) {
	regs := []Registration{chainReg("a"), chainReg("b")}
	if err := ValidateChain(regs, []string{"a", "b"}); err != nil {
		t.Fatalf("chain without null must be accepted; got %v", err)
	}
}

func TestValidateChain_EmptyChainOK(t *testing.T) {
	regs := []Registration{chainReg("a")}
	if err := ValidateChain(regs, nil); err != nil {
		t.Fatalf("empty chain must be accepted; got %v", err)
	}
}
