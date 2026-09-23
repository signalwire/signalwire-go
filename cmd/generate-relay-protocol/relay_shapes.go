// relay_shapes.go — the Go port of porting-sdk's scripts/relay_protocol_shapes.py
// (ledger row R11).
//
// The ten port generators used to consume porting-sdk/relay-protocol/ — a directory
// of standalone JSON-Schema files named <method>.<params|result>.json. That tree is
// being retired in favour of the single combined document combined-specs/relay.yaml,
// which carries the same shapes as
//
//	methods.<name>.request.params_dto       (58 methods)
//	methods.<name>.response.result          (58 methods)
//	param_shapes_unattached.methods.<name>  (6 methods — extracted, not registered)
//	result_shapes_unattached.methods.<name> (6 methods)
//
// …for 64 methods per phase either way.
//
// This is deliberately NOT a directory emulator. It does not synthesise
// <method>.<phase>.json filenames, does not re-derive the phase by splitting a
// suffix, and does not reinstate the per-file envelope keys ($schema, title,
// description, type, x-method, x-phase, additionalProperties). Those keys existed
// only because the legacy tree stored each shape as a standalone schema document
// that had to re-declare its own identity; in the combined document the identity IS
// the position in the tree, so re-synthesising them would rebuild the retired shape
// inside the adapter and leave a second, driftable spelling of every method name.
//
// It exposes the one thing the generator needs: relayShapes(psdk, phase) returning
// the (method, node) pairs in deterministic method-name order.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// combinedRelay is where the combined RELAY document lives, relative to the
// porting-sdk root.
const combinedRelay = "combined-specs/relay.yaml"

// relayShape is one method's carried schema node for a phase.
type relayShape struct {
	method string
	node   *yaml.Node
}

// phaseKeys maps a phase to (containing block, shape key, unattached top-level block).
//
// These two phases are the ONLY phases this reader serves. The legacy tree's
// .event.json files were a third phase (x-phase: event) that no RELAY-protocol
// generator ever emitted — every one filters to these two — so they are out of scope
// here exactly as they were out of scope there.
var phaseKeys = map[string][3]string{
	"params": {"request", "params_dto", "param_shapes_unattached"},
	"result": {"response", "result", "result_shapes_unattached"},
}

// relayDoc caches the parsed combined document per porting-sdk root.
var relayDocCache = map[string]*yaml.Node{}

// loadRelayDoc parses combined-specs/relay.yaml once per porting-sdk root.
//
// Fails LOUD on a missing or malformed document: a generator that silently produced
// zero types because its input vanished would look like a successful surface
// deletion, which is the failure mode this whole row exists to prevent.
func loadRelayDoc(psdk string) (*yaml.Node, error) {
	key, err := filepath.Abs(psdk)
	if err != nil {
		return nil, err
	}
	if cached, ok := relayDocCache[key]; ok {
		return cached, nil
	}
	path := filepath.Join(key, combinedRelay)
	raw, err := os.ReadFile(path) //nolint:gosec // G304: path composed from the resolved porting-sdk root in a developer-run tool, not untrusted input.
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("relay_shapes: %s not found (need a porting-sdk checkout carrying the combined RELAY document)", path)
		}
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("relay_shapes: %s: %w", path, err)
	}
	root := rootOf(&doc)
	if root == nil || root.Kind != yaml.MappingNode || mapChild(root, "methods") == nil {
		return nil, fmt.Errorf("relay_shapes: %s has no `methods` mapping — refusing to emit an empty RELAY surface from a malformed spec", path)
	}
	relayDocCache[key] = root
	return root, nil
}

// mappingPairs walks a YAML mapping node as (key, value) pairs.
func mappingPairs(node *yaml.Node) [][2]*yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	out := make([][2]*yaml.Node, 0, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		out = append(out, [2]*yaml.Node{node.Content[i], node.Content[i+1]})
	}
	return out
}

// relayShapes returns method -> carried schema node for phase, ordered by method name.
//
// Both sources of a shape are merged, attached first:
//   - methods.<name>.request.params_dto / .response.result — the shape attached to a
//     method the vendored spec registers;
//   - <phase>_shapes_unattached.methods.<name> — a shape the extractor found whose
//     method the vendored spec does NOT register. Carried rather than dropped,
//     because dropping them would silently shrink the port surface relative to the
//     legacy tree, which had no such distinction.
func relayShapes(psdk, phase string) ([]relayShape, error) {
	keys, ok := phaseKeys[phase]
	if !ok {
		return nil, fmt.Errorf("relay_shapes: unknown phase %q (want params or result)", phase)
	}
	block, shapeKey, unattachedKey := keys[0], keys[1], keys[2]

	root, err := loadRelayDoc(psdk)
	if err != nil {
		return nil, err
	}

	byMethod := map[string]*yaml.Node{}
	for _, kv := range mappingPairs(mapChild(root, "methods")) {
		method := kv[0].Value
		carrier := mapChild(kv[1], block)
		if carrier == nil {
			continue
		}
		if node := mapChild(carrier, shapeKey); node != nil && node.Kind == yaml.MappingNode {
			byMethod[method] = node
		}
	}

	if un := mapChild(root, unattachedKey); un != nil {
		for _, kv := range mappingPairs(mapChild(un, "methods")) {
			method := kv[0].Value
			if _, seen := byMethod[method]; seen {
				continue
			}
			if kv[1] != nil && kv[1].Kind == yaml.MappingNode {
				byMethod[method] = kv[1]
			}
		}
	}

	names := make([]string, 0, len(byMethod))
	for name := range byMethod {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]relayShape, 0, len(names))
	for _, name := range names {
		out = append(out, relayShape{method: name, node: byMethod[name]})
	}
	return out, nil
}
