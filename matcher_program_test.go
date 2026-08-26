//nolint:testpackage // The private matcher program is its intended test surface.
package testastic

import (
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

type matcherProgramExactMatcher struct {
	want  string
	calls *atomic.Int32
}

func (m matcherProgramExactMatcher) Match(actual any) bool {
	m.calls.Add(1)

	value, ok := actual.(string)

	return ok && value == m.want
}

func (m matcherProgramExactMatcher) String() string {
	return "{{matcherProgramExact " + m.want + "}}"
}

func TestMatcherProgramAdaptersShareCompilation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      string
		preparer    matcherSourcePreparer
		placeholder string
	}{
		{
			name:        "JSON",
			source:      `{"literal":"__TESTASTIC_MATCHER_0__","value":"{{anyString}}"}`,
			preparer:    jsonMatcherPreparer{},
			placeholder: "__TESTASTIC_MATCHER_1__",
		},
		{
			name:        "YAML",
			source:      "literal: __TESTASTIC_YAML_MATCHER_0__\nvalue: {{anyString}}\n",
			preparer:    yamlMatcherPreparer{},
			placeholder: "__TESTASTIC_YAML_MATCHER_1__",
		},
		{
			name:        "HTML",
			source:      `<div data-value="&#95;_TESTASTIC_HTML_MATCHER_0__">{{anyString}}</div>`,
			preparer:    htmlMatcherPreparer{},
			placeholder: "__TESTASTIC_HTML_MATCHER_1__",
		},
		{
			name:        "text",
			source:      "__TESTASTIC_TEXT_MATCHER_0__ {{anyString}}",
			preparer:    fileMatcherPreparer{},
			placeholder: "__TESTASTIC_TEXT_MATCHER_1__",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			program, err := compileMatcherProgram(test.source, test.preparer)
			if err != nil {
				t.Fatalf("compile matcher program: %v", err)
			}

			if program.original != test.source {
				t.Fatalf("original source = %q, want %q", program.original, test.source)
			}

			if len(program.sites) != 1 {
				t.Fatalf("compiled sites = %d, want 1", len(program.sites))
			}

			if program.sites[0].placeholder != test.placeholder {
				t.Fatalf("placeholder = %q, want %q", program.sites[0].placeholder, test.placeholder)
			}

			if !strings.Contains(program.sourceForParser(), test.placeholder) {
				t.Fatalf("parser source %q does not contain %q", program.sourceForParser(), test.placeholder)
			}

			resolved, err := program.resolve(test.placeholder)
			if err != nil {
				t.Fatalf("resolve matcher: %v", err)
			}

			matcher, ok := resolved.(Matcher)
			if !ok || !matcher.Match("value") {
				t.Fatalf("resolved matcher = %T, want accepting Matcher", resolved)
			}
		})
	}
}

func TestMatcherProgramEmbeddedVerdict(t *testing.T) {
	t.Parallel()

	calls := &atomic.Int32{}

	RegisterMatcher("matcherProgramExactCapture", func(arguments string) (Matcher, error) {
		return matcherProgramExactMatcher{want: arguments, calls: calls}, nil
	})

	source := "prefix __TESTASTIC_TEXT_MATCHER_0__ {{regex `(foo)`}}{{matcherProgramExactCapture 123}}"

	program, err := compileMatcherProgram(source, fileMatcherPreparer{})
	if err != nil {
		t.Fatalf("compile matcher program: %v", err)
	}

	resolved, err := program.resolve(program.sourceForParser())
	if err != nil {
		t.Fatalf("resolve matcher program: %v", err)
	}

	embedded, ok := resolved.(*embeddedMatcher)
	if !ok {
		t.Fatalf("resolved matcher = %T, want *embeddedMatcher", resolved)
	}

	actual := "prefix __TESTASTIC_TEXT_MATCHER_0__ foo123"

	verdict := embedded.match(actual)
	if !verdict.matched {
		t.Fatalf("matching verdict = %+v", verdict.captures)
	}

	if len(verdict.captures) != 2 {
		t.Fatalf("captures = %d, want 2", len(verdict.captures))
	}

	if verdict.captures[0].actual != "foo" || verdict.captures[1].actual != "123" {
		t.Fatalf("captures = %+v, want foo followed by 123", verdict.captures)
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("Matcher.Match calls = %d, want 1", got)
	}

	if display := embedded.display(actual, verdict); display != actual {
		t.Fatalf("matching display = %q, want %q", display, actual)
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("display called Matcher.Match, calls = %d", got)
	}

	rejected := embedded.match("prefix __TESTASTIC_TEXT_MATCHER_0__ foo999")
	if rejected.matched {
		t.Fatal("strict custom matcher accepted a rejected capture")
	}

	if display := embedded.display("ignored", rejected); display != source {
		t.Fatalf("rejected display = %q, want %q", display, source)
	}

	if got := calls.Load(); got != 2 {
		t.Fatalf("Matcher.Match calls = %d, want 2", got)
	}
}

