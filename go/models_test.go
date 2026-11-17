package main

import "testing"

func TestStaticModelsThinkingLevels(t *testing.T) {
	models := staticModels()
	byID := map[string]struct {
		levels   []string
		zero     bool
		dynamic  bool
		thinking bool
	}{}
	for _, model := range models {
		entry := struct {
			levels   []string
			zero     bool
			dynamic  bool
			thinking bool
		}{thinking: model.Thinking != nil}
		if model.Thinking != nil {
			entry.levels = model.Thinking.Levels
			entry.zero = model.Thinking.ZeroAllowed
			entry.dynamic = model.Thinking.DynamicAllowed
		}
		byID[model.ID] = entry
	}

	text := byID["doubao"]
	if !text.thinking || !text.zero || !text.dynamic || len(text.levels) != 3 || text.levels[0] != "low" || text.levels[2] != "high" {
		t.Fatalf("unexpected text thinking support: %+v", text)
	}
	video := byID["doubao-video"]
	if video.thinking {
		t.Fatalf("video model must not expose thinking levels: %+v", video)
	}
}

func TestModelsForCredentialFiltersGlobalCatalog(t *testing.T) {
	cn := modelsForCredential(credential{Type: credentialProvider, Site: siteCN})
	if len(cn) < 10 {
		t.Fatalf("domestic catalog unexpectedly short: %d", len(cn))
	}
	global := modelsForCredential(credential{Type: credentialProvider, Site: siteGlobal})
	if len(global) != 1 || global[0].ID != "doubao" {
		t.Fatalf("global catalog should expose only the domestic-compatible text model: %+v", global)
	}
}

func TestModelTokenLimitsFitCombinedContext(t *testing.T) {
	for _, model := range staticModels() {
		if model.InputTokenLimit <= 0 || model.OutputTokenLimit <= 0 || model.ContextLength <= 0 {
			t.Fatalf("model %s has invalid token limits: input=%d output=%d context=%d", model.ID, model.InputTokenLimit, model.OutputTokenLimit, model.ContextLength)
		}
		if model.InputTokenLimit+model.OutputTokenLimit != model.ContextLength {
			t.Fatalf("model %s input+output=%d exceeds combined context=%d", model.ID, model.InputTokenLimit+model.OutputTokenLimit, model.ContextLength)
		}
		if model.MaxCompletionTokens != model.OutputTokenLimit {
			t.Fatalf("model %s completion limit mismatch: %d vs %d", model.ID, model.MaxCompletionTokens, model.OutputTokenLimit)
		}
	}
}
