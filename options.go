package testastic

import (
	"flag"
	"os"
	"slices"
	"strings"
)

// assertType identifies which assertion function is being called.
type assertType string

const (
	assertJSON assertType = "json"
	assertYAML assertType = "yaml"
	assertHTML assertType = "html"
	assertFile assertType = "file"
)

// optionMeta records which option was applied and which assertion types support it.
type optionMeta struct {
	name        string
	supportedBy []assertType
}

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

	// applied tracks which options were set for validation.
	applied []optionMeta
}

func buildConfig(opts []Option) *config {
	cfg := &config{Update: shouldUpdate()}
	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// ShouldIgnoreArrayOrder checks if array order should be ignored at the given path.
// Returns true if global IgnoreArrayOrder is set, or if the path is at or under
// any path specified via [IgnoreArrayOrderAt].
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

// validateOptions returns the names of options that are not supported by the given assertion type.
func (c *config) validateOptions(target assertType) []string {
	var unsupported []string

	for _, opt := range c.applied {
		if !slices.Contains(opt.supportedBy, target) {
			unsupported = append(unsupported, opt.name)
		}
	}

	return unsupported
}

func (c *config) isElementIgnored(tag string) bool {
	for _, t := range c.IgnoredElements {
		if strings.EqualFold(t, tag) {
			return true
		}
	}

	return false
}

func (c *config) isAttributeIgnored(path, attr string) bool {
	for _, a := range c.IgnoredAttributes {
		if strings.EqualFold(a, attr) {
			return true
		}
	}

	pathAttr := path + "@" + attr

	return slices.Contains(c.IgnoredAttributePaths, pathAttr)
}

func (c *config) isHTMLFieldIgnored(path, tag string) bool {
	for _, field := range c.IgnoredFields {
		if field == path {
			return true
		}

		if tag != "" && strings.EqualFold(field, tag) {
			return true
		}
	}

	return false
}

func (c *config) shouldIgnoreChildOrder(path string) bool {
	return c.ShouldIgnoreArrayOrder(path)
}

// shouldUpdate checks if expected files should be updated.
// Checks for -update flag or TESTASTIC_UPDATE environment variable.
func shouldUpdate() bool {
	if env := os.Getenv("TESTASTIC_UPDATE"); env != "" {
		return strings.ToLower(env) == "true" || env == "1"
	}

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
		c.applied = append(c.applied, optionMeta{
			name:        "IgnoreFields",
			supportedBy: []assertType{assertJSON, assertYAML, assertHTML},
		})
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
		c.applied = append(c.applied, optionMeta{
			name:        "IgnoreArrayOrder",
			supportedBy: []assertType{assertJSON, assertYAML},
		})
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
		c.applied = append(c.applied, optionMeta{
			name:        "IgnoreArrayOrderAt",
			supportedBy: []assertType{assertJSON, assertYAML},
		})
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

// IgnoreHTMLComments excludes HTML comment nodes (<!-- ... -->) from comparison.
func IgnoreHTMLComments() Option {
	return func(c *config) {
		c.IgnoreComments = true
		c.applied = append(c.applied, optionMeta{
			name:        "IgnoreHTMLComments",
			supportedBy: []assertType{assertHTML},
		})
	}
}

// PreserveWhitespace disables whitespace normalization in HTML comparison.
func PreserveWhitespace() Option {
	return func(c *config) {
		c.PreserveWhitespace = true
		c.applied = append(c.applied, optionMeta{
			name:        "PreserveWhitespace",
			supportedBy: []assertType{assertHTML},
		})
	}
}

// IgnoreChildOrder makes child element comparison order-insensitive globally in HTML.
// Child elements are compared as sets rather than sequences.
func IgnoreChildOrder() Option {
	return func(c *config) {
		c.IgnoreArrayOrder = true
		c.applied = append(c.applied, optionMeta{
			name:        "IgnoreChildOrder",
			supportedBy: []assertType{assertHTML},
		})
	}
}

// IgnoreChildOrderAt makes child element comparison order-insensitive at the specified HTML path.
func IgnoreChildOrderAt(path string) Option {
	return func(c *config) {
		c.IgnoreArrayOrderPaths = append(c.IgnoreArrayOrderPaths, path)
		c.applied = append(c.applied, optionMeta{
			name:        "IgnoreChildOrderAt",
			supportedBy: []assertType{assertHTML},
		})
	}
}

// IgnoreElements excludes elements matching the specified tag names from HTML comparison.
// Tag matching is case-insensitive.
func IgnoreElements(tags ...string) Option {
	return func(c *config) {
		c.IgnoredElements = append(c.IgnoredElements, tags...)
		c.applied = append(c.applied, optionMeta{
			name:        "IgnoreElements",
			supportedBy: []assertType{assertHTML},
		})
	}
}

// IgnoreAttributes excludes the specified attribute names from HTML comparison globally.
func IgnoreAttributes(attrs ...string) Option {
	return func(c *config) {
		c.IgnoredAttributes = append(c.IgnoredAttributes, attrs...)
		c.applied = append(c.applied, optionMeta{
			name:        "IgnoreAttributes",
			supportedBy: []assertType{assertHTML},
		})
	}
}

// IgnoreAttributeAt excludes a specific attribute at a given HTML path.
// The pathAttr format is "path@attribute", e.g., "html > body > div@class".
func IgnoreAttributeAt(pathAttr string) Option {
	return func(c *config) {
		c.IgnoredAttributePaths = append(c.IgnoredAttributePaths, pathAttr)
		c.applied = append(c.applied, optionMeta{
			name:        "IgnoreAttributeAt",
			supportedBy: []assertType{assertHTML},
		})
	}
}
