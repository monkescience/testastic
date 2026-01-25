package testastic

// YAMLConfig holds the configuration for YAML comparison.
type YAMLConfig struct {
	BaseConfig
}

// YAMLOption is a functional option for configuring YAML comparison.
type YAMLOption func(*YAMLConfig)

// YAMLIgnoreFields excludes the specified fields from comparison.
func YAMLIgnoreFields(fields ...string) YAMLOption {
	return func(c *YAMLConfig) {
		c.BaseConfig.IgnoredFields = append(c.BaseConfig.IgnoredFields, fields...)
	}
}

// YAMLIgnoreArrayOrder makes array comparison order-insensitive globally.
func YAMLIgnoreArrayOrder() YAMLOption {
	return func(c *YAMLConfig) {
		c.BaseConfig.IgnoreArrayOrder = true
	}
}

// YAMLIgnoreArrayOrderAt makes array comparison order-insensitive at the specified YAML path.
func YAMLIgnoreArrayOrderAt(path string) YAMLOption {
	return func(c *YAMLConfig) {
		c.BaseConfig.IgnoreArrayOrderPaths = append(c.BaseConfig.IgnoreArrayOrderPaths, path)
	}
}

// YAMLUpdate forces updating the expected file with the actual value.
func YAMLUpdate() YAMLOption {
	return func(c *YAMLConfig) {
		c.BaseConfig.Update = true
	}
}

// YAMLMessage adds a custom message to the assertion failure output.
func YAMLMessage(msg string) YAMLOption {
	return func(c *YAMLConfig) {
		c.BaseConfig.Message = msg
	}
}

// newYAMLConfig creates a new YAMLConfig with default values and applies options.
func newYAMLConfig(opts ...YAMLOption) *YAMLConfig {
	cfg := &YAMLConfig{
		BaseConfig: BaseConfig{
			Update: shouldUpdate(),
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}
