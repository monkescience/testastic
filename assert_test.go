package testastic_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/monkescience/testastic"
)

// mockT captures test failures without actually failing.
type assertMockT struct {
	testing.TB
	failed  bool
	message string
}

func (m *assertMockT) Helper() {}

func (m *assertMockT) Errorf(format string, args ...any) {
	m.failed = true
	m.message = fmt.Sprintf(format, args...)
}

func (m *assertMockT) Fatalf(format string, args ...any) {
	m.failed = true
	m.message = fmt.Sprintf(format, args...)
}

func (m *assertMockT) Logf(_ string, _ ...any) {}

func newMockT() *assertMockT {
	return &assertMockT{}
}

func TestEqual(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: two equal values of various types
		// when: asserting equality
		// then: the test passes
		testastic.Equal(t, 42, 42)
		testastic.Equal(t, "hello", "hello")
		testastic.Equal(t, true, true)
	})

	t.Run("fail", func(t *testing.T) {
		// given: two unequal integers
		mt := newMockT()

		// when: asserting equality
		testastic.Equal(mt, 42, 43)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Equal to fail")
		}
	})
}

func TestNotEqual(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: two unequal values
		// when: asserting inequality
		// then: the test passes
		testastic.NotEqual(t, 42, 43)
		testastic.NotEqual(t, "hello", "world")
	})

	t.Run("fail", func(t *testing.T) {
		// given: two equal integers
		mt := newMockT()

		// when: asserting inequality
		testastic.NotEqual(mt, 42, 42)

		// then: the test fails
		if !mt.failed {
			t.Error("expected NotEqual to fail")
		}
	})
}

func TestDeepEqual(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: two deeply equal slices and maps
		// when: asserting deep equality
		// then: the test passes
		testastic.DeepEqual(t, []int{1, 2, 3}, []int{1, 2, 3})
		testastic.DeepEqual(t, map[string]int{"a": 1}, map[string]int{"a": 1})
	})

	t.Run("fail", func(t *testing.T) {
		// given: two slices with different content
		mt := newMockT()

		// when: asserting deep equality
		testastic.DeepEqual(mt, []int{1, 2, 3}, []int{1, 2, 4})

		// then: the test fails
		if !mt.failed {
			t.Error("expected DeepEqual to fail")
		}
	})
}

func TestNil(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: nil values of various types
		var ptr *int

		var slice []int

		// when: asserting nil
		// then: the test passes
		testastic.Nil(t, nil)
		testastic.Nil(t, ptr)
		testastic.Nil(t, slice)
	})

	t.Run("fail", func(t *testing.T) {
		// given: a non-nil value
		mt := newMockT()

		// when: asserting nil
		testastic.Nil(mt, 42)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Nil to fail")
		}
	})
}

func TestNotNil(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: non-nil values
		// when: asserting not nil
		// then: the test passes
		testastic.NotNil(t, 42)
		testastic.NotNil(t, "hello")
		testastic.NotNil(t, []int{1, 2, 3})
	})

	t.Run("fail", func(t *testing.T) {
		// given: a nil value
		mt := newMockT()

		// when: asserting not nil
		testastic.NotNil(mt, nil)

		// then: the test fails
		if !mt.failed {
			t.Error("expected NotNil to fail")
		}
	})
}

func TestTrue(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: true boolean values
		// when: asserting true
		// then: the test passes
		testastic.True(t, true)
		testastic.True(t, 1 < 2)
	})

	t.Run("fail", func(t *testing.T) {
		// given: a false value
		mt := newMockT()

		// when: asserting true
		testastic.True(mt, false)

		// then: the test fails
		if !mt.failed {
			t.Error("expected True to fail")
		}
	})
}

func TestFalse(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: false boolean values
		// when: asserting false
		// then: the test passes
		testastic.False(t, false)
		testastic.False(t, 1 == 2)
	})

	t.Run("fail", func(t *testing.T) {
		// given: a true value
		mt := newMockT()

		// when: asserting false
		testastic.False(mt, true)

		// then: the test fails
		if !mt.failed {
			t.Error("expected False to fail")
		}
	})
}

func TestNoError(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: a nil error
		// when: asserting no error
		// then: the test passes
		testastic.NoError(t, nil)
	})

	t.Run("fail", func(t *testing.T) {
		// given: a non-nil error
		mt := newMockT()

		// when: asserting no error
		testastic.NoError(mt, errors.New("some error"))

		// then: the test fails
		if !mt.failed {
			t.Error("expected NoError to fail")
		}
	})
}

