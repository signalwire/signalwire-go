// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Full success+error REST coverage for the `compatibility` (LaML 2010-04-01
// Accounts API) spec group. Every coverable canonical route gets:
//
//   - a SUCCESS test: call the SDK method, assert the response shape AND the
//     journaled request (Method/Path/MatchedRoute == endpoint_id), and
//   - an ERROR test: arm a 4xx/5xx scenario for the route's endpoint_id, call
//     the SDK, assert errors.As(*rest.SignalWireRestError) with the matching
//     StatusCode, plus the journaled ResponseStatus and MatchedRoute.
//
// The single accepted gap (matching python/java/ts) is
// compatibility.list_available_phone_number_resources_by_country — the bare
// /AvailablePhoneNumbers/{IsoCountry} route has no Go SDK method, so it is not
// exercised here.
//
// Helpers keys() and lamlAccountBase() are defined in sibling compat test
// files in this package; they are reused here, not redeclared.

package namespaces_test

import (
	"errors"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/rest"
	"github.com/signalwire/signalwire-go/pkg/rest/internal/mocktest"
)

// assertCompatErr is a shared error-path assertion. It is intentionally NOT a
// full test driver: each Test func arms its own scenario and makes its own SDK
// call (the NO-CHEAT rule). assertCompatErr only verifies the resulting error
// and the journal entry the caller's own call produced.
func assertCompatErr(t *testing.T, mock *mocktest.Harness, err error, wantStatus int, wantRoute string) {
	t.Helper()
	var restErr *rest.SignalWireRestError
	if !errors.As(err, &restErr) {
		t.Fatalf("want *rest.SignalWireRestError, got %T: %v", err, err)
	}
	if restErr.StatusCode != wantStatus {
		t.Errorf("StatusCode = %d, want %d", restErr.StatusCode, wantStatus)
	}
	j := mock.Last(t)
	if j.ResponseStatus == nil || *j.ResponseStatus != wantStatus {
		t.Errorf("response_status = %v, want %d", j.ResponseStatus, wantStatus)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != wantRoute {
		t.Errorf("matched_route = %v, want %q", j.MatchedRoute, wantRoute)
	}
}

// assertMatched asserts the last journal entry's Method/Path/MatchedRoute.
func assertMatched(t *testing.T, mock *mocktest.Harness, wantMethod, wantPath, wantRoute string) {
	t.Helper()
	j := mock.Last(t)
	if j.Method != wantMethod {
		t.Errorf("method = %q, want %q", j.Method, wantMethod)
	}
	if j.Path != wantPath {
		t.Errorf("path = %q, want %q", j.Path, wantPath)
	}
	if j.MatchedRoute == nil || *j.MatchedRoute != wantRoute {
		t.Errorf("matched_route = %v, want %q", j.MatchedRoute, wantRoute)
	}
}

// ============================================================================
// Accounts
// ============================================================================

func TestCompatCov_Accounts_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Accounts.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result == nil {
		t.Fatal("List returned nil")
	}
	assertMatched(t, mock, "GET", "/api/laml/2010-04-01/Accounts", "compatibility.list_accounts")
}

func TestCompatCov_Accounts_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_accounts", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.Accounts.List(nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_accounts")
}

func TestCompatCov_Accounts_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Accounts.Create(map[string]any{"FriendlyName": "Sub"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result == nil {
		t.Fatal("Create returned nil")
	}
	assertMatched(t, mock, "POST", "/api/laml/2010-04-01/Accounts", "compatibility.create_subprojects")
}

func TestCompatCov_Accounts_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.create_subprojects", 422, map[string]any{"error": "invalid"})
	_, err := client.Compat.Accounts.Create(map[string]any{"FriendlyName": "Sub"})
	assertCompatErr(t, mock, err, 422, "compatibility.create_subprojects")
}

func TestCompatCov_Accounts_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Accounts.Get("AC123")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result == nil {
		t.Fatal("Get returned nil")
	}
	assertMatched(t, mock, "GET", "/api/laml/2010-04-01/Accounts/AC123", "compatibility.get_account")
}

func TestCompatCov_Accounts_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.get_account", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Accounts.Get("AC404")
	assertCompatErr(t, mock, err, 404, "compatibility.get_account")
}

func TestCompatCov_Accounts_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Accounts.Update("AC123", map[string]any{"FriendlyName": "Renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result == nil {
		t.Fatal("Update returned nil")
	}
	assertMatched(t, mock, "POST", "/api/laml/2010-04-01/Accounts/AC123", "compatibility.update_account")
}

func TestCompatCov_Accounts_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_account", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Accounts.Update("AC404", map[string]any{"FriendlyName": "x"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_account")
}

// ============================================================================
// Applications
// ============================================================================

func TestCompatCov_Applications_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Applications.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result == nil {
		t.Fatal("List returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Applications", "compatibility.list_applications")
}

func TestCompatCov_Applications_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_applications", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.Applications.List(nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_applications")
}

