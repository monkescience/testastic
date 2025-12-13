package testastic

import (
	"errors"
	"strings"
	"testing"
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
	m.message = format
}

func newMockT() *assertMockT {
	return &assertMockT{}
}

// --- Equal Tests ---

func TestEqual_Pass(t *testing.T) {
	Equal(t, 42, 42)
	Equal(t, "hello", "hello")
	Equal(t, true, true)
}

func TestEqual_Fail(t *testing.T) {
	mt := newMockT()
	Equal(mt, 42, 43)
	if !mt.failed {
		t.Error("expected Equal to fail")
	}
}

func TestNotEqual_Pass(t *testing.T) {
	NotEqual(t, 42, 43)
	NotEqual(t, "hello", "world")
}

func TestNotEqual_Fail(t *testing.T) {
	mt := newMockT()
	NotEqual(mt, 42, 42)
	if !mt.failed {
		t.Error("expected NotEqual to fail")
	}
}

func TestDeepEqual_Pass(t *testing.T) {
	DeepEqual(t, []int{1, 2, 3}, []int{1, 2, 3})
	DeepEqual(t, map[string]int{"a": 1}, map[string]int{"a": 1})
}

func TestDeepEqual_Fail(t *testing.T) {
	mt := newMockT()
	DeepEqual(mt, []int{1, 2, 3}, []int{1, 2, 4})
	if !mt.failed {
		t.Error("expected DeepEqual to fail")
	}
}

// --- Nil Tests ---

func TestNil_Pass(t *testing.T) {
	Nil(t, nil)
	var ptr *int
	Nil(t, ptr)
	var slice []int
	Nil(t, slice)
}

func TestNil_Fail(t *testing.T) {
	mt := newMockT()
	Nil(mt, 42)
	if !mt.failed {
		t.Error("expected Nil to fail")
	}
}

func TestNotNil_Pass(t *testing.T) {
	NotNil(t, 42)
	NotNil(t, "hello")
	NotNil(t, []int{1, 2, 3})
}

func TestNotNil_Fail(t *testing.T) {
	mt := newMockT()
	NotNil(mt, nil)
	if !mt.failed {
		t.Error("expected NotNil to fail")
	}
}

// --- Boolean Tests ---

func TestTrue_Pass(t *testing.T) {
	True(t, true)
	True(t, 1 == 1)
}

func TestTrue_Fail(t *testing.T) {
	mt := newMockT()
	True(mt, false)
	if !mt.failed {
		t.Error("expected True to fail")
	}
}

func TestFalse_Pass(t *testing.T) {
	False(t, false)
	False(t, 1 == 2)
}

func TestFalse_Fail(t *testing.T) {
	mt := newMockT()
	False(mt, true)
	if !mt.failed {
		t.Error("expected False to fail")
	}
}

// --- Error Tests ---

func TestNoError_Pass(t *testing.T) {
	NoError(t, nil)
}

func TestNoError_Fail(t *testing.T) {
	mt := newMockT()
	NoError(mt, errors.New("some error"))
	if !mt.failed {
		t.Error("expected NoError to fail")
	}
}

func TestError_Pass(t *testing.T) {
	Error(t, errors.New("some error"))
}

func TestError_Fail(t *testing.T) {
	mt := newMockT()
	Error(mt, nil)
	if !mt.failed {
		t.Error("expected Error to fail")
	}
}

func TestErrorIs_Pass(t *testing.T) {
	baseErr := errors.New("base error")
	wrappedErr := errors.New("wrapped: base error")
	_ = wrappedErr
	ErrorIs(t, baseErr, baseErr)
}

func TestErrorIs_Fail(t *testing.T) {
	mt := newMockT()
	ErrorIs(mt, errors.New("one"), errors.New("two"))
	if !mt.failed {
		t.Error("expected ErrorIs to fail")
	}
}

func TestErrorContains_Pass(t *testing.T) {
	ErrorContains(t, errors.New("file not found"), "not found")
}

func TestErrorContains_Fail(t *testing.T) {
	mt := newMockT()
	ErrorContains(mt, errors.New("file not found"), "permission denied")
	if !mt.failed {
		t.Error("expected ErrorContains to fail")
	}
}

// --- Comparison Tests ---