func TestError(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: a non-nil error
		// when: asserting error
		// then: the test passes
		testastic.Error(t, errors.New("some error"))
	})

	t.Run("fail", func(t *testing.T) {
		// given: a nil error
		mt := newMockT()

		// when: asserting error
		testastic.Error(mt, nil)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Error to fail")
		}
	})
}

func TestErrorIs(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: an error and its target
		baseErr := errors.New("base error")
		wrappedErr := errors.New("wrapped: base error")
		_ = wrappedErr

		// when: asserting error is the target
		// then: the test passes
		testastic.ErrorIs(t, baseErr, baseErr)
	})

	t.Run("fail", func(t *testing.T) {
		// given: two different errors
		mt := newMockT()

		// when: asserting one error is another
		testastic.ErrorIs(mt, errors.New("one"), errors.New("two"))

		// then: the test fails
		if !mt.failed {
			t.Error("expected ErrorIs to fail")
		}
	})
}

func TestErrorContains(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: an error containing a substring
		// when: asserting error contains the substring
		// then: the test passes
		testastic.ErrorContains(t, errors.New("file not found"), "not found")
	})

	t.Run("fail", func(t *testing.T) {
		// given: an error not containing the substring
		mt := newMockT()

		// when: asserting error contains the substring
		testastic.ErrorContains(mt, errors.New("file not found"), "permission denied")

		// then: the test fails
		if !mt.failed {
			t.Error("expected ErrorContains to fail")
		}
	})
}

func TestGreater(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: values where first is greater than second
		// when: asserting greater
		// then: the test passes
		testastic.Greater(t, 10, 5)
		testastic.Greater(t, "b", "a")
	})

	t.Run("fail", func(t *testing.T) {
		// given: values where first is less than second
		mt := newMockT()

		// when: asserting greater
		testastic.Greater(mt, 5, 10)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Greater to fail")
		}
	})
}

func TestGreaterOrEqual(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: values where first is greater than or equal to second
		// when: asserting greater or equal
		// then: the test passes
		testastic.GreaterOrEqual(t, 10, 5)
		testastic.GreaterOrEqual(t, 10, 10)
	})

	t.Run("fail", func(t *testing.T) {
		// given: values where first is less than second
		mt := newMockT()

		// when: asserting greater or equal
		testastic.GreaterOrEqual(mt, 5, 10)

		// then: the test fails
		if !mt.failed {
			t.Error("expected GreaterOrEqual to fail")
		}
	})
}

func TestLess(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: values where first is less than second
		// when: asserting less
		// then: the test passes
		testastic.Less(t, 5, 10)
		testastic.Less(t, "a", "b")
	})

	t.Run("fail", func(t *testing.T) {
		// given: values where first is greater than second
		mt := newMockT()

		// when: asserting less
		testastic.Less(mt, 10, 5)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Less to fail")
		}
	})
}

func TestLessOrEqual(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: values where first is less than or equal to second
		// when: asserting less or equal
		// then: the test passes
		testastic.LessOrEqual(t, 5, 10)
		testastic.LessOrEqual(t, 10, 10)
	})

	t.Run("fail", func(t *testing.T) {
		// given: values where first is greater than second
		mt := newMockT()

		// when: asserting less or equal
		testastic.LessOrEqual(mt, 10, 5)

		// then: the test fails
		if !mt.failed {
			t.Error("expected LessOrEqual to fail")
		}
	})
}

func TestBetween(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: a value within the range (inclusive)
		// when: asserting between
		// then: the test passes
		testastic.Between(t, 5, 1, 10)
		testastic.Between(t, 1, 1, 10)
		testastic.Between(t, 10, 1, 10)
	})

	t.Run("fail", func(t *testing.T) {
		// given: a value outside the range
		mt := newMockT()

		// when: asserting between
		testastic.Between(mt, 15, 1, 10)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Between to fail")
		}
	})
}

func TestContains(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: a string containing a substring
		// when: asserting contains
		// then: the test passes
		testastic.Contains(t, "hello world", "world")
	})

	t.Run("fail", func(t *testing.T) {
		// given: a string not containing a substring
		mt := newMockT()

		// when: asserting contains
		testastic.Contains(mt, "hello world", "foo")

		// then: the test fails
		if !mt.failed {
			t.Error("expected Contains to fail")
		}
	})
}

