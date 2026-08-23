package testastic_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/monkescience/testastic"
)

// This file contains examples for godoc documentation.
// Since testastic functions require testing.TB, these examples demonstrate
// usage patterns rather than producing output.

func ExampleAssertJSON() {
	// In a test function:
	//
	//   func TestUserAPI(t *testing.T) {
	//       resp, _ := http.Get("/api/user/123")
	//       defer resp.Body.Close()
	//
	//       // Compare response body against expected file
	//       testastic.AssertJSON(t, "testdata/user.expected.json", resp.Body)
	//   }
	//
	// The expected file can contain matchers for dynamic values:
	//
	//   {
	//     "id": "{{anyUUID}}",
	//     "name": "Alice",
	//     "email": "{{regex `^[a-z]+@example\\.com$`}}",
	//     "createdAt": "{{anyDateTime}}",
	//     "role": "{{oneOf \"admin\" \"user\"}}"
	//   }
	//
	// AssertJSON accepts []byte, string, io.Reader, or any struct:
	//
	//   testastic.AssertJSON(t, "testdata/user.expected.json", userStruct)
	//   testastic.AssertJSON(t, "testdata/user.expected.json", jsonBytes)
	//   testastic.AssertJSON(t, "testdata/user.expected.json", jsonString)
	fmt.Println("See godoc for usage examples")
	// Output: See godoc for usage examples
}

func ExampleAssertJSON_withOptions() {
	// Use options to customize comparison behavior:
	//
	//   func TestListUsers(t *testing.T) {
	//       resp := getUsers()
	//
	//       // Ignore array order when comparing lists
	//       testastic.AssertJSON(t, "testdata/users.expected.json", resp,
	//           testastic.IgnoreArrayOrder(),
	//       )
	//
	//       // Ignore order only at specific paths
	//       testastic.AssertJSON(t, "testdata/response.expected.json", resp,
	//           testastic.IgnoreArrayOrderAt("$.data.items"),
	//           testastic.IgnoreArrayOrderAt("$.data.tags"),
	//       )
	//
	//       // Exclude volatile fields from comparison
	//       testastic.AssertJSON(t, "testdata/user.expected.json", resp,
	//           testastic.IgnoreFields("id", "createdAt", "updatedAt"),
	//       )
	//
	//       // Add context to failure messages
	//       testastic.AssertJSON(t, "testdata/user.expected.json", resp,
	//           testastic.Message("user creation response"),
	//       )
	//
	//       // Combine multiple options
	//       testastic.AssertJSON(t, "testdata/complex.expected.json", resp,
	//           testastic.IgnoreArrayOrder(),
	//           testastic.IgnoreFields("timestamp"),
	//           testastic.Message("complex response validation"),
	//       )
	//   }
	//
	// Update expected files when API changes:
	//
	//   go test -update
	//   # or use the Update option programmatically:
	//   testastic.AssertJSON(t, expected, actual, testastic.Update())
	fmt.Println("See godoc for usage examples")
	// Output: See godoc for usage examples
}

func ExampleAssertYAML() {
	// In a test function, compare YAML from a value, string, byte slice, or reader:
	//
	//   func TestConfig(t *testing.T) {
	//       actual := struct {
	//           Host string `yaml:"host"`
	//           Port int    `yaml:"port"`
	//       }{Host: "localhost", Port: 8080}
	//
	//       testastic.AssertYAML(t, "testdata/config.expected.yaml", actual)
	//   }
	//
	// Expected YAML files support the same dynamic matchers as JSON files:
	//
	//   host: localhost
	//   port: "{{anyInt}}"
	fmt.Println("See godoc for AssertYAML usage")
	// Output: See godoc for AssertYAML usage
}

func ExampleAssertHTML() {
	// In a test function, compare an HTML string, byte slice, reader, or
	// fmt.Stringer against an expected file:
	//
	//   func TestPage(t *testing.T) {
	//       actual := `<main><h1>Hello</h1><p>Welcome</p></main>`
	//
	//       testastic.AssertHTML(t, "testdata/page.expected.html", actual,
	//           testastic.IgnoreAttributes("data-request-id"),
	//       )
	//   }
	//
	// Use IgnoreHTMLComments, PreserveWhitespace, or IgnoreChildOrder when the
	// corresponding markup details are not part of the behavior under test.
	fmt.Println("See godoc for AssertHTML usage")
	// Output: See godoc for AssertHTML usage
}