func TestMatcherProgramDeterministicCaptures(t *testing.T) {
	t.Parallel()

	program, err := compileMatcherProgram("{{anyString}}{{anyInt}}", fileMatcherPreparer{})
	if err != nil {
		t.Fatalf("compile matcher program: %v", err)
	}

	resolved, err := program.resolve(program.sourceForParser())
	if err != nil {
		t.Fatalf("resolve matcher program: %v", err)
	}

	embedded, ok := resolved.(*embeddedMatcher)
	if !ok {
		t.Fatalf("resolved matcher = %T, want *embeddedMatcher", resolved)
	}

	verdict := embedded.match("ABC123")
	if !verdict.matched {
		t.Fatalf("adjacent matcher verdict = %+v", verdict.captures)
	}

	if verdict.captures[0].actual != "ABC12" || verdict.captures[1].actual != "3" {
		t.Fatalf("captures = %+v, want ABC12 followed by 3", verdict.captures)
	}
}

func TestMatcherProgramEmptyCustomCapture(t *testing.T) {
	t.Parallel()

	calls := &atomic.Int32{}

	RegisterMatcher("matcherProgramEmptyCapture", func(arguments string) (Matcher, error) {
		return matcherProgramExactMatcher{want: arguments, calls: calls}, nil
	})

	program, err := compileMatcherProgram("pre{{matcherProgramEmptyCapture}}post", fileMatcherPreparer{})
	if err != nil {
		t.Fatalf("compile matcher program: %v", err)
	}

	resolved, err := program.resolve(program.sourceForParser())
	if err != nil {
		t.Fatalf("resolve matcher program: %v", err)
	}

	embedded, ok := resolved.(*embeddedMatcher)
	if !ok {
		t.Fatalf("resolved matcher = %T, want *embeddedMatcher", resolved)
	}

	verdict := embedded.match("prepost")
	if !verdict.matched || len(verdict.captures) != 1 || verdict.captures[0].actual != "" {
		t.Fatalf("empty capture verdict = %+v", verdict.captures)
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("Matcher.Match calls = %d, want 1", got)
	}
}

func TestMatcherProgramBuiltInEmbeddedMatchers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		actual     string
	}{
		{name: "any string", expression: "anyString", actual: ""},
		{name: "any integer", expression: "anyInt", actual: "-12"},
		{name: "any float", expression: "anyFloat", actual: "12.5"},
		{name: "any boolean", expression: "anyBool", actual: "false"},
		{name: "any value", expression: "anyValue", actual: "anything"},
		{name: "ignore", expression: "ignore", actual: "anything"},
		{
			name:       "uppercase UUID",
			expression: "anyUUID",
			actual:     "550E8400-E29B-41D4-A716-446655440000",
		},
		{
			name:       "date time",
			expression: "anyDateTime",
			actual:     "2026-08-26T12:34:56Z",
		},
		{name: "URL", expression: "anyURL", actual: "https://example.com/path"},
		{name: "capture relative regex alternation", expression: "regex `^foo$|^bar$`", actual: "bar"},
		{name: "one of", expression: `oneOf "foo" "bar"`, actual: "bar"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			source := "pre{{" + test.expression + "}}post"

			program, err := compileMatcherProgram(source, fileMatcherPreparer{})
			if err != nil {
				t.Fatalf("compile matcher program: %v", err)
			}

			resolved, err := program.resolve(program.sourceForParser())
			if err != nil {
				t.Fatalf("resolve matcher program: %v", err)
			}

			matcher, ok := resolved.(Matcher)
			if !ok {
				t.Fatalf("resolved matcher = %T, want Matcher", resolved)
			}

			if !matcher.Match("pre" + test.actual + "post") {
				t.Fatalf("%s rejected %q", test.expression, test.actual)
			}
		})
	}
}