func TestCompatCov_Applications_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Applications.Create(map[string]any{"FriendlyName": "App"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result == nil {
		t.Fatal("Create returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Applications", "compatibility.create_application")
}

func TestCompatCov_Applications_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.create_application", 422, map[string]any{"error": "invalid"})
	_, err := client.Compat.Applications.Create(map[string]any{"FriendlyName": "App"})
	assertCompatErr(t, mock, err, 422, "compatibility.create_application")
}

func TestCompatCov_Applications_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Applications.Get("AP1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result == nil {
		t.Fatal("Get returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Applications/AP1", "compatibility.get_application")
}

func TestCompatCov_Applications_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.get_application", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Applications.Get("AP404")
	assertCompatErr(t, mock, err, 404, "compatibility.get_application")
}

func TestCompatCov_Applications_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Applications.Update("AP1", map[string]any{"FriendlyName": "x"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result == nil {
		t.Fatal("Update returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Applications/AP1", "compatibility.update_application")
}

func TestCompatCov_Applications_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_application", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Applications.Update("AP404", map[string]any{"FriendlyName": "x"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_application")
}

func TestCompatCov_Applications_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Applications.Delete("AP1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result == nil {
		t.Fatal("Delete returned nil")
	}
	assertMatched(t, mock, "DELETE", lamlAccountBase(mock)+"/Applications/AP1", "compatibility.delete_application")
}

func TestCompatCov_Applications_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.delete_application", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Applications.Delete("AP404")
	assertCompatErr(t, mock, err, 404, "compatibility.delete_application")
}

// ============================================================================
// Available phone numbers (gap: by_country has no SDK method)
// ============================================================================

func TestCompatCov_AvailableNumbers_ListCountries(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.PhoneNumbers.ListAvailableCountries(nil)
	if err != nil {
		t.Fatalf("ListAvailableCountries: %v", err)
	}
	if result == nil {
		t.Fatal("ListAvailableCountries returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/AvailablePhoneNumbers", "compatibility.list_available_phone_number_resources")
}

func TestCompatCov_AvailableNumbers_ListCountries_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_available_phone_number_resources", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.PhoneNumbers.ListAvailableCountries(nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_available_phone_number_resources")
}

func TestCompatCov_AvailableNumbers_SearchLocal(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.PhoneNumbers.SearchLocal("US", map[string]string{"AreaCode": "415"})
	if err != nil {
		t.Fatalf("SearchLocal: %v", err)
	}
	if result == nil {
		t.Fatal("SearchLocal returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/AvailablePhoneNumbers/US/Local", "compatibility.search_local_available_phone_numbers")
}

func TestCompatCov_AvailableNumbers_SearchLocal_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.search_local_available_phone_numbers", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.PhoneNumbers.SearchLocal("US", nil)
	assertCompatErr(t, mock, err, 500, "compatibility.search_local_available_phone_numbers")
}

func TestCompatCov_AvailableNumbers_SearchTollFree(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.PhoneNumbers.SearchTollFree("US", map[string]string{"AreaCode": "800"})
	if err != nil {
		t.Fatalf("SearchTollFree: %v", err)
	}
	if result == nil {
		t.Fatal("SearchTollFree returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/AvailablePhoneNumbers/US/TollFree", "compatibility.search_toll_free_available_phone_numbers")
}

func TestCompatCov_AvailableNumbers_SearchTollFree_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.search_toll_free_available_phone_numbers", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.PhoneNumbers.SearchTollFree("US", nil)
	assertCompatErr(t, mock, err, 500, "compatibility.search_toll_free_available_phone_numbers")
}

// ============================================================================
// Calls
// ============================================================================

func TestCompatCov_Calls_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Calls.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result == nil {
		t.Fatal("List returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Calls", "compatibility.list_all_calls")
}

func TestCompatCov_Calls_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_all_calls", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.Calls.List(nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_all_calls")
}

func TestCompatCov_Calls_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Calls.Create(map[string]any{"To": "+15551112222", "From": "+15553334444"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result == nil {
		t.Fatal("Create returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Calls", "compatibility.create_a_call")
}

func TestCompatCov_Calls_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.create_a_call", 422, map[string]any{"error": "invalid"})
	_, err := client.Compat.Calls.Create(map[string]any{"To": "x"})
	assertCompatErr(t, mock, err, 422, "compatibility.create_a_call")
}

func TestCompatCov_Calls_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Calls.Get("CA1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result == nil {
		t.Fatal("Get returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Calls/CA1", "compatibility.retrieve_a_call")
}

func TestCompatCov_Calls_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.retrieve_a_call", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Calls.Get("CA404")
	assertCompatErr(t, mock, err, 404, "compatibility.retrieve_a_call")
}

func TestCompatCov_Calls_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Calls.Update("CA1", map[string]any{"Status": "completed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result == nil {
		t.Fatal("Update returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Calls/CA1", "compatibility.update_a_call")
}

func TestCompatCov_Calls_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_a_call", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Calls.Update("CA404", map[string]any{"Status": "completed"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_a_call")
}