func TestGreater_Pass(t *testing.T) {
	Greater(t, 10, 5)
	Greater(t, "b", "a")
}

func TestGreater_Fail(t *testing.T) {
	mt := newMockT()
	Greater(mt, 5, 10)
	if !mt.failed {
		t.Error("expected Greater to fail")
	}
}

func TestGreaterOrEqual_Pass(t *testing.T) {
	GreaterOrEqual(t, 10, 5)
	GreaterOrEqual(t, 10, 10)
}

func TestGreaterOrEqual_Fail(t *testing.T) {
	mt := newMockT()
	GreaterOrEqual(mt, 5, 10)
	if !mt.failed {
		t.Error("expected GreaterOrEqual to fail")
	}
}

func TestLess_Pass(t *testing.T) {
	Less(t, 5, 10)
	Less(t, "a", "b")
}

func TestLess_Fail(t *testing.T) {
	mt := newMockT()
	Less(mt, 10, 5)
	if !mt.failed {
		t.Error("expected Less to fail")
	}
}

func TestLessOrEqual_Pass(t *testing.T) {
	LessOrEqual(t, 5, 10)
	LessOrEqual(t, 10, 10)
}

func TestLessOrEqual_Fail(t *testing.T) {
	mt := newMockT()
	LessOrEqual(mt, 10, 5)
	if !mt.failed {
		t.Error("expected LessOrEqual to fail")
	}
}

func TestBetween_Pass(t *testing.T) {
	Between(t, 5, 1, 10)
	Between(t, 1, 1, 10)
	Between(t, 10, 1, 10)
}

func TestBetween_Fail(t *testing.T) {
	mt := newMockT()
	Between(mt, 15, 1, 10)
	if !mt.failed {
		t.Error("expected Between to fail")
	}
}

// --- String Tests ---

func TestContains_Pass(t *testing.T) {
	Contains(t, "hello world", "world")
}

func TestContains_Fail(t *testing.T) {
	mt := newMockT()
	Contains(mt, "hello world", "foo")
	if !mt.failed {
		t.Error("expected Contains to fail")
	}
}

func TestNotContains_Pass(t *testing.T) {
	NotContains(t, "hello world", "foo")
}

func TestNotContains_Fail(t *testing.T) {
	mt := newMockT()
	NotContains(mt, "hello world", "world")
	if !mt.failed {
		t.Error("expected NotContains to fail")
	}
}

func TestHasPrefix_Pass(t *testing.T) {
	HasPrefix(t, "hello world", "hello")
}

func TestHasPrefix_Fail(t *testing.T) {
	mt := newMockT()
	HasPrefix(mt, "hello world", "world")
	if !mt.failed {
		t.Error("expected HasPrefix to fail")
	}
}

func TestHasSuffix_Pass(t *testing.T) {
	HasSuffix(t, "hello world", "world")
}

func TestHasSuffix_Fail(t *testing.T) {
	mt := newMockT()
	HasSuffix(mt, "hello world", "hello")
	if !mt.failed {
		t.Error("expected HasSuffix to fail")
	}
}

func TestMatches_Pass(t *testing.T) {
	Matches(t, "hello123", `^hello\d+$`)
}

func TestMatches_Fail(t *testing.T) {
	mt := newMockT()
	Matches(mt, "hello", `^\d+$`)
	if !mt.failed {
		t.Error("expected Matches to fail")
	}
}

func TestStringEmpty_Pass(t *testing.T) {
	StringEmpty(t, "")
}

func TestStringEmpty_Fail(t *testing.T) {
	mt := newMockT()
	StringEmpty(mt, "not empty")
	if !mt.failed {
		t.Error("expected StringEmpty to fail")
	}
}

func TestStringNotEmpty_Pass(t *testing.T) {
	StringNotEmpty(t, "not empty")
}

func TestStringNotEmpty_Fail(t *testing.T) {
	mt := newMockT()
	StringNotEmpty(mt, "")
	if !mt.failed {
		t.Error("expected StringNotEmpty to fail")
	}
}

// --- Collection Tests ---

func TestLen_Pass(t *testing.T) {
	Len(t, []int{1, 2, 3}, 3)
	Len(t, "hello", 5)
	Len(t, map[string]int{"a": 1, "b": 2}, 2)
}

