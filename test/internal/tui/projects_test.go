package tui_test

import (
	"testing"

	"github.com/yourusername/ghost-tab/internal/models"
	"github.com/yourusername/ghost-tab/internal/tui"
)

func TestFilterProjects(t *testing.T) {
	projects := []models.Project{
		{Name: "web-app", Path: "/home/user/web-app"},
		{Name: "cli-tool", Path: "/home/user/cli-tool"},
		{Name: "data-service", Path: "/opt/data-service"},
	}

	tests := []struct {
		name     string
		filter   string
		expected int
	}{
		{"no filter", "", 3},
		{"filter web", "web", 1},
		{"filter cli", "cli", 1},
		{"filter data", "data", 1},
		{"no match", "xyz", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := tui.FilterProjects(projects, tt.filter)
			if len(filtered) != tt.expected {
				t.Errorf("FilterProjects with %q: expected %d, got %d", tt.filter, tt.expected, len(filtered))
			}
		})
	}
}
