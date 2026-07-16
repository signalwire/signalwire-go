// Copyright (c) 2025 SignalWire
//
// Tests for the typed relay.Device struct (Tier-3 idiom addition). The Device
// types the {type, params} shape that recurs across connect/refer/dial/tap;
// these tests prove (a) ToMap()/MarshalJSON are BYTE-IDENTICAL to the
// equivalent hand-written map, and (b) a Device round-trips through a real
// calling.dial and calling.connect frame against the shared mock_relay (no
// mocks of the transport).

package relay_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/relay"
	"github.com/signalwire/signalwire-go/v3/pkg/relay/internal/mocktest"
)

// TestDevice_ToMapByteIdenticalToHandWritten proves Device.ToMap() yields the
// exact same map a caller would hand-write for the raw-map device path.
func TestDevice_ToMapByteIdenticalToHandWritten(t *testing.T) {
	dev := relay.NewDevice("phone", map[string]any{
		"to_number":   "+15551112222",
		"from_number": "+15553334444",
	})

	got := dev.ToMap()
	want := map[string]any{
		"type": "phone",
		"params": map[string]any{
			"to_number":   "+15551112222",
			"from_number": "+15553334444",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ToMap() = %#v, want %#v", got, want)
	}

	// And the JSON bytes must match the hand-written map's JSON bytes exactly.
	gotJSON, err := json.Marshal(dev)
	if err != nil {
		t.Fatalf("Marshal(Device): %v", err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal(want): %v", err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("Device JSON = %s, want %s", gotJSON, wantJSON)
	}
}

// TestDevice_NilParamsBecomesEmptyObject proves a Device with nil Params emits
// the canonical "params": {} (matching the hand-written {"type":..,"params":
// map[string]any{}} shape used throughout the suite), not a null/absent params.
func TestDevice_NilParamsBecomesEmptyObject(t *testing.T) {
	dev := relay.Device{Type: "phone"} // Params nil

	want := map[string]any{"type": "phone", "params": map[string]any{}}
	if got := dev.ToMap(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ToMap() = %#v, want %#v", got, want)
	}

	gotJSON, err := json.Marshal(dev)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("JSON = %s, want %s", gotJSON, wantJSON)
	}
}

// TestDevice_ListBuildersMatchHandWrittenNesting proves DeviceList and
// DeviceGroups produce exactly the [][]map[string]any nesting that Connect/Dial
// take, byte-identical to building the nested slices by hand.
func TestDevice_ListBuildersMatchHandWrittenNesting(t *testing.T) {
	phone := relay.NewDevice("phone", map[string]any{"to_number": "+1999"})
	sip := relay.NewDevice("sip", map[string]any{"to": "sip:x@y"})

	// DeviceList = one parallel leg.
	gotList := relay.DeviceList(phone, sip)
	wantList := [][]map[string]any{{
		{"type": "phone", "params": map[string]any{"to_number": "+1999"}},
		{"type": "sip", "params": map[string]any{"to": "sip:x@y"}},
	}}
	if !reflect.DeepEqual(gotList, wantList) {
		t.Errorf("DeviceList = %#v, want %#v", gotList, wantList)
	}

	// DeviceGroups = serial groups of parallel legs.
	gotGroups := relay.DeviceGroups([]relay.Device{phone}, []relay.Device{sip})
	wantGroups := [][]map[string]any{
		{{"type": "phone", "params": map[string]any{"to_number": "+1999"}}},
		{{"type": "sip", "params": map[string]any{"to": "sip:x@y"}}},
	}
	if !reflect.DeepEqual(gotGroups, wantGroups) {
		t.Errorf("DeviceGroups = %#v, want %#v", gotGroups, wantGroups)
	}
}

// TestDevice_RoundTripsThroughRealDialFrame drives a REAL calling.dial through
// the shared mock_relay using a typed Device, and asserts the device that lands
// on the wire is byte-identical to the equivalent hand-written device map.
func TestDevice_RoundTripsThroughRealDialFrame(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}

	handWritten := map[string]any{
		"type":   "phone",
		"params": map[string]any{"to_number": "+15551112222", "from_number": "+15553334444"},
	}
	typed := relay.NewDevice("phone", map[string]any{
		"to_number":   "+15551112222",
		"from_number": "+15553334444",
	})

	h.ArmDial(t, mocktest.DialOpts{
		Tag:          "t-device",
		WinnerCallID: "winner-device",
		States:       []string{"created", "answered"},
		NodeID:       "node-mock-1",
		Device:       handWritten,
	})

	// Dial with the TYPED device via the DeviceList builder.
	_, err := client.Dial(
		relay.DeviceList(typed),
		relay.WithDialTag("t-device"),
		relay.WithDialClientTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	entry := h.JournalLast(t, "calling.dial")
	params, _ := entry.FrameParams()
	devices, ok := params["devices"].([]any)
	if !ok || len(devices) == 0 {
		t.Fatalf("devices missing/empty: %#v", params["devices"])
	}
	leg, ok := devices[0].([]any)
	if !ok || len(leg) == 0 {
		t.Fatalf("first leg empty: %#v", devices[0])
	}
	gotDev, ok := leg[0].(map[string]any)
	if !ok {
		t.Fatalf("device[0] not an object: %#v", leg[0])
	}

	// The device on the wire must equal the hand-written device, JSON-for-JSON
	// (compare via JSON because the wire round-trip normalises numeric/string
	// representations identically for both sides).
	gotJSON, err := json.Marshal(gotDev)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	wantJSON, err := json.Marshal(handWritten)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("wire device = %s, want %s", gotJSON, wantJSON)
	}
}

// TestDevice_RoundTripsThroughRealConnectFrame drives a REAL calling.connect
// (Call.Connect bridge) through the mock using a typed Device and asserts the
// device on the wire is byte-identical to the hand-written equivalent.
func TestDevice_RoundTripsThroughRealConnectFrame(t *testing.T) {
	client, h := mocktest.New(t)
	if client == nil {
		return
	}
	call := answeredInboundCall(t, client, h, "ec-dev-connect")

	handWritten := map[string]any{
		"type":   "phone",
		"params": map[string]any{"to_number": "+15551112222"},
	}
	typed := relay.NewDevice("phone", map[string]any{"to_number": "+15551112222"})

	// Fire the bridge with the typed device. The mock journals the frame
	// regardless of whether it auto-replies; we assert on the journal.
	go func() { _ = call.Connect(relay.DeviceList(typed)) }()

	if !waitFor(5*time.Second, func() bool {
		return len(h.JournalRecv(t, "calling.connect")) > 0
	}) {
		t.Fatal("no calling.connect frame journaled")
	}
	entry := h.JournalLast(t, "calling.connect")

	params, _ := entry.FrameParams()
	devices, ok := params["devices"].([]any)
	if !ok || len(devices) == 0 {
		t.Fatalf("devices missing/empty: %#v", params["devices"])
	}
	leg, ok := devices[0].([]any)
	if !ok || len(leg) == 0 {
		t.Fatalf("first leg empty: %#v", devices[0])
	}
	gotDev, ok := leg[0].(map[string]any)
	if !ok {
		t.Fatalf("device[0] not an object: %#v", leg[0])
	}
	gotJSON, err := json.Marshal(gotDev)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	wantJSON, err := json.Marshal(handWritten)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("wire device = %s, want %s", gotJSON, wantJSON)
	}
}