func TestCompatCov_Calls_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Calls.Delete("CA1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result == nil {
		t.Fatal("Delete returned nil")
	}
	assertMatched(t, mock, "DELETE", lamlAccountBase(mock)+"/Calls/CA1", "compatibility.delete_a_call")
}

func TestCompatCov_Calls_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.delete_a_call", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Calls.Delete("CA404")
	assertCompatErr(t, mock, err, 404, "compatibility.delete_a_call")
}

func TestCompatCov_Calls_StartRecording(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Calls.StartRecording("CA1", map[string]any{"RecordingChannels": "dual"})
	if err != nil {
		t.Fatalf("StartRecording: %v", err)
	}
	if result == nil {
		t.Fatal("StartRecording returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Calls/CA1/Recordings", "compatibility.create_recording")
}

func TestCompatCov_Calls_StartRecording_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.create_recording", 422, map[string]any{"error": "invalid"})
	_, err := client.Compat.Calls.StartRecording("CA1", map[string]any{})
	assertCompatErr(t, mock, err, 422, "compatibility.create_recording")
}

func TestCompatCov_Calls_UpdateRecording(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Calls.UpdateRecording("CA1", "RE1", map[string]any{"Status": "paused"})
	if err != nil {
		t.Fatalf("UpdateRecording: %v", err)
	}
	if result == nil {
		t.Fatal("UpdateRecording returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Calls/CA1/Recordings/RE1", "compatibility.update_recording")
}

func TestCompatCov_Calls_UpdateRecording_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_recording", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Calls.UpdateRecording("CA1", "RE404", map[string]any{"Status": "paused"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_recording")
}

func TestCompatCov_Calls_StartStream(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Calls.StartStream("CA1", map[string]any{"Url": "wss://a.b/s"})
	if err != nil {
		t.Fatalf("StartStream: %v", err)
	}
	if result == nil {
		t.Fatal("StartStream returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Calls/CA1/Streams", "compatibility.create_stream")
}

func TestCompatCov_Calls_StartStream_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.create_stream", 422, map[string]any{"error": "invalid"})
	_, err := client.Compat.Calls.StartStream("CA1", map[string]any{})
	assertCompatErr(t, mock, err, 422, "compatibility.create_stream")
}

func TestCompatCov_Calls_StopStream(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Calls.StopStream("CA1", "ST1", map[string]any{"Status": "stopped"})
	if err != nil {
		t.Fatalf("StopStream: %v", err)
	}
	if result == nil {
		t.Fatal("StopStream returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Calls/CA1/Streams/ST1", "compatibility.update_stream")
}

func TestCompatCov_Calls_StopStream_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_stream", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Calls.StopStream("CA1", "ST404", map[string]any{"Status": "stopped"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_stream")
}

// ============================================================================
// Conferences
// ============================================================================

func TestCompatCov_Conferences_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result == nil {
		t.Fatal("List returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Conferences", "compatibility.list_all_conferences")
}

func TestCompatCov_Conferences_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_all_conferences", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.Conferences.List(nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_all_conferences")
}

func TestCompatCov_Conferences_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.Get("CF1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result == nil {
		t.Fatal("Get returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Conferences/CF1", "compatibility.retrieve_conference")
}

func TestCompatCov_Conferences_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.retrieve_conference", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Conferences.Get("CF404")
	assertCompatErr(t, mock, err, 404, "compatibility.retrieve_conference")
}

func TestCompatCov_Conferences_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.Update("CF1", map[string]any{"Status": "completed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result == nil {
		t.Fatal("Update returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Conferences/CF1", "compatibility.update_conference")
}

func TestCompatCov_Conferences_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_conference", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Conferences.Update("CF404", map[string]any{"Status": "completed"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_conference")
}

func TestCompatCov_Conferences_ListParticipants(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.ListParticipants("CF1", nil)
	if err != nil {
		t.Fatalf("ListParticipants: %v", err)
	}
	if result == nil {
		t.Fatal("ListParticipants returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Conferences/CF1/Participants", "compatibility.list_all_participants")
}

func TestCompatCov_Conferences_ListParticipants_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_all_participants", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.Conferences.ListParticipants("CF1", nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_all_participants")
}

func TestCompatCov_Conferences_GetParticipant(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.GetParticipant("CF1", "CA1")
	if err != nil {
		t.Fatalf("GetParticipant: %v", err)
	}
	if result == nil {
		t.Fatal("GetParticipant returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Conferences/CF1/Participants/CA1", "compatibility.retrieve_participant")
}

func TestCompatCov_Conferences_GetParticipant_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.retrieve_participant", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Conferences.GetParticipant("CF1", "CA404")
	assertCompatErr(t, mock, err, 404, "compatibility.retrieve_participant")
}

