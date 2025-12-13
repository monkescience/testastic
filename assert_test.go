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

func newMockT() *assertMockT {
	return &assertMockT{}
}

// --- Equal Tests ---

func TestEqual_Pass(t *testing.T) {
	testastic.Equal(t, 42, 42)
	testastic.Equal(t, "hello", "hello")
	testastic.Equal(t, true, true)
}

func TestEqual_Fail(t *testing.T) {
	mt := newMockT()

	testastic.Equal(mt, 42, 43)

	if !mt.failed {
		t.Error("expected Equal to fail")
	}
}

func TestNotEqual_Pass(t *testing.T) {
	testastic.NotEqual(t, 42, 43)
	testastic.NotEqual(t, "hello", "world")
}

func TestNotEqual_Fail(t *testing.T) {
	mt := newMockT()

	testastic.NotEqual(mt, 42, 42)

	if !mt.failed {
		t.Error("expected NotEqual to fail")
	}
}

func TestDeepEqual_Pass(t *testing.T) {
	testastic.DeepEqual(t, []int{1, 2, 3}, []int{1, 2, 3})
	testastic.DeepEqual(t, map[string]int{"a": 1}, map[string]int{"a": 1})
}

func TestDeepEqual_Fail(t *testing.T) {
	mt := newMockT()

	testastic.DeepEqual(mt, []int{1, 2, 3}, []int{1, 2, 4})

	if !mt.failed {
		t.Error("expected DeepEqual to fail")
	}
}

// --- Nil Tests ---

func TestNil_Pass(t *testing.T) {
	testastic.Nil(t, nil)

	var ptr *int

	testastic.Nil(t, ptr)

	var slice []int

	testastic.Nil(t, slice)
}

func TestNil_Fail(t *testing.T) {
	mt := newMockT()

	testastic.Nil(mt, 42)

	if !mt.failed {
		t.Error("expected Nil to fail")
	}
}

func TestNotNil_Pass(t *testing.T) {
	testastic.NotNil(t, 42)
	testastic.NotNil(t, "hello")
	testastic.NotNil(t, []int{1, 2, 3})
}

func TestNotNil_Fail(t *testing.T) {
	mt := newMockT()

	testastic.NotNil(mt, nil)

	if !mt.failed {
		t.Error("expected NotNil to fail")
	}
}

// --- Boolean Tests ---

func TestTrue_Pass(t *testing.T) {
	testastic.True(t, true)
	testastic.True(t, 1 < 2)
}

func TestTrue_Fail(t *testing.T) {
	mt := newMockT()

	testastic.True(mt, false)

	if !mt.failed {
		t.Error("expected True to fail")
	}
}

func TestFalse_Pass(t *testing.T) {
	testastic.False(t, false)
	testastic.False(t, 1 == 2)
}

func TestFalse_Fail(t *testing.T) {
	mt := newMockT()

	testastic.False(mt, true)

	if !mt.failed {
		t.Error("expected False to fail")
	}
}

// --- Error Tests ---

func TestNoError_Pass(t *testing.T) {
	testastic.NoError(t, nil)
}

func TestNoError_Fail(t *testing.T) {
	mt := newMockT()

	testastic.NoError(mt, errors.New("some error"))

	if !mt.failed {
		t.Error("expected NoError to fail")
	}
}

func TestError_Pass(t *testing.T) {
	testastic.Error(t, errors.New("some error"))
}

func TestError_Fail(t *testing.T) {
	mt := newMockT()

	testastic.Error(mt, nil)

	if !mt.failed {
		t.Error("expected Error to fail")
	}
}

func TestErrorIs_Pass(t *testing.T) {
	baseErr := errors.New("base error")
	wrappedErr := errors.New("wrapped: base error")
	_ = wrappedErr

	testastic.ErrorIs(t, baseErr, baseErr)
}

func TestErrorIs_Fail(t *testing.T) {
	mt := newMockT()

	testastic.ErrorIs(mt, errors.New("one"), errors.New("two"))

	if !mt.failed {
		t.Error("expected ErrorIs to fail")
	}
}

func TestErrorContains_Pass(t *testing.T) {
	testastic.ErrorContains(t, errors.New("file not found"), "not found")
}

func TestErrorContains_Fail(t *testing.T) {
	mt := newMockT()

	testastic.ErrorContains(mt, errors.New("file not found"), "permission denied")

	if !mt.failed {
		t.Error("expected ErrorContains to fail")
	}
}

// --- Comparison Tests ---

func TestGreater_Pass(t *testing.T) {
	testastic.Greater(t, 10, 5)
	testastic.Greater(t, "b", "a")
}

