package testastic_test

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"github.com/monkescience/testastic"
)

// testError is a custom error type for testing ErrorAs.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

type testStringer struct {
	value string
}

func (s testStringer) String() string {
	return s.value
}

func TestEqual(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: two equal values of various types
		// when: asserting equality
		// then: the test passes
		testastic.Equal(t, 42, 42)
		testastic.Equal(t, "hello", "hello")
		testastic.Equal(t, true, true)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: two unequal integers
		mt := &mockT{}

		// when: asserting equality
		testastic.Equal(mt, 42, 43)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Equal to fail")
		}
	})
}

func TestNotEqual(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: two unequal values
		// when: asserting inequality
		// then: the test passes
		testastic.NotEqual(t, 42, 43)
		testastic.NotEqual(t, "hello", "world")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: two equal integers
		mt := &mockT{}

		// when: asserting inequality
		testastic.NotEqual(mt, 42, 42)

		// then: the test fails
		if !mt.failed {
			t.Error("expected NotEqual to fail")
		}
	})
}

func TestDeepEqual(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: two deeply equal slices and maps
		// when: asserting deep equality
		// then: the test passes
		testastic.DeepEqual(t, []int{1, 2, 3}, []int{1, 2, 3})
		testastic.DeepEqual(t, map[string]int{"a": 1}, map[string]int{"a": 1})
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: two slices with different content
		mt := &mockT{}

		// when: asserting deep equality
		testastic.DeepEqual(mt, []int{1, 2, 3}, []int{1, 2, 4})

		// then: the test fails
		if !mt.failed {
			t.Error("expected DeepEqual to fail")
		}
	})
}

func TestNil(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

		// given: a non-nil value
		mt := &mockT{}

		// when: asserting nil
		testastic.Nil(mt, 42)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Nil to fail")
		}
	})
}

func TestNotNil(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: non-nil values
		// when: asserting not nil
		// then: the test passes
		testastic.NotNil(t, 42)
		testastic.NotNil(t, "hello")
		testastic.NotNil(t, []int{1, 2, 3})
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a nil value
		mt := &mockT{}

		// when: asserting not nil
		testastic.NotNil(mt, nil)

		// then: the test fails
		if !mt.failed {
			t.Error("expected NotNil to fail")
		}
	})
}

func TestTrue(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: true boolean values
		// when: asserting true
		// then: the test passes
		testastic.True(t, true)
		testastic.True(t, 1 < 2)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a false value
		mt := &mockT{}

		// when: asserting true
		testastic.True(mt, false)

		// then: the test fails
		if !mt.failed {
			t.Error("expected True to fail")
		}
	})
}

func TestFalse(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: false boolean values
		// when: asserting false
		// then: the test passes
		testastic.False(t, false)
		testastic.False(t, 1 == 2)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a true value
		mt := &mockT{}

		// when: asserting false
		testastic.False(mt, true)

		// then: the test fails
		if !mt.failed {
			t.Error("expected False to fail")
		}
	})
}

func TestNoError(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: a nil error
		// when: asserting no error
		// then: the test passes
		testastic.NoError(t, nil)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a non-nil error
		mt := &mockT{}

		// when: asserting no error
		testastic.NoError(mt, errors.New("some error"))

		// then: the test fails
		if !mt.failed {
			t.Error("expected NoError to fail")
		}
	})
}

func TestError(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: a non-nil error
		// when: asserting error
		// then: the test passes
		testastic.Error(t, errors.New("some error"))
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a nil error
		mt := &mockT{}

		// when: asserting error
		testastic.Error(mt, nil)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Error to fail")
		}
	})
}

func TestErrorIs(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: an error and its target
		baseErr := errors.New("base error")
		wrappedErr := errors.New("wrapped: base error")
		_ = wrappedErr

		// when: asserting error is the target
		// then: the test passes
		testastic.ErrorIs(t, baseErr, baseErr)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: two different errors
		mt := &mockT{}

		// when: asserting one error is another
		testastic.ErrorIs(mt, errors.New("one"), errors.New("two"))

		// then: the test fails
		if !mt.failed {
			t.Error("expected ErrorIs to fail")
		}
	})
}

