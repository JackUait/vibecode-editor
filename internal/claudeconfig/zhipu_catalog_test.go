package claudeconfig

import (
	"os"
	"path/filepath"
	"testing"
)

// The GLM Coding Plan retired its old lineup onto GLM-5.3. Measured against a
// live coding-plan key on 2026-09-09 through the Anthropic route this provider
// points at (`https://api.z.ai/api/anthropic`), the response's own `model` field
// reports which model actually answered:
//
//	glm-5.3, glm-5.3-flash -> themselves
//	glm-5.2, glm-5.1, glm-5 -> glm-5.3
//	glm-4.7, glm-4.6, glm-4.5-air, glm-5-turbo, glm-4.5 -> glm-5.3-flash
//
// So the two 5.3 ids are the only offered rows whose price and window describe
// the model that runs, and they are what the four aliases must name.
func TestZhipuCatalog_offersTheModelsTheCodingPlanActuallyServes(t *testing.T) {
	provider, ok := ProviderByKey("zhipu")
	if !ok {
		t.Fatal("catalog is missing the zhipu provider")
	}
	byID := make(map[string]Model, len(provider.Models))
	for _, m := range provider.Models {
		byID[m.ID] = m
	}

	// Context 1000000: both ids accepted a 1,020,017-token prompt, so nothing
	// smaller may be declared. Output 131072: the endpoint names its own range
	// in the 400 it answers max_tokens=131073 with,
	// "[1210][The max_tokens parameter is illegal.：限制数值范围[1,131072]]".
	// Prices are models.dev's zai entry for the metered API.
	for _, want := range []Model{
		{"glm-5.3", 1.40, 4.40, 1000000, 131072},
		{"glm-5.3-flash", 0.075, 0.25, 1000000, 131072},
	} {
		got, present := byID[want.ID]
		if !present {
			t.Errorf("zhipu catalog is missing %q", want.ID)
			continue
		}
		if got != want {
			t.Errorf("zhipu model %q = %+v, want %+v", want.ID, got, want)
		}
	}

	// Opus and Sonnet on the flagship, Haiku and Fable on the cheap tier — the
	// same split the plan's own remap makes.
	wantDefaults := [4]string{"glm-5.3", "glm-5.3", "glm-5.3-flash", "glm-5.3-flash"}
	if provider.DefaultModels != wantDefaults {
		t.Errorf("zhipu defaults = %v, want %v", provider.DefaultModels, wantDefaults)
	}
}

// The retired ids keep their rows: they still answer 200, so a profile written
// before the remap must still resolve its stored model in the cycler and in the
// Stats cost calculator.
func TestZhipuCatalog_keepsTheRetiredIdsSelectable(t *testing.T) {
	models := ModelsForConfig("Zhipu GLM")
	present := make(map[string]bool, len(models))
	for _, id := range models {
		present[id] = true
	}
	for _, id := range []string{"glm-5.2", "glm-5.1", "glm-5", "glm-4.7", "glm-4.6", "glm-4.5-air"} {
		if !present[id] {
			t.Errorf("zhipu catalog dropped %q, stranding profiles that name it", id)
		}
	}
}

// A shipped default is copied verbatim on a fresh install, so a retired model
// left in one launches every new profile on it. Nothing tied the two together
// before: the catalog moved to the 5.3 tier while defaults/claude-configs/
// still named glm-4.7.
func TestShippedDefaults_nameTheirProvidersDefaultModels(t *testing.T) {
	dir := filepath.Join("..", "..", "defaults", "claude-configs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read defaults: %v", err)
	}
	seen := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			env := readEnvMap(t, filepath.Join(dir, entry.Name()))
			provider, ok := ProviderByKey(env["WISP_DECK_SUBSCRIPTION_PROVIDER"])
			if !ok {
				provider = ProviderForName(entry.Name())
			}
			for i, key := range envKeys {
				// An alias a default deliberately leaves unmapped is its own
				// choice; only a mapping that names the wrong model is drift.
				got, mapped := env[key]
				if !mapped {
					continue
				}
				seen++
				if got != provider.DefaultModels[i] {
					t.Errorf("%s = %q, want provider %q default %q",
						key, got, provider.Key, provider.DefaultModels[i])
				}
			}
		})
	}
	if seen == 0 {
		t.Error("no shipped default mapped an alias — the guard checked nothing")
	}
}
