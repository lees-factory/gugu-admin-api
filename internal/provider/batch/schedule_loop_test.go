package batch

import (
	"testing"
	"time"
)

func TestShouldAlignToMidnight(t *testing.T) {
	testCases := []struct {
		name     string
		interval time.Duration
		expected bool
	}{
		{name: "12 hours", interval: 12 * time.Hour, expected: false},
		{name: "24 hours", interval: 24 * time.Hour, expected: true},
		{name: "36 hours", interval: 36 * time.Hour, expected: false},
		{name: "48 hours", interval: 48 * time.Hour, expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if actual := shouldAlignToMidnight(tc.interval); actual != tc.expected {
				t.Fatalf("shouldAlignToMidnight(%s) = %t, want %t", tc.interval, actual, tc.expected)
			}
		})
	}
}

func TestNextMidnightRun(t *testing.T) {
	loc := time.FixedZone("KST", 9*60*60)
	now := time.Date(2026, 4, 8, 23, 59, 2, 0, loc)

	nextRun := nextMidnightRun(now, loc)

	expected := time.Date(2026, 4, 9, 0, 0, 0, 0, loc)
	if !nextRun.Equal(expected) {
		t.Fatalf("nextMidnightRun() = %s, want %s", nextRun, expected)
	}
}
