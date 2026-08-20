package dsl

import "path/filepath"

// Discover returns the sorted list of *.yaml files in dir/workflows/.
// A missing directory (or no matching files) returns an empty slice and nil
// error. Results are in lexical order (filepath.Glob guarantee).
// Starting a runtime is not in scope (T6; IMP §27.M10; PRD §18).
func Discover(dir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "workflows", "*.yaml"))
	if err != nil {
		return nil, err
	}
	if matches == nil {
		return []string{}, nil
	}
	return matches, nil
}
