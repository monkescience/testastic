package testastic

import (
	"flag"
	"os"
	"strings"
)

// Config holds the configuration for JSON comparison.
type Config struct {
	IgnoreArrayOrder      bool
	IgnoreArrayOrderPaths []string
	IgnoredFields         []string
	Update                bool
}

// Option is a functional option for configuring JSON comparison.
type Option func(*Config)

// IgnoreFields excludes the specified fields from comparison.
// Fields can be simple names or JSON paths (e.g., "$.user.id").
func IgnoreFields(fields ...string) Option {
	return func(c *Config) {
		c.IgnoredFields = append(c.IgnoredFields, fields...)
	}
}

// IgnoreArrayOrder makes array comparison order-insensitive globally.
func IgnoreArrayOrder() Option {
	return func(c *Config) {
		c.IgnoreArrayOrder = true
	}
}

// IgnoreArrayOrderAt makes array comparison order-insensitive at the specified JSON path.
func IgnoreArrayOrderAt(path string) Option {
	return func(c *Config) {
		c.IgnoreArrayOrderPaths = append(c.IgnoreArrayOrderPaths, path)
	}
}

// Update forces updating the expected file with the actual value.
func Update() Option {
	return func(c *Config) {
		c.Update = true
	}
}

// newConfig creates a new Config with default values and applies options.
func newConfig(opts ...Option) *Config {
	cfg := &Config{
		Update: shouldUpdate(),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
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

// shouldIgnoreArrayOrder checks if array order should be ignored at the given path.
func (c *Config) shouldIgnoreArrayOrder(path string) bool {
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

// isFieldIgnored checks if a field at the given path should be ignored.
func (c *Config) isFieldIgnored(path string) bool {
	for _, f := range c.IgnoredFields {
		// Exact match
		if f == path {
			return true
		}
		// Match by field name (last segment)
		parts := strings.Split(path, ".")
		if len(parts) > 0 && parts[len(parts)-1] == f {
			return true
		}
	}

	return false
}