func TestMatcherProgramCompileErrors(t *testing.T) {
	t.Parallel()

	whitespaceTests := []struct {
		name     string
		source   string
		preparer matcherSourcePreparer
	}{
		{name: "JSON", source: `{"value":"{{ }}"}`, preparer: jsonMatcherPreparer{}},
		{name: "YAML", source: "value: {{ }}\n", preparer: yamlMatcherPreparer{}},
		{name: "HTML", source: "<p>{{ }}</p>", preparer: htmlMatcherPreparer{}},
		{name: "text", source: "{{ }}", preparer: fileMatcherPreparer{}},
	}

	for _, test := range whitespaceTests {
		t.Run(test.name+" whitespace", func(t *testing.T) {
			t.Parallel()

			_, err := compileMatcherProgram(test.source, test.preparer)
			if !errors.Is(err, errEmptyMatcherExpression) {
				t.Fatalf("compile error = %v, want empty matcher expression", err)
			}
		})
	}

	expressionTests := []struct {
		name   string
		source string
		target error
	}{
		{name: "invalid oneOf", source: `{{oneOf unquoted}}`, target: errInvalidOneOfSyntax},
		{name: "unknown", source: `{{matcherProgramUnknown}}`, target: errUnknownMatcher},
	}

	for _, test := range expressionTests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := compileMatcherProgram(test.source, fileMatcherPreparer{})
			if !errors.Is(err, test.target) {
				t.Fatalf("compile error = %v, want %v", err, test.target)
			}
		})
	}
}

func TestMatcherProgramPreservesLegacyFactoryErrors(t *testing.T) {
	t.Parallel()

	factoryErr := errors.New("matcher program factory failure")

	RegisterMatcher("matcherProgramFactoryFailure", func(string) (Matcher, error) {
		return nil, factoryErr
	})

	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{
			name: "JSON",
			run: func() error {
				_, err := parseExpectedJSONString(`{"value":"{{matcherProgramFactoryFailure}}"}`)

				return err
			},
			want: `failed to parse matcher "matcherProgramFactoryFailure": matcher program factory failure`,
		},
		{
			name: "YAML",
			run: func() error {
				_, err := parseExpectedYAMLString("value: {{matcherProgramFactoryFailure}}\n")

				return err
			},
			want: `failed to parse matcher "matcherProgramFactoryFailure": matcher program factory failure`,
		},
		{
			name: "text",
			run: func() error {
				_, err := parseLine("value: {{matcherProgramFactoryFailure}}")

				return err
			},
			want: `line "value: {{matcherProgramFactoryFailure}}": matcher program factory failure`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := test.run()
			if err == nil || err.Error() != test.want {
				t.Fatalf("error = %q, want %q", err, test.want)
			}
		})
	}
}

func TestMatcherStringRoundTripsEscapedQuotes(t *testing.T) {
	t.Parallel()

	regex, err := Regex("a`\"b")
	if err != nil {
		t.Fatalf("create regex matcher: %v", err)
	}

	tests := []Matcher{
		regex,
		OneOf(`a"b`, `c\d`),
	}

	for _, matcher := range tests {
		encoded, err := json.Marshal(map[string]string{"value": matcher.String()})
		if err != nil {
			t.Fatalf("encode matcher source: %v", err)
		}

		expected, err := parseExpectedJSONString(string(encoded))
		if err != nil {
			t.Fatalf("parse matcher source %s: %v", encoded, err)
		}

		values, ok := expected.Data.(map[string]any)
		if !ok {
			t.Fatalf("resolved data = %T, want map[string]any", expected.Data)
		}

		resolved, ok := values["value"].(Matcher)
		if !ok {
			t.Fatalf("resolved value = %T, want Matcher", values["value"])
		}

		if resolved.String() != matcher.String() {
			t.Fatalf("round trip = %q, want %q", resolved.String(), matcher.String())
		}
	}
}

func TestMatcherProgramMalformedQuotedArguments(t *testing.T) {
	t.Parallel()

	t.Run("JSON remains fatal", func(t *testing.T) {
		t.Parallel()

		_, err := parseExpectedJSONString(`{"value":"{{regex \"unterminated}}"}`)
		if !errors.Is(err, errInvalidRegexSyntax) {
			t.Fatalf("parse error = %v, want invalid regex syntax", err)
		}

		want := `failed to parse matcher "regex `
		if !strings.HasPrefix(err.Error(), want) {
			t.Fatalf("parse error = %q, want prefix %q", err, want)
		}
	})

	t.Run("text remains literal", func(t *testing.T) {
		t.Parallel()

		line := `value: {{regex "unterminated}}`

		parsed, err := parseLine(line)
		if err != nil {
			t.Fatalf("parse line: %v", err)
		}

		if parsed.pattern != nil || parsed.original != line {
			t.Fatalf("parsed line = %+v, want unchanged literal", parsed)
		}
	})

	t.Run("JSON unclosed backtick remains literal", func(t *testing.T) {
		t.Parallel()

		value := "{{regex `unterminated}}"

		source, err := json.Marshal(map[string]string{"value": value})
		if err != nil {
			t.Fatalf("encode source: %v", err)
		}

		expected, err := parseExpectedJSONString(string(source))
		if err != nil {
			t.Fatalf("parse expected JSON: %v", err)
		}

		values, ok := expected.Data.(map[string]any)
		if !ok || values["value"] != value {
			t.Fatalf("resolved data = %#v, want literal %q", expected.Data, value)
		}
	})
}