func TestCompatCov_Conferences_UpdateParticipant(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.UpdateParticipant("CF1", "CA1", map[string]any{"Muted": "true"})
	if err != nil {
		t.Fatalf("UpdateParticipant: %v", err)
	}
	if result == nil {
		t.Fatal("UpdateParticipant returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Conferences/CF1/Participants/CA1", "compatibility.update_participant")
}

func TestCompatCov_Conferences_UpdateParticipant_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_participant", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Conferences.UpdateParticipant("CF1", "CA404", map[string]any{"Muted": "true"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_participant")
}

func TestCompatCov_Conferences_RemoveParticipant(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.RemoveParticipant("CF1", "CA1")
	if err != nil {
		t.Fatalf("RemoveParticipant: %v", err)
	}
	if result == nil {
		t.Fatal("RemoveParticipant returned nil")
	}
	assertMatched(t, mock, "DELETE", lamlAccountBase(mock)+"/Conferences/CF1/Participants/CA1", "compatibility.delete_participant")
}

func TestCompatCov_Conferences_RemoveParticipant_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.delete_participant", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Conferences.RemoveParticipant("CF1", "CA404")
	assertCompatErr(t, mock, err, 404, "compatibility.delete_participant")
}

func TestCompatCov_Conferences_ListRecordings(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.ListRecordings("CF1", nil)
	if err != nil {
		t.Fatalf("ListRecordings: %v", err)
	}
	if result == nil {
		t.Fatal("ListRecordings returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Conferences/CF1/Recordings", "compatibility.list_conference_recordings")
}

func TestCompatCov_Conferences_ListRecordings_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_conference_recordings", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.Conferences.ListRecordings("CF1", nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_conference_recordings")
}

func TestCompatCov_Conferences_GetRecording(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.GetRecording("CF1", "RE1")
	if err != nil {
		t.Fatalf("GetRecording: %v", err)
	}
	if result == nil {
		t.Fatal("GetRecording returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Conferences/CF1/Recordings/RE1", "compatibility.get_conference_recording")
}

func TestCompatCov_Conferences_GetRecording_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.get_conference_recording", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Conferences.GetRecording("CF1", "RE404")
	assertCompatErr(t, mock, err, 404, "compatibility.get_conference_recording")
}

func TestCompatCov_Conferences_UpdateRecording(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.UpdateRecording("CF1", "RE1", map[string]any{"Status": "paused"})
	if err != nil {
		t.Fatalf("UpdateRecording: %v", err)
	}
	if result == nil {
		t.Fatal("UpdateRecording returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Conferences/CF1/Recordings/RE1", "compatibility.update_conference_recording")
}

func TestCompatCov_Conferences_UpdateRecording_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_conference_recording", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Conferences.UpdateRecording("CF1", "RE404", map[string]any{"Status": "paused"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_conference_recording")
}

func TestCompatCov_Conferences_DeleteRecording(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.DeleteRecording("CF1", "RE1")
	if err != nil {
		t.Fatalf("DeleteRecording: %v", err)
	}
	if result == nil {
		t.Fatal("DeleteRecording returned nil")
	}
	assertMatched(t, mock, "DELETE", lamlAccountBase(mock)+"/Conferences/CF1/Recordings/RE1", "compatibility.delete_conference_recording")
}

func TestCompatCov_Conferences_DeleteRecording_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.delete_conference_recording", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Conferences.DeleteRecording("CF1", "RE404")
	assertCompatErr(t, mock, err, 404, "compatibility.delete_conference_recording")
}

func TestCompatCov_Conferences_StartStream(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.StartStream("CF1", map[string]any{"Url": "wss://a.b/s"})
	if err != nil {
		t.Fatalf("StartStream: %v", err)
	}
	if result == nil {
		t.Fatal("StartStream returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Conferences/CF1/Streams", "compatibility.create_conference_stream")
}

func TestCompatCov_Conferences_StartStream_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.create_conference_stream", 422, map[string]any{"error": "invalid"})
	_, err := client.Compat.Conferences.StartStream("CF1", map[string]any{})
	assertCompatErr(t, mock, err, 422, "compatibility.create_conference_stream")
}

func TestCompatCov_Conferences_StopStream(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Conferences.StopStream("CF1", "ST1", map[string]any{"Status": "stopped"})
	if err != nil {
		t.Fatalf("StopStream: %v", err)
	}
	if result == nil {
		t.Fatal("StopStream returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Conferences/CF1/Streams/ST1", "compatibility.update_conference_stream")
}

func TestCompatCov_Conferences_StopStream_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_conference_stream", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Conferences.StopStream("CF1", "ST404", map[string]any{"Status": "stopped"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_conference_stream")
}

// ============================================================================
// Faxes
// ============================================================================

func TestCompatCov_Faxes_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Faxes.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result == nil {
		t.Fatal("List returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Faxes", "compatibility.list_all_faxes")
}

func TestCompatCov_Faxes_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_all_faxes", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.Faxes.List(nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_all_faxes")
}

