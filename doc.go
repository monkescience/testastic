// Package testastic provides expressive testing utilities for Go.
//
// Testastic offers structured comparison for JSON, YAML, HTML, and plain text
// documents with support for template-based matchers, alongside general-purpose
// assertions for values, errors, strings, and collections.
//
// # JSON, YAML, HTML, and File Assertions
//
// Compare API responses or rendered documents against expected files with flexible matching:
//
//	testastic.AssertJSON(t, "testdata/user.expected.json", resp.Body)
//	testastic.AssertYAML(t, "testdata/config.expected.yaml", configBytes)
//	testastic.AssertHTML(t, "testdata/page.expected.html", renderedHTML)
//	testastic.AssertFile(t, "testdata/output.expected.txt", actualString)
//
// Input types by assertion:
//   - AssertJSON and AssertYAML: string, []byte, io.Reader, or any struct (auto-marshaled)
//   - AssertHTML: string, []byte, io.Reader, or fmt.Stringer
//   - AssertFile: string, []byte, or io.Reader
//
// Expected files support template matchers for dynamic values:
//
//	{
//	  "id": "{{anyUUID}}",
//	  "count": "{{anyInt}}",
//	  "email": "{{regex `^[a-z]+@example\\.com$`}}",
//	  "status": "{{oneOf \"pending\" \"active\"}}",
//	  "timestamp": "{{ignore}}"
//	}
//
// Available built-in matchers:
//   - {{anyString}} - matches any string
//   - {{anyInt}} - matches any integer
//   - {{anyFloat}} - matches any number
//   - {{anyBool}} - matches any boolean
//   - {{anyValue}} - matches any value including null
//   - {{anyUUID}} - matches UUID strings (RFC 4122)
//   - {{anyDateTime}} - matches ISO 8601 datetime strings
//   - {{anyURL}} - matches URL strings
//   - {{ignore}} - skips the field during comparison
//   - {{regex `pattern`}} - matches against a regular expression
//   - {{oneOf "a" "b"}} - matches one of the specified values
//
// # Comparison Options
//
// Configure comparison behavior with options:
//
//	testastic.AssertJSON(t, expected, actual,
//	    testastic.IgnoreArrayOrder(),           // ignore order globally
//	    testastic.IgnoreArrayOrderAt("$.items"), // ignore order at path
//	    testastic.IgnoreFields("id", "timestamp"), // exclude fields
//	    testastic.Message("user creation response"), // custom failure message
//	)
//
// # HTML-Specific Options
//
// Additional options are available for HTML comparison:
//
//	testastic.AssertHTML(t, expected, actual,
//	    testastic.IgnoreHTMLComments(),                     // exclude comments
//	    testastic.PreserveWhitespace(),                     // disable whitespace normalization
//	    testastic.IgnoreChildOrder(),                       // order-insensitive globally
//	    testastic.IgnoreChildOrderAt("html > body > ul"),   // order-insensitive at path
//	    testastic.IgnoreElements("script", "style"),        // exclude elements by tag
//	    testastic.IgnoreAttributes("class", "style"),       // exclude attributes globally
//	    testastic.IgnoreAttributeAt("html > body > div@id"), // exclude attribute at path
//	)
//
// # Updating Expected Files
//
// When API responses change, update expected files automatically:
//
//	go test -update
//	# or
//	TESTASTIC_UPDATE=true go test
//
// # Basic Assertions
//
// General-purpose assertions for common test scenarios:
//
//	testastic.Equal(t, expected, actual)
//	testastic.NotEqual(t, unexpected, actual)
//	testastic.DeepEqual(t, expected, actual)
//
//	testastic.Nil(t, value)
//	testastic.NotNil(t, value)
//	testastic.True(t, condition)
//	testastic.False(t, condition)
//
// # Error Assertions
//
//	testastic.NoError(t, err)
//	testastic.Error(t, err)
//	testastic.ErrorIs(t, err, target)
//	testastic.ErrorAs(t, err, &pathErr)
//	testastic.ErrorContains(t, err, "substring")
//
// # Panic Assertions
//
//	testastic.Panics(t, func() { panic("boom") })
//	testastic.NotPanics(t, func() { safeFunc() })
//
// # Comparison Assertions
//
//	testastic.Greater(t, a, b)
//	testastic.GreaterOrEqual(t, a, b)
//	testastic.Less(t, a, b)
//	testastic.LessOrEqual(t, a, b)
//	testastic.Between(t, value, min, max)
//
// # String Assertions
//
//	testastic.Contains(t, s, "substring")
//	testastic.NotContains(t, s, "substring")
//	testastic.HasPrefix(t, s, "prefix")
//	testastic.NotHasPrefix(t, s, "prefix")
//	testastic.HasSuffix(t, s, "suffix")
//	testastic.NotHasSuffix(t, s, "suffix")
//	testastic.Matches(t, s, `^\d+$`)
//	testastic.StringEmpty(t, s)
//	testastic.StringNotEmpty(t, s)
//
// Contains, NotContains, HasPrefix, NotHasPrefix, HasSuffix, NotHasSuffix,
// and Matches accept string, []byte, or fmt.Stringer inputs.
//
// # Collection Assertions
//
//	testastic.Len(t, slice, 3)
//	testastic.Empty(t, slice)
//	testastic.NotEmpty(t, slice)
//	testastic.SliceContains(t, slice, element)
//	testastic.SliceNotContains(t, slice, element)
//	testastic.SliceEqual(t, expected, actual)
//	testastic.MapHasKey(t, m, "key")
//	testastic.MapNotHasKey(t, m, "key")
//	testastic.MapHasValue(t, m, "value")
//	testastic.MapNotHasValue(t, m, "value")
//	testastic.MapEqual(t, expected, actual)
//
// # Eventual Assertions
//
// For asynchronous operations, retry until a condition is met or timeout is reached.
// The condition is checked immediately, then at regular intervals (default 100ms).
//
//	testastic.Eventually(t, func() bool {
//	    return server.IsReady()
//	}, 5*time.Second)
//
//	testastic.EventuallyTrue(t, func() bool {
//	    return isReady
//	}, 3*time.Second)
//
//	testastic.EventuallyFalse(t, func() bool {
//	    return server.IsProcessing()
//	}, 5*time.Second)
//
//	testastic.EventuallyEqual(t, "ready", func() string {
//	    return service.Status()
//	}, 3*time.Second, testastic.WithInterval(50*time.Millisecond))
//
//	testastic.EventuallyNil(t, func() any {
//	    return cache.Get("key")
//	}, 2*time.Second)
//
//	testastic.EventuallyNotNil(t, func() any {
//	    return cache.Get("key")
//	}, 2*time.Second)
//
//	testastic.EventuallyNoError(t, func() error {
//	    _, err := client.Ping()
//	    return err
//	}, 5*time.Second)
//
//	testastic.EventuallyError(t, func() error {
//	    return service.HealthCheck()
//	}, 3*time.Second)
//
// Configure polling with [WithInterval] and [WithMessage]:
//
//	testastic.Eventually(t, condition, 5*time.Second,
//	    testastic.WithInterval(50*time.Millisecond),
//	    testastic.WithMessage("waiting for server startup"),
//	)
//
// # Custom Matchers
//
// Register custom matchers for domain-specific validation:
//
//	testastic.RegisterMatcher("customID", func(args string) (testastic.Matcher, error) {
//	    return myCustomMatcher{}, nil
//	})
//
// Then use in expected files: "id": "{{customID}}"
package testastic
