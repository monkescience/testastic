package testastic

// AssertionOption is a common interface for all assertion options.
type AssertionOption interface {
	applyToConfig(cfg *Config)
	applyToYAMLConfig(cfg *YAMLConfig)
	applyToHTMLConfig(cfg *HTMLConfig)
}

// Unified option functions that work with all assertion types.

// IgnoreFields excludes the specified fields from comparison.
func IgnoreFields(fields ...string) AssertionOption {
	return &ignoreFieldsOpt{fields: fields}
}

// IgnoreArrayOrder makes array comparison order-insensitive globally.
func IgnoreArrayOrder() AssertionOption {
	return &ignoreArrayOrderOpt{}
}

// IgnoreArrayOrderAt makes array comparison order-insensitive at the specified path.
func IgnoreArrayOrderAt(path string) AssertionOption {
	return &ignoreArrayOrderAtOpt{path: path}
}

// Update forces updating the expected file with the actual value.
func Update() AssertionOption {
	return &updateOpt{}
}

// Message adds a custom message to the assertion failure output.
func Message(msg string) AssertionOption {
	return &messageOpt{msg: msg}
}

// ConvertToOption converts an AssertionOption to an Option.
func ConvertToOption(opt AssertionOption) Option {
	return func(c *Config) {
		opt.applyToConfig(c)
	}
}

// ConvertToYAMLOption converts an AssertionOption to a YAMLOption.
func ConvertToYAMLOption(opt AssertionOption) YAMLOption {
	return func(c *YAMLConfig) {
		opt.applyToYAMLConfig(c)
	}
}

// ConvertToHTMLOption converts an AssertionOption to an HTMLOption.
func ConvertToHTMLOption(opt AssertionOption) HTMLOption {
	return func(c *HTMLConfig) {
		opt.applyToHTMLConfig(c)
	}
}

// optionAdapter wraps an Option to implement AssertionOption.
type optionAdapter struct {
	opt Option
}

func (a *optionAdapter) applyToConfig(cfg *Config) {
	a.opt(cfg)
}

func (a *optionAdapter) applyToYAMLConfig(cfg *YAMLConfig) {
	// Create a temporary Config and apply the option
	tempCfg := &Config{BaseConfig: cfg.BaseConfig}
	a.opt(tempCfg)
	cfg.BaseConfig = tempCfg.BaseConfig
}

func (a *optionAdapter) applyToHTMLConfig(cfg *HTMLConfig) {
	// Create a temporary Config and apply the option
	tempCfg := &Config{BaseConfig: cfg.BaseConfig}
	a.opt(tempCfg)
	cfg.BaseConfig = tempCfg.BaseConfig
}

// yamlOptionAdapter wraps a YAMLOption to implement AssertionOption.
type yamlOptionAdapter struct {
	opt YAMLOption
}

func (a *yamlOptionAdapter) applyToConfig(cfg *Config) {
	// Create a temporary YAMLConfig and apply the option
	tempCfg := &YAMLConfig{BaseConfig: cfg.BaseConfig}
	a.opt(tempCfg)
	cfg.BaseConfig = tempCfg.BaseConfig
}

func (a *yamlOptionAdapter) applyToYAMLConfig(cfg *YAMLConfig) {
	a.opt(cfg)
}

func (a *yamlOptionAdapter) applyToHTMLConfig(cfg *HTMLConfig) {
	// Create a temporary YAMLConfig and apply the option
	tempCfg := &YAMLConfig{BaseConfig: cfg.BaseConfig}
	a.opt(tempCfg)
	cfg.BaseConfig = tempCfg.BaseConfig
}

// htmlOptionAdapter wraps an HTMLOption to implement AssertionOption.
type htmlOptionAdapter struct {
	opt HTMLOption
}

func (a *htmlOptionAdapter) applyToConfig(cfg *Config) {
	// Create a temporary HTMLConfig and apply the option
	tempCfg := &HTMLConfig{BaseConfig: cfg.BaseConfig}
	a.opt(tempCfg)
	cfg.BaseConfig = tempCfg.BaseConfig
}

func (a *htmlOptionAdapter) applyToYAMLConfig(cfg *YAMLConfig) {
	// Create a temporary HTMLConfig and apply the option
	tempCfg := &HTMLConfig{BaseConfig: cfg.BaseConfig}
	a.opt(tempCfg)
	cfg.BaseConfig = tempCfg.BaseConfig
}

func (a *htmlOptionAdapter) applyToHTMLConfig(cfg *HTMLConfig) {
	a.opt(cfg)
}

// WrapOption wraps an Option to implement AssertionOption.
func WrapOption(opt Option) AssertionOption {
	return &optionAdapter{opt: opt}
}

// WrapYAMLOption wraps a YAMLOption to implement AssertionOption.
func WrapYAMLOption(opt YAMLOption) AssertionOption {
	return &yamlOptionAdapter{opt: opt}
}

// WrapHTMLOption wraps an HTMLOption to implement AssertionOption.
func WrapHTMLOption(opt HTMLOption) AssertionOption {
	return &htmlOptionAdapter{opt: opt}
}

