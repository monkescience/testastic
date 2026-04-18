package testastic_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monkescience/testastic"
)

type htmlStringer string

func (s htmlStringer) String() string {
	return string(s)
}

func TestAssertHTML(t *testing.T) {
	t.Parallel()

	t.Run("exact match", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with exact content
		mt := &mockT{}

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, "testdata/html/exact_match.html",
			`<div class="card"><span>Hello</span></div>`)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure, got: %s", mt.message)
		}
	})

	t.Run("exact match full document", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with a full HTML document
		mt := &mockT{}
		actual := `<!DOCTYPE html><html><head><title>Test</title></head>` +
			`<body><p>Hello</p></body></html>`

		// when: asserting with matching full document
		testastic.AssertHTML(mt, "testdata/html/full_document.html", actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure, got: %s", mt.message)
		}
	})

	t.Run("with anyString matcher", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with anyString matcher in text content
		mt := &mockT{}

		// when: asserting with any string in the span
		testastic.AssertHTML(mt, "testdata/html/with_anystring.html",
			`<div class="card"><span>Hello World</span></div>`)

		// then: the test passes (matcher accepts any string)
		if mt.failed {
			t.Errorf("expected no failure with anyString matcher, got: %s", mt.message)
		}
	})

	t.Run("with regex matcher", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with regex matcher
		mt := &mockT{}

		// when: asserting with a value matching the regex
		testastic.AssertHTML(mt, "testdata/html/with_regex.html",
			`<div><span>user-123</span></div>`)

		// then: the test passes (regex matches)
		if mt.failed {
			t.Errorf("expected no failure with regex matcher, got: %s", mt.message)
		}
	})

	t.Run("with regex matcher fails", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with regex matcher
		mt := &mockT{}

		// when: asserting with a value not matching the regex
		testastic.AssertHTML(mt, "testdata/html/with_regex_fails.html",
			`<div><span>invalid-format</span></div>`)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure with non-matching regex")
		}
	})

	t.Run("with ignore matcher", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with ignore matcher
		mt := &mockT{}
		actual := `<div><span class="timestamp">2024-01-01 12:00:00</span>` +
			`<span>Content</span></div>`

		// when: asserting with any value in the ignored span
		testastic.AssertHTML(mt, "testdata/html/with_ignore.html", actual)

		// then: the test passes (ignored content is not compared)
		if mt.failed {
			t.Errorf("expected no failure with ignore matcher, got: %s", mt.message)
		}
	})

	t.Run("matcher in attribute", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with matcher in an attribute
		mt := &mockT{}

		// when: asserting with any string in the attribute
		testastic.AssertHTML(mt, "testdata/html/matcher_in_attribute.html",
			`<div data-id="abc-123"><span>Content</span></div>`)

		// then: the test passes (matcher accepts any string)
		if mt.failed {
			t.Errorf("expected no failure with matcher in attribute, got: %s", mt.message)
		}
	})

	t.Run("missing element", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with two span elements
		mt := &mockT{}

		// when: asserting with HTML missing the second span
		testastic.AssertHTML(mt, "testdata/html/missing_element.html",
			`<div><span>First</span></div>`)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for missing element")
		}
	})

	t.Run("extra element", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with one span element
		mt := &mockT{}

		// when: asserting with HTML containing an extra span
		testastic.AssertHTML(mt, "testdata/html/extra_element.html",
			`<div><span>First</span><span>Second</span></div>`)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for extra element")
		}
	})

	t.Run("wrong tag", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with a span element
		mt := &mockT{}

		// when: asserting with HTML using a different tag
		testastic.AssertHTML(mt, "testdata/html/wrong_tag.html",
			`<div><p>Content</p></div>`)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for wrong tag")
		}
	})

	t.Run("wrong attribute", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with a specific class attribute
		mt := &mockT{}

		// when: asserting with HTML using a different class value
		testastic.AssertHTML(mt, "testdata/html/wrong_attribute.html",
			`<div class="box"><span>Content</span></div>`)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for wrong attribute value")
		}
	})

	t.Run("missing attribute", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with class and id attributes
		mt := &mockT{}

		// when: asserting with HTML missing the id attribute
		testastic.AssertHTML(mt, "testdata/html/missing_attribute.html",
			`<div class="card"><span>Content</span></div>`)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for missing attribute")
		}
	})

	t.Run("extra attribute", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with only class attribute
		mt := &mockT{}

		// when: asserting with HTML containing an extra id attribute
		testastic.AssertHTML(mt, "testdata/html/extra_attribute.html",
			`<div class="card" id="extra"><span>Content</span></div>`)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for extra attribute")
		}
	})

	t.Run("whitespace normalization", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with normalized whitespace
		mt := &mockT{}

		// when: asserting with HTML containing extra whitespace
		testastic.AssertHTML(mt, "testdata/html/whitespace.html",
			`<div><span>Hello   World</span></div>`)

		// then: the test passes (whitespace is normalized by default)
		if mt.failed {
			t.Errorf("expected whitespace to be normalized, got: %s", mt.message)
		}
	})

	t.Run("preserve whitespace", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with specific whitespace
		mt := &mockT{}
		actual := `<div><span>Hello   World</span></div>`

		// when: asserting with PreserveWhitespace option
		testastic.AssertHTML(mt, "testdata/html/whitespace_preserve.html", actual,
			testastic.PreserveWhitespace())

		// then: the test fails (whitespace differences are detected)
		if !mt.failed {
			t.Error("expected failure with PreserveWhitespace option")
		}
	})

	t.Run("ignore comments", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with a comment
		mt := &mockT{}
		actual := `<div><!-- different comment --><span>Content</span></div>`

		// when: asserting with IgnoreHTMLComments option
		testastic.AssertHTML(mt, "testdata/html/with_comment.html", actual,
			testastic.IgnoreHTMLComments())

		// then: the test passes (comments are ignored)
		if mt.failed {
			t.Errorf("expected comments to be ignored, got: %s", mt.message)
		}
	})

	t.Run("ignore elements", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with a script element
		mt := &mockT{}
		actual := `<div><script>console.log('different')</script><span>Content</span></div>`

		// when: asserting with IgnoreElements option for script
		testastic.AssertHTML(mt, "testdata/html/with_script.html", actual,
			testastic.IgnoreElements("script"))

		// then: the test passes (script element is ignored)
		if mt.failed {
			t.Errorf("expected script element to be ignored, got: %s", mt.message)
		}
	})

	t.Run("ignore attributes", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with class and data-testid attributes
		mt := &mockT{}
		actual := `<div class="box" data-testid="different"><span>Content</span></div>`

		// when: asserting with IgnoreAttributes option
		testastic.AssertHTML(mt, "testdata/html/ignore_attributes.html", actual,
			testastic.IgnoreAttributes("class", "data-testid"))

		// then: the test passes (specified attributes are ignored)
		if mt.failed {
			t.Errorf("expected attributes to be ignored, got: %s", mt.message)
		}
	})

	t.Run("ignore child order", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with ordered children
		expectedFile := filepath.Join(t.TempDir(), "ignore-child-order.html")

		err := os.WriteFile(expectedFile, []byte("<ul><li>first</li><li>second</li></ul>"), 0o644)
		if err != nil {
			t.Fatalf("write expected file: %v", err)
		}

		mt := &mockT{}
		actual := `<ul><li>second</li><li>first</li></ul>`

		// when: asserting with IgnoreChildOrder
		testastic.AssertHTML(mt, expectedFile, actual, testastic.IgnoreChildOrder())

		// then: the children are compared without order sensitivity
		if mt.failed {
			t.Errorf("expected child order to be ignored, got: %s", mt.message)
		}
	})

	t.Run("ignore child order at", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with one ordered and one unordered list
		expectedFile := filepath.Join(t.TempDir(), "ignore-child-order-at.html")

		err := os.WriteFile(expectedFile, []byte(
			"<div><ol><li>one</li><li>two</li></ol><ul><li>a</li><li>b</li></ul></div>",
		), 0o644)
		if err != nil {
			t.Fatalf("write expected file: %v", err)
		}

		mt := &mockT{}
		actual := `<div><ol><li>one</li><li>two</li></ol><ul><li>b</li><li>a</li></ul></div>`

		// when: asserting with IgnoreChildOrderAt for the ul path
		testastic.AssertHTML(mt, expectedFile, actual, testastic.IgnoreChildOrderAt("html > body > div > ul"))

		// then: only the targeted subtree ignores child order
		if mt.failed {
			t.Errorf("expected child order to be ignored at the targeted path, got: %s", mt.message)
		}
	})

	t.Run("ignore attribute at", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with an attribute that changes at one path
		expectedFile := filepath.Join(t.TempDir(), "ignore-attribute-at.html")

		err := os.WriteFile(expectedFile, []byte(`<div data-id="expected"><span>Content</span></div>`), 0o644)
		if err != nil {
			t.Fatalf("write expected file: %v", err)
		}

		mt := &mockT{}
		actual := `<div data-id="actual"><span>Content</span></div>`

		// when: asserting with IgnoreAttributeAt for the data-id attribute
		testastic.AssertHTML(mt, expectedFile, actual, testastic.IgnoreAttributeAt("html > body > div@data-id"))

		// then: the targeted attribute is ignored
		if mt.failed {
			t.Errorf("expected attribute to be ignored at the targeted path, got: %s", mt.message)
		}
	})

	t.Run("ignore fields by path", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with a subtree that should be ignored
		expectedFile := filepath.Join(t.TempDir(), "ignore-fields-path.html")

		err := os.WriteFile(expectedFile, []byte(`<div><span>expected</span><p>stable</p></div>`), 0o644)
		if err != nil {
			t.Fatalf("write expected file: %v", err)
		}

		mt := &mockT{}
		actual := `<div><span>actual</span><p>stable</p></div>`

		// when: asserting with IgnoreFields for the span path
		testastic.AssertHTML(mt, expectedFile, actual, testastic.IgnoreFields("html > body > div > span"))

		// then: the targeted subtree is excluded from comparison
		if mt.failed {
			t.Errorf("expected ignored HTML field path to be skipped, got: %s", mt.message)
		}
	})

	t.Run("create expected file", func(t *testing.T) {
		t.Parallel()

		// given: a non-existent expected file path
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "new-expected.html")

		mt := &mockT{}
		actual := `<div class="card"><span>Content</span></div>`

		// when: asserting with HTMLUpdate option
		testastic.AssertHTML(mt, expectedFile, actual, testastic.Update())

		// then: the test passes and the file is created
		if mt.failed {
			t.Errorf("expected no failure when creating file, got: %s", mt.message)
		}

		content, err := os.ReadFile(expectedFile)
		if err != nil {
			t.Fatalf("expected file was not created: %v", err)
		}

		if !strings.Contains(string(content), "card") {
			t.Errorf("expected file content incorrect: %s", content)
		}
	})

	t.Run("create expected file in nested directory", func(t *testing.T) {
		t.Parallel()

		// given: a non-existent expected file path in nested directories
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "nested", "path", "new-expected.html")

		mt := &mockT{}
		actual := `<div class="card"><span>Content</span></div>`

		// when: asserting with update option
		testastic.AssertHTML(mt, expectedFile, actual, testastic.Update())

		// then: the test passes and the nested file is created
		if mt.failed {
			t.Errorf("expected no failure when creating nested file, got: %s", mt.message)
		}

		content, err := os.ReadFile(expectedFile)
		if err != nil {
			t.Fatalf("expected nested file was not created: %v", err)
		}

		if !strings.Contains(string(content), "card") {
			t.Errorf("expected file content incorrect: %s", content)
		}
	})

	t.Run("byte slice input", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file and actual as []byte
		mt := &mockT{}

		// when: asserting with []byte input
		testastic.AssertHTML(mt, "testdata/html/bytes.html",
			[]byte(`<div><span>Hello</span></div>`))

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with []byte input, got: %s", mt.message)
		}
	})

	t.Run("reader input", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file and actual as io.Reader
		mt := &mockT{}

		// when: asserting with io.Reader input
		testastic.AssertHTML(mt, "testdata/html/bytes_reader.html",
			strings.NewReader(`<article><span>Hello Reader</span></article>`))

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with io.Reader input, got: %s", mt.message)
		}
	})

	t.Run("stringer input", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file and actual as fmt.Stringer
		mt := &mockT{}
		actual := htmlStringer(`<div><span>Hello</span></div>`)

		// when: asserting with fmt.Stringer input
		testastic.AssertHTML(mt, "testdata/html/bytes.html", actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with fmt.Stringer input, got: %s", mt.message)
		}
	})

	t.Run("nested elements", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with nested elements
		mt := &mockT{}

		// when: asserting with matching nested structure
		testastic.AssertHTML(mt, "testdata/html/nested.html",
			`<div><ul><li>Item 1</li><li>Item 2</li></ul></div>`)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with nested elements, got: %s", mt.message)
		}
	})

	t.Run("void elements", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with void elements
		mt := &mockT{}

		// when: asserting with matching void elements
		testastic.AssertHTML(mt, "testdata/html/void_elements.html",
			`<div><img src="test.jpg"><br><input type="text"></div>`)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with void elements, got: %s", mt.message)
		}
	})

	t.Run("embedded matcher in attribute", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with embedded matcher in attribute
		mt := &mockT{}

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, "testdata/html/embedded_matcher_attr.html",
			`<div style="border-left: 6px solid #ff0000;">Content</div>`)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded matcher, got: %s", mt.message)
		}
	})

	t.Run("embedded regex in attribute", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with embedded regex in attribute
		mt := &mockT{}

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, "testdata/html/embedded_regex_attr.html",
			`<div style="border-left: 6px solid #ff0000;">Content</div>`)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded regex, got: %s", mt.message)
		}
	})

	t.Run("embedded matcher mismatch", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with embedded regex that won't match
		mt := &mockT{}

		// when: asserting with non-matching HTML
		testastic.AssertHTML(mt, "testdata/html/embedded_regex_attr_mismatch.html",
			`<div style="border-left: 6px solid red;">Content</div>`)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure with non-matching embedded regex")
		}
	})

	t.Run("multiple embedded matchers", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with multiple embedded matchers
		mt := &mockT{}

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, "testdata/html/multiple_embedded.html",
			`<div data-info="user-john-id-12345">Content</div>`)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with multiple embedded matchers, got: %s", mt.message)
		}
	})

	t.Run("embedded matcher in text content", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with embedded matcher in text content
		mt := &mockT{}

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, "testdata/html/embedded_text.html",
			`<div>Hello World, your ID is 12345</div>`)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded matcher in text, got: %s", mt.message)
		}
	})

	t.Run("embedded anyInt", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with embedded anyInt matcher
		mt := &mockT{}

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, "testdata/html/embedded_anyint.html",
			`<div data-count="Total: 42 items">Content</div>`)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded anyInt, got: %s", mt.message)
		}
	})

	t.Run("embedded anyFloat", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with embedded anyFloat matcher
		mt := &mockT{}

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, "testdata/html/embedded_anyfloat.html",
			`<div data-price="Price: $19.99">Content</div>`)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded anyFloat, got: %s", mt.message)
		}
	})

	t.Run("embedded anyBool", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with embedded anyBool matcher
		mt := &mockT{}

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, "testdata/html/embedded_anybool.html",
			`<div data-state="enabled=true">Content</div>`)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded anyBool, got: %s", mt.message)
		}
	})

	t.Run("embedded anyValue", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with embedded anyValue matcher
		mt := &mockT{}

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, "testdata/html/embedded_anyvalue.html",
			`<div data-info="key=anything-here-123">Content</div>`)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded anyValue, got: %s", mt.message)
		}
	})

	t.Run("embedded ignore", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with embedded ignore matcher
		mt := &mockT{}

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, "testdata/html/embedded_ignore.html",
			`<div data-timestamp="created=2024-01-15T10:30:00Z">Content</div>`)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded ignore, got: %s", mt.message)
		}
	})

	t.Run("embedded oneOf", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with embedded oneOf matcher
		mt := &mockT{}

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, "testdata/html/embedded_oneof.html",
			`<div class="btn btn-secondary">Content</div>`)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded oneOf, got: %s", mt.message)
		}
	})

	t.Run("embedded oneOf mismatch", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file with embedded oneOf matcher
		mt := &mockT{}

		// when: asserting with non-matching HTML
		testastic.AssertHTML(mt, "testdata/html/embedded_oneof_limited.html",
			`<div class="btn btn-danger">Content</div>`)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure with non-matching oneOf value")
		}
	})

	t.Run("with message", func(t *testing.T) {
		t.Parallel()

		// given: an expected HTML file and mismatched actual HTML
		mt := &mockT{}

		// when: asserting with a custom message
		testastic.AssertHTML(mt, "testdata/html/wrong_tag.html", `<div><p>Content</p></div>`,
			testastic.Message("custom html message"))

		// then: the failure includes the custom message
		if !mt.failed {
			t.Error("expected failure for wrong HTML tag")
		}

		if !strings.Contains(mt.message, "custom html message") {
			t.Errorf("expected custom message in failure, got: %s", mt.message)
		}
	})
}