func TestNotContains(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: a string not containing a substring
		// when: asserting not contains
		// then: the test passes
		testastic.NotContains(t, "hello world", "foo")
	})

	t.Run("fail", func(t *testing.T) {
		// given: a string containing a substring
		mt := newMockT()

		// when: asserting not contains
		testastic.NotContains(mt, "hello world", "world")

		// then: the test fails
		if !mt.failed {
			t.Error("expected NotContains to fail")
		}
	})
}

func TestHasPrefix(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: a string with a specific prefix
		// when: asserting has prefix
		// then: the test passes
		testastic.HasPrefix(t, "hello world", "hello")
	})

	t.Run("fail", func(t *testing.T) {
		// given: a string without the specified prefix
		mt := newMockT()

		// when: asserting has prefix
		testastic.HasPrefix(mt, "hello world", "world")

		// then: the test fails
		if !mt.failed {
			t.Error("expected HasPrefix to fail")
		}
	})
}

func TestHasSuffix(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: a string with a specific suffix
		// when: asserting has suffix
		// then: the test passes
		testastic.HasSuffix(t, "hello world", "world")
	})

	t.Run("fail", func(t *testing.T) {
		// given: a string without the specified suffix
		mt := newMockT()

		// when: asserting has suffix
		testastic.HasSuffix(mt, "hello world", "hello")

		// then: the test fails
		if !mt.failed {
			t.Error("expected HasSuffix to fail")
		}
	})
}

func TestMatches(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: a string matching a regex pattern
		// when: asserting matches
		// then: the test passes
		testastic.Matches(t, "hello123", `^hello\d+$`)
	})

	t.Run("fail", func(t *testing.T) {
		// given: a string not matching a regex pattern
		mt := newMockT()

		// when: asserting matches
		testastic.Matches(mt, "hello", `^\d+$`)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Matches to fail")
		}
	})
}

func TestStringEmpty(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: an empty string
		// when: asserting string empty
		// then: the test passes
		testastic.StringEmpty(t, "")
	})

	t.Run("fail", func(t *testing.T) {
		// given: a non-empty string
		mt := newMockT()

		// when: asserting string empty
		testastic.StringEmpty(mt, "not empty")

		// then: the test fails
		if !mt.failed {
			t.Error("expected StringEmpty to fail")
		}
	})
}

func TestStringNotEmpty(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: a non-empty string
		// when: asserting string not empty
		// then: the test passes
		testastic.StringNotEmpty(t, "not empty")
	})

	t.Run("fail", func(t *testing.T) {
		// given: an empty string
		mt := newMockT()

		// when: asserting string not empty
		testastic.StringNotEmpty(mt, "")

		// then: the test fails
		if !mt.failed {
			t.Error("expected StringNotEmpty to fail")
		}
	})
}

func TestLen(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: collections with known lengths
		// when: asserting length
		// then: the test passes
		testastic.Len(t, []int{1, 2, 3}, 3)
		testastic.Len(t, "hello", 5)
		testastic.Len(t, map[string]int{"a": 1, "b": 2}, 2)
	})

	t.Run("fail", func(t *testing.T) {
		// given: a collection with a different length than expected
		mt := newMockT()

		// when: asserting length
		testastic.Len(mt, []int{1, 2, 3}, 5)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Len to fail")
		}
	})
}

func TestEmpty(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: empty collections
		// when: asserting empty
		// then: the test passes
		testastic.Empty(t, []int{})
		testastic.Empty(t, "")
		testastic.Empty(t, map[string]int{})
	})

	t.Run("fail", func(t *testing.T) {
		// given: a non-empty collection
		mt := newMockT()

		// when: asserting empty
		testastic.Empty(mt, []int{1})

		// then: the test fails
		if !mt.failed {
			t.Error("expected Empty to fail")
		}
	})
}

func TestNotEmpty(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: non-empty collections
		// when: asserting not empty
		// then: the test passes
		testastic.NotEmpty(t, []int{1})
		testastic.NotEmpty(t, "hello")
		testastic.NotEmpty(t, map[string]int{"a": 1})
	})

	t.Run("fail", func(t *testing.T) {
		// given: an empty collection
		mt := newMockT()

		// when: asserting not empty
		testastic.NotEmpty(mt, []int{})

		// then: the test fails
		if !mt.failed {
			t.Error("expected NotEmpty to fail")
		}
	})
}

func TestSliceContains(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: a slice containing a specific element
		// when: asserting slice contains
		// then: the test passes
		testastic.SliceContains(t, []int{1, 2, 3}, 2)
		testastic.SliceContains(t, []string{"a", "b", "c"}, "b")
	})

	t.Run("fail", func(t *testing.T) {
		// given: a slice not containing a specific element
		mt := newMockT()

		// when: asserting slice contains
		testastic.SliceContains(mt, []int{1, 2, 3}, 5)

		// then: the test fails
		if !mt.failed {
			t.Error("expected SliceContains to fail")
		}
	})
}

