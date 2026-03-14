// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package rest

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-agents-go/pkg/rest/namespaces"
)

// SignalWireClient is the top-level REST client for the SignalWire platform.
// It provides namespaced access to all SignalWire API domains.
//
// Usage:
//
//	client, err := rest.NewSignalWireClient("project-id", "api-token", "your-space.signalwire.com")
//	// or use environment variables SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, SIGNALWIRE_SPACE
//	client, err := rest.NewSignalWireClient("", "", "")
//
//	agents, err := client.Fabric.AIAgents.List(nil)
//	client.Calling.Play("call-id", map[string]any{"play": [...]})
type SignalWireClient struct {
	http      *HttpClient
	projectID string

	// Fabric API
	Fabric *namespaces.FabricNamespace

	// Calling API (REST-based call control)
	Calling *namespaces.CallingNamespace

	// Relay REST resources
	PhoneNumbers    *namespaces.PhoneNumbersNamespace
	Addresses       *namespaces.AddressesNamespace
	Queues          *namespaces.QueuesNamespace
	Recordings      *namespaces.RecordingsNamespace
	NumberGroups    *namespaces.NumberGroupsNamespace
	VerifiedCallers *namespaces.VerifiedCallersNamespace
	SipProfile      *namespaces.SipProfileNamespace
	Lookup          *namespaces.LookupNamespace
	ShortCodes      *namespaces.ShortCodesNamespace
	ImportedNumbers *namespaces.ImportedNumbersNamespace
	MFA             *namespaces.MFANamespace

	// 10DLC Campaign Registry
	Registry *namespaces.RegistryNamespace

	// Datasphere API
	Datasphere *namespaces.DatasphereNamespace

	// Video API
	Video *namespaces.VideoNamespace

	// Compatibility (Twilio-compatible) LAML API
	Compat *namespaces.CompatNamespace

	// Logs
	Logs *namespaces.LogsNamespace

	// Project management
	Project *namespaces.ProjectNamespace

	// PubSub & Chat
	PubSub *namespaces.PubSubNamespace
	Chat   *namespaces.ChatNamespace
}

// NewSignalWireClient creates a new SignalWireClient. If project, token, or
// space are empty strings the corresponding environment variables are used:
//
//	SIGNALWIRE_PROJECT_ID
//	SIGNALWIRE_API_TOKEN
//	SIGNALWIRE_SPACE
//
// An error is returned when any of the three values is still empty after the
// environment lookup.
func NewSignalWireClient(project, token, space string) (*SignalWireClient, error) {
	if project == "" {
		project = os.Getenv("SIGNALWIRE_PROJECT_ID")
	}
	if token == "" {
		token = os.Getenv("SIGNALWIRE_API_TOKEN")
	}
	if space == "" {
		space = os.Getenv("SIGNALWIRE_SPACE")
	}

	if project == "" || token == "" || space == "" {
		return nil, fmt.Errorf(
			"project, token, and space are required; provide them as arguments " +
				"or set SIGNALWIRE_PROJECT_ID, SIGNALWIRE_API_TOKEN, and SIGNALWIRE_SPACE environment variables",
		)
	}

	h := NewHttpClient(project, token, space)

	// Wrap the HttpClient in a namespaces.HTTPClient adapter so namespaces
	// can use it without importing the rest package (avoiding a cycle).
	adapter := &httpAdapter{h}

	c := &SignalWireClient{
		http:      h,
		projectID: project,
	}

	// Initialize all namespace objects
	c.Fabric = namespaces.NewFabricNamespace(adapter)
	c.Calling = namespaces.NewCallingNamespace(adapter)
	c.PhoneNumbers = namespaces.NewPhoneNumbersNamespace(adapter)
	c.Addresses = namespaces.NewAddressesNamespace(adapter)
	c.Queues = namespaces.NewQueuesNamespace(adapter)
	c.Recordings = namespaces.NewRecordingsNamespace(adapter)
	c.NumberGroups = namespaces.NewNumberGroupsNamespace(adapter)
	c.VerifiedCallers = namespaces.NewVerifiedCallersNamespace(adapter)
	c.SipProfile = namespaces.NewSipProfileNamespace(adapter)
	c.Lookup = namespaces.NewLookupNamespace(adapter)
	c.ShortCodes = namespaces.NewShortCodesNamespace(adapter)
	c.ImportedNumbers = namespaces.NewImportedNumbersNamespace(adapter)
	c.MFA = namespaces.NewMFANamespace(adapter)
	c.Registry = namespaces.NewRegistryNamespace(adapter)
	c.Datasphere = namespaces.NewDatasphereNamespace(adapter)
	c.Video = namespaces.NewVideoNamespace(adapter)
	c.Compat = namespaces.NewCompatNamespace(adapter, project)
	c.Logs = namespaces.NewLogsNamespace(adapter)
	c.Project = namespaces.NewProjectNamespace(adapter)
	c.PubSub = namespaces.NewPubSubNamespace(adapter)
	c.Chat = namespaces.NewChatNamespace(adapter)

	return c, nil
}

// ---------- httpAdapter ----------

// httpAdapter wraps *HttpClient to satisfy the namespaces.HTTPClient interface.
type httpAdapter struct {
	c *HttpClient
}

func (a *httpAdapter) Get(path string, params map[string]string) (map[string]any, error) {
	return a.c.Get(path, params)
}
func (a *httpAdapter) Post(path string, body map[string]any) (map[string]any, error) {
	return a.c.Post(path, body)
}
func (a *httpAdapter) Put(path string, body map[string]any) (map[string]any, error) {
	return a.c.Put(path, body)
}
func (a *httpAdapter) Patch(path string, body map[string]any) (map[string]any, error) {
	return a.c.Patch(path, body)
}
func (a *httpAdapter) Delete(path string) error {
	return a.c.Delete(path)
}
