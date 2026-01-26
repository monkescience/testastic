package testastic

// AssertionOption is a common interface for all assertion options.
type AssertionOption interface {
	applyToConfig(cfg *JSONConfig)
	applyToYAMLConfig(cfg *YAMLConfig)
	applyToHTMLConfig(cfg *HTMLConfig)
	applyToFileConfig(cfg *FileConfig)
}

// Unified option functions that work with all assertion types.

// IgnoreFields excludes the specified fields from comparison.
func IgnoreFields(fields ...string) AssertionOption {
	return newIgnoreFieldsOpt(fields)
}

// IgnoreArrayOrder makes array comparison order-insensitive globally.
func IgnoreArrayOrder() AssertionOption {
	return newIgnoreArrayOrderOpt()
}

// IgnoreArrayOrderAt makes array comparison order-insensitive at the specified path.
func IgnoreArrayOrderAt(path string) AssertionOption {
	return newIgnoreArrayOrderAtOpt(path)
}

// Update forces updating the expected file with the actual value.
func Update() AssertionOption {
	return newUpdateOpt()
}

// Message adds a custom message to the assertion failure output.
func Message(msg string) AssertionOption {
	return newMessageOpt(msg)
}

// ConvertToJSONOption converts an AssertionOption to an Option.
func ConvertToJSONOption(opt AssertionOption) JSONOption {
	return func(c *JSONConfig) {
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

// Helper functions for applying options via temporary configs.

func applyJSONOptionViaConfig(opt JSONOption, base *BaseConfig) {
	temp := &JSONConfig{BaseConfig: *base}
	opt(temp)
	*base = temp.BaseConfig
}

func applyYAMLOptionViaConfig(opt YAMLOption, base *BaseConfig) {
	temp := &YAMLConfig{BaseConfig: *base}
	opt(temp)
	*base = temp.BaseConfig
}

func applyHTMLOptionViaConfig(opt HTMLOption, base *BaseConfig) {
	temp := &HTMLConfig{BaseConfig: *base}
	opt(temp)
	*base = temp.BaseConfig
}

// jsonOptionAdapter wraps an Option to implement AssertionOption.
type jsonOptionAdapter struct {
	opt JSONOption
}

func (a *jsonOptionAdapter) applyToConfig(cfg *JSONConfig) { a.opt(cfg) }

func (a *jsonOptionAdapter) applyToYAMLConfig(cfg *YAMLConfig) {
	applyJSONOptionViaConfig(a.opt, &cfg.BaseConfig)
}

func (a *jsonOptionAdapter) applyToHTMLConfig(cfg *HTMLConfig) {
	applyJSONOptionViaConfig(a.opt, &cfg.BaseConfig)
}

func (a *jsonOptionAdapter) applyToFileConfig(cfg *FileConfig) {
	applyJSONOptionViaConfig(a.opt, &cfg.BaseConfig)
}

// yamlOptionAdapter wraps a YAMLOption to implement AssertionOption.
type yamlOptionAdapter struct {
	opt YAMLOption
}

func (a *yamlOptionAdapter) applyToConfig(cfg *JSONConfig) {
	applyYAMLOptionViaConfig(a.opt, &cfg.BaseConfig)
}
func (a *yamlOptionAdapter) applyToYAMLConfig(cfg *YAMLConfig) { a.opt(cfg) }
func (a *yamlOptionAdapter) applyToHTMLConfig(cfg *HTMLConfig) {
	applyYAMLOptionViaConfig(a.opt, &cfg.BaseConfig)
}

func (a *yamlOptionAdapter) applyToFileConfig(cfg *FileConfig) {
	applyYAMLOptionViaConfig(a.opt, &cfg.BaseConfig)
}

// htmlOptionAdapter wraps an HTMLOption to implement AssertionOption.
type htmlOptionAdapter struct {
	opt HTMLOption
}

func (a *htmlOptionAdapter) applyToConfig(cfg *JSONConfig) {
	applyHTMLOptionViaConfig(a.opt, &cfg.BaseConfig)
}

func (a *htmlOptionAdapter) applyToYAMLConfig(cfg *YAMLConfig) {
	applyHTMLOptionViaConfig(a.opt, &cfg.BaseConfig)
}
func (a *htmlOptionAdapter) applyToHTMLConfig(cfg *HTMLConfig) { a.opt(cfg) }

func (a *htmlOptionAdapter) applyToFileConfig(cfg *FileConfig) {
	applyHTMLOptionViaConfig(a.opt, &cfg.BaseConfig)
}

// WrapJSONOption wraps an Option to implement AssertionOption.
func WrapJSONOption(opt JSONOption) AssertionOption {
	return &jsonOptionAdapter{opt: opt}
}

// WrapYAMLOption wraps a YAMLOption to implement AssertionOption.
func WrapYAMLOption(opt YAMLOption) AssertionOption {
	return &yamlOptionAdapter{opt: opt}
}

// WrapHTMLOption wraps an HTMLOption to implement AssertionOption.
func WrapHTMLOption(opt HTMLOption) AssertionOption {
	return &htmlOptionAdapter{opt: opt}
}

// baseConfigOpt is a generic AssertionOption that operates only on BaseConfig.
// Since all config types embed BaseConfig, this eliminates duplicate implementations.
type baseConfigOpt struct {
	apply func(*BaseConfig)
}

func (o *baseConfigOpt) applyToConfig(cfg *JSONConfig)     { o.apply(&cfg.BaseConfig) }
func (o *baseConfigOpt) applyToYAMLConfig(cfg *YAMLConfig) { o.apply(&cfg.BaseConfig) }
func (o *baseConfigOpt) applyToHTMLConfig(cfg *HTMLConfig) { o.apply(&cfg.BaseConfig) }
func (o *baseConfigOpt) applyToFileConfig(cfg *FileConfig) { o.apply(&cfg.BaseConfig) }

// ignoreFieldsOpt implements AssertionOption for ignoring fields.
type ignoreFieldsOpt struct{ baseConfigOpt }

func newIgnoreFieldsOpt(fields []string) *ignoreFieldsOpt {
	opt := &ignoreFieldsOpt{}
	opt.apply = func(b *BaseConfig) {
		b.IgnoredFields = append(b.IgnoredFields, fields...)
	}

	return opt
}

// ignoreArrayOrderOpt implements AssertionOption for ignoring array order.
type ignoreArrayOrderOpt struct{ baseConfigOpt }

func newIgnoreArrayOrderOpt() *ignoreArrayOrderOpt {
	opt := &ignoreArrayOrderOpt{}
	opt.apply = func(b *BaseConfig) { b.IgnoreArrayOrder = true }

	return opt
}

// ignoreArrayOrderAtOpt implements AssertionOption for ignoring array order at a path.
type ignoreArrayOrderAtOpt struct{ baseConfigOpt }

func newIgnoreArrayOrderAtOpt(path string) *ignoreArrayOrderAtOpt {
	opt := &ignoreArrayOrderAtOpt{}
	opt.apply = func(b *BaseConfig) {
		b.IgnoreArrayOrderPaths = append(b.IgnoreArrayOrderPaths, path)
	}

	return opt
}

// updateOpt implements AssertionOption for update mode.
type updateOpt struct{ baseConfigOpt }

func newUpdateOpt() *updateOpt {
	opt := &updateOpt{}
	opt.apply = func(b *BaseConfig) { b.Update = true }

	return opt
}

// messageOpt implements AssertionOption for custom messages.
type messageOpt struct{ baseConfigOpt }

func newMessageOpt(msg string) *messageOpt {
	opt := &messageOpt{}
	opt.apply = func(b *BaseConfig) { b.Message = msg }

	return opt
}
