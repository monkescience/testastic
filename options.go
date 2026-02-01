package testastic

import (
	"flag"
	"os"
	"strings"
)

// baseConfig holds common configuration shared across all assertion types.
// This struct is embedded in JSONConfig, YAMLConfig, HTMLConfig, and FileConfig.
type baseConfig struct {
	IgnoreArrayOrder      bool
	IgnoreArrayOrderPaths []string
	IgnoredFields         []string
	Update                bool
	Message               string
}

// compareConfig defines methods for querying comparison configuration.
type compareConfig interface {
	ShouldIgnoreArrayOrder(path string) bool
	IsFieldIgnored(path string) bool
}

// ShouldIgnoreArrayOrder checks if array order should be ignored at the given path.
func (c *baseConfig) ShouldIgnoreArrayOrder(path string) bool {
	if c.IgnoreArrayOrder {
		return true
	}

	for _, p := range c.IgnoreArrayOrderPaths {
		if p == path || strings.HasPrefix(path, p+".") || strings.HasPrefix(path, p+"[") {
			return true
		}
	}

	return false
}

// IsFieldIgnored checks if a field at the given path should be ignored.
// Fields can be matched by exact path (e.g., "$.user.id") or by field name only (e.g., "id").
// When matching by field name, all fields with that name at any depth are ignored.
func (c *baseConfig) IsFieldIgnored(path string) bool {
	if len(c.IgnoredFields) == 0 {
		return false
	}

	parts := strings.Split(path, ".")

	lastPart := ""
	if len(parts) > 0 {
		lastPart = parts[len(parts)-1]
	}

	for _, f := range c.IgnoredFields {
		if f == path || f == lastPart {
			return true
		}
	}

	return false
}

// shouldUpdate checks if expected files should be updated.
// Checks for -update flag or TESTASTIC_UPDATE environment variable.
func shouldUpdate() bool {
	// Check environment variable
	if env := os.Getenv("TESTASTIC_UPDATE"); env != "" {
		return strings.ToLower(env) == "true" || env == "1"
	}

	// Check for -update flag
	for _, arg := range os.Args[1:] {
		if arg == "-update" || arg == "--update" {
			return true
		}
	}

	// Check if flag is registered and set
	if f := flag.Lookup("update"); f != nil {
		return f.Value.String() == "true"
	}

	return false
}