func TestErrorContains(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: an error containing a substring
		// when: asserting error contains the substring
		// then: the test passes
		testastic.ErrorContains(t, errors.New("file not found"), "not found")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: an error not containing the substring
		mt := &mockT{}

		// when: asserting error contains the substring
		testastic.ErrorContains(mt, errors.New("file not found"), "permission denied")

		// then: the test fails
		if !mt.failed {
			t.Error("expected ErrorContains to fail")
		}
	})
}

func TestGreater(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: values where first is greater than second
		// when: asserting greater
		// then: the test passes
		testastic.Greater(t, 10, 5)
		testastic.Greater(t, "b", "a")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: values where first is less than second
		mt := &mockT{}

		// when: asserting greater
		testastic.Greater(mt, 5, 10)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Greater to fail")
		}
	})
}

func TestGreaterOrEqual(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: values where first is greater than or equal to second
		// when: asserting greater or equal
		// then: the test passes
		testastic.GreaterOrEqual(t, 10, 5)
		testastic.GreaterOrEqual(t, 10, 10)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: values where first is less than second
		mt := &mockT{}

		// when: asserting greater or equal
		testastic.GreaterOrEqual(mt, 5, 10)

		// then: the test fails
		if !mt.failed {
			t.Error("expected GreaterOrEqual to fail")
		}
	})
}

func TestLess(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: values where first is less than second
		// when: asserting less
		// then: the test passes
		testastic.Less(t, 5, 10)
		testastic.Less(t, "a", "b")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: values where first is greater than second
		mt := &mockT{}

		// when: asserting less
		testastic.Less(mt, 10, 5)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Less to fail")
		}
	})
}

func TestLessOrEqual(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: values where first is less than or equal to second
		// when: asserting less or equal
		// then: the test passes
		testastic.LessOrEqual(t, 5, 10)
		testastic.LessOrEqual(t, 10, 10)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: values where first is greater than second
		mt := &mockT{}

		// when: asserting less or equal
		testastic.LessOrEqual(mt, 10, 5)

		// then: the test fails
		if !mt.failed {
			t.Error("expected LessOrEqual to fail")
		}
	})
}

func TestBetween(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: a value within the range (inclusive)
		// when: asserting between
		// then: the test passes
		testastic.Between(t, 5, 1, 10)
		testastic.Between(t, 1, 1, 10)
		testastic.Between(t, 10, 1, 10)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a value outside the range
		mt := &mockT{}

		// when: asserting between
		testastic.Between(mt, 15, 1, 10)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Between to fail")
		}
	})
}

func TestContains(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: supported string-like values containing a substring
		// when: asserting contains
		// then: the test passes
		testastic.Contains(t, "hello world", "world")
		testastic.Contains(t, []byte("hello world"), "world")
		testastic.Contains(t, testStringer{value: "hello world"}, "world")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a string not containing a substring
		mt := &mockT{}

		// when: asserting contains
		testastic.Contains(mt, "hello world", "foo")

		// then: the test fails
		if !mt.failed {
			t.Error("expected Contains to fail")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		t.Parallel()

		// given: a non-string-like value
		mt := &mockT{}

		// when: asserting contains
		testastic.Contains(mt, 123, "23")

		// then: the test fails with a type error
		if !mt.failed {
			t.Error("expected Contains to fail for unsupported type")
		}

		if !strings.Contains(mt.message, "unsupported type int") {
			t.Error("expected Contains to report unsupported type")
		}
	})
}

func TestNotContains(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: supported string-like values not containing a substring
		// when: asserting not contains
		// then: the test passes
		testastic.NotContains(t, "hello world", "foo")
		testastic.NotContains(t, []byte("hello world"), "foo")
		testastic.NotContains(t, testStringer{value: "hello world"}, "foo")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a string containing a substring
		mt := &mockT{}

		// when: asserting not contains
		testastic.NotContains(mt, "hello world", "world")

		// then: the test fails
		if !mt.failed {
			t.Error("expected NotContains to fail")
		}
	})
}

