package builtin

import "testing"

// as checks a type assertion and fails the test on mismatch, so a wrong wire
// shape surfaces as a clear test failure instead of a panic.
func as[T any](t *testing.T, v any) T {
	t.Helper()
	tv, ok := v.(T)
	if !ok {
		t.Fatalf("expected %T, got %T", tv, v)
	}
	return tv
}
