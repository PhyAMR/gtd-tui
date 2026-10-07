package scheduler

import (
	"testing"
	"time"

	"github.com/phuniverse/gtd/internal/db"
)

func TestCalculateNextReview(t *testing.T) {
	tests := []struct {
		name     string
		interval int
		wantDays int
	}{
		{"Interval 1", 1, 1},
		{"Interval 2", 2, 3},
		{"Interval 3", 3, 7},
		{"Interval 4", 4, 14},
		{"Interval 5", 5, 35},
	}

	s := &Scheduler{}
	now := time.Now()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &db.Task{
				Interval: tt.interval,
			}
			next := s.calculateNextReview(task)
			diff := next.Sub(now)
			expectedDiff := time.Duration(tt.wantDays) * 24 * time.Hour

			if diff < expectedDiff-time.Hour || diff > expectedDiff+time.Hour {
				t.Errorf("calculateNextReview() diff = %v, want %v", diff, expectedDiff)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 10, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"this is a very long string", 10, "this is..."},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.max)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
		}
	}
}
