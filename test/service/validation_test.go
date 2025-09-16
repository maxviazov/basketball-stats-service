package service_test

import (
	"testing"

	"github.com/maxviazov/basketball-stats-service/internal/service"
)

func TestIsValidSeason(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"Valid format", "2023-24", true},
		{"Valid format with leading space", " 2023-24", true},
		{"Valid format with trailing space", "2023-24 ", true},
		{"Invalid year format", "2023-2024", false},
		{"Invalid separator", "2023/24", false},
		{"Too short", "2023-2", false},
		{"Too long", "2023-245", false},
		{"Letters instead of numbers", "abcd-ef", false},
		{"Empty string", "", false},
		{"Only spaces", "   ", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Now we call the exported function from the service package.
			got := service.IsValidSeason(tc.input)
			if got != tc.want {
				t.Errorf("IsValidSeason(%q) = %v; want %v", tc.input, got, tc.want)
			}
		})
	}
}