func TestHasPrefix(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: supported string-like values with a specific prefix
		// when: asserting has prefix
		// then: the test passes
		testastic.HasPrefix(t, "hello world", "hello")
		testastic.HasPrefix(t, []byte("hello world"), "hello")
		testastic.HasPrefix(t, testStringer{value: "hello world"}, "hello")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a string without the specified prefix
		mt := &mockT{}

		// when: asserting has prefix
		testastic.HasPrefix(mt, "hello world", "world")

		// then: the test fails
		if !mt.failed {
			t.Error("expected HasPrefix to fail")
		}
	})
}

func TestNotHasPrefix(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: supported string-like values without a specific prefix
		// when: asserting not has prefix
		// then: the test passes
		testastic.NotHasPrefix(t, "hello world", "world")
		testastic.NotHasPrefix(t, []byte("hello world"), "world")
		testastic.NotHasPrefix(t, testStringer{value: "hello world"}, "world")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a string with the specified prefix
		mt := &mockT{}

		// when: asserting not has prefix
		testastic.NotHasPrefix(mt, "hello world", "hello")

		// then: the test fails
		if !mt.failed {
			t.Error("expected NotHasPrefix to fail")
		}
	})
}

func TestHasSuffix(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: supported string-like values with a specific suffix
		// when: asserting has suffix
		// then: the test passes
		testastic.HasSuffix(t, "hello world", "world")
		testastic.HasSuffix(t, []byte("hello world"), "world")
		testastic.HasSuffix(t, testStringer{value: "hello world"}, "world")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a string without the specified suffix
		mt := &mockT{}

		// when: asserting has suffix
		testastic.HasSuffix(mt, "hello world", "hello")

		// then: the test fails
		if !mt.failed {
			t.Error("expected HasSuffix to fail")
		}
	})
}

func TestNotHasSuffix(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: supported string-like values without a specific suffix
		// when: asserting not has suffix
		// then: the test passes
		testastic.NotHasSuffix(t, "hello world", "hello")
		testastic.NotHasSuffix(t, []byte("hello world"), "hello")
		testastic.NotHasSuffix(t, testStringer{value: "hello world"}, "hello")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a string with the specified suffix
		mt := &mockT{}

		// when: asserting not has suffix
		testastic.NotHasSuffix(mt, "hello world", "world")

		// then: the test fails
		if !mt.failed {
			t.Error("expected NotHasSuffix to fail")
		}
	})
}

func TestMatches(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: supported string-like values matching a regex pattern
		// when: asserting matches
		// then: the test passes
		testastic.Matches(t, "hello123", `^hello\d+$`)
		testastic.Matches(t, []byte("hello123"), `^hello\d+$`)
		testastic.Matches(t, testStringer{value: "hello123"}, `^hello\d+$`)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a string not matching a regex pattern
		mt := &mockT{}

		// when: asserting matches
		testastic.Matches(mt, "hello", `^\d+$`)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Matches to fail")
		}
	})
}

func TestStringEmpty(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: an empty string
		// when: asserting string empty
		// then: the test passes
		testastic.StringEmpty(t, "")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a non-empty string
		mt := &mockT{}

		// when: asserting string empty
		testastic.StringEmpty(mt, "not empty")

		// then: the test fails
		if !mt.failed {
			t.Error("expected StringEmpty to fail")
		}
	})
}

func TestStringNotEmpty(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: a non-empty string
		// when: asserting string not empty
		// then: the test passes
		testastic.StringNotEmpty(t, "not empty")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: an empty string
		mt := &mockT{}

		// when: asserting string not empty
		testastic.StringNotEmpty(mt, "")

		// then: the test fails
		if !mt.failed {
			t.Error("expected StringNotEmpty to fail")
		}
	})
}

func TestLen(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: collections with known lengths
		// when: asserting length
		// then: the test passes
		testastic.Len(t, []int{1, 2, 3}, 3)
		testastic.Len(t, "hello", 5)
		testastic.Len(t, map[string]int{"a": 1, "b": 2}, 2)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a collection with a different length than expected
		mt := &mockT{}

		// when: asserting length
		testastic.Len(mt, []int{1, 2, 3}, 5)

		// then: the test fails
		if !mt.failed {
			t.Error("expected Len to fail")
		}
	})
}

