package testastic

// JSONConfig holds the configuration for JSON comparison.
type JSONConfig struct {
	baseConfig
}

// JSONOption is a functional option for configuring JSON comparison.
type JSONOption func(*JSONConfig)

// JSONIgnoreFields excludes the specified fields from comparison in JSON.
// Fields can be simple names or JSON paths (e.g., "$.user.id").
func JSONIgnoreFields(fields ...string) JSONOption {
	return func(c *JSONConfig) {
		c.IgnoredFields = append(c.IgnoredFields, fields...)
	}
}

// JSONIgnoreArrayOrder makes array comparison order-insensitive globally in JSON.
func JSONIgnoreArrayOrder() JSONOption {
	return func(c *JSONConfig) {
		c.IgnoreArrayOrder = true
	}
}

// JSONIgnoreArrayOrderAt makes array comparison order-insensitive at the specified JSON path.
func JSONIgnoreArrayOrderAt(path string) JSONOption {
	return func(c *JSONConfig) {
		c.IgnoreArrayOrderPaths = append(c.IgnoreArrayOrderPaths, path)
	}
}

// JSONUpdate forces updating the expected file with the actual value in JSON.
func JSONUpdate() JSONOption {
	return func(c *JSONConfig) {
		c.Update = true
	}
}

// JSONMessage adds a custom message to the assertion failure output in JSON.
func JSONMessage(msg string) JSONOption {
	return func(c *JSONConfig) {
		c.Message = msg
	}
}
