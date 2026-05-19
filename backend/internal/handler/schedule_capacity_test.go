package handler

import "testing"

func TestResolveSetScheduleCapacity(t *testing.T) {
	t.Run("omitted uses default 10", func(t *testing.T) {
		if got := resolveSetScheduleCapacity(nil); got != defaultSetScheduleCapacity {
			t.Fatalf("got %d, want %d", got, defaultSetScheduleCapacity)
		}
	})

	t.Run("explicit zero is preserved", func(t *testing.T) {
		zero := 0
		if got := resolveSetScheduleCapacity(&zero); got != 0 {
			t.Fatalf("got %d, want 0", got)
		}
	})

	t.Run("explicit value is preserved", func(t *testing.T) {
		five := 5
		if got := resolveSetScheduleCapacity(&five); got != 5 {
			t.Fatalf("got %d, want 5", got)
		}
	})
}