func TestGreater_Fail(t *testing.T) {
	mt := newMockT()

	testastic.Greater(mt, 5, 10)

	if !mt.failed {
		t.Error("expected Greater to fail")
	}
}

func TestGreaterOrEqual_Pass(t *testing.T) {
	testastic.GreaterOrEqual(t, 10, 5)
	testastic.GreaterOrEqual(t, 10, 10)
}

func TestGreaterOrEqual_Fail(t *testing.T) {
	mt := newMockT()

	testastic.GreaterOrEqual(mt, 5, 10)

	if !mt.failed {
		t.Error("expected GreaterOrEqual to fail")
	}
}

func TestLess_Pass(t *testing.T) {
	testastic.Less(t, 5, 10)
	testastic.Less(t, "a", "b")
}

func TestLess_Fail(t *testing.T) {
	mt := newMockT()

	testastic.Less(mt, 10, 5)

	if !mt.failed {
		t.Error("expected Less to fail")
	}
}

func TestLessOrEqual_Pass(t *testing.T) {
	testastic.LessOrEqual(t, 5, 10)
	testastic.LessOrEqual(t, 10, 10)
}

func TestLessOrEqual_Fail(t *testing.T) {
	mt := newMockT()

	testastic.LessOrEqual(mt, 10, 5)

	if !mt.failed {
		t.Error("expected LessOrEqual to fail")
	}
}

func TestBetween_Pass(t *testing.T) {
	testastic.Between(t, 5, 1, 10)
	testastic.Between(t, 1, 1, 10)
	testastic.Between(t, 10, 1, 10)
}

func TestBetween_Fail(t *testing.T) {
	mt := newMockT()

	testastic.Between(mt, 15, 1, 10)

	if !mt.failed {
		t.Error("expected Between to fail")
	}
}

// --- String Tests ---

func TestContains_Pass(t *testing.T) {
	testastic.Contains(t, "hello world", "world")
}

func TestContains_Fail(t *testing.T) {
	mt := newMockT()

	testastic.Contains(mt, "hello world", "foo")

	if !mt.failed {
		t.Error("expected Contains to fail")
	}
}

func TestNotContains_Pass(t *testing.T) {
	testastic.NotContains(t, "hello world", "foo")
}

func TestNotContains_Fail(t *testing.T) {
	mt := newMockT()

	testastic.NotContains(mt, "hello world", "world")

	if !mt.failed {
		t.Error("expected NotContains to fail")
	}
}

func TestHasPrefix_Pass(t *testing.T) {
	testastic.HasPrefix(t, "hello world", "hello")
}

func TestHasPrefix_Fail(t *testing.T) {
	mt := newMockT()

	testastic.HasPrefix(mt, "hello world", "world")

	if !mt.failed {
		t.Error("expected HasPrefix to fail")
	}
}

func TestHasSuffix_Pass(t *testing.T) {
	testastic.HasSuffix(t, "hello world", "world")
}

func TestHasSuffix_Fail(t *testing.T) {
	mt := newMockT()

	testastic.HasSuffix(mt, "hello world", "hello")

	if !mt.failed {
		t.Error("expected HasSuffix to fail")
	}
}

func TestMatches_Pass(t *testing.T) {
	testastic.Matches(t, "hello123", `^hello\d+$`)
}

func TestMatches_Fail(t *testing.T) {
	mt := newMockT()

	testastic.Matches(mt, "hello", `^\d+$`)

	if !mt.failed {
		t.Error("expected Matches to fail")
	}
}

func TestStringEmpty_Pass(t *testing.T) {
	testastic.StringEmpty(t, "")
}

func TestStringEmpty_Fail(t *testing.T) {
	mt := newMockT()

	testastic.StringEmpty(mt, "not empty")

	if !mt.failed {
		t.Error("expected StringEmpty to fail")
	}
}

func TestStringNotEmpty_Pass(t *testing.T) {
	testastic.StringNotEmpty(t, "not empty")
}

func TestStringNotEmpty_Fail(t *testing.T) {
	mt := newMockT()

	testastic.StringNotEmpty(mt, "")

	if !mt.failed {
		t.Error("expected StringNotEmpty to fail")
	}
}

// --- Collection Tests ---

func TestLen_Pass(t *testing.T) {
	testastic.Len(t, []int{1, 2, 3}, 3)
	testastic.Len(t, "hello", 5)
	testastic.Len(t, map[string]int{"a": 1, "b": 2}, 2)
}

func TestLen_Fail(t *testing.T) {
	mt := newMockT()

	testastic.Len(mt, []int{1, 2, 3}, 5)

	if !mt.failed {
		t.Error("expected Len to fail")
	}
}

