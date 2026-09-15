package domain

import (
	"slices"
	"testing"
)

func TestCanTransition(t *testing.T) {
	// docs/api_design.md のステータス遷移表。ここに無い組はすべて不許可
	allowed := map[[2]string]bool{
		{"pending", "approved"}:   true,
		{"pending", "rejected"}:   true,
		{"pending", "cancelled"}:  true,
		{"approved", "no_show"}:   true,
		{"approved", "cancelled"}: true,
	}

	for _, from := range ValidReservationStatuses {
		for _, to := range ValidReservationStatuses {
			want := allowed[[2]string{from, to}]
			name := from + " to " + to
			if want {
				name = "allows " + name
			} else {
				name = "rejects " + name
			}
			t.Run(name, func(t *testing.T) {
				if got := CanTransition(from, to); got != want {
					t.Fatalf("CanTransition(%q, %q) = %v, want %v", from, to, got, want)
				}
			})
		}
	}

	unknown := []struct {
		name string
		from string
		to   string
	}{
		{name: "rejects unknown source status", from: "unknown", to: "approved"},
		{name: "rejects unknown target status", from: "pending", to: "unknown"},
		{name: "rejects empty source status", from: "", to: "approved"},
		{name: "rejects empty target status", from: "pending", to: ""},
		{name: "rejects case-mismatched status", from: "Pending", to: "approved"},
	}
	for _, tt := range unknown {
		t.Run(tt.name, func(t *testing.T) {
			if CanTransition(tt.from, tt.to) {
				t.Fatalf("CanTransition(%q, %q) = true, want false", tt.from, tt.to)
			}
		})
	}
}

func TestReservationEnums(t *testing.T) {
	// docs/api_design.md・docs/table_design.md の列挙値と一致すること
	tests := []struct {
		name string
		got  []string
		want []string
	}{
		{
			name: "valid statuses match reservations.status",
			got:  ValidReservationStatuses,
			want: []string{"pending", "approved", "rejected", "cancelled", "no_show"},
		},
		{
			name: "valid sources match reservations.source",
			got:  ValidReservationSources,
			want: []string{"web", "instagram", "phone", "walk_in", "other"},
		},
		{
			name: "update status targets exclude pending",
			got:  UpdateStatusTargets,
			want: []string{"approved", "rejected", "cancelled", "no_show"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !slices.Equal(tt.got, tt.want) {
				t.Fatalf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestUpdateStatusTargetsCoverAllTransitionTargets(t *testing.T) {
	// 遷移表の遷移先は PATCH で指定可能な値に含まれていなければならない
	for from, targets := range statusTransitions {
		for _, to := range targets {
			if !slices.Contains(UpdateStatusTargets, to) {
				t.Fatalf("transition %s -> %s is not in UpdateStatusTargets %v", from, to, UpdateStatusTargets)
			}
		}
	}
}
