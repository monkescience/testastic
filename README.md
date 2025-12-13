# testastic

A Go testing toolkit with JSON comparison and general assertions.

## Install

```bash
go get github.com/monkescience/testastic
```

## JSON Assertions

Compare API responses against expected JSON files with template matchers:

```go
testastic.AssertJSON(t, "testdata/user.expected.json", resp.Body)
```

**Expected file with matchers:**
```json
{
  "id": "{{anyString}}",
  "count": "{{anyInt}}",
  "email": "{{regex `^[a-z]+@example\\.com$`}}",
  "status": "{{oneOf \"pending\" \"active\"}}",
  "timestamp": "{{ignore}}"
}
```

**Available matchers:** `{{anyString}}`, `{{anyInt}}`, `{{anyFloat}}`, `{{anyBool}}`, `{{anyValue}}`, `{{ignore}}`, `{{regex ``}}`, `{{oneOf ""}}`

**Options:**
```go
AssertJSON(t, expected, actual, IgnoreArrayOrder())
AssertJSON(t, expected, actual, IgnoreArrayOrderAt("$.items"))
AssertJSON(t, expected, actual, IgnoreFields("id", "timestamp"))
```

Update expected files: `go test -update`

## General Assertions

```go
// Equality
testastic.Equal(t, expected, actual)
testastic.NotEqual(t, unexpected, actual)
testastic.DeepEqual(t, expected, actual)

// Nil/Boolean
testastic.Nil(t, value)
testastic.NotNil(t, value)
testastic.True(t, value)
testastic.False(t, value)

// Errors
testastic.NoError(t, err)
testastic.Error(t, err)
testastic.ErrorIs(t, err, target)
testastic.ErrorContains(t, err, "substring")

// Comparison
testastic.Greater(t, a, b)
testastic.GreaterOrEqual(t, a, b)
testastic.Less(t, a, b)
testastic.LessOrEqual(t, a, b)
testastic.Between(t, value, min, max)

// Strings
testastic.Contains(t, s, substring)
testastic.NotContains(t, s, substring)
testastic.HasPrefix(t, s, prefix)
testastic.HasSuffix(t, s, suffix)
testastic.Matches(t, s, `^\d+$`)
testastic.StringEmpty(t, s)
testastic.StringNotEmpty(t, s)

// Collections
testastic.Len(t, collection, expected)
testastic.Empty(t, collection)
testastic.NotEmpty(t, collection)
testastic.SliceContains(t, slice, element)
testastic.SliceNotContains(t, slice, element)
testastic.SliceEqual(t, expected, actual)
testastic.MapHasKey(t, m, key)
testastic.MapNotHasKey(t, m, key)
testastic.MapEqual(t, expected, actual)
```

## Output

Colored diff output (red for expected, green for actual):

```
testastic: assertion failed

  Equal
    expected: "Alice"
    actual:   "Bob"
```

JSON mismatches show git-style inline diff:

```diff
testastic: assertion failed

  AssertJSON (testdata/user.expected.json)
  {
-   "name": "Alice",
+   "name": "Bob",
    "age": 30
  }
```
