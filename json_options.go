package testastic

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
		c.IgnoredFields = append(c.IgnoredFields, fields...)
	}
}

// JSONIgnoreArrayOrder makes array comparison order-insensitive globally in JSON.
func JSONIgnoreArrayOrder() Option {
	return func(c *Config) {
		c.IgnoreArrayOrder = true
	}
}

// JSONIgnoreArrayOrderAt makes array comparison order-insensitive at the specified JSON path.
func JSONIgnoreArrayOrderAt(path string) Option {
	return func(c *Config) {
		c.IgnoreArrayOrderPaths = append(c.IgnoreArrayOrderPaths, path)
	}
}

// JSONUpdate forces updating the expected file with the actual value in JSON.
func JSONUpdate() Option {
	return func(c *Config) {
		c.Update = true
	}
}

// JSONMessage adds a custom message to the assertion failure output in JSON.
func JSONMessage(msg string) Option {
	return func(c *Config) {
		c.Message = msg
	}
}
