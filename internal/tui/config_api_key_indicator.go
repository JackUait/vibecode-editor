package tui

import (
	"fmt"

	"github.com/jackuait/wisp-deck/internal/claudeconfig"
)

// configAPIKeyIndicator returns a display string counting how many of the four
// Anthropic aliases a config maps. Mappings are counted against the config's own
// provider models, so a value belonging to a different provider isn't
// mis-counted.
//
// All-In alone reports nothing when it maps nothing. It routes by picker row and
// writes no alias keys, so its count can never be anything but zero and the
// "unmapped" it used to render was permanent. Every other provider keeps the
// word: there, an empty count is a state the user can still change.
func configAPIKeyIndicator(configsDir, file, name string) string {
	if file == "" {
		return ""
	}
	provider := claudeconfig.ProviderForConfig(configsDir, claudeconfig.Config{Name: name, File: file})
	if provider.Key == claudeconfig.AllInProvider.Key {
		return ""
	}
	mappings := claudeconfig.ReadModelMappings(configsDir, file, claudeconfig.ProviderModels[provider.Key])
	mapped := 0
	for _, v := range mappings {
		if v >= 0 {
			mapped++
		}
	}
	if mapped > 0 {
		return fmt.Sprintf("%d mapped", mapped)
	}
	return "unmapped"
}