func TestEmpty(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: empty collections
		// when: asserting empty
		// then: the test passes
		testastic.Empty(t, []int{})
		testastic.Empty(t, "")
		testastic.Empty(t, map[string]int{})
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a non-empty collection
		mt := &mockT{}

		// when: asserting empty
		testastic.Empty(mt, []int{1})

		// then: the test fails
		if !mt.failed {
			t.Error("expected Empty to fail")
		}
	})
}

func TestNotEmpty(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: non-empty collections
		// when: asserting not empty
		// then: the test passes
		testastic.NotEmpty(t, []int{1})
		testastic.NotEmpty(t, "hello")
		testastic.NotEmpty(t, map[string]int{"a": 1})
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: an empty collection
		mt := &mockT{}

		// when: asserting not empty
		testastic.NotEmpty(mt, []int{})

		// then: the test fails
		if !mt.failed {
			t.Error("expected NotEmpty to fail")
		}
	})
}

func TestSliceContains(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: a slice containing a specific element
		// when: asserting slice contains
		// then: the test passes
		testastic.SliceContains(t, []int{1, 2, 3}, 2)
		testastic.SliceContains(t, []string{"a", "b", "c"}, "b")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a slice not containing a specific element
		mt := &mockT{}

		// when: asserting slice contains
		testastic.SliceContains(mt, []int{1, 2, 3}, 5)

		// then: the test fails
		if !mt.failed {
			t.Error("expected SliceContains to fail")
		}
	})
}

func TestSliceNotContains(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: a slice not containing a specific element
		// when: asserting slice not contains
		// then: the test passes
		testastic.SliceNotContains(t, []int{1, 2, 3}, 5)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a slice containing a specific element
		mt := &mockT{}

		// when: asserting slice not contains
		testastic.SliceNotContains(mt, []int{1, 2, 3}, 2)

		// then: the test fails
		if !mt.failed {
			t.Error("expected SliceNotContains to fail")
		}
	})
}

func TestSliceEqual(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: two equal slices
		// when: asserting slice equal
		// then: the test passes
		testastic.SliceEqual(t, []int{1, 2, 3}, []int{1, 2, 3})
		testastic.SliceEqual(t, []string{"a", "b"}, []string{"a", "b"})
	})

	t.Run("fail length", func(t *testing.T) {
		t.Parallel()

		// given: two slices of different lengths
		mt := &mockT{}

		// when: asserting slice equal
		testastic.SliceEqual(mt, []int{1, 2, 3}, []int{1, 2})

		// then: the test fails
		if !mt.failed {
			t.Error("expected SliceEqual to fail due to length")
		}
	})

	t.Run("fail content", func(t *testing.T) {
		t.Parallel()

		// given: two slices with different content
		mt := &mockT{}

		// when: asserting slice equal
		testastic.SliceEqual(mt, []int{1, 2, 3}, []int{1, 2, 4})

		// then: the test fails
		if !mt.failed {
			t.Error("expected SliceEqual to fail due to content")
		}
	})
}

func TestMapHasKey(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: a map containing a specific key
		// when: asserting map has key
		// then: the test passes
		testastic.MapHasKey(t, map[string]int{"a": 1, "b": 2}, "a")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a map not containing a specific key
		mt := &mockT{}

		// when: asserting map has key
		testastic.MapHasKey(mt, map[string]int{"a": 1}, "b")

		// then: the test fails
		if !mt.failed {
			t.Error("expected MapHasKey to fail")
		}
	})
}

func TestMapNotHasKey(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: a map not containing a specific key
		// when: asserting map not has key
		// then: the test passes
		testastic.MapNotHasKey(t, map[string]int{"a": 1}, "b")
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a map containing a specific key
		mt := &mockT{}

		// when: asserting map not has key
		testastic.MapNotHasKey(mt, map[string]int{"a": 1}, "a")

		// then: the test fails
		if !mt.failed {
			t.Error("expected MapNotHasKey to fail")
		}
	})
}