func TestSliceNotContains(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: a slice not containing a specific element
		// when: asserting slice not contains
		// then: the test passes
		testastic.SliceNotContains(t, []int{1, 2, 3}, 5)
	})

	t.Run("fail", func(t *testing.T) {
		// given: a slice containing a specific element
		mt := newMockT()

		// when: asserting slice not contains
		testastic.SliceNotContains(mt, []int{1, 2, 3}, 2)

		// then: the test fails
		if !mt.failed {
			t.Error("expected SliceNotContains to fail")
		}
	})
}

func TestSliceEqual(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: two equal slices
		// when: asserting slice equal
		// then: the test passes
		testastic.SliceEqual(t, []int{1, 2, 3}, []int{1, 2, 3})
		testastic.SliceEqual(t, []string{"a", "b"}, []string{"a", "b"})
	})

	t.Run("fail length", func(t *testing.T) {
		// given: two slices of different lengths
		mt := newMockT()

		// when: asserting slice equal
		testastic.SliceEqual(mt, []int{1, 2, 3}, []int{1, 2})

		// then: the test fails
		if !mt.failed {
			t.Error("expected SliceEqual to fail due to length")
		}
	})

	t.Run("fail content", func(t *testing.T) {
		// given: two slices with different content
		mt := newMockT()

		// when: asserting slice equal
		testastic.SliceEqual(mt, []int{1, 2, 3}, []int{1, 2, 4})

		// then: the test fails
		if !mt.failed {
			t.Error("expected SliceEqual to fail due to content")
		}
	})
}

func TestMapHasKey(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: a map containing a specific key
		// when: asserting map has key
		// then: the test passes
		testastic.MapHasKey(t, map[string]int{"a": 1, "b": 2}, "a")
	})

	t.Run("fail", func(t *testing.T) {
		// given: a map not containing a specific key
		mt := newMockT()

		// when: asserting map has key
		testastic.MapHasKey(mt, map[string]int{"a": 1}, "b")

		// then: the test fails
		if !mt.failed {
			t.Error("expected MapHasKey to fail")
		}
	})
}

func TestMapNotHasKey(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: a map not containing a specific key
		// when: asserting map not has key
		// then: the test passes
		testastic.MapNotHasKey(t, map[string]int{"a": 1}, "b")
	})

	t.Run("fail", func(t *testing.T) {
		// given: a map containing a specific key
		mt := newMockT()

		// when: asserting map not has key
		testastic.MapNotHasKey(mt, map[string]int{"a": 1}, "a")

		// then: the test fails
		if !mt.failed {
			t.Error("expected MapNotHasKey to fail")
		}
	})
}

func TestMapEqual(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		// given: two equal maps
		// when: asserting map equal
		// then: the test passes
		testastic.MapEqual(t, map[string]int{"a": 1, "b": 2}, map[string]int{"a": 1, "b": 2})
	})

	t.Run("fail length", func(t *testing.T) {
		// given: two maps of different sizes
		mt := newMockT()

		// when: asserting map equal
		testastic.MapEqual(mt, map[string]int{"a": 1}, map[string]int{"a": 1, "b": 2})

		// then: the test fails
		if !mt.failed {
			t.Error("expected MapEqual to fail due to length")
		}
	})

	t.Run("fail value", func(t *testing.T) {
		// given: two maps with different values
		mt := newMockT()

		// when: asserting map equal
		testastic.MapEqual(mt, map[string]int{"a": 1}, map[string]int{"a": 2})

		// then: the test fails
		if !mt.failed {
			t.Error("expected MapEqual to fail due to value")
		}
	})
}

func TestErrorMessageFormat(t *testing.T) {
	t.Run("contains testastic prefix and assertion name", func(t *testing.T) {
		// given: two unequal values
		mt := newMockT()

		// when: asserting equality
		testastic.Equal(mt, "expected", "actual")

		// then: the test fails with proper error message format
		if !mt.failed {
			t.Error("expected Equal to fail")
		}

		if !strings.Contains(mt.message, "testastic:") {
			t.Error("expected error message to contain 'testastic:'")
		}

		if !strings.Contains(mt.message, "Equal") {
			t.Error("expected error message to contain assertion name")
		}
	})
}
