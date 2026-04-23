// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package swml

// VerbHandler defines the contract for specialized SWML verb handlers.
//
// Implementations provide verb-specific validation and configuration-building
// logic for complex SWML verbs that cannot be handled generically. This is
// the Go equivalent of the Python SWMLVerbHandler abstract base class.
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
	// params contains keyword arguments specific to this verb, mirroring the
	// **kwargs pattern from Python. It returns the constructed configuration
	// map, or an error if the provided parameters are insufficient or
	// contradictory.
	BuildConfig(params map[string]any) (map[string]any, error)
}

// RegisterVerbHandler registers a custom handler for a SWML verb, keyed by
// the name returned by h.GetVerbName(). A subsequent call with the same verb
// name replaces the previous handler. This is the Go equivalent of Python's
// VerbHandlerRegistry.register_handler.
func (s *Service) RegisterVerbHandler(h VerbHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.verbHandlers == nil {
		s.verbHandlers = make(map[string]VerbHandler)
	}
	s.verbHandlers[h.GetVerbName()] = h
}

// GetVerbHandler returns the registered handler for verbName, or nil if no
// handler has been registered for that verb. This is the Go equivalent of
// Python's VerbHandlerRegistry.get_handler.
func (s *Service) GetVerbHandler(verbName string) VerbHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.verbHandlers[verbName]
}

// HasVerbHandler reports whether a custom handler is registered for verbName.
// This is the Go equivalent of Python's VerbHandlerRegistry.has_handler.
func (s *Service) HasVerbHandler(verbName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.verbHandlers[verbName]
	return ok
}