func TestLen_Fail(t *testing.T) {
	mt := newMockT()
	Len(mt, []int{1, 2, 3}, 5)
	if !mt.failed {
		t.Error("expected Len to fail")
	}
}

func TestEmpty_Pass(t *testing.T) {
	Empty(t, []int{})
	Empty(t, "")
	Empty(t, map[string]int{})
}

func TestEmpty_Fail(t *testing.T) {
	mt := newMockT()
	Empty(mt, []int{1})
	if !mt.failed {
		t.Error("expected Empty to fail")
	}
}

func TestNotEmpty_Pass(t *testing.T) {
	NotEmpty(t, []int{1})
	NotEmpty(t, "hello")
	NotEmpty(t, map[string]int{"a": 1})
}

func TestNotEmpty_Fail(t *testing.T) {
	mt := newMockT()
	NotEmpty(mt, []int{})
	if !mt.failed {
		t.Error("expected NotEmpty to fail")
	}
}

func TestSliceContains_Pass(t *testing.T) {
	SliceContains(t, []int{1, 2, 3}, 2)
	SliceContains(t, []string{"a", "b", "c"}, "b")
}

func TestSliceContains_Fail(t *testing.T) {
	mt := newMockT()
	SliceContains(mt, []int{1, 2, 3}, 5)
	if !mt.failed {
		t.Error("expected SliceContains to fail")
	}
}

func TestSliceNotContains_Pass(t *testing.T) {
	SliceNotContains(t, []int{1, 2, 3}, 5)
}

func TestSliceNotContains_Fail(t *testing.T) {
	mt := newMockT()
	SliceNotContains(mt, []int{1, 2, 3}, 2)
	if !mt.failed {
		t.Error("expected SliceNotContains to fail")
	}
}

func TestSliceEqual_Pass(t *testing.T) {
	SliceEqual(t, []int{1, 2, 3}, []int{1, 2, 3})
	SliceEqual(t, []string{"a", "b"}, []string{"a", "b"})
}

func TestSliceEqual_Fail_Length(t *testing.T) {
	mt := newMockT()
	SliceEqual(mt, []int{1, 2, 3}, []int{1, 2})
	if !mt.failed {
		t.Error("expected SliceEqual to fail due to length")
	}
}

func TestSliceEqual_Fail_Content(t *testing.T) {
	mt := newMockT()
	SliceEqual(mt, []int{1, 2, 3}, []int{1, 2, 4})
	if !mt.failed {
		t.Error("expected SliceEqual to fail due to content")
	}
}

func TestMapHasKey_Pass(t *testing.T) {
	MapHasKey(t, map[string]int{"a": 1, "b": 2}, "a")
}

func TestMapHasKey_Fail(t *testing.T) {
	mt := newMockT()
	MapHasKey(mt, map[string]int{"a": 1}, "b")
	if !mt.failed {
		t.Error("expected MapHasKey to fail")
	}
}

func TestMapNotHasKey_Pass(t *testing.T) {
	MapNotHasKey(t, map[string]int{"a": 1}, "b")
}

func TestMapNotHasKey_Fail(t *testing.T) {
	mt := newMockT()
	MapNotHasKey(mt, map[string]int{"a": 1}, "a")
	if !mt.failed {
		t.Error("expected MapNotHasKey to fail")
	}
}

func TestMapEqual_Pass(t *testing.T) {
	MapEqual(t, map[string]int{"a": 1, "b": 2}, map[string]int{"a": 1, "b": 2})
}

func TestMapEqual_Fail_Length(t *testing.T) {
	mt := newMockT()
	MapEqual(mt, map[string]int{"a": 1}, map[string]int{"a": 1, "b": 2})
	if !mt.failed {
		t.Error("expected MapEqual to fail due to length")
	}
}

func TestMapEqual_Fail_Value(t *testing.T) {
	mt := newMockT()
	MapEqual(mt, map[string]int{"a": 1}, map[string]int{"a": 2})
	if !mt.failed {
		t.Error("expected MapEqual to fail due to value")
	}
}

// --- Error Message Format Test ---

func TestErrorMessageFormat(t *testing.T) {
	mt := newMockT()
	Equal(mt, "expected", "actual")
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
