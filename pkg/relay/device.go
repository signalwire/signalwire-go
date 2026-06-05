package relay

import "encoding/json"

// Device is a typed view of the {type, params} object that RELAY passes
// across connect / refer / dial / tap. It types the SHAPE only — the
// discriminant `type` stays a string because the set of device types is NOT
// schema-enumerated (the wire schema declares `type` as a free string;
// grounding: porting-sdk/relay-protocol/calling.{connect,dial,refer,tap}.
// params.json, where each device is {required:["type"], properties:{type:
// {type:string}, params:{}}}).
//
// Device is purely ADDITIVE: the raw-map path is unchanged — Call.Connect,
// Client.Dial/DialContext, Call.Refer and the tap helpers all still take
// map[string]any / [][]map[string]any. ToMap() (and json.Marshal via
// MarshalJSON) yield the IDENTICAL wire shape — {"type": <type>, "params":
// <params>} — so a Device can be dropped in anywhere a hand-written device map
// is used, byte-for-byte. Params is left `any` to carry whatever the specific
// device type expects (e.g. phone: {to_number, from_number}; sip: {to, from,
// headers}); when nil it serialises as an empty object to match the canonical
// hand-written `"params": map[string]any{}`.
type Device struct {
	// Type is the device discriminant (e.g. "phone", "sip"). Kept a string —
	// the value set is open / not schema-enumerated.
	Type string
	// Params is the device-type-specific parameter object. Typically a
	// map[string]any; left `any` so callers pass the shape the type needs.
	Params any
}

// NewDevice constructs a Device from a type discriminant and its params.
func NewDevice(deviceType string, params any) Device {
	return Device{Type: deviceType, Params: params}
}

// paramsOrEmpty returns d.Params, or an empty map[string]any when nil, so the
// emitted shape matches the canonical hand-written device map (which always
// carries a "params" object, e.g. {"type":"phone","params":{...}}).
func (d Device) paramsOrEmpty() any {
	if d.Params == nil {
		return map[string]any{}
	}
	return d.Params
}

// ToMap renders the Device as the raw wire map {"type":…, "params":…},
// byte-identical to the hand-written map[string]any callers pass to
// Connect/Dial/Refer/tap. A nil Params becomes an empty object.
func (d Device) ToMap() map[string]any {
	return map[string]any{
		"type":   d.Type,
		"params": d.paramsOrEmpty(),
	}
}

// MarshalJSON makes a Device serialise identically to its ToMap() form, so a
// Device used directly inside a params payload produces the same bytes as the
// equivalent raw map.
func (d Device) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.ToMap())
}

// DeviceList converts a flat list of typed Devices into ONE parallel device
// leg in the [][]map[string]any shape that Connect/Dial expect — i.e. all the
// given devices are tried in parallel (a single serial group). Equivalent to
// the hand-written [][]map[string]any{{dev1, dev2, …}}.
func DeviceList(devices ...Device) [][]map[string]any {
	leg := make([]map[string]any, len(devices))
	for i, d := range devices {
		leg[i] = d.ToMap()
	}
	return [][]map[string]any{leg}
}

// DeviceGroups converts serial groups of parallel typed Devices into the
// [][]map[string]any shape Connect/Dial expect: the outer slice is tried
// serially, the inner slice in parallel — the same semantics as the raw
// [][]map[string]any field. Byte-identical to building the nested maps by hand.
func DeviceGroups(groups ...[]Device) [][]map[string]any {
	out := make([][]map[string]any, len(groups))
	for i, group := range groups {
		leg := make([]map[string]any, len(group))
		for j, d := range group {
			leg[j] = d.ToMap()
		}
		out[i] = leg
	}
	return out
}
