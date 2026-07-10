package plugin

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// semverRE matches a semver string (coordinate: internal/dsl semverRE).
var semverRE = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)`)

// Manifest is the parsed representation of an awis-plugin.yaml file.
// Field names match the Blueprint §11 YAML verbatim.
type Manifest struct {
	Name         string       `yaml:"name"`
	Version      string       `yaml:"version"`
	Description  string       `yaml:"description"`
	Author       string       `yaml:"author"`
	Capabilities []Capability `yaml:"capabilities"`
	Runtime      Runtime      `yaml:"runtime"`
}

// Capability declares a single capability exported by a plugin.
// Field names match the Blueprint §11 YAML verbatim.
type Capability struct {
	ID        string         `yaml:"id"`
	Inputs    map[string]any `yaml:"inputs"`
	Outputs   map[string]any `yaml:"outputs"`
	TimeoutMS int            `yaml:"timeout_ms"`
}

// Runtime holds the spawn configuration for a plugin.
// Field names match the Blueprint §11 YAML verbatim.
type Runtime struct {
	Command      string            `yaml:"command"`
	Args         []string          `yaml:"args"`
	Env          map[string]string `yaml:"env"`
	IdleTimeoutS int               `yaml:"idle_timeout_s"`
}

// ParseManifest opens path and parses it as an awis-plugin.yaml manifest.
// Errors carry the file path and, where available, a source line number
// (the yaml.v3 decoder includes "line N:" in decode errors).
func ParseManifest(path string) (*Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("%s: read: %w", path, err)
	}
	return ParseManifestBytes(data, path)
}

// ParseManifestBytes parses a manifest from raw YAML bytes.
// filename is used in error messages.
func ParseManifestBytes(data []byte, filename string) (*Manifest, error) {
	var m Manifest
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}
	if err := validateManifest(&m, filename); err != nil {
		return nil, err
	}
	return &m, nil
}

// validateManifest enforces all manifest validation rules (SPEC §1).
func validateManifest(m *Manifest, filename string) error {
	if m.Name == "" {
		return fmt.Errorf("%s: manifest name must not be empty", filename)
	}
	if err := validateSemver(m.Version); err != nil {
		return fmt.Errorf("%s: %w", filename, err)
	}
	if len(m.Capabilities) == 0 {
		return fmt.Errorf("%s: manifest must declare at least one capability", filename)
	}
	seen := make(map[string]bool, len(m.Capabilities))
	for i, c := range m.Capabilities {
		if c.ID == "" {
			return fmt.Errorf("%s: capability[%d] id must not be empty", filename, i)
		}
		if seen[c.ID] {
			return fmt.Errorf("%s: duplicate capability id %q", filename, c.ID)
		}
		seen[c.ID] = true
	}
	if m.Runtime.Command == "" {
		return fmt.Errorf("%s: runtime.command must not be empty", filename)
	}
	return nil
}

func validateSemver(version string) error {
	if version == "" {
		return fmt.Errorf("version must not be empty")
	}
	if !semverRE.MatchString(version) {
		return fmt.Errorf("version %q does not match semver format (e.g. \"1.0.0\")", version)
	}
	return nil
}
