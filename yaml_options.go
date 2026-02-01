package testastic

// YAMLConfig holds the configuration for YAML comparison.
type YAMLConfig struct {
	baseConfig
}

// YAMLOption is a functional option for configuring YAML comparison.
type YAMLOption func(*YAMLConfig)

// YAMLIgnoreFields excludes the specified fields from comparison.
func YAMLIgnoreFields(fields ...string) YAMLOption {
	return func(c *YAMLConfig) {
		c.IgnoredFields = append(c.IgnoredFields, fields...)
	}
}

// YAMLIgnoreArrayOrder makes array comparison order-insensitive globally.
func YAMLIgnoreArrayOrder() YAMLOption {
	return func(c *YAMLConfig) {
		c.IgnoreArrayOrder = true
	}
}

// YAMLIgnoreArrayOrderAt makes array comparison order-insensitive at the specified YAML path.
func YAMLIgnoreArrayOrderAt(path string) YAMLOption {
	return func(c *YAMLConfig) {
		c.IgnoreArrayOrderPaths = append(c.IgnoreArrayOrderPaths, path)
	}
}

// YAMLUpdate forces updating the expected file with the actual value.
func YAMLUpdate() YAMLOption {
	return func(c *YAMLConfig) {
		c.Update = true
	}
}

// YAMLMessage adds a custom message to the assertion failure output.
func YAMLMessage(msg string) YAMLOption {
	return func(c *YAMLConfig) {
		c.Message = msg
	}
}
