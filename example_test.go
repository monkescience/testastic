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
