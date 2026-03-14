// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package namespaces

// ---------- MessageLogs ----------

// MessageLogs provides message log queries.
type MessageLogs struct {
	Resource
}

// List lists message logs.
func (r *MessageLogs) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Get retrieves a specific message log entry.
func (r *MessageLogs) Get(logID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(logID), nil)
}

// ---------- VoiceLogs ----------

// VoiceLogs provides voice log queries.
type VoiceLogs struct {
	Resource
}

// List lists voice logs.
func (r *VoiceLogs) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Get retrieves a specific voice log entry.
func (r *VoiceLogs) Get(logID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(logID), nil)
}

// ListEvents lists events for a voice log entry.
func (r *VoiceLogs) ListEvents(logID string, params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(logID, "events"), params)
}

// ---------- FaxLogs ----------

// FaxLogs provides fax log queries.
type FaxLogs struct {
	Resource
}

// List lists fax logs.
func (r *FaxLogs) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// Get retrieves a specific fax log entry.
func (r *FaxLogs) Get(logID string) (map[string]any, error) {
	return r.HTTP.Get(r.Path(logID), nil)
}

// ---------- ConferenceLogs ----------

// ConferenceLogs provides conference log queries.
type ConferenceLogs struct {
	Resource
}

// List lists conference logs.
func (r *ConferenceLogs) List(params map[string]string) (map[string]any, error) {
	return r.HTTP.Get(r.Base, params)
}

// ---------- LogsNamespace ----------

// LogsNamespace groups all log query resources.
type LogsNamespace struct {
	Messages    *MessageLogs
	Voice       *VoiceLogs
	Fax         *FaxLogs
	Conferences *ConferenceLogs
}

// NewLogsNamespace creates a new LogsNamespace with all sub-resources initialized.
func NewLogsNamespace(client HTTPClient) *LogsNamespace {
	return &LogsNamespace{
		Messages:    &MessageLogs{Resource{HTTP: client, Base: "/api/messaging/logs"}},
		Voice:       &VoiceLogs{Resource{HTTP: client, Base: "/api/voice/logs"}},
		Fax:         &FaxLogs{Resource{HTTP: client, Base: "/api/fax/logs"}},
		Conferences: &ConferenceLogs{Resource{HTTP: client, Base: "/api/logs/conferences"}},
	}
}
