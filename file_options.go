package testastic

// FileConfig holds configuration for file assertions.
type FileConfig struct {
	BaseConfig
}

// FileOption is a functional option for configuring file comparison.
type FileOption func(*FileConfig)