func TestCompatCov_Faxes_Send(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Faxes.Create(map[string]any{"To": "+15551112222", "MediaUrl": "https://a.b/f.pdf"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result == nil {
		t.Fatal("Create returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Faxes", "compatibility.send_fax")
}

func TestCompatCov_Faxes_Send_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.send_fax", 422, map[string]any{"error": "invalid"})
	_, err := client.Compat.Faxes.Create(map[string]any{"To": "x"})
	assertCompatErr(t, mock, err, 422, "compatibility.send_fax")
}

func TestCompatCov_Faxes_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Faxes.Get("FX1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result == nil {
		t.Fatal("Get returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Faxes/FX1", "compatibility.retrieve_fax")
}

func TestCompatCov_Faxes_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.retrieve_fax", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Faxes.Get("FX404")
	assertCompatErr(t, mock, err, 404, "compatibility.retrieve_fax")
}

func TestCompatCov_Faxes_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Faxes.Update("FX1", map[string]any{"Status": "canceled"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result == nil {
		t.Fatal("Update returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Faxes/FX1", "compatibility.update_fax")
}

func TestCompatCov_Faxes_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_fax", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Faxes.Update("FX404", map[string]any{"Status": "canceled"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_fax")
}

func TestCompatCov_Faxes_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Faxes.Delete("FX1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result == nil {
		t.Fatal("Delete returned nil")
	}
	assertMatched(t, mock, "DELETE", lamlAccountBase(mock)+"/Faxes/FX1", "compatibility.delete_fax")
}

func TestCompatCov_Faxes_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.delete_fax", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Faxes.Delete("FX404")
	assertCompatErr(t, mock, err, 404, "compatibility.delete_fax")
}

func TestCompatCov_Faxes_ListMedia(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Faxes.ListMedia("FX1", nil)
	if err != nil {
		t.Fatalf("ListMedia: %v", err)
	}
	if result == nil {
		t.Fatal("ListMedia returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Faxes/FX1/Media", "compatibility.list_all_fax_media")
}

func TestCompatCov_Faxes_ListMedia_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_all_fax_media", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.Faxes.ListMedia("FX1", nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_all_fax_media")
}

func TestCompatCov_Faxes_GetMedia(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Faxes.GetMedia("FX1", "ME1")
	if err != nil {
		t.Fatalf("GetMedia: %v", err)
	}
	if result == nil {
		t.Fatal("GetMedia returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Faxes/FX1/Media/ME1", "compatibility.retrieve_medias")
}

func TestCompatCov_Faxes_GetMedia_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.retrieve_medias", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Faxes.GetMedia("FX1", "ME404")
	assertCompatErr(t, mock, err, 404, "compatibility.retrieve_medias")
}

func TestCompatCov_Faxes_DeleteMedia(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Faxes.DeleteMedia("FX1", "ME1")
	if err != nil {
		t.Fatalf("DeleteMedia: %v", err)
	}
	if result == nil {
		t.Fatal("DeleteMedia returned nil")
	}
	assertMatched(t, mock, "DELETE", lamlAccountBase(mock)+"/Faxes/FX1/Media/ME1", "compatibility.delete_fax_media")
}

func TestCompatCov_Faxes_DeleteMedia_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.delete_fax_media", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Faxes.DeleteMedia("FX1", "ME404")
	assertCompatErr(t, mock, err, 404, "compatibility.delete_fax_media")
}

// ============================================================================
// Incoming phone numbers
// ============================================================================

func TestCompatCov_IncomingNumbers_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.PhoneNumbers.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result == nil {
		t.Fatal("List returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/IncomingPhoneNumbers", "compatibility.list_incoming_phone_numbers")
}

func TestCompatCov_IncomingNumbers_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_incoming_phone_numbers", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.PhoneNumbers.List(nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_incoming_phone_numbers")
}

func TestCompatCov_IncomingNumbers_Purchase(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.PhoneNumbers.Purchase(map[string]any{"PhoneNumber": "+15555550100"})
	if err != nil {
		t.Fatalf("Purchase: %v", err)
	}
	if result == nil {
		t.Fatal("Purchase returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/IncomingPhoneNumbers", "compatibility.create_incoming_phone_number")
}

func TestCompatCov_IncomingNumbers_Purchase_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.create_incoming_phone_number", 422, map[string]any{"error": "invalid"})
	_, err := client.Compat.PhoneNumbers.Purchase(map[string]any{"PhoneNumber": "x"})
	assertCompatErr(t, mock, err, 422, "compatibility.create_incoming_phone_number")
}

func TestCompatCov_IncomingNumbers_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.PhoneNumbers.Get("PN1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result == nil {
		t.Fatal("Get returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/IncomingPhoneNumbers/PN1", "compatibility.retrieve_incoming_phone_number")
}

func TestCompatCov_IncomingNumbers_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.retrieve_incoming_phone_number", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.PhoneNumbers.Get("PN404")
	assertCompatErr(t, mock, err, 404, "compatibility.retrieve_incoming_phone_number")
}

func TestCompatCov_IncomingNumbers_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.PhoneNumbers.Update("PN1", map[string]any{"FriendlyName": "x"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result == nil {
		t.Fatal("Update returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/IncomingPhoneNumbers/PN1", "compatibility.update_incoming_phone_number")
}

func TestCompatCov_IncomingNumbers_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_incoming_phone_number", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.PhoneNumbers.Update("PN404", map[string]any{"FriendlyName": "x"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_incoming_phone_number")
}

