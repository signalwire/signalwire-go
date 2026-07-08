// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package rest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNewRestClient_ExplicitArgs(t *testing.T) {
	client, err := NewRestClient("proj-123", "tok-456", "example.signalwire.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.projectID != "proj-123" {
		t.Errorf("projectID = %q, want %q", client.projectID, "proj-123")
	}
}

func TestNewRestClient_EnvVars(t *testing.T) {
	t.Setenv("SIGNALWIRE_PROJECT_ID", "env-proj")
	t.Setenv("SIGNALWIRE_API_TOKEN", "env-tok")
	t.Setenv("SIGNALWIRE_SPACE", "env.signalwire.com")

	client, err := NewRestClient("", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.projectID != "env-proj" {
		t.Errorf("projectID = %q, want %q", client.projectID, "env-proj")
	}
}

func TestNewRestClient_ExplicitOverridesEnv(t *testing.T) {
	t.Setenv("SIGNALWIRE_PROJECT_ID", "env-proj")
	t.Setenv("SIGNALWIRE_API_TOKEN", "env-tok")
	t.Setenv("SIGNALWIRE_SPACE", "env.signalwire.com")

	client, err := NewRestClient("explicit-proj", "explicit-tok", "explicit.signalwire.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.projectID != "explicit-proj" {
		t.Errorf("projectID = %q, want %q", client.projectID, "explicit-proj")
	}
}

func TestNewRestClient_MissingCredentials(t *testing.T) {
	// Clear env vars
	_ = os.Unsetenv("SIGNALWIRE_PROJECT_ID")
	_ = os.Unsetenv("SIGNALWIRE_API_TOKEN")
	_ = os.Unsetenv("SIGNALWIRE_SPACE")

	_, err := NewRestClient("", "", "")
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
}

func TestNewRestClient_PartialCredentials(t *testing.T) {
	_ = os.Unsetenv("SIGNALWIRE_PROJECT_ID")
	_ = os.Unsetenv("SIGNALWIRE_API_TOKEN")
	_ = os.Unsetenv("SIGNALWIRE_SPACE")

	_, err := NewRestClient("proj-123", "", "")
	if err == nil {
		t.Fatal("expected error for partial credentials")
	}
}

func TestAllNamespacesInitialized(t *testing.T) {
	client, err := NewRestClient("proj", "tok", "space.signalwire.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []struct {
		name string
		val  any
	}{
		{"Fabric", client.Fabric},
		{"Calling", client.Calling},
		{"PhoneNumbers", client.PhoneNumbers},
		{"Addresses", client.Addresses},
		{"Queues", client.Queues},
		{"Recordings", client.Recordings},
		{"NumberGroups", client.NumberGroups},
		{"VerifiedCallers", client.VerifiedCallers},
		{"SIPProfile", client.SIPProfile},
		{"Lookup", client.Lookup},
		{"ShortCodes", client.ShortCodes},
		{"ImportedNumbers", client.ImportedNumbers},
		{"MFA", client.MFA},
		{"Registry", client.Registry},
		{"Datasphere", client.Datasphere},
		{"Video", client.Video},
		{"Logs", client.Logs},
		{"Project", client.Project},
		{"PubSub", client.PubSub},
		{"Chat", client.Chat},
	}

	for _, check := range checks {
		if check.val == nil {
			t.Errorf("namespace %s is nil", check.name)
		}
	}
}

func TestFabricSubResources(t *testing.T) {
	client, err := NewRestClient("proj", "tok", "space.signalwire.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	f := client.Fabric
	checks := []struct {
		name string
		val  any
	}{
		{"AIAgents", f.AIAgents},
		{"CallFlows", f.CallFlows},
		{"Subscribers", f.Subscribers},
		{"SWMLScripts", f.SWMLScripts},
		{"SWMLWebhooks", f.SWMLWebhooks},
		{"RelayApplications", f.RelayApplications},
		{"ConferenceRooms", f.ConferenceRooms},
		{"FreeSwitchConnectors", f.FreeSwitchConnectors},
		{"SIPEndpoints", f.SIPEndpoints},
		{"CXMLScripts", f.CXMLScripts},
		{"CXMLApplications", f.CXMLApplications},
		{"SIPGateways", f.SIPGateways},
		{"CXMLWebhooks", f.CXMLWebhooks},
		{"Resources", f.Resources},
		{"Addresses", f.Addresses},
		{"Tokens", f.Tokens},
	}

	for _, check := range checks {
		if check.val == nil {
			t.Errorf("Fabric.%s is nil", check.name)
		}
	}
}

func TestVideoSubResources(t *testing.T) {
	client, err := NewRestClient("proj", "tok", "space.signalwire.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v := client.Video
	checks := []struct {
		name string
		val  any
	}{
		{"Rooms", v.Rooms},
		{"RoomTokens", v.RoomTokens},
		{"RoomSessions", v.RoomSessions},
		{"RoomRecordings", v.RoomRecordings},
		{"Conferences", v.Conferences},
		{"ConferenceTokens", v.ConferenceTokens},
		{"Streams", v.Streams},
	}

	for _, check := range checks {
		if check.val == nil {
			t.Errorf("Video.%s is nil", check.name)
		}
	}
}

func TestRegistrySubResources(t *testing.T) {
	client, err := NewRestClient("proj", "tok", "space.signalwire.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r := client.Registry
	checks := []struct {
		name string
		val  any
	}{
		{"Brands", r.Brands},
		{"Campaigns", r.Campaigns},
		{"Orders", r.Orders},
		{"Numbers", r.Numbers},
	}

	for _, check := range checks {
		if check.val == nil {
			t.Errorf("Registry.%s is nil", check.name)
		}
	}
}

func TestLogsSubResources(t *testing.T) {
	client, err := NewRestClient("proj", "tok", "space.signalwire.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	l := client.Logs
	checks := []struct {
		name string
		val  any
	}{
		{"Messages", l.Messages},
		{"Voice", l.Voice},
		{"Fax", l.Fax},
		{"Conferences", l.Conferences},
	}

	for _, check := range checks {
		if check.val == nil {
			t.Errorf("Logs.%s is nil", check.name)
		}
	}
}

func TestDatasphereSubResources(t *testing.T) {
	client, err := NewRestClient("proj", "tok", "space.signalwire.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.Datasphere.Documents == nil {
		t.Error("Datasphere.Documents is nil")
	}
}

func TestProjectSubResources(t *testing.T) {
	client, err := NewRestClient("proj", "tok", "space.signalwire.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.Project.Tokens == nil {
		t.Error("Project.Tokens is nil")
	}
}

func TestSignalWireRestError_Format(t *testing.T) {
	err := &SignalWireRestError{
		StatusCode: 404,
		Body:       `{"error":"not found"}`,
		URL:        "/api/resource/123",
		Method:     "GET",
	}

	expected := `GET /api/resource/123 returned 404: {"error":"not found"}`
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestSignalWireRestError_ImplementsError(t *testing.T) {
	// Assigning to `error` proves *SignalWireRestError satisfies the interface at
	// compile time; exercise Error() to prove the method is wired and usable.
	var err error = &SignalWireRestError{
		StatusCode: 500,
		Body:       "internal server error",
		URL:        "/api/test",
		Method:     "POST",
	}

	got := err.Error()
	want := `POST /api/test returned 500: internal server error`
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestHttpClient_URLConstruction(t *testing.T) {
	c := NewHTTPClient("proj-id", "token", "my-space.signalwire.com")
	expected := "https://my-space.signalwire.com"
	if c.BaseURL() != expected {
		t.Errorf("BaseURL() = %q, want %q", c.BaseURL(), expected)
	}
}

func TestHttpClient_GetContext_CancelledBeforeRequest(t *testing.T) {
	// An already-cancelled context must abort the request before it is sent.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("server handler was called despite a cancelled context")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewHTTPClient("proj", "tok", "ignored")
	c.SetBaseURL(srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.GetContext(ctx, "/anything", nil)
	if err == nil {
		t.Fatal("expected an error from a cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled in error chain, got %v", err)
	}
}

func TestHttpClient_GetContext_Succeeds(t *testing.T) {
	// A live context lets the request through and returns the body.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewHTTPClient("proj", "tok", "ignored")
	c.SetBaseURL(srv.URL)

	resp, err := c.GetContext(context.Background(), "/anything", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("resp[ok] = %v, want true", resp["ok"])
	}
}

func TestHttpClient_BasicFields(t *testing.T) {
	c := NewHTTPClient("proj-id", "my-token", "space.signalwire.com")
	if c.projectID != "proj-id" {
		t.Errorf("projectID = %q, want %q", c.projectID, "proj-id")
	}
	if c.token != "my-token" {
		t.Errorf("token = %q, want %q", c.token, "my-token")
	}
	if c.httpClient == nil {
		t.Error("httpClient is nil")
	}
	if c.logger == nil {
		t.Error("logger is nil")
	}
}
