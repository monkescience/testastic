package testastic

import (
	"flag"
	"os"
	"slices"
	"strings"
)

// Option configures the behavior of assertion functions.
// Use the provided option constructors (IgnoreFields, IgnoreArrayOrder, etc.) to create options.
type Option func(*config)

// config holds all configuration for assertion comparison.
// Shared fields apply to all assertion types. HTML-specific fields
// are only used by AssertHTML and are ignored by other assertion types.
type config struct {
	// Shared fields.
	IgnoreArrayOrder      bool
	IgnoreArrayOrderPaths []string
	IgnoredFields         []string
	Update                bool
	Message               string

	// HTML-specific fields.
	IgnoreComments        bool
	PreserveWhitespace    bool
	IgnoredElements       []string
	IgnoredAttributes     []string
	IgnoredAttributePaths []string
}

// buildConfig creates a config from the provided options.
func buildConfig(opts []Option) *config {
	cfg := &config{Update: shouldUpdate()}
	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// ShouldIgnoreArrayOrder checks if array order should be ignored at the given path.
func (c *config) ShouldIgnoreArrayOrder(path string) bool {
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
func (c *config) IsFieldIgnored(path string) bool {
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

// isElementIgnored checks if an element with the given tag should be ignored (HTML only).
func (c *config) isElementIgnored(tag string) bool {
	for _, t := range c.IgnoredElements {
		if strings.EqualFold(t, tag) {
			return true
		}
	}

	return false
}

// isAttributeIgnored checks if an attribute should be ignored (HTML only).
func (c *config) isAttributeIgnored(path, attr string) bool {
	for _, a := range c.IgnoredAttributes {
		if strings.EqualFold(a, attr) {
			return true
		}
	}

	pathAttr := path + "@" + attr

	return slices.Contains(c.IgnoredAttributePaths, pathAttr)
}

// shouldIgnoreChildOrder checks if child order should be ignored at the given path (HTML only).
func (c *config) shouldIgnoreChildOrder(path string) bool {
	return c.ShouldIgnoreArrayOrder(path)
}

// shouldUpdate checks if expected files should be updated.
// Checks for -update flag or TESTASTIC_UPDATE environment variable.
func shouldUpdate() bool {
	// Check environment variable.
	if env := os.Getenv("TESTASTIC_UPDATE"); env != "" {
		return strings.ToLower(env) == "true" || env == "1"
	}

	// Check for -update flag.
	for _, arg := range os.Args[1:] {
		if arg == "-update" || arg == "--update" {
			return true
		}
	}

	// Check if flag is registered and set.
	if f := flag.Lookup("update"); f != nil {
		return f.Value.String() == "true"
	}

	return false
}

// IgnoreFields excludes the specified fields from comparison.
// Fields can be specified by name (matches at any depth) or by full path.
//
// By name (matches all occurrences):
//
//	IgnoreFields("id", "timestamp")  // ignores all "id" and "timestamp" fields
//
// By path (matches specific locations):
//
//	IgnoreFields("$.user.id", "$.metadata.createdAt")
//
// Path format uses JSONPath-like syntax:
//   - $ represents the root
//   - .field accesses object properties
//   - [0] accesses array elements
func IgnoreFields(fields ...string) Option {
	return func(c *config) {
		c.IgnoredFields = append(c.IgnoredFields, fields...)
	}
}

// IgnoreArrayOrder makes array comparison order-insensitive globally.
// When enabled, arrays are compared as sets: [1, 2, 3] equals [3, 1, 2].
//
// Use [IgnoreArrayOrderAt] to apply order-insensitive comparison
// only at specific paths.
func IgnoreArrayOrder() Option {
	return func(c *config) {
		c.IgnoreArrayOrder = true
	}
}

// IgnoreArrayOrderAt makes array comparison order-insensitive at the specified path.
// Unlike [IgnoreArrayOrder], this only affects the array at the given path.
//
// Path format uses JSONPath-like syntax:
//
//	IgnoreArrayOrderAt("$.items")          // root-level items array
//	IgnoreArrayOrderAt("$.users[0].roles") // roles array in first user
//
// The option also applies to nested arrays within the specified path.
func IgnoreArrayOrderAt(path string) Option {
	return func(c *config) {
		c.IgnoreArrayOrderPaths = append(c.IgnoreArrayOrderPaths, path)
	}
}

// Update forces updating the expected file with the actual value.
// This is equivalent to running tests with the -update flag or
// setting TESTASTIC_UPDATE=true.
//
// Use this option when you need to programmatically update expected files:
//
//	if shouldUpdateGoldenFiles {
//	    testastic.AssertJSON(t, expected, actual, testastic.Update())
//	}
func Update() Option {
	return func(c *config) {
		c.Update = true
	}
}

// Message adds a custom message to the assertion failure output.
// This helps identify which assertion failed when a test has multiple assertions.
//
//	testastic.AssertJSON(t, expected, actual,
//	    testastic.Message("user creation response"),
//	)
func Message(msg string) Option {
	return func(c *config) {
		c.Message = msg
	}
}

// IgnoreHTMLComments excludes HTML comments from comparison.
func IgnoreHTMLComments() Option {
	return func(c *config) {
		c.IgnoreComments = true
	}
}

// PreserveWhitespace disables whitespace normalization in HTML comparison.
func PreserveWhitespace() Option {
	return func(c *config) {
		c.PreserveWhitespace = true
	}
}

// IgnoreChildOrder makes child element comparison order-insensitive globally in HTML.
func IgnoreChildOrder() Option {
	return func(c *config) {
		c.IgnoreArrayOrder = true
	}
}

// IgnoreChildOrderAt makes child comparison order-insensitive at the specified HTML path.
func IgnoreChildOrderAt(path string) Option {
	return func(c *config) {
		c.IgnoreArrayOrderPaths = append(c.IgnoreArrayOrderPaths, path)
	}
}

// IgnoreElements excludes elements matching the specified tag names from HTML comparison.
func IgnoreElements(tags ...string) Option {
	return func(c *config) {
		c.IgnoredElements = append(c.IgnoredElements, tags...)
	}
}

// IgnoreAttributes excludes the specified attribute names from HTML comparison globally.
func IgnoreAttributes(attrs ...string) Option {
	return func(c *config) {
		c.IgnoredAttributes = append(c.IgnoredAttributes, attrs...)
	}
}

// IgnoreAttributeAt excludes a specific attribute at a given HTML path.
func IgnoreAttributeAt(pathAttr string) Option {
	return func(c *config) {
		c.IgnoredAttributePaths = append(c.IgnoredAttributePaths, pathAttr)
	}
}