func ExampleAssertFile() {
	// In a test function, compare text from a string, byte slice, or reader:
	//
	//   func TestReport(t *testing.T) {
	//       actual := "order: ORD-123456\nstatus: ready\n"
	//
	//       testastic.AssertFile(t, "testdata/report.expected.txt", actual)
	//   }
	//
	// Matchers can appear inline in the expected text file:
	//
	//   order: {{regex `ORD-[0-9]{6}`}}
	//   status: ready
	fmt.Println("See godoc for AssertFile usage")
	// Output: See godoc for AssertFile usage
}

func ExampleEqual() {
	// In a test function:
	//
	//   func TestCalculation(t *testing.T) {
	//       result := Calculate(10, 20)
	//       testastic.Equal(t, 30, result)
	//
	//       name := GetUserName(123)
	//       testastic.Equal(t, "Alice", name)
	//   }
	//
	// For complex types, use DeepEqual:
	//
	//   func TestStructEquality(t *testing.T) {
	//       expected := User{Name: "Alice", Age: 30}
	//       actual := GetUser(123)
	//       testastic.DeepEqual(t, expected, actual)
	//   }
	//
	// Related assertions:
	//
	//   testastic.NotEqual(t, unexpected, actual)  // values must differ
	//   testastic.Nil(t, value)                    // value must be nil
	//   testastic.NotNil(t, value)                 // value must not be nil
	//   testastic.True(t, condition)               // condition must be true
	//   testastic.False(t, condition)              // condition must be false
	fmt.Println("See godoc for usage examples")
	// Output: See godoc for usage examples
}

func ExampleEventually() {
	// In a test function, wait for an async condition:
	//
	//   func TestServerStartup(t *testing.T) {
	//       go startServer()
	//
	//       // Wait up to 5 seconds for server to be ready
	//       testastic.Eventually(t, func() bool {
	//           resp, err := http.Get("http://localhost:8080/health")
	//           return err == nil && resp.StatusCode == 200
	//       }, 5*time.Second)
	//   }
	//
	// Customize polling interval:
	//
	//   testastic.Eventually(t, condition, 5*time.Second,
	//       testastic.WithInterval(50*time.Millisecond),  // check every 50ms
	//       testastic.WithMessage("waiting for server"),  // add context to failures
	//   )
	//
	// Type-specific variants provide better error messages:
	//
	//   // Wait for specific value
	//   testastic.EventuallyEqual(t, "ready", func() string {
	//       return service.Status()
	//   }, 3*time.Second)
	//
	//   // Wait for no error
	//   testastic.EventuallyNoError(t, func() error {
	//       _, err := client.Ping()
	//       return err
	//   }, 5*time.Second)
	//
	//   // Wait for condition to become false
	//   testastic.EventuallyFalse(t, func() bool {
	//       return server.IsProcessing()
	//   }, 10*time.Second)
	fmt.Println("See godoc for usage examples")
	// Output: See godoc for usage examples
}

func ExampleRegisterMatcher() {
	// Register a custom matcher for domain-specific validation.
	// This is typically done in TestMain or an init function.

	// Example: matcher for order IDs with format "ORD-XXXXXX"
	testastic.RegisterMatcher("orderID", func(args string) (testastic.Matcher, error) {
		return &orderIDMatcher{}, nil
	})

	// Example: matcher with arguments for currency amounts
	testastic.RegisterMatcher("currency", func(args string) (testastic.Matcher, error) {
		// args contains everything after the matcher name
		// e.g., for {{currency USD}}, args would be "USD"
		return &currencyMatcher{currency: args}, nil
	})

	// Now use in expected JSON files:
	//
	//   {
	//     "orderId": "{{orderID}}",
	//     "total": "{{currency USD}}"
	//   }

	fmt.Println("Custom matchers registered")
	// Output: Custom matchers registered
}

// orderIDMatcher matches order IDs with format "ORD-XXXXXX".
type orderIDMatcher struct{}

func (m *orderIDMatcher) Match(actual any) bool {
	s, ok := actual.(string)
	if !ok || len(s) != 10 {
		return false
	}

	return s[:4] == "ORD-"
}

func (m *orderIDMatcher) String() string {
	return "{{orderID}}"
}

// currencyMatcher matches currency amounts for a specific currency.
type currencyMatcher struct {
	currency string
}

func (m *currencyMatcher) Match(actual any) bool {
	// Simplified example: just check it's a positive number
	switch v := actual.(type) {
	case float64:
		return v >= 0
	case int:
		return v >= 0
	}

	return false
}

func (m *currencyMatcher) String() string {
	return fmt.Sprintf("{{currency %s}}", m.currency)
}

// Ensure matchers implement the interface.
var (
	_ testastic.Matcher = (*orderIDMatcher)(nil)
	_ testastic.Matcher = (*currencyMatcher)(nil)
)

// Silence unused variable warnings for example imports.
var (
	_ = testing.T{}
	_ = time.Second
)