func TestAssertHTML_UnsupportedOptions(t *testing.T) {
	t.Parallel()

	t.Run("json options rejected", func(t *testing.T) {
		t.Parallel()

		// given: JSON/YAML-only options passed to AssertHTML
		mt := &mockT{}

		// when: asserting with unsupported options
		testastic.AssertHTML(mt, "testdata/html/exact_match_unsupported_options.html",
			`<section class="card"><span>Hello Unsupported</span></section>`, testastic.IgnoreArrayOrder())

		// then: the test fatals with a message listing the unsupported option
		if !mt.fatal {
			t.Error("expected fatal error for unsupported options")
		}

		expectedMsg := "testastic: unsupported options for AssertHTML: IgnoreArrayOrder"
		if mt.message != expectedMsg {
			t.Errorf("expected message %q, got: %q", expectedMsg, mt.message)
		}
	})

	t.Run("supported options accepted", func(t *testing.T) {
		t.Parallel()

		// given: supported options for AssertHTML
		mt := &mockT{}

		// when: asserting with IgnoreHTMLComments
		testastic.AssertHTML(mt, "testdata/html/exact_match_supported_options.html",
			`<div class="panel"><span>Hello Options</span></div>`, testastic.IgnoreHTMLComments())

		// then: the test passes without unsupported option error
		if mt.failed {
			t.Errorf("expected no failure, got: %s", mt.message)
		}
	})
}