func TestCompatCov_IncomingNumbers_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.PhoneNumbers.Delete("PN1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result == nil {
		t.Fatal("Delete returned nil")
	}
	assertMatched(t, mock, "DELETE", lamlAccountBase(mock)+"/IncomingPhoneNumbers/PN1", "compatibility.delete_incoming_phone_number")
}

func TestCompatCov_IncomingNumbers_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.delete_incoming_phone_number", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.PhoneNumbers.Delete("PN404")
	assertCompatErr(t, mock, err, 404, "compatibility.delete_incoming_phone_number")
}

func TestCompatCov_IncomingNumbers_Import(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.PhoneNumbers.ImportNumber(map[string]any{"PhoneNumber": "+15555550111"})
	if err != nil {
		t.Fatalf("ImportNumber: %v", err)
	}
	if result == nil {
		t.Fatal("ImportNumber returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/ImportedPhoneNumbers", "compatibility.create_imported_phone_number")
}

func TestCompatCov_IncomingNumbers_Import_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.create_imported_phone_number", 422, map[string]any{"error": "invalid"})
	_, err := client.Compat.PhoneNumbers.ImportNumber(map[string]any{"PhoneNumber": "x"})
	assertCompatErr(t, mock, err, 422, "compatibility.create_imported_phone_number")
}

// ============================================================================
// LamlBins (cXML scripts)
// ============================================================================

func TestCompatCov_LamlBins_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.LamlBins.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result == nil {
		t.Fatal("List returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/LamlBins", "compatibility.list_cxml_scripts")
}

func TestCompatCov_LamlBins_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_cxml_scripts", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.LamlBins.List(nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_cxml_scripts")
}

func TestCompatCov_LamlBins_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.LamlBins.Create(map[string]any{"Name": "bin", "Contents": "<Response/>"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result == nil {
		t.Fatal("Create returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/LamlBins", "compatibility.create_cxml_script")
}

func TestCompatCov_LamlBins_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.create_cxml_script", 422, map[string]any{"error": "invalid"})
	_, err := client.Compat.LamlBins.Create(map[string]any{"Name": "x"})
	assertCompatErr(t, mock, err, 422, "compatibility.create_cxml_script")
}

func TestCompatCov_LamlBins_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.LamlBins.Get("LB1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result == nil {
		t.Fatal("Get returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/LamlBins/LB1", "compatibility.retrieve_cxml_script")
}

func TestCompatCov_LamlBins_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.retrieve_cxml_script", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.LamlBins.Get("LB404")
	assertCompatErr(t, mock, err, 404, "compatibility.retrieve_cxml_script")
}

func TestCompatCov_LamlBins_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.LamlBins.Update("LB1", map[string]any{"Name": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result == nil {
		t.Fatal("Update returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/LamlBins/LB1", "compatibility.update_cxml_script")
}

func TestCompatCov_LamlBins_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_cxml_script", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.LamlBins.Update("LB404", map[string]any{"Name": "x"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_cxml_script")
}

func TestCompatCov_LamlBins_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.LamlBins.Delete("LB1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result == nil {
		t.Fatal("Delete returned nil")
	}
	assertMatched(t, mock, "DELETE", lamlAccountBase(mock)+"/LamlBins/LB1", "compatibility.delete_cxml_script")
}

func TestCompatCov_LamlBins_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.delete_cxml_script", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.LamlBins.Delete("LB404")
	assertCompatErr(t, mock, err, 404, "compatibility.delete_cxml_script")
}

// ============================================================================
// Messages
// ============================================================================

func TestCompatCov_Messages_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Messages.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result == nil {
		t.Fatal("List returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Messages", "compatibility.list_messages")
}

func TestCompatCov_Messages_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_messages", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.Messages.List(nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_messages")
}

func TestCompatCov_Messages_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Messages.Create(map[string]any{"To": "+15551112222", "From": "+15553334444", "Body": "hi"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result == nil {
		t.Fatal("Create returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Messages", "compatibility.create_message")
}

func TestCompatCov_Messages_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.create_message", 422, map[string]any{"error": "invalid"})
	_, err := client.Compat.Messages.Create(map[string]any{"Body": "x"})
	assertCompatErr(t, mock, err, 422, "compatibility.create_message")
}

func TestCompatCov_Messages_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Messages.Get("MM1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result == nil {
		t.Fatal("Get returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Messages/MM1", "compatibility.retrieve_message")
}

func TestCompatCov_Messages_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.retrieve_message", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Messages.Get("MM404")
	assertCompatErr(t, mock, err, 404, "compatibility.retrieve_message")
}

func TestCompatCov_Messages_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Messages.Update("MM1", map[string]any{"Body": "edited"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result == nil {
		t.Fatal("Update returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Messages/MM1", "compatibility.update_message")
}

func TestCompatCov_Messages_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_message", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Messages.Update("MM404", map[string]any{"Body": "x"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_message")
}