// ignoreFieldsOpt implements AssertionOption for ignoring fields.
type ignoreFieldsOpt struct {
	fields []string
}

func (o *ignoreFieldsOpt) applyToConfig(cfg *Config) {
	cfg.BaseConfig.IgnoredFields = append(cfg.BaseConfig.IgnoredFields, o.fields...)
}

func (o *ignoreFieldsOpt) applyToYAMLConfig(cfg *YAMLConfig) {
	cfg.BaseConfig.IgnoredFields = append(cfg.BaseConfig.IgnoredFields, o.fields...)
}

func (o *ignoreFieldsOpt) applyToHTMLConfig(cfg *HTMLConfig) {
	cfg.BaseConfig.IgnoredFields = append(cfg.BaseConfig.IgnoredFields, o.fields...)
}

// ignoreArrayOrderOpt implements AssertionOption for ignoring array order.
type ignoreArrayOrderOpt struct{}

func (o *ignoreArrayOrderOpt) applyToConfig(cfg *Config) {
	cfg.BaseConfig.IgnoreArrayOrder = true
}

func (o *ignoreArrayOrderOpt) applyToYAMLConfig(cfg *YAMLConfig) {
	cfg.BaseConfig.IgnoreArrayOrder = true
}

func (o *ignoreArrayOrderOpt) applyToHTMLConfig(cfg *HTMLConfig) {
	cfg.BaseConfig.IgnoreArrayOrder = true
}

// ignoreArrayOrderAtOpt implements AssertionOption for ignoring array order at a path.
type ignoreArrayOrderAtOpt struct {
	path string
}

func (o *ignoreArrayOrderAtOpt) applyToConfig(cfg *Config) {
	cfg.BaseConfig.IgnoreArrayOrderPaths = append(cfg.BaseConfig.IgnoreArrayOrderPaths, o.path)
}

func (o *ignoreArrayOrderAtOpt) applyToYAMLConfig(cfg *YAMLConfig) {
	cfg.BaseConfig.IgnoreArrayOrderPaths = append(cfg.BaseConfig.IgnoreArrayOrderPaths, o.path)
}

func (o *ignoreArrayOrderAtOpt) applyToHTMLConfig(cfg *HTMLConfig) {
	cfg.BaseConfig.IgnoreArrayOrderPaths = append(cfg.BaseConfig.IgnoreArrayOrderPaths, o.path)
}

// updateOpt implements AssertionOption for update mode.
type updateOpt struct{}

func (o *updateOpt) applyToConfig(cfg *Config) {
	cfg.BaseConfig.Update = true
}

func (o *updateOpt) applyToYAMLConfig(cfg *YAMLConfig) {
	cfg.BaseConfig.Update = true
}

func (o *updateOpt) applyToHTMLConfig(cfg *HTMLConfig) {
	cfg.BaseConfig.Update = true
}

// messageOpt implements AssertionOption for custom messages.
type messageOpt struct {
	msg string
}

func (o *messageOpt) applyToConfig(cfg *Config) {
	cfg.BaseConfig.Message = o.msg
}

func (o *messageOpt) applyToYAMLConfig(cfg *YAMLConfig) {
	cfg.BaseConfig.Message = o.msg
}

func (o *messageOpt) applyToHTMLConfig(cfg *HTMLConfig) {
	cfg.BaseConfig.Message = o.msg
}

// ignoreChildOrderOpt implements AssertionOption for ignoring child order in HTML.
type ignoreChildOrderOpt struct{}

func (o *ignoreChildOrderOpt) applyToConfig(cfg *Config) {
	cfg.BaseConfig.IgnoreArrayOrder = true
}

func (o *ignoreChildOrderOpt) applyToYAMLConfig(cfg *YAMLConfig) {
	cfg.BaseConfig.IgnoreArrayOrder = true
}

func (o *ignoreChildOrderOpt) applyToHTMLConfig(cfg *HTMLConfig) {
	cfg.BaseConfig.IgnoreArrayOrder = true
}

// ignoreChildOrderAtOpt implements AssertionOption for ignoring child order at a path in HTML.
type ignoreChildOrderAtOpt struct {
	path string
}

func (o *ignoreChildOrderAtOpt) applyToConfig(cfg *Config) {
	cfg.BaseConfig.IgnoreArrayOrderPaths = append(cfg.BaseConfig.IgnoreArrayOrderPaths, o.path)
}

func (o *ignoreChildOrderAtOpt) applyToYAMLConfig(cfg *YAMLConfig) {
	cfg.BaseConfig.IgnoreArrayOrderPaths = append(cfg.BaseConfig.IgnoreArrayOrderPaths, o.path)
}

func (o *ignoreChildOrderAtOpt) applyToHTMLConfig(cfg *HTMLConfig) {
	cfg.BaseConfig.IgnoreArrayOrderPaths = append(cfg.BaseConfig.IgnoreArrayOrderPaths, o.path)
}