func TestMapEqual(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: two equal maps
		// when: asserting map equal
		// then: the test passes
		testastic.MapEqual(t, map[string]int{"a": 1, "b": 2}, map[string]int{"a": 1, "b": 2})
	})

	t.Run("fail length", func(t *testing.T) {
		t.Parallel()

		// given: two maps of different sizes
		mt := &mockT{}

		// when: asserting map equal
		testastic.MapEqual(mt, map[string]int{"a": 1}, map[string]int{"a": 1, "b": 2})

		// then: the test fails
		if !mt.failed {
			t.Error("expected MapEqual to fail due to length")
		}
	})

	t.Run("fail value", func(t *testing.T) {
		t.Parallel()

		// given: two maps with different values
		mt := &mockT{}

		// when: asserting map equal
		testastic.MapEqual(mt, map[string]int{"a": 1}, map[string]int{"a": 2})

		// then: the test fails
		if !mt.failed {
			t.Error("expected MapEqual to fail due to value")
		}
	})
}

func TestErrorAs(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: a wrapped error matching target type
		var target *testError

		err := &testError{msg: "something failed"}

		// when: asserting error as target type
		// then: the test passes
		testastic.ErrorAs(t, err, &target)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: an error that does not match target type
		mt := &mockT{}

		var target *testError

		// when: asserting a plain error as a custom type
		testastic.ErrorAs(mt, errors.New("plain error"), &target)

		// then: the test fails
		if !mt.failed {
			t.Error("expected ErrorAs to fail")
		}
	})
}

func TestErrorIsWithNilTarget(t *testing.T) {
	t.Parallel()

	t.Run("nil error matches nil target", func(t *testing.T) {
		t.Parallel()

		// given: nil error and nil target
		// when: asserting ErrorIs
		// then: the test passes (errors.Is(nil, nil) == true)
		testastic.ErrorIs(t, nil, nil)
	})

	t.Run("non-nil error does not match nil target", func(t *testing.T) {
		t.Parallel()

		// given: non-nil error and nil target
		mt := &mockT{}

		// when: asserting ErrorIs
		testastic.ErrorIs(mt, errors.New("some error"), nil)

		// then: the test fails without panicking
		if !mt.failed {
			t.Error("expected ErrorIs to fail")
		}
	})
}

func TestPanics(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: a function that panics
		// when: asserting panics
		// then: the test passes
		testastic.Panics(t, func() {
			panic("boom")
		})
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a function that does not panic
		mt := &mockT{}

		// when: asserting panics
		testastic.Panics(mt, func() {})

		// then: the test fails
		if !mt.failed {
			t.Error("expected Panics to fail")
		}
	})
}

func TestNotPanics(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: a function that does not panic
		// when: asserting not panics
		// then: the test passes
		testastic.NotPanics(t, func() {})
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a function that panics
		mt := &mockT{}

		// when: asserting not panics
		testastic.NotPanics(mt, func() {
			panic("boom")
		})

		// then: the test fails
		if !mt.failed {
			t.Error("expected NotPanics to fail")
		}
	})
}

func TestMapHasValue(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: a map containing the value
		// when: asserting map has value
		// then: the test passes
		testastic.MapHasValue(t, map[string]int{"a": 1, "b": 2}, 2)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a map not containing the value
		mt := &mockT{}

		// when: asserting map has value
		testastic.MapHasValue(mt, map[string]int{"a": 1, "b": 2}, 5)

		// then: the test fails
		if !mt.failed {
			t.Error("expected MapHasValue to fail")
		}
	})
}

func TestMapNotHasValue(t *testing.T) {
	t.Parallel()

	t.Run("pass", func(t *testing.T) {
		t.Parallel()

		// given: a map not containing the value
		// when: asserting map not has value
		// then: the test passes
		testastic.MapNotHasValue(t, map[string]int{"a": 1, "b": 2}, 5)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()

		// given: a map containing the value
		mt := &mockT{}

		// when: asserting map not has value
		testastic.MapNotHasValue(mt, map[string]int{"a": 1, "b": 2}, 2)

		// then: the test fails
		if !mt.failed {
			t.Error("expected MapNotHasValue to fail")
		}
	})
}

