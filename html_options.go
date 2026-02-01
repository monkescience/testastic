package testastic

import (
	"slices"
	"strings"
)

// HTMLConfig holds the configuration for HTML comparison.
type HTMLConfig struct {
	baseConfig
	IgnoreComments        bool
	PreserveWhitespace    bool
	IgnoredElements       []string
	IgnoredAttributes     []string
	IgnoredAttributePaths []string
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
func PreserveWhitespace() HTMLOption {
	return func(c *HTMLConfig) {
		c.PreserveWhitespace = true
	}
}

// IgnoreChildOrder makes child element comparison order-insensitive globally.
func IgnoreChildOrder() HTMLOption {
	return func(c *HTMLConfig) {
		c.IgnoreArrayOrder = true
	}
}

// IgnoreChildOrderAt makes child comparison order-insensitive at the specified HTML path.
func IgnoreChildOrderAt(path string) HTMLOption {
	return func(c *HTMLConfig) {
		c.IgnoreArrayOrderPaths = append(c.IgnoreArrayOrderPaths, path)
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

// HTMLMessage adds a custom message to the assertion failure output.
func HTMLMessage(msg string) HTMLOption {
	return func(c *HTMLConfig) {
		c.Message = msg
	}
}

// IsFieldIgnored checks if an element should be ignored.
func (c *HTMLConfig) IsFieldIgnored(path string) bool {
	parts := strings.Split(path, " > ")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		if c.isElementIgnored(lastPart) {
			return true
		}
	}

	return c.baseConfig.IsFieldIgnored(path)
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

// shouldIgnoreChildOrder checks if child order should be ignored at the given path.
func (c *HTMLConfig) shouldIgnoreChildOrder(path string) bool {
	return c.ShouldIgnoreArrayOrder(path)
}

// isAttributeIgnored checks if an attribute should be ignored.
func (c *HTMLConfig) isAttributeIgnored(path, attr string) bool {
	for _, a := range c.IgnoredAttributes {
		if strings.EqualFold(a, attr) {
			return true
		}
	}

	pathAttr := path + "@" + attr

	return slices.Contains(c.IgnoredAttributePaths, pathAttr)
}
