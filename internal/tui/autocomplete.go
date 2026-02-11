package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackuait/ghost-tab/internal/util"
)

// SuggestionProvider is a function that returns suggestions for a given input.
type SuggestionProvider func(input string) []string

// PathSuggestionProvider returns a SuggestionProvider that suggests directory paths.
// Empty input defaults to ~/. Results are sorted alphabetically and capped at maxResults.
// Matching is case-insensitive and supports substring (glob-style) matching.
func PathSuggestionProvider(maxResults int) SuggestionProvider {
	return func(input string) []string {
		if input == "" {
			input = "~/"
		}

		expanded := util.ExpandPath(input)

		var dir string
		var prefix string

		if strings.HasSuffix(input, "/") {
			dir = expanded
			prefix = ""
		} else {
			dir = filepath.Dir(expanded)
			prefix = filepath.Base(expanded)
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}

		lowerPrefix := strings.ToLower(prefix)
		var suggestions []string

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			lowerName := strings.ToLower(name)
			// Glob-style: match if prefix appears anywhere in name
			if prefix == "" || strings.Contains(lowerName, lowerPrefix) {
				var suggestion string
				if strings.HasSuffix(input, "/") {
					suggestion = input + name + "/"
				} else {
					parentInput := input[:len(input)-len(filepath.Base(input))]
					suggestion = parentInput + name + "/"
				}
				suggestions = append(suggestions, suggestion)
			}
		}

		sort.Strings(suggestions)

		if len(suggestions) > maxResults {
			suggestions = suggestions[:maxResults]
		}

		return suggestions
	}
}