func TestErrorMessageFormat(t *testing.T) {
	t.Parallel()

	t.Run("contains testastic prefix and assertion name", func(t *testing.T) {
		t.Parallel()

		// given: two unequal values
		mt := &mockT{}

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

type nilDerefStringer struct{ n int }

func (s *nilDerefStringer) String() string { return strconv.Itoa(s.n) }

func TestOrderedComparisons_FailOnNaN(t *testing.T) {
	t.Parallel()

	nan := math.NaN()

	t.Run("Between", func(t *testing.T) {
		t.Parallel()

		// given: a NaN value; when: asserting it lies between bounds
		mt := &mockT{}
		testastic.Between(mt, nan, 0.0, 10.0)

		// then: the assertion fails (NaN satisfies no ordering)
		if !mt.failed {
			t.Error("expected Between(NaN, 0, 10) to fail")
		}
	})

	t.Run("LessOrEqual", func(t *testing.T) {
		t.Parallel()

		mt := &mockT{}
		testastic.LessOrEqual(mt, nan, 5.0)

		if !mt.failed {
			t.Error("expected LessOrEqual(NaN, 5) to fail")
		}
	})

	t.Run("Greater", func(t *testing.T) {
		t.Parallel()

		mt := &mockT{}
		testastic.Greater(mt, nan, 5.0)

		if !mt.failed {
			t.Error("expected Greater(NaN, 5) to fail")
		}
	})

	t.Run("non-float ordered types unaffected", func(t *testing.T) {
		t.Parallel()

		mt := &mockT{}
		testastic.Greater(mt, 5, 3)

		if mt.failed {
			t.Errorf("expected Greater(5, 3) to pass, got: %s", mt.message)
		}
	})
}

func TestNil_HandlesUnsafePointer(t *testing.T) {
	t.Parallel()

	var p unsafe.Pointer

	// given: a nil unsafe.Pointer; when/then: Nil passes
	mt := &mockT{}

	testastic.Nil(mt, p)

	if mt.failed {
		t.Errorf("expected Nil(nil unsafe.Pointer) to pass, got: %s", mt.message)
	}

	// and: NotNil fails
	mt = &mockT{}

	testastic.NotNil(mt, p)

	if !mt.failed {
		t.Error("expected NotNil(nil unsafe.Pointer) to fail")
	}
}

func TestContains_TypedNilStringerDoesNotPanic(t *testing.T) {
	t.Parallel()

	// given: a typed-nil pointer whose String() dereferences the receiver
	var s *nilDerefStringer

	// when: asserting Contains on it; then: it reports a failure instead of panicking
	mt := &mockT{}

	testastic.Contains(mt, s, "1")

	if !mt.failed {
		t.Error("expected Contains on a typed-nil Stringer to report a failure")
	}
}

func TestMisuseErrorsAreFatal(t *testing.T) {
	t.Parallel()

	t.Run("Len on non-collection", func(t *testing.T) {
		t.Parallel()

		mt := &mockT{}

		testastic.Len(mt, 42, 3)

		if !mt.fatal {
			t.Error("expected Len misuse to be fatal")
		}
	})

	t.Run("Contains on unsupported type", func(t *testing.T) {
		t.Parallel()

		mt := &mockT{}

		testastic.Contains(mt, struct{}{}, "x")

		if !mt.fatal {
			t.Error("expected Contains misuse to be fatal")
		}
	})

	t.Run("Matches with invalid pattern", func(t *testing.T) {
		t.Parallel()

		mt := &mockT{}

		testastic.Matches(mt, "value", "[invalid")

		if !mt.fatal {
			t.Error("expected Matches invalid pattern to be fatal")
		}
	})
}

func TestSliceContains_FailureShowsModerateSliceInFull(t *testing.T) {
	t.Parallel()

	// given: a moderate-length slice missing the element
	mt := &mockT{}
	testastic.SliceContains(mt, []int{10, 20, 30, 40, 50, 60, 70, 80}, 99)

	// then: the failure message shows every element, not just the first few
	if !mt.failed {
		t.Fatal("expected SliceContains to fail")
	}

	if !strings.Contains(mt.message, "80") {
		t.Errorf("expected failure message to show all elements, got: %s", mt.message)
	}
}
