package main

import "testing"

// TestNamespacePredicate pins the mapping from a CLI namespace argument to a
// storage predicate, including the "default"-as-wildcard overloading that the
// read commands have always applied.
func TestNamespacePredicate(t *testing.T) {
	cases := []struct {
		in   string
		want string
		why  string
	}{
		{"", "", "unset means every namespace"},
		{"default", "", "the global flag's default value is treated as a wildcard"},
		{"examples", "examples", "a real namespace filters to itself"},
		{"nonexistent", "nonexistent", "an unknown namespace filters to nothing, it does not fall back to all"},
	}
	for _, tc := range cases {
		if got := namespacePredicate(tc.in); got != tc.want {
			t.Errorf("namespacePredicate(%q) = %q, want %q (%s)", tc.in, got, tc.want, tc.why)
		}
	}
}