func TestCompatCov_Messages_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Messages.Delete("MM1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result == nil {
		t.Fatal("Delete returned nil")
	}
	assertMatched(t, mock, "DELETE", lamlAccountBase(mock)+"/Messages/MM1", "compatibility.delete_message")
}

func TestCompatCov_Messages_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.delete_message", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Messages.Delete("MM404")
	assertCompatErr(t, mock, err, 404, "compatibility.delete_message")
}

func TestCompatCov_Messages_ListMedia(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Messages.ListMedia("MM1", nil)
	if err != nil {
		t.Fatalf("ListMedia: %v", err)
	}
	if result == nil {
		t.Fatal("ListMedia returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Messages/MM1/Media", "compatibility.list_media")
}

func TestCompatCov_Messages_ListMedia_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_media", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.Messages.ListMedia("MM1", nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_media")
}

func TestCompatCov_Messages_GetMedia(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Messages.GetMedia("MM1", "ME1")
	if err != nil {
		t.Fatalf("GetMedia: %v", err)
	}
	if result == nil {
		t.Fatal("GetMedia returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Messages/MM1/Media/ME1", "compatibility.retrieve_media")
}

func TestCompatCov_Messages_GetMedia_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.retrieve_media", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Messages.GetMedia("MM1", "ME404")
	assertCompatErr(t, mock, err, 404, "compatibility.retrieve_media")
}

func TestCompatCov_Messages_DeleteMedia(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Messages.DeleteMedia("MM1", "ME1")
	if err != nil {
		t.Fatalf("DeleteMedia: %v", err)
	}
	if result == nil {
		t.Fatal("DeleteMedia returned nil")
	}
	assertMatched(t, mock, "DELETE", lamlAccountBase(mock)+"/Messages/MM1/Media/ME1", "compatibility.delete_message_media")
}

func TestCompatCov_Messages_DeleteMedia_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.delete_message_media", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Messages.DeleteMedia("MM1", "ME404")
	assertCompatErr(t, mock, err, 404, "compatibility.delete_message_media")
}

// ============================================================================
// Queues
// ============================================================================

func TestCompatCov_Queues_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Queues.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result == nil {
		t.Fatal("List returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Queues", "compatibility.list_queues")
}

func TestCompatCov_Queues_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_queues", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.Queues.List(nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_queues")
}

func TestCompatCov_Queues_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Queues.Create(map[string]any{"FriendlyName": "Q"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result == nil {
		t.Fatal("Create returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Queues", "compatibility.create_queue")
}

func TestCompatCov_Queues_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.create_queue", 422, map[string]any{"error": "invalid"})
	_, err := client.Compat.Queues.Create(map[string]any{"FriendlyName": "Q"})
	assertCompatErr(t, mock, err, 422, "compatibility.create_queue")
}

func TestCompatCov_Queues_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Queues.Get("QU1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result == nil {
		t.Fatal("Get returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Queues/QU1", "compatibility.retrieve_queue")
}

func TestCompatCov_Queues_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.retrieve_queue", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Queues.Get("QU404")
	assertCompatErr(t, mock, err, 404, "compatibility.retrieve_queue")
}

func TestCompatCov_Queues_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Queues.Update("QU1", map[string]any{"FriendlyName": "renamed"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result == nil {
		t.Fatal("Update returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Queues/QU1", "compatibility.update_queue")
}

func TestCompatCov_Queues_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_queue", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Queues.Update("QU404", map[string]any{"FriendlyName": "x"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_queue")
}

func TestCompatCov_Queues_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Queues.Delete("QU1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result == nil {
		t.Fatal("Delete returned nil")
	}
	assertMatched(t, mock, "DELETE", lamlAccountBase(mock)+"/Queues/QU1", "compatibility.delete_queue")
}

func TestCompatCov_Queues_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.delete_queue", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Queues.Delete("QU404")
	assertCompatErr(t, mock, err, 404, "compatibility.delete_queue")
}

func TestCompatCov_Queues_ListMembers(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Queues.ListMembers("QU1", nil)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if result == nil {
		t.Fatal("ListMembers returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Queues/QU1/Members", "compatibility.list_all_queue_members")
}

func TestCompatCov_Queues_ListMembers_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_all_queue_members", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.Queues.ListMembers("QU1", nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_all_queue_members")
}

func TestCompatCov_Queues_GetMember(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Queues.GetMember("QU1", "CA1")
	if err != nil {
		t.Fatalf("GetMember: %v", err)
	}
	if result == nil {
		t.Fatal("GetMember returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Queues/QU1/Members/CA1", "compatibility.retrieve_queue_member")
}

func TestCompatCov_Queues_GetMember_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.retrieve_queue_member", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Queues.GetMember("QU1", "CA404")
	assertCompatErr(t, mock, err, 404, "compatibility.retrieve_queue_member")
}

