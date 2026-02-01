package testastic

// AssertionOption configures the behavior of assertion functions.
// Options work with all assertion types (JSON, YAML, HTML, and File).
//
// Use the provided option functions to create options:
//   - [IgnoreFields] - exclude fields from comparison
//   - [IgnoreArrayOrder] - compare arrays without regard to order
//   - [IgnoreArrayOrderAt] - ignore order at specific paths
//   - [Update] - update expected files with actual values
//   - [Message] - add context to failure messages
type AssertionOption interface {
	applyToConfig(cfg *JSONConfig)
	applyToYAMLConfig(cfg *YAMLConfig)
	applyToHTMLConfig(cfg *HTMLConfig)
	applyToFileConfig(cfg *FileConfig)
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
func IgnoreFields(fields ...string) AssertionOption {
	return newIgnoreFieldsOpt(fields)
}

// IgnoreArrayOrder makes array comparison order-insensitive globally.
// When enabled, arrays are compared as sets: [1, 2, 3] equals [3, 1, 2].
//
// Use [IgnoreArrayOrderAt] to apply order-insensitive comparison
// only at specific paths.
func IgnoreArrayOrder() AssertionOption {
	return newIgnoreArrayOrderOpt()
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
func IgnoreArrayOrderAt(path string) AssertionOption {
	return newIgnoreArrayOrderAtOpt(path)
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
func Update() AssertionOption {
	return newUpdateOpt()
}

// Message adds a custom message to the assertion failure output.
// This helps identify which assertion failed when a test has multiple assertions.
//
//	testastic.AssertJSON(t, expected, actual,
//	    testastic.Message("user creation response"),
//	)
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

func applyJSONOptionViaConfig(opt JSONOption, base *baseConfig) {
	temp := &JSONConfig{baseConfig: *base}
	opt(temp)
	*base = temp.baseConfig
}

func applyYAMLOptionViaConfig(opt YAMLOption, base *baseConfig) {
	temp := &YAMLConfig{baseConfig: *base}
	opt(temp)
	*base = temp.baseConfig
}

func applyHTMLOptionViaConfig(opt HTMLOption, base *baseConfig) {
	temp := &HTMLConfig{baseConfig: *base}
	opt(temp)
	*base = temp.baseConfig
}

// jsonOptionAdapter wraps an Option to implement AssertionOption.
type jsonOptionAdapter struct {
	opt JSONOption
}

func (a *jsonOptionAdapter) applyToConfig(cfg *JSONConfig) { a.opt(cfg) }

func (a *jsonOptionAdapter) applyToYAMLConfig(cfg *YAMLConfig) {
	applyJSONOptionViaConfig(a.opt, &cfg.baseConfig)
}

func (a *jsonOptionAdapter) applyToHTMLConfig(cfg *HTMLConfig) {
	applyJSONOptionViaConfig(a.opt, &cfg.baseConfig)
}

func (a *jsonOptionAdapter) applyToFileConfig(cfg *FileConfig) {
	applyJSONOptionViaConfig(a.opt, &cfg.baseConfig)
}

// yamlOptionAdapter wraps a YAMLOption to implement AssertionOption.
type yamlOptionAdapter struct {
	opt YAMLOption
}

func (a *yamlOptionAdapter) applyToConfig(cfg *JSONConfig) {
	applyYAMLOptionViaConfig(a.opt, &cfg.baseConfig)
}
func (a *yamlOptionAdapter) applyToYAMLConfig(cfg *YAMLConfig) { a.opt(cfg) }
func (a *yamlOptionAdapter) applyToHTMLConfig(cfg *HTMLConfig) {
	applyYAMLOptionViaConfig(a.opt, &cfg.baseConfig)
}

func (a *yamlOptionAdapter) applyToFileConfig(cfg *FileConfig) {
	applyYAMLOptionViaConfig(a.opt, &cfg.baseConfig)
}

// htmlOptionAdapter wraps an HTMLOption to implement AssertionOption.
type htmlOptionAdapter struct {
	opt HTMLOption
}

func (a *htmlOptionAdapter) applyToConfig(cfg *JSONConfig) {
	applyHTMLOptionViaConfig(a.opt, &cfg.baseConfig)
}

func (a *htmlOptionAdapter) applyToYAMLConfig(cfg *YAMLConfig) {
	applyHTMLOptionViaConfig(a.opt, &cfg.baseConfig)
}
func (a *htmlOptionAdapter) applyToHTMLConfig(cfg *HTMLConfig) { a.opt(cfg) }

func (a *htmlOptionAdapter) applyToFileConfig(cfg *FileConfig) {
	applyHTMLOptionViaConfig(a.opt, &cfg.baseConfig)
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

// baseConfigOpt is a generic AssertionOption that operates only on baseConfig.
// Since all config types embed baseConfig, this eliminates duplicate implementations.
type baseConfigOpt struct {
	apply func(*baseConfig)
}

func (o *baseConfigOpt) applyToConfig(cfg *JSONConfig)     { o.apply(&cfg.baseConfig) }
func (o *baseConfigOpt) applyToYAMLConfig(cfg *YAMLConfig) { o.apply(&cfg.baseConfig) }
func (o *baseConfigOpt) applyToHTMLConfig(cfg *HTMLConfig) { o.apply(&cfg.baseConfig) }
func (o *baseConfigOpt) applyToFileConfig(cfg *FileConfig) { o.apply(&cfg.baseConfig) }

// ignoreFieldsOpt implements AssertionOption for ignoring fields.
type ignoreFieldsOpt struct{ baseConfigOpt }

func newIgnoreFieldsOpt(fields []string) *ignoreFieldsOpt {
	opt := &ignoreFieldsOpt{}
	opt.apply = func(b *baseConfig) {
		b.IgnoredFields = append(b.IgnoredFields, fields...)
	}

	return opt
}

// ignoreArrayOrderOpt implements AssertionOption for ignoring array order.
type ignoreArrayOrderOpt struct{ baseConfigOpt }

func newIgnoreArrayOrderOpt() *ignoreArrayOrderOpt {
	opt := &ignoreArrayOrderOpt{}
	opt.apply = func(b *baseConfig) { b.IgnoreArrayOrder = true }

	return opt
}

// ignoreArrayOrderAtOpt implements AssertionOption for ignoring array order at a path.
type ignoreArrayOrderAtOpt struct{ baseConfigOpt }

func newIgnoreArrayOrderAtOpt(path string) *ignoreArrayOrderAtOpt {
	opt := &ignoreArrayOrderAtOpt{}
	opt.apply = func(b *baseConfig) {
		b.IgnoreArrayOrderPaths = append(b.IgnoreArrayOrderPaths, path)
	}

	return opt
}

// updateOpt implements AssertionOption for update mode.
type updateOpt struct{ baseConfigOpt }

func newUpdateOpt() *updateOpt {
	opt := &updateOpt{}
	opt.apply = func(b *baseConfig) { b.Update = true }

	return opt
}

// messageOpt implements AssertionOption for custom messages.
type messageOpt struct{ baseConfigOpt }

func newMessageOpt(msg string) *messageOpt {
	opt := &messageOpt{}
	opt.apply = func(b *baseConfig) { b.Message = msg }

	return opt
}
