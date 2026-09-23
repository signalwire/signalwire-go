// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package swml

import "sort"

// VerbHandler defines the contract for specialized SWML verb handlers.
//
// Implementations provide verb-specific validation and configuration-building
// logic for complex SWML verbs that cannot be handled generically.
type VerbHandler interface {
	// GetVerbName returns the name of the SWML verb this handler handles.
	//
	// The returned name must match the verb name used in SWML documents
	// (e.g., "ai", "play", "record").
	GetVerbName() string

	// ValidateConfig validates the configuration map for this verb.
	//
	// config is the configuration dictionary for this verb. It returns
	// (isValid, errorMessages): isValid is true when the config passes all
	// validation checks, and errorMessages contains human-readable descriptions
	// of any validation failures. When isValid is true, errorMessages will be
	// empty.
	ValidateConfig(config map[string]any) (bool, []string)

	// BuildConfig builds a configuration map for this verb from the provided
	// parameters.
	//
	// params contains the verb-specific named arguments, keyed by the name
	// each one carries in the emitted verb config. It returns the constructed
	// configuration map, or an error if the provided parameters are
	// insufficient or contradictory.
	BuildConfig(params map[string]any) (map[string]any, error)
}

// RegisterVerbHandler registers a custom handler for a SWML verb, keyed by
// the name returned by h.GetVerbName(). A subsequent call with the same verb
// name replaces the previous handler.
func (s *Service) RegisterVerbHandler(h VerbHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.verbHandlers == nil {
		s.verbHandlers = make(map[string]VerbHandler)
	}
	s.verbHandlers[h.GetVerbName()] = h
}

// GetVerbHandler returns the registered handler for verbName, or nil if no
// handler has been registered for that verb.
func (s *Service) GetVerbHandler(verbName string) VerbHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.verbHandlers[verbName]
}

// HasVerbHandler reports whether a custom handler is registered for verbName.
func (s *Service) HasVerbHandler(verbName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.verbHandlers[verbName]
	return ok
}

// VerbHandlerNames returns the names of the registered verb handlers, sorted
// lexically so the result is stable across calls.
func (s *Service) VerbHandlerNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.verbHandlers))
	for name := range s.verbHandlers {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
