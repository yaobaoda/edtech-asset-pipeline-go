package main

import "testing"

func TestDecideDeliveryPrioritizesLearnerDeadline(t *testing.T) {
	tests := []struct {
		name string
		job  AssetJob
		want string
	}{
		{name: "deadline course", job: AssetJob{CourseID: "math", Deadline: "2026-10-02"}, want: "deadline"},
		{name: "no deadline marker", job: AssetJob{CourseID: "history", Deadline: "2027-01-02"}, want: "standard"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decideDelivery(tt.job).Priority; got != tt.want {
				t.Fatalf("priority = %q, want %q", got, tt.want)
			}
		})
	}
}
