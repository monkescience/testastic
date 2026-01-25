package testastic

import (
	"flag"
	"os"
	"strings"
)

// BaseConfig holds common configuration shared across all assertion types.
type BaseConfig struct {
	IgnoreArrayOrder      bool
	IgnoreArrayOrderPaths []string
	IgnoredFields         []string
	Update                bool
	Message               string
}

// CompareConfig is the interface for comparison configuration.
type CompareConfig interface {
	ShouldIgnoreArrayOrder(path string) bool
	IsFieldIgnored(path string) bool
}

// ShouldIgnoreArrayOrder checks if array order should be ignored at the given path.
func (c *BaseConfig) ShouldIgnoreArrayOrder(path string) bool {
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
func (c *BaseConfig) IsFieldIgnored(path string) bool {
	for _, f := range c.IgnoredFields {
		if f == path {
			return true
		}
		parts := strings.Split(path, ".")
		if len(parts) > 0 && parts[len(parts)-1] == f {
			return true
		}
	}

	return false
}

// Config holds the configuration for JSON comparison.
type Config struct {
	BaseConfig
}

// Option is a functional option for configuring JSON comparison.
type Option func(*Config)

// JSONIgnoreFields excludes the specified fields from comparison in JSON.
// Fields can be simple names or JSON paths (e.g., "$.user.id").
func JSONIgnoreFields(fields ...string) Option {
	return func(c *Config) {
		c.BaseConfig.IgnoredFields = append(c.BaseConfig.IgnoredFields, fields...)
	}
}

// JSONIgnoreArrayOrder makes array comparison order-insensitive globally in JSON.
func JSONIgnoreArrayOrder() Option {
	return func(c *Config) {
		c.BaseConfig.IgnoreArrayOrder = true
	}
}

// JSONIgnoreArrayOrderAt makes array comparison order-insensitive at the specified JSON path.
func JSONIgnoreArrayOrderAt(path string) Option {
	return func(c *Config) {
		c.BaseConfig.IgnoreArrayOrderPaths = append(c.BaseConfig.IgnoreArrayOrderPaths, path)
	}
}

// JSONUpdate forces updating the expected file with the actual value in JSON.
func JSONUpdate() Option {
	return func(c *Config) {
		c.BaseConfig.Update = true
	}
}

// JSONMessage adds a custom message to the assertion failure output in JSON.
func JSONMessage(msg string) Option {
	return func(c *Config) {
		c.BaseConfig.Message = msg
	}
}

// newConfig creates a new Config with default values and applies options.
func newConfig(opts ...Option) *Config {
	cfg := &Config{
		BaseConfig: BaseConfig{
			Update: shouldUpdate(),
		},
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
