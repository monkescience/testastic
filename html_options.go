package testastic

import (
	"slices"
	"strings"
)

// HTMLConfig holds the configuration for HTML comparison.
type HTMLConfig struct {
	IgnoreComments        bool
	PreserveWhitespace    bool
	IgnoreChildOrder      bool
	IgnoreChildOrderPaths []string
	IgnoredElements       []string
	IgnoredAttributes     []string
	IgnoredAttributePaths []string
	Update                bool
}

// HTMLOption is a functional option for configuring HTML comparison.
type HTMLOption func(*HTMLConfig)

// IgnoreHTMLComments excludes HTML comments from comparison.
func IgnoreHTMLComments() HTMLOption {
	return func(c *HTMLConfig) {
		c.IgnoreComments = true
	}
}

// PreserveWhitespace disables whitespace normalization.
// By default, insignificant whitespace is collapsed.
func PreserveWhitespace() HTMLOption {
	return func(c *HTMLConfig) {
		c.PreserveWhitespace = true
	}
}

// IgnoreChildOrder makes child element comparison order-insensitive globally.
func IgnoreChildOrder() HTMLOption {
	return func(c *HTMLConfig) {
		c.IgnoreChildOrder = true
	}
}

// IgnoreChildOrderAt makes child comparison order-insensitive at the specified HTML path.
func IgnoreChildOrderAt(path string) HTMLOption {
	return func(c *HTMLConfig) {
		c.IgnoreChildOrderPaths = append(c.IgnoreChildOrderPaths, path)
	}
}

// IgnoreElements excludes elements matching the specified tag names from comparison.
func IgnoreElements(tags ...string) HTMLOption {
	return func(c *HTMLConfig) {
		c.IgnoredElements = append(c.IgnoredElements, tags...)
	}
}

// IgnoreAttributes excludes the specified attribute names from comparison globally.
func IgnoreAttributes(attrs ...string) HTMLOption {
	return func(c *HTMLConfig) {
		c.IgnoredAttributes = append(c.IgnoredAttributes, attrs...)
	}
}

// IgnoreAttributeAt excludes a specific attribute at a given path.
// Format: "path@attribute" e.g., "html > body > div@class".
func IgnoreAttributeAt(pathAttr string) HTMLOption {
	return func(c *HTMLConfig) {
		c.IgnoredAttributePaths = append(c.IgnoredAttributePaths, pathAttr)
	}
}

// HTMLUpdate forces updating the expected file with the actual value.
func HTMLUpdate() HTMLOption {
	return func(c *HTMLConfig) {
		c.Update = true
	}
}

// newHTMLConfig creates a new HTMLConfig with default values and applies options.
func newHTMLConfig(opts ...HTMLOption) *HTMLConfig {
	cfg := &HTMLConfig{
		Update: shouldUpdate(),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// shouldIgnoreChildOrder checks if child order should be ignored at the given path.
func (c *HTMLConfig) shouldIgnoreChildOrder(path string) bool {
	if c.IgnoreChildOrder {
		return true
	}

	for _, p := range c.IgnoreChildOrderPaths {
		if p == path || strings.HasPrefix(path, p+" > ") {
			return true
		}
	}

	return false
}

// isElementIgnored checks if an element with the given tag should be ignored.
func (c *HTMLConfig) isElementIgnored(tag string) bool {
	for _, t := range c.IgnoredElements {
		if strings.EqualFold(t, tag) {
			return true
		}
	}

	return false
}

// isAttributeIgnored checks if an attribute should be ignored.
func (c *HTMLConfig) isAttributeIgnored(path, attr string) bool {
	// Check global attribute ignores
	for _, a := range c.IgnoredAttributes {
		if strings.EqualFold(a, attr) {
			return true
		}
	}

	// Check path-specific attribute ignores
	pathAttr := path + "@" + attr

	return slices.Contains(c.IgnoredAttributePaths, pathAttr)
}