func TestCompatCov_Queues_DequeueMember(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Queues.DequeueMember("QU1", "CA1", map[string]any{"Url": "https://a.b/d"})
	if err != nil {
		t.Fatalf("DequeueMember: %v", err)
	}
	if result == nil {
		t.Fatal("DequeueMember returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/Queues/QU1/Members/CA1", "compatibility.update_queue_member")
}

func TestCompatCov_Queues_DequeueMember_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_queue_member", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Queues.DequeueMember("QU1", "CA404", map[string]any{"Url": "https://a.b/d"})
	assertCompatErr(t, mock, err, 404, "compatibility.update_queue_member")
}

// ============================================================================
// Recordings
// ============================================================================

func TestCompatCov_Recordings_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Recordings.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result == nil {
		t.Fatal("List returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Recordings", "compatibility.list_recordings")
}

func TestCompatCov_Recordings_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_recordings", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.Recordings.List(nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_recordings")
}

func TestCompatCov_Recordings_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Recordings.Get("RE1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result == nil {
		t.Fatal("Get returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Recordings/RE1", "compatibility.retrieve_recording")
}

func TestCompatCov_Recordings_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.retrieve_recording", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Recordings.Get("RE404")
	assertCompatErr(t, mock, err, 404, "compatibility.retrieve_recording")
}

func TestCompatCov_Recordings_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Recordings.Delete("RE1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result == nil {
		t.Fatal("Delete returned nil")
	}
	assertMatched(t, mock, "DELETE", lamlAccountBase(mock)+"/Recordings/RE1", "compatibility.delete_recording")
}

func TestCompatCov_Recordings_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.delete_recording", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Recordings.Delete("RE404")
	assertCompatErr(t, mock, err, 404, "compatibility.delete_recording")
}

// ============================================================================
// Transcriptions
// ============================================================================

func TestCompatCov_Transcriptions_List(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Transcriptions.List(nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if result == nil {
		t.Fatal("List returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Transcriptions", "compatibility.list_transcriptions")
}

func TestCompatCov_Transcriptions_List_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.list_transcriptions", 500, map[string]any{"error": "boom"})
	_, err := client.Compat.Transcriptions.List(nil)
	assertCompatErr(t, mock, err, 500, "compatibility.list_transcriptions")
}

func TestCompatCov_Transcriptions_Get(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Transcriptions.Get("TR1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if result == nil {
		t.Fatal("Get returned nil")
	}
	assertMatched(t, mock, "GET", lamlAccountBase(mock)+"/Transcriptions/TR1", "compatibility.retrieve_transcription")
}

func TestCompatCov_Transcriptions_Get_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.retrieve_transcription", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Transcriptions.Get("TR404")
	assertCompatErr(t, mock, err, 404, "compatibility.retrieve_transcription")
}

func TestCompatCov_Transcriptions_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Transcriptions.Delete("TR1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result == nil {
		t.Fatal("Delete returned nil")
	}
	assertMatched(t, mock, "DELETE", lamlAccountBase(mock)+"/Transcriptions/TR1", "compatibility.delete_transcription")
}

func TestCompatCov_Transcriptions_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.delete_transcription", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Transcriptions.Delete("TR404")
	assertCompatErr(t, mock, err, 404, "compatibility.delete_transcription")
}

// ============================================================================
// Tokens
// ============================================================================

func TestCompatCov_Tokens_Create(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Tokens.Create(map[string]any{"Ttl": 3600})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result == nil {
		t.Fatal("Create returned nil")
	}
	assertMatched(t, mock, "POST", lamlAccountBase(mock)+"/tokens", "compatibility.create_token")
}

func TestCompatCov_Tokens_Create_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.create_token", 422, map[string]any{"error": "invalid"})
	_, err := client.Compat.Tokens.Create(map[string]any{"Ttl": -1})
	assertCompatErr(t, mock, err, 422, "compatibility.create_token")
}

func TestCompatCov_Tokens_Update(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Tokens.Update("TK1", map[string]any{"Ttl": 7200})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result == nil {
		t.Fatal("Update returned nil")
	}
	assertMatched(t, mock, "PATCH", lamlAccountBase(mock)+"/tokens/TK1", "compatibility.update_token")
}

func TestCompatCov_Tokens_Update_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.update_token", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Tokens.Update("TK404", map[string]any{"Ttl": 1})
	assertCompatErr(t, mock, err, 404, "compatibility.update_token")
}

func TestCompatCov_Tokens_Delete(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)

	result, err := client.Compat.Tokens.Delete("TK1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result == nil {
		t.Fatal("Delete returned nil")
	}
	assertMatched(t, mock, "DELETE", lamlAccountBase(mock)+"/tokens/TK1", "compatibility.delete_token")
}

func TestCompatCov_Tokens_Delete_Error(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}
	mock.Reset(t)
	mock.PushScenario(t, "compatibility.delete_token", 404, map[string]any{"error": "not found"})
	_, err := client.Compat.Tokens.Delete("TK404")
	assertCompatErr(t, mock, err, 404, "compatibility.delete_token")
}
