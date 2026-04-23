// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package swml

import "fmt"

// AIVerbHandler is a concrete VerbHandler for the SWML "ai" verb.
//
// It implements the VerbHandler interface and provides validation and
// configuration-building logic for the AI verb. This is the Go equivalent
// of the Python AIVerbHandler class in core/swml_handler.py.
//
// The AI verb is complex and requires specialized handling, particularly
// for managing prompts, SWAIG functions, and AI configurations.
type AIVerbHandler struct{}

// NewAIVerbHandler returns a new AIVerbHandler ready for registration.
//
// Example:
//
//	svc.RegisterVerbHandler(swml.NewAIVerbHandler())
func NewAIVerbHandler() *AIVerbHandler {
	return &AIVerbHandler{}
}

// GetVerbName returns "ai", the name of the SWML verb this handler handles.
func (h *AIVerbHandler) GetVerbName() string {
	return "ai"
}

// ValidateConfig validates the configuration map for the AI verb.
//
// Validation rules (ported from Python AIVerbHandler.validate_config):
//   - "prompt" key must be present and must be a map[string]any.
//   - "prompt" must contain exactly one of "text" or "pom" (mutually exclusive).
//   - If "prompt.contexts" is present it must be a map[string]any.
//   - If "SWAIG" is present it must be a map[string]any.
//
// Returns (true, nil) when the config is valid; (false, errors) when it is not.
func (h *AIVerbHandler) ValidateConfig(config map[string]any) (bool, []string) {
	var errors []string

	// Require "prompt" key.
	rawPrompt, ok := config["prompt"]
	if !ok {
		errors = append(errors, "Missing required field 'prompt'")
		return false, errors
	}

	prompt, ok := rawPrompt.(map[string]any)
	if !ok {
		errors = append(errors, "'prompt' must be an object")
		return false, errors
	}

	// Exactly one of "text" or "pom" must be present (mutually exclusive).
	_, hasText := prompt["text"]
	_, hasPom := prompt["pom"]
	baseCount := 0
	if hasText {
		baseCount++
	}
	if hasPom {
		baseCount++
	}
	switch {
	case baseCount == 0:
		errors = append(errors, "'prompt' must contain either 'text' or 'pom' as base prompt")
	case baseCount > 1:
		errors = append(errors, "'prompt' can only contain one of: 'text' or 'pom' (mutually exclusive)")
	}

	// "prompt.contexts" is optional but must be a map when present.
	if rawContexts, exists := prompt["contexts"]; exists {
		if _, ok := rawContexts.(map[string]any); !ok {
			errors = append(errors, "'prompt.contexts' must be an object")
		}
	}

	// "SWAIG" is optional but must be a map when present.
	if rawSWAIG, exists := config["SWAIG"]; exists {
		if _, ok := rawSWAIG.(map[string]any); !ok {
			errors = append(errors, "'SWAIG' must be an object")
		}
	}

	return len(errors) == 0, errors
}

// BuildConfig assembles an AI verb configuration map from the provided params.
//
// Recognised keys in params (ported from Python AIVerbHandler.build_config):
//
//   - "prompt_text"     (string)          — text prompt; mutually exclusive with "prompt_pom"
//   - "prompt_pom"      ([]any or similar) — POM structure; mutually exclusive with "prompt_text"
//   - "contexts"        (map[string]any)  — optional contexts / steps configuration
//   - "post_prompt"     (string)          — optional post-prompt text; wrapped in {"text": value}
//   - "post_prompt_url" (string)          — optional post-prompt URL
//   - "swaig"           (map[string]any)  — optional SWAIG configuration; emitted as "SWAIG"
//
// Additional keys in params are handled as follows (matching Python **kwargs logic):
//   - "languages", "hints", "pronounce", "global_data" — emitted as top-level keys.
//   - All other extra keys are collected under a nested "params" map.
//
// Returns (configMap, nil) on success, or (nil, error) if the parameters are
// contradictory (e.g. both prompt_text and prompt_pom supplied, or neither).
func (h *AIVerbHandler) BuildConfig(params map[string]any) (map[string]any, error) {
	config := make(map[string]any)

	// Extract named params — mirroring Python's named arguments.
	promptText, _ := params["prompt_text"]
	promptPom, _ := params["prompt_pom"]
	contexts, _ := params["contexts"]
	postPrompt, _ := params["post_prompt"]
	postPromptURL, _ := params["post_prompt_url"]
	swaig, _ := params["swaig"]

	// Exactly one of prompt_text / prompt_pom is required.
	hasText := promptText != nil
	hasPom := promptPom != nil
	baseCount := 0
	if hasText {
		baseCount++
	}
	if hasPom {
		baseCount++
	}
	switch {
	case baseCount == 0:
		return nil, fmt.Errorf("either prompt_text or prompt_pom must be provided as base prompt")
	case baseCount > 1:
		return nil, fmt.Errorf("prompt_text and prompt_pom are mutually exclusive")
	}

	// Build prompt object.
	promptConfig := make(map[string]any)
	if hasText {
		promptConfig["text"] = promptText
	} else {
		promptConfig["pom"] = promptPom
	}
	if contexts != nil {
		promptConfig["contexts"] = contexts
	}
	config["prompt"] = promptConfig

	// Post-prompt: wrapped in {"text": value} to match Python behaviour.
	if postPrompt != nil {
		config["post_prompt"] = map[string]any{"text": postPrompt}
	}

	// Post-prompt URL.
	if postPromptURL != nil {
		config["post_prompt_url"] = postPromptURL
	}

	// SWAIG: emitted as top-level "SWAIG" key.
	if swaig != nil {
		config["SWAIG"] = swaig
	}

	// Pass-through kwargs — mirroring Python's **kwargs handling.
	// "languages", "hints", "pronounce", "global_data" go to the top level;
	// everything else accumulates under "params".
	topLevelKeys := map[string]bool{
		"prompt_text": true, "prompt_pom": true, "contexts": true,
		"post_prompt": true, "post_prompt_url": true, "swaig": true,
	}
	extraParams := make(map[string]any)
	for k, v := range params {
		if topLevelKeys[k] {
			continue
		}
		switch k {
		case "languages", "hints", "pronounce", "global_data":
			config[k] = v
		default:
			extraParams[k] = v
		}
	}
	if len(extraParams) > 0 {
		config["params"] = extraParams
	}

	return config, nil
}
