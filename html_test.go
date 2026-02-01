package testastic_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monkescience/testastic"
)

func TestAssertHTML(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
		// given: an expected HTML file with exact content
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		html := `<div class="card"><span>Hello</span></div>`

		err := os.WriteFile(expectedFile, []byte(html), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, expectedFile, html)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure, got: %s", mt.message)
		}
	})

	t.Run("exact match full document", func(t *testing.T) {
		// given: an expected HTML file with a full HTML document
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		html := `<!DOCTYPE html><html><head><title>Test</title></head><body><p>Hello</p></body></html>`

		err := os.WriteFile(expectedFile, []byte(html), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}

		// when: asserting with matching full document
		testastic.AssertHTML(mt, expectedFile, html)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure, got: %s", mt.message)
		}
	})

	t.Run("with anyString matcher", func(t *testing.T) {
		// given: an expected HTML file with anyString matcher in text content
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div class="card"><span>{{anyString}}</span></div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div class="card"><span>Hello World</span></div>`

		// when: asserting with any string in the span
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes (matcher accepts any string)
		if mt.failed {
			t.Errorf("expected no failure with anyString matcher, got: %s", mt.message)
		}
	})

	t.Run("with regex matcher", func(t *testing.T) {
		// given: an expected HTML file with regex matcher
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := "<div><span>{{regex `^user-\\d+$`}}</span></div>"

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div><span>user-123</span></div>`

		// when: asserting with a value matching the regex
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes (regex matches)
		if mt.failed {
			t.Errorf("expected no failure with regex matcher, got: %s", mt.message)
		}
	})

	t.Run("with regex matcher fails", func(t *testing.T) {
		// given: an expected HTML file with regex matcher
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := "<div><span>{{regex `^user-\\d+$`}}</span></div>"

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div><span>invalid-format</span></div>`

		// when: asserting with a value not matching the regex
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure with non-matching regex")
		}
	})

	t.Run("with ignore matcher", func(t *testing.T) {
		// given: an expected HTML file with ignore matcher
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div><span class="timestamp">{{ignore}}</span><span>Content</span></div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div><span class="timestamp">2024-01-01 12:00:00</span><span>Content</span></div>`

		// when: asserting with any value in the ignored span
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes (ignored content is not compared)
		if mt.failed {
			t.Errorf("expected no failure with ignore matcher, got: %s", mt.message)
		}
	})

	t.Run("matcher in attribute", func(t *testing.T) {
		// given: an expected HTML file with matcher in an attribute
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div data-id="{{anyString}}"><span>Content</span></div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div data-id="abc-123"><span>Content</span></div>`

		// when: asserting with any string in the attribute
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes (matcher accepts any string)
		if mt.failed {
			t.Errorf("expected no failure with matcher in attribute, got: %s", mt.message)
		}
	})

	t.Run("missing element", func(t *testing.T) {
		// given: an expected HTML file with two span elements
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div><span>First</span><span>Second</span></div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div><span>First</span></div>`

		// when: asserting with HTML missing the second span
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for missing element")
		}
	})

	t.Run("extra element", func(t *testing.T) {
		// given: an expected HTML file with one span element
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div><span>First</span></div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div><span>First</span><span>Second</span></div>`

		// when: asserting with HTML containing an extra span
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for extra element")
		}
	})

	t.Run("wrong tag", func(t *testing.T) {
		// given: an expected HTML file with a span element
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div><span>Content</span></div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div><p>Content</p></div>`

		// when: asserting with HTML using a different tag
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for wrong tag")
		}
	})

	t.Run("wrong attribute", func(t *testing.T) {
		// given: an expected HTML file with a specific class attribute
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div class="card"><span>Content</span></div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div class="box"><span>Content</span></div>`

		// when: asserting with HTML using a different class value
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for wrong attribute value")
		}
	})

	t.Run("missing attribute", func(t *testing.T) {
		// given: an expected HTML file with class and id attributes
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div class="card" id="main"><span>Content</span></div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div class="card"><span>Content</span></div>`

		// when: asserting with HTML missing the id attribute
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for missing attribute")
		}
	})

	t.Run("extra attribute", func(t *testing.T) {
		// given: an expected HTML file with only class attribute
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div class="card"><span>Content</span></div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div class="card" id="extra"><span>Content</span></div>`

		// when: asserting with HTML containing an extra id attribute
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for extra attribute")
		}
	})

	t.Run("whitespace normalization", func(t *testing.T) {
		// given: an expected HTML file with normalized whitespace
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div><span>Hello World</span></div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div><span>Hello   World</span></div>` // Extra whitespace

		// when: asserting with HTML containing extra whitespace
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes (whitespace is normalized by default)
		if mt.failed {
			t.Errorf("expected whitespace to be normalized, got: %s", mt.message)
		}
	})

	t.Run("preserve whitespace", func(t *testing.T) {
		// given: an expected HTML file with specific whitespace
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div><span>Hello World</span></div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div><span>Hello   World</span></div>`

		// when: asserting with PreserveWhitespace option
		testastic.AssertHTML(mt, expectedFile, actual, testastic.WrapHTMLOption(testastic.PreserveWhitespace()))

		// then: the test fails (whitespace differences are detected)
		if !mt.failed {
			t.Error("expected failure with PreserveWhitespace option")
		}
	})

	t.Run("ignore comments", func(t *testing.T) {
		// given: an expected HTML file with a comment
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div><!-- comment --><span>Content</span></div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div><!-- different comment --><span>Content</span></div>`

		// when: asserting with IgnoreHTMLComments option
		testastic.AssertHTML(mt, expectedFile, actual, testastic.WrapHTMLOption(testastic.IgnoreHTMLComments()))

		// then: the test passes (comments are ignored)
		if mt.failed {
			t.Errorf("expected comments to be ignored, got: %s", mt.message)
		}
	})

	t.Run("ignore elements", func(t *testing.T) {
		// given: an expected HTML file with a script element
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div><script>console.log('test')</script><span>Content</span></div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div><script>console.log('different')</script><span>Content</span></div>`

		// when: asserting with IgnoreElements option for script
		testastic.AssertHTML(mt, expectedFile, actual, testastic.WrapHTMLOption(testastic.IgnoreElements("script")))

		// then: the test passes (script element is ignored)
		if mt.failed {
			t.Errorf("expected script element to be ignored, got: %s", mt.message)
		}
	})

	t.Run("ignore attributes", func(t *testing.T) {
		// given: an expected HTML file with class and data-testid attributes
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div class="card" data-testid="test"><span>Content</span></div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div class="box" data-testid="different"><span>Content</span></div>`

		// when: asserting with IgnoreAttributes option
		testastic.AssertHTML(
			mt, expectedFile, actual,
			testastic.WrapHTMLOption(testastic.IgnoreAttributes("class", "data-testid")),
		)

		// then: the test passes (specified attributes are ignored)
		if mt.failed {
			t.Errorf("expected attributes to be ignored, got: %s", mt.message)
		}
	})

	t.Run("create expected file", func(t *testing.T) {
		// given: a non-existent expected file path
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "new-expected.html")

		mt := &htmlMockT{}
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

	t.Run("byte slice input", func(t *testing.T) {
		// given: an expected HTML file and actual as []byte
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		html := `<div><span>Hello</span></div>`

		err := os.WriteFile(expectedFile, []byte(html), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}

		// when: asserting with []byte input
		testastic.AssertHTML(mt, expectedFile, []byte(html))

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with []byte input, got: %s", mt.message)
		}
	})

	t.Run("reader input", func(t *testing.T) {
		// given: an expected HTML file and actual as io.Reader
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		html := `<div><span>Hello</span></div>`

		err := os.WriteFile(expectedFile, []byte(html), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}

		// when: asserting with io.Reader input
		testastic.AssertHTML(mt, expectedFile, strings.NewReader(html))

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with io.Reader input, got: %s", mt.message)
		}
	})

	t.Run("nested elements", func(t *testing.T) {
		// given: an expected HTML file with nested elements
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		html := `<div><ul><li>Item 1</li><li>Item 2</li></ul></div>`

		err := os.WriteFile(expectedFile, []byte(html), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}

		// when: asserting with matching nested structure
		testastic.AssertHTML(mt, expectedFile, html)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with nested elements, got: %s", mt.message)
		}
	})

	t.Run("void elements", func(t *testing.T) {
		// given: an expected HTML file with void elements
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		html := `<div><img src="test.jpg"><br><input type="text"></div>`

		err := os.WriteFile(expectedFile, []byte(html), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}

		// when: asserting with matching void elements
		testastic.AssertHTML(mt, expectedFile, html)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with void elements, got: %s", mt.message)
		}
	})

	t.Run("embedded matcher in attribute", func(t *testing.T) {
		// given: an expected HTML file with embedded matcher in attribute
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div style="border-left: 6px solid {{anyString}};">Content</div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div style="border-left: 6px solid #ff0000;">Content</div>`

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded matcher, got: %s", mt.message)
		}
	})

	t.Run("embedded regex in attribute", func(t *testing.T) {
		// given: an expected HTML file with embedded regex in attribute
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := "<div style=\"border-left: 6px solid {{regex `#[0-9a-fA-F]{6}`}};\">Content</div>"

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div style="border-left: 6px solid #ff0000;">Content</div>`

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded regex, got: %s", mt.message)
		}
	})

	t.Run("embedded matcher mismatch", func(t *testing.T) {
		// given: an expected HTML file with embedded regex that won't match
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := "<div style=\"border-left: 6px solid {{regex `#[0-9a-fA-F]{6}`}};\">Content</div>"

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div style="border-left: 6px solid red;">Content</div>`

		// when: asserting with non-matching HTML
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure with non-matching embedded regex")
		}
	})

	t.Run("multiple embedded matchers", func(t *testing.T) {
		// given: an expected HTML file with multiple embedded matchers
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div data-info="user-{{anyString}}-id-{{anyString}}">Content</div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div data-info="user-john-id-12345">Content</div>`

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with multiple embedded matchers, got: %s", mt.message)
		}
	})

	t.Run("embedded matcher in text content", func(t *testing.T) {
		// given: an expected HTML file with embedded matcher in text content
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div>Hello {{anyString}}, your ID is {{anyString}}</div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div>Hello World, your ID is 12345</div>`

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded matcher in text, got: %s", mt.message)
		}
	})

	t.Run("embedded anyInt", func(t *testing.T) {
		// given: an expected HTML file with embedded anyInt matcher
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div data-count="Total: {{anyInt}} items">Content</div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div data-count="Total: 42 items">Content</div>`

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded anyInt, got: %s", mt.message)
		}
	})

	t.Run("embedded anyFloat", func(t *testing.T) {
		// given: an expected HTML file with embedded anyFloat matcher
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div data-price="Price: ${{anyFloat}}">Content</div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div data-price="Price: $19.99">Content</div>`

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded anyFloat, got: %s", mt.message)
		}
	})

	t.Run("embedded anyBool", func(t *testing.T) {
		// given: an expected HTML file with embedded anyBool matcher
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div data-state="enabled={{anyBool}}">Content</div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div data-state="enabled=true">Content</div>`

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded anyBool, got: %s", mt.message)
		}
	})

	t.Run("embedded anyValue", func(t *testing.T) {
		// given: an expected HTML file with embedded anyValue matcher
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div data-info="key={{anyValue}}">Content</div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div data-info="key=anything-here-123">Content</div>`

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded anyValue, got: %s", mt.message)
		}
	})

	t.Run("embedded ignore", func(t *testing.T) {
		// given: an expected HTML file with embedded ignore matcher
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div data-timestamp="created={{ignore}}">Content</div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div data-timestamp="created=2024-01-15T10:30:00Z">Content</div>`

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded ignore, got: %s", mt.message)
		}
	})

	t.Run("embedded oneOf", func(t *testing.T) {
		// given: an expected HTML file with embedded oneOf matcher
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div class="btn btn-{{oneOf "primary" "secondary" "danger"}}">Content</div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div class="btn btn-secondary">Content</div>`

		// when: asserting with matching HTML
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with embedded oneOf, got: %s", mt.message)
		}
	})

	t.Run("embedded oneOf mismatch", func(t *testing.T) {
		// given: an expected HTML file with embedded oneOf matcher
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.html")

		expected := `<div class="btn btn-{{oneOf "primary" "secondary"}}">Content</div>`

		err := os.WriteFile(expectedFile, []byte(expected), 0o644)
		if err != nil {
			t.Fatalf("failed to create expected file: %v", err)
		}

		mt := &htmlMockT{}
		actual := `<div class="btn btn-danger">Content</div>`

		// when: asserting with non-matching HTML
		testastic.AssertHTML(mt, expectedFile, actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure with non-matching oneOf value")
		}
	})
}

// htmlMockT is a mock testing.TB for testing HTML assertions.
type htmlMockT struct {
	testing.TB
	failed  bool
	message string
}

func (m *htmlMockT) Helper() {}

func (m *htmlMockT) Fatalf(format string, args ...any) {
	m.failed = true
	m.message = format
}

func (m *htmlMockT) Errorf(format string, args ...any) {
	m.failed = true
	m.message = format
}

func (m *htmlMockT) Logf(format string, args ...any) {}
