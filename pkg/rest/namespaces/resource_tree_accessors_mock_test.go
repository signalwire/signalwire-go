// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire AI Agents SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

// Behavioural proof that every namespace accessor the reference exposes on
// signalwire.rest.client.RestClient is reachable in Go through the embedded
// _GeneratedResourceTree — client.Calling, client.Fabric, client.Video, ….
//
// Go promotes an embedded struct's fields, so the 22 tree fields resolve on
// *RestClient exactly as the reference's 22 properties resolve on its client.
// A source-level enumerator that does not follow the embed sees zero of them
// and reports 22 phantom "missing-port" drifts; this test is the runtime
// regression guard that keeps the accessors honest independent of whatever
// the enumerator can or cannot see.

package namespaces_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/v3/pkg/rest/internal/mocktest"
	"github.com/signalwire/signalwire-go/v3/pkg/rest/namespaces"
)

// isNilPointer reports whether v holds a nil pointer. A typed-nil pointer boxed
// into an `any` compares non-nil against the untyped nil, so the accessor check
// needs this to catch a declared-but-unwired namespace field.
func isNilPointer(v any) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

// TestResourceTreeAccessors_AllPromotedAndWired asserts that all 22 namespace
// accessors the reference declares on RestClient are reachable on the Go client
// through the embed AND are non-nil (i.e. wireGeneratedTree ran for each).
//
// A nil field here means the tree declared a namespace the constructor never
// wired — a caller would nil-panic on first use.
func TestResourceTreeAccessors_AllPromotedAndWired(t *testing.T) {
	t.Parallel()
	client, _ := mocktest.New(t)
	if client == nil {
		return
	}

	// Each entry is (reference property name, the promoted Go field value).
	// Referencing client.X here IS the compile-time proof that the embed
	// promotes the field to the client's own selector namespace.
	accessors := []struct {
		reference string
		value     any
	}{
		{"addresses", client.Addresses},
		{"calling", client.Calling},
		{"chat", client.Chat},
		{"datasphere", client.Datasphere},
		{"fabric", client.Fabric},
		{"imported_numbers", client.ImportedNumbers},
		{"logs", client.Logs},
		{"lookup", client.Lookup},
		{"messages", client.Messages},
		{"mfa", client.MFA},
		{"number_groups", client.NumberGroups},
		{"phone_numbers", client.PhoneNumbers},
		{"project", client.Project},
		{"projects", client.Projects},
		{"pubsub", client.PubSub},
		{"queues", client.Queues},
		{"recordings", client.Recordings},
		{"registry", client.Registry},
		{"short_codes", client.ShortCodes},
		{"sip_profile", client.SIPProfile},
		{"verified_callers", client.VerifiedCallers},
		{"video", client.Video},
	}

	if len(accessors) != 22 {
		t.Fatalf("expected 22 reference accessors, listed %d", len(accessors))
	}

	for _, a := range accessors {
		if a.value == nil {
			t.Errorf("client.%s (reference %q) is nil — declared on the tree but never wired", a.reference, a.reference)
			continue
		}
		// A typed-nil pointer stuffed into an `any` is non-nil as an interface,
		// so check the concrete pointer too.
		if isNilPointer(a.value) {
			t.Errorf("client.%s (reference %q) is a nil pointer — wireGeneratedTree did not construct it", a.reference, a.reference)
		}
	}
}

// TestResourceTreeAccessors_RequestsLandThroughAccessorPath drives real requests
// through three of the promoted accessors and asserts each one lands on the mock
// with the right method and path. This is the behavioural half of the proof: the
// accessors are not merely non-nil, they carry the HTTP client down to a resource
// that actually issues the request.
func TestResourceTreeAccessors_RequestsLandThroughAccessorPath(t *testing.T) {
	t.Parallel()
	client, mock := mocktest.New(t)
	if client == nil {
		return
	}

	cases := []struct {
		name       string
		reference  string
		call       func() error
		wantMethod string
		wantPath   string
	}{
		{
			name:      "fabric",
			reference: "client.fabric.addresses.list()",
			call: func() error {
				_, err := client.Fabric.Addresses.List(context.Background(), nil)
				return err
			},
			wantMethod: "GET",
			wantPath:   "/api/fabric/addresses",
		},
		{
			name:      "video",
			reference: "client.video.rooms.list()",
			call: func() error {
				_, err := client.Video.Rooms.List(context.Background(), nil)
				return err
			},
			wantMethod: "GET",
			wantPath:   "/api/video/rooms",
		},
		{
			name:      "calling",
			reference: "client.calling.dial(...)",
			call: func() error {
				_, err := client.Calling.Dial(context.Background(), namespaces.CallingNamespaceDialParams{
					Extras: map[string]any{"from": "+15551110000", "to": "+15552220000"},
				})
				return err
			},
			wantMethod: "POST",
			wantPath:   "/api/calling/calls",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock.Reset(t)

			if err := tc.call(); err != nil {
				t.Fatalf("%s through the promoted accessor: %v", tc.reference, err)
			}

			j := mock.Last(t)
			if j.Method != tc.wantMethod {
				t.Errorf("%s: method = %q, want %q", tc.reference, j.Method, tc.wantMethod)
			}
			if !strings.HasPrefix(j.Path, tc.wantPath) {
				t.Errorf("%s: path = %q, want prefix %q", tc.reference, j.Path, tc.wantPath)
			}
		})
	}
}
