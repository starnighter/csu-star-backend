package repo

import "testing"

func TestUserContributionLevelByScore(t *testing.T) {
	tests := []struct {
		name  string
		score int
		want  int
	}{
		{name: "zero score", score: 0, want: 1},
		{name: "first upgrade", score: 5, want: 2},
		{name: "reach level 10", score: 45, want: 10},
		{name: "reach level 11", score: 55, want: 11},
		{name: "reach level 20", score: 145, want: 20},
		{name: "reach level 21", score: 165, want: 21},
		{name: "cap level 100", score: 100000, want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := userContributionLevelByScore(tt.score); got != tt.want {
				t.Fatalf("userContributionLevelByScore(%d) = %d, want %d", tt.score, got, tt.want)
			}
		})
	}
}