func TestEmpty_Pass(t *testing.T) {
	testastic.Empty(t, []int{})
	testastic.Empty(t, "")
	testastic.Empty(t, map[string]int{})
}

func TestEmpty_Fail(t *testing.T) {
	mt := newMockT()

	testastic.Empty(mt, []int{1})

	if !mt.failed {
		t.Error("expected Empty to fail")
	}
}

func TestNotEmpty_Pass(t *testing.T) {
	testastic.NotEmpty(t, []int{1})
	testastic.NotEmpty(t, "hello")
	testastic.NotEmpty(t, map[string]int{"a": 1})
}

func TestNotEmpty_Fail(t *testing.T) {
	mt := newMockT()

	testastic.NotEmpty(mt, []int{})

	if !mt.failed {
		t.Error("expected NotEmpty to fail")
	}
}

func TestSliceContains_Pass(t *testing.T) {
	testastic.SliceContains(t, []int{1, 2, 3}, 2)
	testastic.SliceContains(t, []string{"a", "b", "c"}, "b")
}

func TestSliceContains_Fail(t *testing.T) {
	mt := newMockT()

	testastic.SliceContains(mt, []int{1, 2, 3}, 5)

	if !mt.failed {
		t.Error("expected SliceContains to fail")
	}
}

func TestSliceNotContains_Pass(t *testing.T) {
	testastic.SliceNotContains(t, []int{1, 2, 3}, 5)
}

func TestSliceNotContains_Fail(t *testing.T) {
	mt := newMockT()

	testastic.SliceNotContains(mt, []int{1, 2, 3}, 2)

	if !mt.failed {
		t.Error("expected SliceNotContains to fail")
	}
}

func TestSliceEqual_Pass(t *testing.T) {
	testastic.SliceEqual(t, []int{1, 2, 3}, []int{1, 2, 3})
	testastic.SliceEqual(t, []string{"a", "b"}, []string{"a", "b"})
}

func TestSliceEqual_Fail_Length(t *testing.T) {
	mt := newMockT()

	testastic.SliceEqual(mt, []int{1, 2, 3}, []int{1, 2})

	if !mt.failed {
		t.Error("expected SliceEqual to fail due to length")
	}
}

func TestSliceEqual_Fail_Content(t *testing.T) {
	mt := newMockT()

	testastic.SliceEqual(mt, []int{1, 2, 3}, []int{1, 2, 4})

	if !mt.failed {
		t.Error("expected SliceEqual to fail due to content")
	}
}

func TestMapHasKey_Pass(t *testing.T) {
	testastic.MapHasKey(t, map[string]int{"a": 1, "b": 2}, "a")
}

func TestMapHasKey_Fail(t *testing.T) {
	mt := newMockT()

	testastic.MapHasKey(mt, map[string]int{"a": 1}, "b")

	if !mt.failed {
		t.Error("expected MapHasKey to fail")
	}
}

func TestMapNotHasKey_Pass(t *testing.T) {
	testastic.MapNotHasKey(t, map[string]int{"a": 1}, "b")
}

func TestMapNotHasKey_Fail(t *testing.T) {
	mt := newMockT()

	testastic.MapNotHasKey(mt, map[string]int{"a": 1}, "a")

	if !mt.failed {
		t.Error("expected MapNotHasKey to fail")
	}
}

func TestMapEqual_Pass(t *testing.T) {
	testastic.MapEqual(t, map[string]int{"a": 1, "b": 2}, map[string]int{"a": 1, "b": 2})
}

func TestMapEqual_Fail_Length(t *testing.T) {
	mt := newMockT()

	testastic.MapEqual(mt, map[string]int{"a": 1}, map[string]int{"a": 1, "b": 2})

	if !mt.failed {
		t.Error("expected MapEqual to fail due to length")
	}
}

func TestMapEqual_Fail_Value(t *testing.T) {
	mt := newMockT()

	testastic.MapEqual(mt, map[string]int{"a": 1}, map[string]int{"a": 2})

	if !mt.failed {
		t.Error("expected MapEqual to fail due to value")
	}
}

// --- Error Message Format Test ---

func TestErrorMessageFormat(t *testing.T) {
	mt := newMockT()

	testastic.Equal(mt, "expected", "actual")

	if !mt.failed {
		t.Error("expected Equal to fail")
	}

	if !strings.Contains(mt.message, "testastic:") {
		t.Error("expected error message to contain 'testastic:'")
	}

	if !strings.Contains(mt.message, "Equal") {
		t.Error("expected error message to contain assertion name")
	}
}
