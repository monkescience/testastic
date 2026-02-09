// Package testastic provides expressive testing utilities for Go.
//
// Testastic offers structured comparison for JSON, YAML, and HTML documents
// with support for template-based matchers, alongside general-purpose assertions
// for values, errors, strings, and collections.
//
// # JSON, YAML, and HTML Assertions
//
// Compare API responses or documents against expected files with flexible matching:
//
//	testastic.AssertJSON(t, "testdata/user.expected.json", resp.Body)
//	testastic.AssertYAML(t, "testdata/config.expected.yaml", configBytes)
//	testastic.AssertHTML(t, "testdata/page.expected.html", renderedHTML)
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
//	testastic.NoError(t, err)
//	testastic.Error(t, err)
//	testastic.ErrorIs(t, err, target)
//	testastic.ErrorAs(t, err, &pathErr)
//	testastic.ErrorContains(t, err, "substring")
//
//	testastic.Panics(t, func() { panic("boom") })
//	testastic.NotPanics(t, func() { safeFunc() })
//
// # String Assertions
//
//	testastic.Contains(t, s, "substring")
//	testastic.HasPrefix(t, s, "prefix")
//	testastic.HasSuffix(t, s, "suffix")
//	testastic.Matches(t, s, `^\d+$`)
//
// # Collection Assertions
//
//	testastic.Len(t, slice, 3)
//	testastic.Empty(t, slice)
//	testastic.SliceContains(t, slice, element)
//	testastic.MapHasKey(t, m, "key")
//	testastic.MapHasValue(t, m, "value")
//
// # Eventual Assertions
//
// For asynchronous operations, retry until a condition is met:
//
//	testastic.Eventually(t, func() bool {
//	    return server.IsReady()
//	}, 5*time.Second)
//
//	testastic.EventuallyEqual(t, "ready", func() string {
//	    return service.Status()
//	}, 3*time.Second, testastic.WithInterval(50*time.Millisecond))
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
