// Command generate-rest emits the SignalWire REST namespace resource layer
// (per-resource structs + methods + the client namespace tree) from the
// authoritative vendored OpenAPI specs in porting-sdk:
//
//	rest-apis/<ns>/openapi.yaml  (+ x-sdk-* markup)
//	rest-apis/x-sdk-bases.yaml   (shared base method-sets)
//
// It is the Go realization of REST_GENERATOR_RULES.md — the language-neutral
// contract of the REST resource generator (bases, x-sdk-resource markup, path
// composition, command-dispatch, set_methods, cross-spec client-tree
// placement, fail-loud invariants).
//
// Idiom note (PORT_PHILOSOPHY_GO.md): the Go REST port carries the LOOSE
// pre-strict-typing surface — resource methods take map[string]any bodies and
// map[string]string query params, NOT the closed typed create/update params the
// Python reference emits (§5). The bases (Resource/CrudResource/CrudWithAddresses
// + their List/Create/Get/Update/Delete/ListAddresses bodies) are HAND-WRITTEN in
// pkg/rest/namespaces/common.go; this generator emits ONLY the per-resource
// structs that embed those bases, the extra declared methods, the command-dispatch
// verbs, the set_methods wrappers, and the namespace container tree that
// rest_client.go wires. The base method BODIES are never emitted here.
//
// The generated struct/method/param shapes reproduce the CURRENT committed
// hand-written pkg/rest/namespaces/*.go surface (the parity target) so a later
// turn can swap them in with DRIFT staying 0; GEN-FRESH proves fidelity.
//
// GEN-FRESH-gated: `--check` reproduces the committed output and exits non-zero
// if any file differs. Resolves porting-sdk via $PORTING_SDK or sibling.
//
// Usage:
//
//	go run ./cmd/generate-rest          # (re)write the generated REST files
//	go run ./cmd/generate-rest --check  # GEN-FRESH: fail if any is stale
//	go run ./cmd/generate-rest --out DIR # emit to DIR instead of the repo tree
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/signalwire/signalwire-go/cmd/internal/overlay"
	"gopkg.in/yaml.v3"
)

// restOverlay is the SDK-surface policy (x-sdk-overlay.yaml), loaded once in
// run() from porting-sdk. It is consulted at every struct-field emission site to
// drop hidden fields / flag deprecated ones. A nil value (porting-sdk absent) is
// a no-op. Package-level, like emittedTypeNames, so the emit helpers reach it
// without threading it through every call.
var restOverlay *overlay.Overlay

// ---------------------------------------------------------------------------
// Spec model (a minimal ordered OpenAPI view; mirrors generate-payloads).
// ---------------------------------------------------------------------------

// specDoc is one parsed openapi.yaml: its server path prefix, its ordered path
// items, and the whole-spec x-sdk-namespace attr (if any).
type specDoc struct {
	name          string // the spec dir name (e.g. "relay-rest")
	rawPath       string // absolute path to the openapi.yaml (for schema re-reads)
	serverPath    string // path portion of servers[0].url (e.g. "/api/relay/rest")
	namespaceAttr string // whole-spec x-sdk-namespace.attr, "" if none
	paths         []pathItem
	// opIndex maps operationId -> (verb, path) for path composition.
	opIndex map[string]opInfo
}

type opInfo struct {
	verb     string // get/post/put/patch/delete
	path     string // the path key the op lives under (server-relative)
	hasBody  bool   // whether the op declares a requestBody
	hasQuery bool   // whether the op declares any query parameters
}

// pathItem is one "  /path:" entry that may carry an x-sdk-resource.
type pathItem struct {
	path string
	res  *resourceMarkup // nil when the path has no x-sdk-resource
}

// resourceMarkup is the parsed x-sdk-resource block.
type resourceMarkup struct {
	name         string
	base         string
	updateMethod string // PUT / PATCH
	collection   string // base-path override; "" is meaningful (explicit empty)
	hasCollKey   bool   // whether "collection" key was present at all
	namespace    string // per-resource container (registry/logs/video/project/datasphere/fabric)
	attr         string // accessor name override
	kind         string // "command-dispatch" or ""
	request      string // command-dispatch request schema name
	exclude      bool
	methods      []methodMarkup   // ordered
	setMethods   []setMethodBlock // ordered
	specPath     string           // the path key this resource is anchored on
}

type methodMarkup struct {
	name string // SDK method name (snake_case as declared)
	op   string // operationId
}

type setMethodBlock struct {
	name    string // e.g. set_swml_webhook
	handler string
	args    []setArg // ordered
}

type setArg struct {
	name     string // arg identifier (snake_case)
	field    string // bound update-request wire field
	required bool
}

// baseSpec is a parsed x-sdk-bases method-set.
type baseSpec struct {
	extends string
	methods []string // method names in this base (ordered as declared)
}

// ---------------------------------------------------------------------------
// YAML helpers (ordered node walk; mirrors generate-payloads).
// ---------------------------------------------------------------------------

func mapChild(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func scalarChild(node *yaml.Node, key string) string {
	c := mapChild(node, key)
	if c == nil {
		return ""
	}
	return c.Value
}

func rootOf(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	return doc
}

// ---------------------------------------------------------------------------
// Base loading.
// ---------------------------------------------------------------------------

func loadBases(psdk string) (map[string]*baseSpec, error) {
	raw, err := os.ReadFile(filepath.Join(psdk, "rest-apis", "x-sdk-bases.yaml"))
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	basesNode := mapChild(rootOf(&doc), "x-sdk-bases")
	if basesNode == nil {
		return nil, fmt.Errorf("x-sdk-bases.yaml: missing x-sdk-bases")
	}
	out := map[string]*baseSpec{}
	for i := 0; i+1 < len(basesNode.Content); i += 2 {
		name := basesNode.Content[i].Value
		body := basesNode.Content[i+1]
		bs := &baseSpec{extends: scalarChild(body, "extends")}
		if m := mapChild(body, "methods"); m != nil && m.Kind == yaml.MappingNode {
			for j := 0; j+1 < len(m.Content); j += 2 {
				bs.methods = append(bs.methods, m.Content[j].Value)
			}
		}
		out[name] = bs
	}
	// FabricResource is defined in the per-namespace fabric bases file; load it too.
	fabRaw, err := os.ReadFile(filepath.Join(psdk, "rest-apis", "fabric", "x-sdk-bases.yaml"))
	if err == nil {
		var fdoc yaml.Node
		if err := yaml.Unmarshal(fabRaw, &fdoc); err == nil {
			if fb := mapChild(rootOf(&fdoc), "x-sdk-bases"); fb != nil {
				for i := 0; i+1 < len(fb.Content); i += 2 {
					name := fb.Content[i].Value
					body := fb.Content[i+1]
					bs := &baseSpec{extends: scalarChild(body, "extends")}
					if m := mapChild(body, "methods"); m != nil && m.Kind == yaml.MappingNode {
						for j := 0; j+1 < len(m.Content); j += 2 {
							bs.methods = append(bs.methods, m.Content[j].Value)
						}
					}
					out[name] = bs
				}
			}
		}
	}
	// Flatten extends (fail loud on undefined/cyclic).
	flat := map[string][]string{}
	var resolve func(name string, seen map[string]bool) ([]string, error)
	resolve = func(name string, seen map[string]bool) ([]string, error) {
		if seen[name] {
			return nil, fmt.Errorf("x-sdk-bases: cyclic extends at %s", name)
		}
		bs, ok := out[name]
		if !ok {
			return nil, fmt.Errorf("x-sdk-bases: undefined base %q", name)
		}
		seen[name] = true
		var methods []string
		if bs.extends != "" {
			parent, err := resolve(bs.extends, seen)
			if err != nil {
				return nil, err
			}
			methods = append(methods, parent...)
		}
		methods = append(methods, bs.methods...)
		return methods, nil
	}
	for name := range out {
		m, err := resolve(name, map[string]bool{})
		if err != nil {
			return nil, err
		}
		flat[name] = m
	}
	for name, m := range flat {
		out[name].methods = m
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Spec loading.
// ---------------------------------------------------------------------------

func loadSpec(psdk, ns string) (*specDoc, error) {
	rawPath := filepath.Join(psdk, "rest-apis", ns, "openapi.yaml")
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	root := rootOf(&doc)
	sd := &specDoc{name: ns, rawPath: rawPath, opIndex: map[string]opInfo{}}

	// servers[0].url -> path prefix; fail loud on trailing slash (§9).
	servers := mapChild(root, "servers")
	if servers == nil || servers.Kind != yaml.SequenceNode || len(servers.Content) == 0 {
		return nil, fmt.Errorf("%s: missing servers", ns)
	}
	url := scalarChild(servers.Content[0], "url")
	sd.serverPath = urlPath(url)
	if sd.serverPath != "/" && strings.HasSuffix(sd.serverPath, "/") {
		return nil, fmt.Errorf("%s: servers[0].url path %q has a trailing slash", ns, sd.serverPath)
	}

	// whole-spec x-sdk-namespace.attr.
	if nsNode := mapChild(root, "x-sdk-namespace"); nsNode != nil {
		sd.namespaceAttr = scalarChild(nsNode, "attr")
	}

	// paths + operation index.
	pathsNode := mapChild(root, "paths")
	if pathsNode == nil || pathsNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: missing paths", ns)
	}
	for i := 0; i+1 < len(pathsNode.Content); i += 2 {
		pathKey := pathsNode.Content[i].Value
		body := pathsNode.Content[i+1]
		// index operations
		for _, verb := range []string{"get", "post", "put", "patch", "delete"} {
			opNode := mapChild(body, verb)
			if opNode == nil {
				continue
			}
			if opID := scalarChild(opNode, "operationId"); opID != "" {
				info := opInfo{verb: verb, path: pathKey}
				if mapChild(opNode, "requestBody") != nil {
					info.hasBody = true
				}
				if p := mapChild(opNode, "parameters"); p != nil && p.Kind == yaml.SequenceNode && len(p.Content) > 0 {
					info.hasQuery = true
				}
				sd.opIndex[opID] = info
			}
		}
		pi := pathItem{path: pathKey}
		if rm := parseResourceMarkup(mapChild(body, "x-sdk-resource"), pathKey); rm != nil {
			pi.res = rm
		}
		sd.paths = append(sd.paths, pi)
	}
	return sd, nil
}

// urlPath extracts the path component from a (possibly templated) server URL.
func urlPath(url string) string {
	// Strip scheme.
	if i := strings.Index(url, "://"); i >= 0 {
		url = url[i+3:]
	}
	// The first "/" begins the path.
	if i := strings.Index(url, "/"); i >= 0 {
		return url[i:]
	}
	return "/"
}

func parseResourceMarkup(node *yaml.Node, pathKey string) *resourceMarkup {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	rm := &resourceMarkup{
		name:         scalarChild(node, "name"),
		base:         scalarChild(node, "base"),
		updateMethod: scalarChild(node, "update_method"),
		namespace:    scalarChild(node, "namespace"),
		attr:         scalarChild(node, "attr"),
		kind:         scalarChild(node, "kind"),
		request:      scalarChild(node, "request"),
		specPath:     pathKey,
	}
	if c := mapChild(node, "collection"); c != nil {
		rm.collection = c.Value
		rm.hasCollKey = true
	}
	if e := mapChild(node, "exclude"); e != nil {
		rm.exclude = e.Value == "true"
	}
	if m := mapChild(node, "methods"); m != nil && m.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(m.Content); i += 2 {
			mm := methodMarkup{name: m.Content[i].Value, op: scalarChild(m.Content[i+1], "op")}
			rm.methods = append(rm.methods, mm)
		}
	}
	if sm := mapChild(node, "set_methods"); sm != nil && sm.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(sm.Content); i += 2 {
			blk := setMethodBlock{name: sm.Content[i].Value, handler: scalarChild(sm.Content[i+1], "handler")}
			if args := mapChild(sm.Content[i+1], "args"); args != nil && args.Kind == yaml.MappingNode {
				for j := 0; j+1 < len(args.Content); j += 2 {
					a := setArg{name: args.Content[j].Value, field: scalarChild(args.Content[j+1], "field")}
					if r := mapChild(args.Content[j+1], "required"); r != nil && r.Value == "true" {
						a.required = true
					}
					blk.args = append(blk.args, a)
				}
			}
			rm.setMethods = append(rm.setMethods, blk)
		}
	}
	return rm
}

// ---------------------------------------------------------------------------
// Reconciliation tables — spec name: -> the CURRENT hand-written Go surface.
//
// The Go REST port predates the strict-typed generator; its hand names, method
// spellings, base embeddings and container field names are IRREGULAR and
// load-bearing (they are the DRIFT parity target — see tables.go keys). The
// generator therefore reads the markup for STRUCTURE (which resources exist,
// their method-sets, their paths, their placement) but maps every spec name and
// method to the exact hand Go identifier via these tables. A markup name with no
// table entry is a NEW resource the hand surface lacks -> fail loud (a real
// finding, not silently skipped).
// ---------------------------------------------------------------------------

// goStructName maps an x-sdk-resource.name to the hand Go struct name.
var goStructName = map[string]string{
	// relay-rest flat
	"Addresses":       "AddressesNamespace",
	"ImportedNumbers": "ImportedNumbersNamespace",
	"Lookup":          "LookupNamespace",
	"Mfa":             "MFANamespace",
	"NumberGroups":    "NumberGroupsNamespace",
	"PhoneNumbers":    "PhoneNumbersNamespace",
	"Queues":          "QueuesNamespace",
	"Recordings":      "RecordingsNamespace",
	"ShortCodes":      "ShortCodesNamespace",
	"SipProfile":      "SIPProfileNamespace",
	"VerifiedCallers": "VerifiedCallersNamespace",
	// calling (command-dispatch)
	"Calling": "CallingNamespace",
	// chat / pubsub
	"Chat":   "ChatNamespace",
	"PubSub": "PubSubNamespace",
	// datasphere
	"DatasphereDocuments": "DatasphereDocuments",
	// project
	"ProjectTokens": "ProjectTokens",
	// video
	"VideoRooms":            "VideoRooms",
	"VideoRoomTokens":       "VideoRoomTokens",
	"VideoRoomSessions":     "VideoRoomSessions",
	"VideoRoomRecordings":   "VideoRoomRecordings",
	"VideoConferences":      "VideoConferences",
	"VideoConferenceTokens": "VideoConferenceTokens",
	"VideoStreams":          "VideoStreams",
	// registry
	"RegistryBrands":    "RegistryBrands",
	"RegistryCampaigns": "RegistryCampaigns",
	"RegistryOrders":    "RegistryOrders",
	"RegistryNumbers":   "RegistryNumbers",
	// logs (cross-spec)
	"MessageLogs":    "MessageLogs",
	"VoiceLogs":      "VoiceLogs",
	"FaxLogs":        "FaxLogs",
	"ConferenceLogs": "ConferenceLogs",
	// fabric
	"FabricAddresses":      "FabricAddresses",
	"GenericResources":     "GenericResources",
	"CallFlows":            "CallFlowsResource",
	"ConferenceRooms":      "ConferenceRoomsResource",
	"CxmlApplications":     "CxmlApplicationsResource",
	"Subscribers":          "SubscribersResource",
	"FabricTokens":         "FabricTokens",
	"AiAgents":             "AIAgents",
	"CxmlScripts":          "CXMLScripts",
	"CxmlWebhooks":         "CXMLWebhooks",
	"FreeswitchConnectors": "FreeSwitchConnectors",
	"RelayApplications":    "RelayApplications",
	"SipEndpoints":         "SIPEndpoints",
	"SipGateways":          "SIPGateways",
	"SwmlScripts":          "SWMLScripts",
	"SwmlWebhooks":         "SWMLWebhooks",
}

// goMethodName maps a declared markup method name (snake_case) to the hand Go
// method identifier for that resource. Keyed by "<specName>.<methodName>" with a
// bare "<methodName>" fallback (the common CRUD verbs + shared spellings).
var goMethodName = map[string]string{
	// generic CRUD verbs (base-provided or explicit)
	"list":   "List",
	"create": "Create",
	"get":    "Get",
	"update": "Update",
	"delete": "Delete",
	// shared extras
	"list_addresses":  "ListAddresses",
	"list_events":     "ListEvents",
	"list_streams":    "ListStreams",
	"create_stream":   "CreateStream",
	"list_members":    "ListMembers",
	"list_recordings": "ListRecordings",
	// resource-specific irregular spellings
	"PhoneNumbers.search":                        "Search",
	"Lookup.phone_number":                        "PhoneNumber",
	"Mfa.sms":                                    "SMS",
	"Mfa.call":                                   "Call",
	"Mfa.verify":                                 "Verify",
	"NumberGroups.list_memberships":              "ListMemberships",
	"NumberGroups.add_membership":                "AddMembership",
	"NumberGroups.get_membership":                "GetMembership",
	"NumberGroups.delete_membership":             "DeleteMembership",
	"Queues.list_members":                        "ListMembers",
	"Queues.get_next_member":                     "GetNextMember",
	"Queues.get_member":                          "GetMember",
	"VerifiedCallers.redial_verification":        "RedialVerification",
	"VerifiedCallers.submit_verification":        "SubmitVerification",
	"Chat.create_token":                          "CreateToken",
	"PubSub.create_token":                        "CreateToken",
	"DatasphereDocuments.search":                 "Search",
	"DatasphereDocuments.list_chunks":            "ListChunks",
	"DatasphereDocuments.get_chunk":              "GetChunk",
	"DatasphereDocuments.delete_chunk":           "DeleteChunk",
	"ProjectTokens.create":                       "Create",
	"ProjectTokens.update":                       "Update",
	"ProjectTokens.delete":                       "Delete",
	"VideoRooms.list_streams":                    "ListStreams",
	"VideoRooms.create_stream":                   "CreateStream",
	"VideoRoomTokens.create":                     "Create",
	"VideoRoomSessions.list_events":              "ListEvents",
	"VideoRoomSessions.list_members":             "ListMembers",
	"VideoRoomSessions.list_recordings":          "ListRecordings",
	"VideoRoomRecordings.list_events":            "ListEvents",
	"VideoConferences.list_conference_tokens":    "ListConferenceTokens",
	"VideoConferences.list_streams":              "ListStreams",
	"VideoConferences.create_stream":             "CreateStream",
	"VideoConferenceTokens.get":                  "Get",
	"VideoConferenceTokens.reset":                "Reset",
	"VideoStreams.get":                           "Get",
	"VideoStreams.update":                        "Update",
	"VideoStreams.delete":                        "Delete",
	"RegistryBrands.list_campaigns":              "ListCampaigns",
	"RegistryBrands.create_campaign":             "CreateCampaign",
	"RegistryCampaigns.list_numbers":             "ListNumbers",
	"RegistryCampaigns.list_orders":              "ListOrders",
	"RegistryCampaigns.create_order":             "CreateOrder",
	"CallFlows.list_versions":                    "ListVersions",
	"CallFlows.deploy_version":                   "DeployVersion",
	"GenericResources.assign_phone_route":        "AssignPhoneRoute",
	"GenericResources.assign_domain_application": "AssignDomainApplication",
	"Subscribers.list_sip_endpoints":             "ListSIPEndpoints",
	"Subscribers.create_sip_endpoint":            "CreateSIPEndpoint",
	"Subscribers.get_sip_endpoint":               "GetSIPEndpoint",
	"Subscribers.update_sip_endpoint":            "UpdateSIPEndpoint",
	"Subscribers.delete_sip_endpoint":            "DeleteSIPEndpoint",
	"FabricTokens.create_subscriber_token":       "CreateSubscriberToken",
	"FabricTokens.refresh_subscriber_token":      "RefreshSubscriberToken",
	"FabricTokens.create_invite_token":           "CreateInviteToken",
	"FabricTokens.create_guest_token":            "CreateGuestToken",
	"FabricTokens.create_embed_token":            "CreateEmbedToken",
}

// resolveMethodName returns the hand Go method identifier for a markup method
// on a resource. Fail loud when no mapping exists.
func resolveMethodName(specName, methodName string) (string, error) {
	if v, ok := goMethodName[specName+"."+methodName]; ok {
		return v, nil
	}
	if v, ok := goMethodName[methodName]; ok {
		return v, nil
	}
	return "", fmt.Errorf("no Go method name for %s.%s (add to goMethodName)", specName, methodName)
}

// ---------------------------------------------------------------------------
// Path composition (§4).
// ---------------------------------------------------------------------------

// collectionPath returns the resource's base path: serverPath + collection.
// The collection is the explicit override if present, else derived from the
// anchor path's leading segment(s).
func (rm *resourceMarkup) collectionSegment() string {
	if rm.hasCollKey {
		return rm.collection // may be "" (FabricTokens) — explicit
	}
	// Derive from the anchor path: strip a trailing "/{param}" or "/{param}/..".
	p := rm.specPath
	if i := strings.Index(p, "/{"); i >= 0 {
		p = p[:i]
	}
	return p
}

func (rm *resourceMarkup) basePath(serverPath string) string {
	return joinPath(serverPath, rm.collectionSegment())
}

func joinPath(a, b string) string {
	if b == "" {
		return a
	}
	return strings.TrimRight(a, "/") + "/" + strings.TrimLeft(b, "/")
}

// tailBelow returns the op path's segments below the resource collection (the
// under-collection relative form) as literal segments and {param} markers, OR
// signals sibling (absolute) placement when the op path does not start with the
// collection.
func (rm *resourceMarkup) relativeTail(serverPath, opPath string) (segs []string, sibling bool) {
	coll := rm.collectionSegment()
	full := joinPath(serverPath, coll) // resource base
	abs := joinPath(serverPath, opPath)
	if coll != "" && strings.HasPrefix(abs, full+"/") {
		rest := strings.TrimPrefix(abs, full+"/")
		return splitSegs(rest), false
	}
	if coll != "" && abs == full {
		return nil, false
	}
	// Sibling: the op lives outside the collection.
	return splitSegs(strings.TrimPrefix(abs, "/")), true
}

func splitSegs(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "/")
}

// ---------------------------------------------------------------------------
// Emission — file header.
// ---------------------------------------------------------------------------

const genHeader = `// Code generated by cmd/generate-rest; DO NOT EDIT.
//
// AUTO-GENERATED from porting-sdk/rest-apis/ (x-sdk-* markup) — regenerate with:
//   go run ./cmd/generate-rest
//
// %s

package namespaces
`

// paramName maps a wire path-param brace name to the hand Go arg identifier.
// The hand code uses domain-specific arg names (queueID, groupID, documentID…);
// they are cosmetic (the value is positional), but reproduced for fidelity.
var paramArgName = map[string]string{
	"id":                   "id",
	"queue_id":             "queueID",
	"NumberGroupId":        "groupID",
	"documentId":           "documentID",
	"chunkId":              "chunkID",
	"mfa_request_id":       "requestID",
	"e164_number":          "e164",
	"fabric_subscriber_id": "subscriberID",
	"ai_agent_id":          "id",
	"cxml_webhook_id":      "id",
	"swml_webhook_id":      "id",
	"token_id":             "tokenID",
}

func argFor(brace string) string {
	if v, ok := paramArgName[brace]; ok {
		return v
	}
	return "id"
}

// escapeIdent escapes a Go keyword-colliding identifier (§5). Builtins like
// id/list/type stay unescaped (valid Go identifiers).
var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true, "interface": true,
	"map": true, "package": true, "range": true, "return": true, "select": true,
	"struct": true, "switch": true, "type": true, "var": true,
}

func escapeIdent(s string) string {
	if goKeywords[s] {
		return s + "_"
	}
	return s
}

// ---------------------------------------------------------------------------
// Emission — one method (declared methods:).
//
// Signature category comes from the op verb + path shape, reproducing the hand
// loose-typed surface:
//   GET  with a trailing collection segment (list-ish) -> (…ids, params map[string]string)
//   GET  single item                                   -> (…ids)
//   POST/PUT/PATCH                                      -> (…ids, data map[string]any)
//   DELETE                                             -> (…ids)
// All return (map[string]any, error). The HTTP call uses the base relative
// r.Path(...) helper (under-collection) or an absolute server-rooted path (sibling).
// ---------------------------------------------------------------------------

func emitMethod(b *strings.Builder, recv, goName string, rm *resourceMarkup, serverPath string, mm methodMarkup, sd *specDoc) error {
	op, ok := sd.opIndex[mm.op]
	if !ok {
		return fmt.Errorf("%s.%s: op %q not in spec", rm.name, mm.name, mm.op)
	}
	segs, sibling := rm.relativeTail(serverPath, op.path)

	// Split segments into positional id args + literal path pieces.
	var idArgs []string
	var pathExpr []string // r.Path(...) arg exprs, or absolute build
	for _, s := range segs {
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
			arg := argFor(s[1 : len(s)-1])
			// de-dup arg names within one signature (rare)
			for containsStr(idArgs, arg) {
				arg += "2"
			}
			idArgs = append(idArgs, arg)
			pathExpr = append(pathExpr, arg)
		} else {
			pathExpr = append(pathExpr, fmt.Sprintf("%q", s))
		}
	}

	verb := op.verb
	// Determine trailing-body vs query. A write verb takes a `data` body only
	// when the op declares a requestBody (else it POSTs nil — matches the hand
	// no-body form for reset/redial). A GET takes query params only when the op
	// declares query parameters (spec-driven, not path-shape-guessed).
	writeVerb := verb == "post" || verb == "put" || verb == "patch"

	var params []string
	for _, a := range idArgs {
		params = append(params, a+" string")
	}
	// bodyFields holds (wire field -> struct field ident) for an exploded write body.
	var bodyOrder []string
	bodyParam := map[string]string{}
	// bodyFieldPtr marks a body field whose Go type is nilable (pointer/slice/map/
	// any) — assembled with an `if != nil` guard; a value-typed (required) field is
	// assigned unconditionally.
	bodyFieldPtr := map[string]bool{}
	// structDef, when set, is the params-struct type definition emitted before the
	// method (the idiomatic Go named-options-struct form — §5, §4a). structName is
	// the parameter variable's type; every optional call-site field lives on it.
	var structDef string
	var tail string
	switch {
	case writeVerb:
		if op.hasBody {
			// §5/§4a: collect the requestBody schema fields (required-first, then
			// optionals) into a per-method params STRUCT — the idiomatic Go named-
			// options form (aws-sdk-go-v2 input-struct style), NOT flat positionals.
			// Each field becomes a struct field; an `Extras map[string]any` door
			// closes it. The wire body is assembled from the struct's non-nil fields
			// + merged Extras — SAME wire keys/body as the old flat form. Struct
			// fields are `any` (loose surface; the drift gate compares param
			// count+kind, and the enumerator unfolds the struct back into the flat
			// keyword set).
			fields, oneOfBody, err := operationBodyFields(sd, op)
			if err != nil {
				return err
			}
			if oneOfBody {
				// A top-level oneOf-union body is NOT exploded (the reference passes a
				// single whole-body object) — emit the loose single-`data` form.
				params = append(params, "data map[string]any")
				tail = "data"
				break
			}
			// Field types from the requestBody schema (§4 strict typing): a
			// required field is a value type, an optional field a pointer *T (nil =
			// unset). The enumerator maps *T→optional<T>, []T→list<T>, a generated
			// type name→class:<...>, keeping the field-level types real against the
			// oracle (count: optional<int>, distance: optional<float>, …) instead of
			// the old `any`. Falls back to `any` for a field the type walk can't
			// resolve (none today).
			fieldTypes, fieldReq, ftErr := operationBodyFieldTypes(sd, op)
			if ftErr != nil {
				return ftErr
			}
			structName := recv + goName + "Params"
			used := map[string]bool{"Extras": true}
			var fieldDefs []string
			for _, f := range fields {
				fn := structFieldName(f)
				for used[fn] {
					fn += "_"
				}
				used[fn] = true
				bodyParam[f] = fn
				bodyOrder = append(bodyOrder, f)
				ftype := paramFieldType(fieldTypes[f], fieldReq[f])
				bodyFieldPtr[f] = isNilableGoType(ftype)
				fieldDefs = append(fieldDefs, "\t"+fn+" "+ftype)
			}
			fieldDefs = append(fieldDefs, "\tExtras map[string]any")
			structDef = fmt.Sprintf("// %s holds the named optional parameters for %s.%s.\ntype %s struct {\n%s\n}\n\n",
				structName, recv, goName, structName, strings.Join(fieldDefs, "\n"))
			params = append(params, "params "+structName)
			tail = "body"
		}
	case verb == "get":
		// §5.3: a GET has no request body, so every generated GET operation
		// method takes an optional query-params tail (the Python oracle records
		// a `params` var_keyword on EVERY declared GET method — verified against
		// python_signatures.json across all specs, zero exceptions). This is the
		// general rule, NOT a per-method opt-in: it is independent of whether the
		// spec happens to declare a query parameter today (`include` on Lookup,
		// `media_ttl` on room recordings) and independent of path shape — a
		// {param}-terminal single-item GET (RegistryBrands.get, VideoStreams.get,
		// Subscribers.get_sip_endpoint, DatasphereDocuments.get_chunk, …) takes
		// the tail exactly like a collection-style GET. Only the base-provided
		// Get(id) (in common.go / synthesized for ReadResource) omits it, matching
		// the oracle's fixed base get(resource_id).
		params = append(params, "params map[string]string")
		tail = "params"
	}

	// Path expression.
	var pathCode string
	switch {
	case sibling:
		// Absolute server-rooted path (§4 sibling): build the literal + args.
		pathCode = absolutePath(serverPath, op.path, idArgs)
	case len(pathExpr) == 0:
		pathCode = "r.Base"
	default:
		pathCode = "r.Path(" + strings.Join(pathExpr, ", ") + ")"
	}

	// Return type (§4 typed output): the op's 200/201 $ref response type, else the
	// open map when the spec has no typed response (inline/array/absent). A typed
	// response wraps the base map result in decodeResult[Resp].
	respType, rtErr := operationResponseType(sd, op)
	if rtErr != nil {
		return rtErr
	}
	retSig := "(map[string]any, error)"
	if respType != "" {
		retSig = "(*" + respType + ", error)"
	}

	if structDef != "" {
		b.WriteString(structDef)
	}
	fmt.Fprintf(b, "func (r *%s) %s(%s) %s {\n", recv, goName, strings.Join(params, ", "), retSig)
	// Assemble the write body: exploded from the params struct (only non-nil fields
	// + merged params.Extras) or the loose single-`data` object (oneOf-union bodies).
	dataExpr := "nil"
	switch tail {
	case "body":
		b.WriteString("\tbody := map[string]any{}\n")
		for _, f := range bodyOrder {
			// A required field is a value type (always sent); an optional field is a
			// pointer / reference type sent only when non-nil (matches the reference,
			// which omits an unset optional so the server applies its default). Value
			// types have no nil to test, so they are assigned unconditionally.
			if bodyFieldPtr[f] {
				fmt.Fprintf(b, "\tif params.%s != nil {\n\t\tbody[%q] = params.%s\n\t}\n", bodyParam[f], f, bodyParam[f])
			} else {
				fmt.Fprintf(b, "\tbody[%q] = params.%s\n", f, bodyParam[f])
			}
		}
		b.WriteString("\tmergeExtra(body, []map[string]any{params.Extras})\n")
		dataExpr = "body"
	case "data":
		dataExpr = "data"
	}
	// wrap emits the HTTP call, wrapping it in decodeResult[Resp] for a typed return.
	wrap := func(call string) string {
		if respType != "" {
			return "decodeResult[" + respType + "](" + call + ")"
		}
		return call
	}
	switch verb {
	case "get":
		if tail == "params" {
			fmt.Fprintf(b, "\treturn %s\n", wrap(fmt.Sprintf("r.HTTP.Get(%s, params)", pathCode)))
		} else {
			fmt.Fprintf(b, "\treturn %s\n", wrap(fmt.Sprintf("r.HTTP.Get(%s, nil)", pathCode)))
		}
	case "post":
		fmt.Fprintf(b, "\treturn %s\n", wrap(fmt.Sprintf("r.HTTP.Post(%s, %s, nil)", pathCode, dataExpr)))
	case "put":
		fmt.Fprintf(b, "\treturn %s\n", wrap(fmt.Sprintf("r.HTTP.Put(%s, %s)", pathCode, dataExpr)))
	case "patch":
		fmt.Fprintf(b, "\treturn %s\n", wrap(fmt.Sprintf("r.HTTP.Patch(%s, %s)", pathCode, dataExpr)))
	case "delete":
		fmt.Fprintf(b, "\treturn %s\n", wrap(fmt.Sprintf("r.HTTP.Delete(%s)", pathCode)))
	}
	b.WriteString("}\n\n")
	return nil
}

func containsStr(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// absolutePath builds an absolute server-rooted path literal for a sibling op,
// substituting {params} with "+arg+" concatenation (matches the hand form
// "/api/relay/rest/number_group_memberships/"+membershipID).
func absolutePath(serverPath, opPath string, idArgs []string) string {
	full := joinPath(serverPath, strings.TrimPrefix(opPath, "/"))
	var out strings.Builder
	out.WriteByte('"')
	ai := 0
	i := 0
	for i < len(full) {
		if full[i] == '{' {
			j := strings.IndexByte(full[i:], '}')
			out.WriteString(`"`)
			if ai < len(idArgs) {
				out.WriteString(" + " + idArgs[ai] + " + ")
			}
			ai++
			out.WriteByte('"')
			i += j + 1
			continue
		}
		out.WriteByte(full[i])
		i++
	}
	out.WriteByte('"')
	// Clean trailing ` + ""`.
	s := strings.TrimSuffix(out.String(), ` + ""`)
	return s
}

// ---------------------------------------------------------------------------
// Emission — resource struct + its declared methods.
//
// Base embedding maps the markup base + update_method to the hand Go embed:
//   BaseResource    -> Resource            (value embed; partial surfaces)
//   ReadResource    -> Resource            (list/get only; hand uses Resource)
//   CrudResource    -> *CrudResource       (NewCrudResource / NewCrudResourcePUT)
//   FabricResource  -> *CrudWithAddresses  (NewCrudWithAddresses / …PUT)
// ---------------------------------------------------------------------------

type embedInfo struct {
	field string // embedded type as written in the struct
	ctor  string // constructor expr fragment producing it, given (client, base)
	crud  bool   // whether the base provides CRUD (for set_methods validation)
}

// hasSiblingAddressOverride reports whether the resource RE-DECLARES list_addresses
// on a SINGULAR sibling sub-path (the fabric call_flow/conference_room platform
// quirk — a real wire quirk documented in the fabric spec). When it does, the
// resource must NOT also inherit the base CrudWithAddresses.ListAddresses (which
// routes to the PLURAL collection path), or the class would expose TWO routes for
// one canonical op (the embedded plural + the singular override) — a route split.
// The reference (Python) has no such split: an overriding method fully REPLACES the
// inherited one. So when the override is present we embed the plain *CrudResource
// (no addresses) and let the singular override be the class's ONLY list_addresses.
func hasSiblingAddressOverride(rm *resourceMarkup, sd *specDoc) bool {
	for _, mm := range rm.methods {
		if mm.name != "list_addresses" {
			continue
		}
		if op, ok := sd.opIndex[mm.op]; ok {
			if _, sibling := rm.relativeTail(sd.serverPath, op.path); sibling {
				return true
			}
		}
	}
	return false
}

func embedFor(rm *resourceMarkup, sd *specDoc) (embedInfo, error) {
	switch rm.base {
	case "BaseResource", "ReadResource":
		return embedInfo{field: "Resource", ctor: "Resource{HTTP: client, Base: %s}"}, nil
	case "CrudResource":
		if rm.updateMethod == "" {
			return embedInfo{}, fmt.Errorf("%s: CrudResource requires update_method", rm.name)
		}
		if rm.updateMethod == "PUT" {
			return embedInfo{field: "*CrudResource", ctor: "NewCrudResourcePUT(client, %s)", crud: true}, nil
		}
		return embedInfo{field: "*CrudResource", ctor: "NewCrudResource(client, %s)", crud: true}, nil
	case "FabricResource":
		// A resource that overrides list_addresses onto a singular sibling path
		// embeds the plain *CrudResource (no base ListAddresses) so its ONLY
		// list_addresses route is the singular override — no plural/singular split.
		if hasSiblingAddressOverride(rm, sd) {
			if rm.updateMethod == "PUT" {
				return embedInfo{field: "*CrudResource", ctor: "NewCrudResourcePUT(client, %s)", crud: true}, nil
			}
			return embedInfo{field: "*CrudResource", ctor: "NewCrudResource(client, %s)", crud: true}, nil
		}
		if rm.updateMethod == "PUT" {
			return embedInfo{field: "*CrudWithAddresses", ctor: "NewCrudWithAddressesPUT(client, %s)", crud: true}, nil
		}
		return embedInfo{field: "*CrudWithAddresses", ctor: "NewCrudWithAddresses(client, %s)", crud: true}, nil
	}
	return embedInfo{}, fmt.Errorf("%s: unknown base %q", rm.name, rm.base)
}

// goEmbedProvides is the set of method names the CHOSEN GO EMBED already
// supplies (so no per-resource code is emitted for them). This is keyed on the
// Go embed, NOT the markup base — because the Go realization maps some markup
// bases to embeds with fewer methods:
//
//	Resource            -> {}                  (BaseResource + ReadResource map here)
//	*CrudResource       -> list/create/get/update/delete
//	*CrudWithAddresses  -> …CRUD… + list_addresses
//
// A declared method that the embed does NOT provide is emitted. A declared
// method the embed DOES provide is inherited — EXCEPT list_addresses, which the
// fabric singular-path resources RE-DECLARE as an override (hand-written the
// same way): if list_addresses is explicitly in the markup methods AND its op
// path is a sibling of the collection, it is emitted to shadow the base.
func extraMethods(rm *resourceMarkup, emb embedInfo, sd *specDoc) []methodMarkup {
	provided := map[string]bool{}
	switch emb.field {
	case "*CrudResource":
		for _, m := range []string{"list", "create", "get", "update", "delete"} {
			provided[m] = true
		}
	case "*CrudWithAddresses":
		for _, m := range []string{"list", "create", "get", "update", "delete", "list_addresses"} {
			provided[m] = true
		}
	}
	var out []methodMarkup
	for _, mm := range rm.methods {
		if provided[mm.name] {
			// Inherited — unless it is an explicit sibling-path override
			// (list_addresses on a singular sub-path).
			if mm.name == "list_addresses" {
				if op, ok := sd.opIndex[mm.op]; ok {
					if _, sibling := rm.relativeTail(sd.serverPath, op.path); sibling {
						out = append(out, mm)
						continue
					}
				}
			}
			continue
		}
		out = append(out, mm)
	}
	return out
}

func emitResource(b *strings.Builder, rm *resourceMarkup, sd *specDoc, bases map[string]*baseSpec) error {
	goName, ok := goStructName[rm.name]
	if !ok {
		return fmt.Errorf("resource %q (spec %s) has no Go struct mapping (add to goStructName)", rm.name, sd.name)
	}
	if rm.kind == "command-dispatch" {
		return emitCommandDispatch(b, rm, sd, goName)
	}
	emb, err := embedFor(rm, sd)
	if err != nil {
		return err
	}
	fmt.Fprintf(b, "// %s is a client for the %q resource of the SignalWire %s API.\n", goName, rm.name, sd.name)
	fmt.Fprintf(b, "type %s struct {\n\t%s\n}\n\n", goName, emb.field)

	// Per-resource constructor (§4): bakes the resource's base path into the
	// embedded base. The base path is serverPath + collection, computed once.
	base := rm.basePath(sd.serverPath)
	ctorExpr := fmt.Sprintf(emb.ctor, fmt.Sprintf("%q", base))
	fmt.Fprintf(b, "// New%s constructs a %s bound to base path %q.\n", goName, goName, base)
	fmt.Fprintf(b, "func New%s(client HTTPClient) *%s {\n\treturn &%s{%s}\n}\n\n", goName, goName, goName, ctorExpr)

	// ReadResource maps to the method-less Go `Resource` embed, so the base's
	// list+get are synthesized here (the hand code writes them out explicitly).
	if rm.base == "ReadResource" {
		fmt.Fprintf(b, "func (r *%s) List(params map[string]string) (map[string]any, error) {\n\treturn r.HTTP.Get(r.Base, params)\n}\n\n", goName)
		fmt.Fprintf(b, "func (r *%s) Get(id string) (map[string]any, error) {\n\treturn r.HTTP.Get(r.Path(id), nil)\n}\n\n", goName)
	}

	for _, mm := range extraMethods(rm, emb, sd) {
		mName, err := resolveMethodName(rm.name, mm.name)
		if err != nil {
			return err
		}
		if err := emitMethod(b, goName, mName, rm, sd.serverPath, mm, sd); err != nil {
			return err
		}
	}

	// set_methods (§7): require a CRUD base; each arg field must be in the
	// update request schema (verified structurally against the markup bindings).
	if len(rm.setMethods) > 0 {
		if !emb.crud {
			return fmt.Errorf("%s: set_methods require a CRUD base, got %s", rm.name, rm.base)
		}
		for _, sm := range rm.setMethods {
			if err := emitSetMethod(b, goName, rm, sm); err != nil {
				return err
			}
		}
	}
	return nil
}

// emitSetMethod emits a typed update() wrapper binding a fixed handler + args.
// Reproduces the hand loose form: (sid string, <required-args string>,
// extra ...map[string]any). Optional args funnel through the extra door in the
// hand surface (the hand code uses *Options structs for some; the generated
// form uses the uniform extra-map door and flags the *Options divergence).
func emitSetMethod(b *strings.Builder, recv string, rm *resourceMarkup, sm setMethodBlock) error {
	if sm.handler == "" {
		return fmt.Errorf("%s.%s: set_method missing handler", rm.name, sm.name)
	}
	goName := setMethodGoName(sm.name)
	// Required args become positional string params (folded into body
	// unconditionally); optional args become trailing *string params (nil ->
	// omit the wire field), matching the Python reference's
	// ``<arg>: str`` (required) / ``<arg>: Optional[str] = None`` (optional)
	// keyword shape. Both go before the ``extra ...map[string]any`` **kwargs
	// door. Ordering: all required, then all optional, then extra.
	var params []string
	params = append(params, "sid string")
	var bodyLines []string     // unconditional (required) body assignments
	var optionalLines []string // conditional (optional) body assignments
	bodyLines = append(bodyLines, fmt.Sprintf("\t\t%q: %q,", "call_handler", sm.handler))
	var optParams []string
	for _, a := range sm.args {
		if a.field == "" {
			return fmt.Errorf("%s.%s: arg %q missing field", rm.name, sm.name, a.name)
		}
		if a.required {
			params = append(params, escapeIdent(a.name)+" string")
			bodyLines = append(bodyLines, fmt.Sprintf("\t\t%q: %s,", a.field, escapeIdent(a.name)))
		} else {
			id := escapeIdent(a.name)
			optParams = append(optParams, id+" *string")
			optionalLines = append(optionalLines, fmt.Sprintf("\tif %s != nil {\n\t\tbody[%q] = *%s\n\t}\n", id, a.field, id))
		}
	}
	params = append(params, optParams...)
	params = append(params, "extra ...map[string]any")
	fmt.Fprintf(b, "func (r *%s) %s(%s) (map[string]any, error) {\n", recv, goName, strings.Join(params, ", "))
	b.WriteString("\tbody := map[string]any{\n")
	for _, l := range bodyLines {
		b.WriteString(l + "\n")
	}
	b.WriteString("\t}\n")
	for _, l := range optionalLines {
		b.WriteString(l)
	}
	b.WriteString("\tmergeExtra(body, extra)\n")
	b.WriteString("\treturn r.Update(sid, body)\n}\n\n")
	return nil
}

func setMethodGoName(snake string) string {
	// set_swml_webhook -> SetSwmlWebhook; SIP/AI/etc initialisms handled by table.
	if v, ok := setMethodNames[snake]; ok {
		return v
	}
	return pascal(snake)
}

var setMethodNames = map[string]string{
	"set_swml_webhook":      "SetSwmlWebhook",
	"set_cxml_webhook":      "SetCxmlWebhook",
	"set_cxml_application":  "SetCxmlApplication",
	"set_ai_agent":          "SetAiAgent",
	"set_call_flow":         "SetCallFlow",
	"set_relay_application": "SetRelayApplication",
	"set_relay_topic":       "SetRelayTopic",
}

func pascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' || r == '.' })
	var b strings.Builder
	for _, w := range parts {
		if w == "" {
			continue
		}
		b.WriteString(strings.ToUpper(w[:1]) + w[1:])
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Emission — command-dispatch resource (§6, calling).
//
// Reads the request oneOf discriminator.mapping and emits one typed method per
// command. The hand Go form is loose: each verb is
//   func (c *CallingNamespace) <Verb>(callID string, params map[string]any)
// dispatching via a shared execute(command, callID, params). Commands with no
// id (dial/update) omit the callID positional. Method names come from the
// callingMethodName table (the command->Go-name mapping is declared, matching
// tables.go). Union params are NOT flattened in the loose Go surface (params is
// map[string]any); this is a documented divergence from §6.
// ---------------------------------------------------------------------------

// callingMethodName maps a discriminator command string to the hand Go method.
var callingMethodName = map[string]string{
	"dial":                               "Dial",
	"update":                             "Update",
	"calling.end":                        "End",
	"calling.transfer":                   "Transfer",
	"calling.disconnect":                 "Disconnect",
	"calling.play":                       "Play",
	"calling.play.pause":                 "PlayPause",
	"calling.play.resume":                "PlayResume",
	"calling.play.stop":                  "PlayStop",
	"calling.play.volume":                "PlayVolume",
	"calling.record":                     "Record",
	"calling.record.pause":               "RecordPause",
	"calling.record.resume":              "RecordResume",
	"calling.record.stop":                "RecordStop",
	"calling.collect":                    "Collect",
	"calling.collect.stop":               "CollectStop",
	"calling.collect.start_input_timers": "CollectStartInputTimers",
	"calling.detect":                     "Detect",
	"calling.detect.stop":                "DetectStop",
	"calling.tap":                        "Tap",
	"calling.tap.stop":                   "TapStop",
	"calling.stream":                     "Stream",
	"calling.stream.stop":                "StreamStop",
	"calling.denoise":                    "Denoise",
	"calling.denoise.stop":               "DenoiseStop",
	"calling.transcribe":                 "Transcribe",
	"calling.transcribe.stop":            "TranscribeStop",
	"calling.ai_message":                 "AIMessage",
	"calling.ai_hold":                    "AIHold",
	"calling.ai_unhold":                  "AIUnhold",
	"calling.ai.stop":                    "AIStop",
	"calling.live_transcribe":            "LiveTranscribe",
	"calling.live_translate":             "LiveTranslate",
	"calling.send_fax.stop":              "SendFaxStop",
	"calling.receive_fax.stop":           "ReceiveFaxStop",
	"calling.refer":                      "Refer",
	"calling.user_event":                 "UserEvent",
}

// commandsWithoutID are the commands whose schema has no id property (dial/update).
var commandsWithoutID = map[string]bool{"dial": true, "update": true}

func emitCommandDispatch(b *strings.Builder, rm *resourceMarkup, sd *specDoc, goName string) error {
	if rm.request == "" {
		return fmt.Errorf("%s: command-dispatch requires request", rm.name)
	}
	mapping, schemaByCmd, err := loadDiscriminatorMappingSchemas(sd, rm.request)
	if err != nil {
		return err
	}
	op, ok := sd.opIndex["call-commands"]
	base := joinPath(sd.serverPath, "calls")
	if ok {
		base = joinPath(sd.serverPath, strings.TrimPrefix(op.path, "/"))
	}
	fmt.Fprintf(b, "// %s is a client for the %q resource of the SignalWire %s API (command-dispatch endpoint).\n", goName, rm.name, sd.name)
	fmt.Fprintf(b, "type %s struct {\n\tResource\n}\n\n", goName)
	// Constructor bakes the command endpoint base path (§4).
	fmt.Fprintf(b, "// New%s constructs a %s bound to base path %q.\n", goName, goName, base)
	fmt.Fprintf(b, "func New%s(client HTTPClient) *%s {\n\treturn &%s{Resource{HTTP: client, Base: %q}}\n}\n\n", goName, goName, goName, base)
	// execute helper.
	fmt.Fprintf(b, "func (c *%s) execute(command string, callID string, params map[string]any) (map[string]any, error) {\n", goName)
	b.WriteString("\tbody := map[string]any{\"command\": command, \"params\": params}\n")
	b.WriteString("\tif callID != \"\" {\n\t\tbody[\"id\"] = callID\n\t}\n")
	b.WriteString("\treturn c.HTTP.Post(c.Base, body, nil)\n}\n\n")

	for _, cmd := range mapping {
		mName, ok := callingMethodName[cmd]
		if !ok {
			return fmt.Errorf("command %q has no Go method name (add to callingMethodName)", cmd)
		}
		// §5/§6: emit one typed param per command params-sub-schema field
		// (union-flattened), a leading callID positional when the command carries
		// a top-level id, and a trailing `extras` door. The wire body is the
		// `params` object assembled from the provided fields + merged extras.
		_, fields, err := commandFields(sd, schemaByCmd[cmd])
		if err != nil {
			return err
		}
		cmdFieldTypes, cmdFieldReq, ctErr := commandFieldTypes(sd, schemaByCmd[cmd])
		if ctErr != nil {
			return ctErr
		}
		withID := !commandsWithoutID[cmd]
		// §5/§6/§4a: the command's typed params collapse into a named params STRUCT
		// (idiomatic Go options struct, not flat positionals); a leading callID stays
		// positional when the command carries a top-level id. The wire `params` object
		// is assembled from the struct's non-nil fields + merged Extras — SAME body as
		// the old flat form. The enumerator unfolds the struct back into the flat
		// keyword set (drift-neutral).
		structName := goName + mName + "Params"
		fieldParam := map[string]string{} // wire field -> struct field ident
		fieldPtr := map[string]bool{}     // wire field -> nilable Go type
		used := map[string]bool{"Extras": true}
		var fieldDefs []string
		for _, f := range fields {
			fn := structFieldName(f)
			for used[fn] {
				fn += "_"
			}
			used[fn] = true
			fieldParam[f] = fn
			// §4 strict typing: type each command param from its params-sub-schema
			// field (required → value, optional → *T); a $ref field → the generated
			// type (e.g. swml → *SWMLObject), a union → any, an array → []T.
			ftype := paramFieldType(cmdFieldTypes[f], cmdFieldReq[f])
			fieldPtr[f] = isNilableGoType(ftype)
			fieldDefs = append(fieldDefs, "\t"+fn+" "+ftype)
		}
		fieldDefs = append(fieldDefs, "\tExtras map[string]any")
		fmt.Fprintf(b, "// %s holds the named optional parameters for %s.%s.\ntype %s struct {\n%s\n}\n\n",
			structName, goName, mName, structName, strings.Join(fieldDefs, "\n"))
		var sigParams []string
		if withID {
			sigParams = append(sigParams, "callID string")
		}
		sigParams = append(sigParams, "params "+structName)
		// Command methods return the typed CallResponse (the call-commands op's
		// 200 response is a $ref to CallResponse for every command).
		fmt.Fprintf(b, "func (c *%s) %s(%s) (*CallResponse, error) {\n", goName, mName, strings.Join(sigParams, ", "))
		b.WriteString("\tbody := map[string]any{}\n")
		for _, f := range fields {
			if fieldPtr[f] {
				fmt.Fprintf(b, "\tif params.%s != nil {\n\t\tbody[%q] = params.%s\n\t}\n", fieldParam[f], f, fieldParam[f])
			} else {
				fmt.Fprintf(b, "\tbody[%q] = params.%s\n", f, fieldParam[f])
			}
		}
		b.WriteString("\tmergeExtra(body, []map[string]any{params.Extras})\n")
		callID := `""`
		if withID {
			callID = "callID"
		}
		fmt.Fprintf(b, "\treturn decodeResult[CallResponse](c.execute(%q, %s, body))\n}\n\n", cmd, callID)
	}
	return nil
}

// loadDiscriminatorMapping returns the ordered command strings from the request
// schema's discriminator.mapping. Fail loud if absent (§9).
func loadDiscriminatorMapping(sd *specDoc, schemaName string) ([]string, error) {
	schemas, err := componentsSchemas(sd)
	if err != nil {
		return nil, err
	}
	sch := mapChild(schemas, schemaName)
	if sch == nil {
		return nil, fmt.Errorf("command-dispatch request %q not in components.schemas", schemaName)
	}
	disc := mapChild(sch, "discriminator")
	mapping := mapChild(disc, "mapping")
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("command-dispatch request %q has no discriminator.mapping", schemaName)
	}
	var out []string
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		out = append(out, mapping.Content[i].Value)
	}
	return out, nil
}

// loadDiscriminatorMappingSchemas returns the ordered command strings AND a
// command->request-schema-name map (the discriminator.mapping value, a $ref).
func loadDiscriminatorMappingSchemas(sd *specDoc, schemaName string) ([]string, map[string]string, error) {
	schemas, err := componentsSchemas(sd)
	if err != nil {
		return nil, nil, err
	}
	sch := mapChild(schemas, schemaName)
	if sch == nil {
		return nil, nil, fmt.Errorf("command-dispatch request %q not in components.schemas", schemaName)
	}
	mapping := mapChild(mapChild(sch, "discriminator"), "mapping")
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("command-dispatch request %q has no discriminator.mapping", schemaName)
	}
	var out []string
	byCmd := map[string]string{}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		cmd := mapping.Content[i].Value
		out = append(out, cmd)
		byCmd[cmd] = refLeaf(mapping.Content[i+1].Value)
	}
	return out, byCmd, nil
}

// ---------------------------------------------------------------------------
// Typed param extraction (§5 closed params) — read the request/params schema
// properties from the spec so operation + command methods can emit one Go
// param per wire field (required-first, then optionals, in property order),
// with a trailing `extras map[string]any` door. Matches the Python oracle's
// keyword-only body fields + `extras`/`**kwargs` tail (the port enumerator
// reclassifies the exploded positionals to keyword/var_keyword — see
// cmd/enumerate-signatures). Body fields are emitted as `any`: the Go REST port
// carries the loose surface, and the drift gate compares param COUNT + KIND
// (not the field's static type), so `any` is both idiom-correct and drift-safe.
// ---------------------------------------------------------------------------

// schemaCache memoizes the parsed components/schemas node per spec (the raw doc
// is re-read once and the node tree kept alive).
var schemaCache = map[string]*yaml.Node{}

// componentsSchemas returns the components.schemas mapping node for a spec.
func componentsSchemas(sd *specDoc) (*yaml.Node, error) {
	if n, ok := schemaCache[sd.rawPath]; ok {
		return n, nil
	}
	raw, err := os.ReadFile(sd.rawPath)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	schemas := mapChild(mapChild(rootOf(&doc), "components"), "schemas")
	if schemas == nil {
		return nil, fmt.Errorf("%s: missing components.schemas", sd.name)
	}
	schemaCache[sd.rawPath] = schemas
	return schemas, nil
}

// refLeaf returns the final component name of a "#/components/schemas/Foo" ref.
func refLeaf(ref string) string {
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[i+1:]
	}
	return ref
}

// resolveSchema follows a $ref (one level, within components.schemas) to the
// concrete schema node; a non-ref node is returned as-is.
func resolveSchema(schemas, node *yaml.Node) *yaml.Node {
	seen := map[string]bool{}
	for node != nil {
		if ref := scalarChild(node, "$ref"); ref != "" {
			leaf := refLeaf(ref)
			if seen[leaf] {
				return node
			}
			seen[leaf] = true
			node = mapChild(schemas, leaf)
			continue
		}
		return node
	}
	return node
}

// schemaFields returns the ordered property names of an object schema, flattening
// allOf/anyOf/oneOf unions (dedup, first-seen order) and following $refs, then
// hoisting required fields (in property order) ahead of the optionals. Mirrors
// the reference generator: a closed method signature lists required params first,
// then optionals, both in schema property order.
func schemaFields(schemas, node *yaml.Node) []string {
	node = resolveSchema(schemas, node)
	if node == nil {
		return nil
	}
	// The required set uses the SAME semantics as schemaFieldTypes (allOf merges,
	// anyOf/oneOf INTERSECT — a field required only if EVERY variant requires it),
	// so the required-first ORDER agrees with the field TYPES (a field typed as
	// optional *T is never hoisted ahead as if required). This matches the oracle's
	// order (calling.dial: from/to required first, then caller_id…url…swml in
	// property order). Names are collected in property order (dedup, first-seen).
	_, required, _ := schemaFieldTypes(schemas, node)
	var order []string
	seen := map[string]bool{}
	var walk func(n *yaml.Node)
	walk = func(n *yaml.Node) {
		n = resolveSchema(schemas, n)
		if n == nil {
			return
		}
		for _, comb := range []string{"allOf", "anyOf", "oneOf"} {
			if lst := mapChild(n, comb); lst != nil && lst.Kind == yaml.SequenceNode {
				for _, br := range lst.Content {
					walk(br)
				}
			}
		}
		if props := mapChild(n, "properties"); props != nil && props.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(props.Content); i += 2 {
				name := props.Content[i].Value
				if !seen[name] {
					seen[name] = true
					order = append(order, name)
				}
			}
		}
	}
	walk(node)
	// required-first (in property order), then the rest.
	var out []string
	for _, n := range order {
		if required[n] {
			out = append(out, n)
		}
	}
	for _, n := range order {
		if !required[n] {
			out = append(out, n)
		}
	}
	return out
}

// operationBodyFields returns the ordered typed-param field names of an operation's
// requestBody schema (empty if the op declares no body). The second result is true
// when the body is a top-level oneOf union of DISTINCT request types: the reference
// generator does NOT explode such a body — it passes a single `body` object — so the
// caller emits the loose single-`data` form instead of exploded params.
func operationBodyFields(sd *specDoc, op opInfo) (fields []string, oneOfBody bool, err error) {
	schemas, err := componentsSchemas(sd)
	if err != nil {
		return nil, false, err
	}
	// Re-read the op node to reach requestBody.content.*.schema.
	body := mapChild(sd.rawOp(op), "requestBody")
	if body == nil {
		return nil, false, nil
	}
	content := mapChild(body, "content")
	if content == nil || content.Kind != yaml.MappingNode || len(content.Content) < 2 {
		return nil, false, nil
	}
	// first media type
	sch := mapChild(content.Content[1], "schema")
	if sch == nil {
		return nil, false, nil
	}
	resolved := resolveSchema(schemas, sch)
	if isUnionBody(resolved) {
		return nil, true, nil
	}
	return schemaFields(schemas, sch), false, nil
}

// isUnionBody reports whether an operation's request-body schema is a top-level
// oneOf/anyOf union of DISTINCT request types (e.g. CreateManagedCampaignRequest |
// CreatePartnerCampaignRequest, or a CallFlow deploy oneOf). The reference
// generator does NOT explode such a body — it passes a single whole-body object —
// so the operation method keeps the loose single-`data` form. (A command's `params`
// sub-schema union IS flattened+exploded — that path does not go through here.)
func isUnionBody(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	for _, comb := range []string{"oneOf", "anyOf"} {
		u := mapChild(node, comb)
		if u != nil && u.Kind == yaml.SequenceNode && len(u.Content) > 0 {
			return true
		}
	}
	return false
}

// rawOp returns the raw operation node (verb map) for an opInfo, re-reading the
// spec doc's paths. Used to reach requestBody schema refs.
func (sd *specDoc) rawOp(op opInfo) *yaml.Node {
	raw, err := os.ReadFile(sd.rawPath)
	if err != nil {
		return nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	paths := mapChild(rootOf(&doc), "paths")
	if paths == nil {
		return nil
	}
	pathNode := mapChild(paths, op.path)
	if pathNode == nil {
		return nil
	}
	return mapChild(pathNode, op.verb)
}

// commandFields returns, for a command-dispatch command schema, whether the
// request carries a top-level `id` (the call_id positional) plus the ordered
// typed field names of its `params` sub-schema (union-flattened).
func commandFields(sd *specDoc, requestSchema string) (hasID bool, fields []string, err error) {
	schemas, err := componentsSchemas(sd)
	if err != nil {
		return false, nil, err
	}
	sch := mapChild(schemas, requestSchema)
	if sch == nil {
		return false, nil, fmt.Errorf("command request schema %q not in components.schemas", requestSchema)
	}
	sch = resolveSchema(schemas, sch)
	props := mapChild(sch, "properties")
	if mapChild(props, "id") != nil {
		hasID = true
	}
	params := mapChild(props, "params")
	if params != nil {
		fields = schemaFields(schemas, params)
	}
	return hasID, fields, nil
}

// goParamName maps a wire field name to a Go parameter identifier: snake->camel,
// with Go-keyword collisions escaped (§5). E.g. "from" -> "from_", "status_url"
// -> "statusURL"-ish; here we keep it simple (camelCase) since the param name is
// cosmetic — the enumerator snake_cases it back and the diff ignores names.
func goParamName(field string) string {
	parts := strings.Split(field, "_")
	var b strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			b.WriteString(p)
		} else {
			b.WriteString(strings.ToUpper(p[:1]) + p[1:])
		}
	}
	s := b.String()
	if s == "" {
		s = "field"
	}
	return escapeIdent(s)
}

// structFieldName maps a wire field name to an EXPORTED Go struct-field identifier
// for a params struct (§5/§4a). It reuses goParamName (snake->camel, keyword-safe)
// and upper-cases the first rune so the field is exported — e.g. "query_string" ->
// "QueryString", "from" -> "From_" (goParamName escapes the keyword to "from_",
// which pascals to "From_"). Crucially the enumerator's goNameToSnake round-trips
// this back to the SAME wire/snake name the old flat-positional form recorded
// ("From_" -> "from_"), keeping port_signatures.json byte-identical (drift 0).
func structFieldName(field string) string {
	s := goParamName(field)
	r := []rune(s)
	if len(r) > 0 && r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 32
	}
	return string(r)
}

// ---------------------------------------------------------------------------
// Client tree (§8): resolve flat vs containered placement and emit the
// namespace container structs + constructors + the flat RestClient wiring.
//
// A resource is CONTAINERED when its spec has a whole-spec x-sdk-namespace.attr
// OR the resource carries x-sdk-resource.namespace (cross-spec: logs/registry).
// Otherwise it is FLAT (client.<attr>). The accessor name = attr override else
// the snake->Pascal of the class with the container prefix stripped.
// ---------------------------------------------------------------------------

type placedResource struct {
	specName  string // x-sdk-resource.name
	goStruct  string // hand Go struct name
	container string // "" for flat, else the container attr (fabric/video/…)
	accessor  string // field/accessor name in the client or container
	sd        *specDoc
	rm        *resourceMarkup
}

func resolvePlacement(specs []*specDoc) []placedResource {
	var out []placedResource
	for _, sd := range specs {
		for _, pi := range sd.paths {
			rm := pi.res
			if rm == nil || rm.exclude || rm.name == "" {
				continue
			}
			container := rm.namespace
			if container == "" {
				container = sd.namespaceAttr
			}
			accessor := rm.attr
			out = append(out, placedResource{
				specName:  rm.name,
				goStruct:  goStructName[rm.name],
				container: container,
				accessor:  accessor,
				sd:        sd,
				rm:        rm,
			})
		}
	}
	return out
}

// containerInfo describes one client namespace container: its Go struct type,
// the RestClient field it is exposed as, and the ordered fields it holds.
// Keyed by the placement `container` attr (fabric/video/logs/registry/project/
// datasphere). The container struct/field names reproduce the CURRENT hand
// FabricNamespace/VideoNamespace/… surface (the DRIFT parity target) — the
// adoption turn swaps the generated tree in with these exact names so
// rest_client.go and examples keep compiling.
type containerInfo struct {
	structName string // e.g. "FabricNamespace"
	clientAttr string // RestClient field, e.g. "Fabric"
}

// containers maps a placement container attr -> its Go container type + the
// RestClient field name. Order of client fields follows containerOrder below.
var containers = map[string]containerInfo{
	"fabric":     {structName: "FabricNamespace", clientAttr: "Fabric"},
	"video":      {structName: "VideoNamespace", clientAttr: "Video"},
	"logs":       {structName: "LogsNamespace", clientAttr: "Logs"},
	"registry":   {structName: "RegistryNamespace", clientAttr: "Registry"},
	"project":    {structName: "ProjectNamespace", clientAttr: "Project"},
	"datasphere": {structName: "DatasphereNamespace", clientAttr: "Datasphere"},
}

// containerFieldName maps a contained resource's Go struct name to the field
// name it takes inside its container. Reproduces the hand container field
// names (irregular: GenericResources -> Resources, VideoRooms -> Rooms,
// MessageLogs -> Messages, RegistryBrands -> Brands, …). A containered resource
// with no entry falls back to its Go struct name.
var containerFieldName = map[string]string{
	// fabric
	"GenericResources":         "Resources",
	"FabricAddresses":          "Addresses",
	"FabricTokens":             "Tokens",
	"CxmlApplicationsResource": "CXMLApplications",
	"CallFlowsResource":        "CallFlows",
	"ConferenceRoomsResource":  "ConferenceRooms",
	"SubscribersResource":      "Subscribers",
	// video
	"VideoRooms":            "Rooms",
	"VideoRoomTokens":       "RoomTokens",
	"VideoRoomSessions":     "RoomSessions",
	"VideoRoomRecordings":   "RoomRecordings",
	"VideoConferences":      "Conferences",
	"VideoConferenceTokens": "ConferenceTokens",
	"VideoStreams":          "Streams",
	// logs
	"MessageLogs":    "Messages",
	"VoiceLogs":      "Voice",
	"FaxLogs":        "Fax",
	"ConferenceLogs": "Conferences",
	// registry
	"RegistryBrands":    "Brands",
	"RegistryCampaigns": "Campaigns",
	"RegistryOrders":    "Orders",
	"RegistryNumbers":   "Numbers",
	// project
	"ProjectTokens": "Tokens",
	// datasphere
	"DatasphereDocuments": "Documents",
}

func fieldNameFor(goStruct string) string {
	if v, ok := containerFieldName[goStruct]; ok {
		return v
	}
	return goStruct
}

// flatClientField maps a FLAT resource's Go struct name to the RestClient field
// it is exposed as (the hand RestClient names: PhoneNumbers, SIPProfile, MFA,
// Calling, …). Reproduces the committed rest_client.go field names so the
// generated tree drops in with no example churn.
var flatClientField = map[string]string{
	"AddressesNamespace":       "Addresses",
	"ImportedNumbersNamespace": "ImportedNumbers",
	"LookupNamespace":          "Lookup",
	"MFANamespace":             "MFA",
	"NumberGroupsNamespace":    "NumberGroups",
	"PhoneNumbersNamespace":    "PhoneNumbers",
	"QueuesNamespace":          "Queues",
	"RecordingsNamespace":      "Recordings",
	"ShortCodesNamespace":      "ShortCodes",
	"SIPProfileNamespace":      "SIPProfile",
	"VerifiedCallersNamespace": "VerifiedCallers",
	"CallingNamespace":         "Calling",
	"ChatNamespace":            "Chat",
	"PubSubNamespace":          "PubSub",
}

// clientFieldOrder is the field order on the generated resource tree (flat
// resources first, then containers), matching the committed rest_client.go
// layout so the diff stays reviewable.
var clientFieldOrder = []string{
	"Fabric", "Calling",
	"PhoneNumbers", "Addresses", "Queues", "Recordings", "NumberGroups",
	"VerifiedCallers", "SIPProfile", "Lookup", "ShortCodes", "ImportedNumbers", "MFA",
	"Registry", "Datasphere", "Video", "Logs", "Project", "PubSub", "Chat",
}

// emitClientTree emits the generated REST client tree (§8) as TWO source files.
//
// The FIRST return (package namespaces, client_tree_generated.go) holds the
// per-namespace container structs + their New<Container>Namespace constructors,
// each wired by calling the per-resource New<Struct> constructors. These
// reference namespaces.* types directly so they live in package namespaces.
//
// The SECOND return (package rest, rest_tree_generated.go) holds the
// _GeneratedResourceTree struct (fields typed as namespaces.* — qualified via the
// namespaces import) + its wireGeneratedTree(client namespaces.HTTPClient) method.
// It lives in package rest so the hand RestClient can EMBED _GeneratedResourceTree
// (Go cannot embed a cross-package underscore-unexported type, so this cannot stay
// in package namespaces). The leading underscore keeps it off the enumerated
// oracle surface (mirroring TS's _GeneratedResourceTree). The hand RestClient
// keeps only the non-spec-derivable bits (auth, HTTP construction, env-var
// handling, the httpAdapter import-cycle breaker).
func emitClientTree(placed []placedResource) (namespacesFile, restFile string) {
	var b strings.Builder
	fmt.Fprintf(&b, genHeader,
		"The generated REST client-namespace tree (§8): container structs +\n// constructors, wired by calling the per-resource New<Struct> constructors.\n// Placement resolved from x-sdk-namespace.attr + per-resource\n// x-sdk-resource.namespace/attr; base paths per §4. The _GeneratedResourceTree\n// + wireGeneratedTree live in package rest (rest_tree_generated.go) so the hand\n// RestClient can embed the tree (Go forbids embedding a cross-package\n// underscore-unexported type).")
	b.WriteString("\n")

	// Group containered resources by container attr, preserving encounter order
	// (spec order). Also collect flat resources.
	type member struct {
		field    string // field name inside the container
		goStruct string
	}
	byContainer := map[string][]member{}
	var containerAttrs []string
	seenContainer := map[string]bool{}
	type flat struct {
		field    string
		goStruct string
	}
	var flats []flat
	seenFlat := map[string]bool{}
	for _, p := range placed {
		if p.container == "" {
			f := flatClientField[p.goStruct]
			if f == "" {
				f = p.goStruct
			}
			if seenFlat[f] {
				continue
			}
			seenFlat[f] = true
			flats = append(flats, flat{field: f, goStruct: p.goStruct})
			continue
		}
		if !seenContainer[p.container] {
			seenContainer[p.container] = true
			containerAttrs = append(containerAttrs, p.container)
		}
		fld := p.accessor
		if fld == "" {
			fld = fieldNameFor(p.goStruct)
		} else {
			fld = pascal(fld)
		}
		byContainer[p.container] = append(byContainer[p.container], member{field: fld, goStruct: p.goStruct})
	}
	sort.Strings(containerAttrs)

	// Emit each container struct + constructor.
	for _, attr := range containerAttrs {
		ci, ok := containers[attr]
		if !ok {
			// A container attr with no registered Go container type is a real
			// finding (add to the containers table), not silently skipped.
			panic(fmt.Sprintf("container attr %q has no Go container type (add to containers table)", attr))
		}
		members := byContainer[attr]
		fmt.Fprintf(&b, "// %s groups the %s namespace resources (§8 container).\n", ci.structName, attr)
		fmt.Fprintf(&b, "type %s struct {\n", ci.structName)
		for _, m := range members {
			fmt.Fprintf(&b, "\t%s *%s\n", m.field, m.goStruct)
		}
		b.WriteString("}\n\n")
		fmt.Fprintf(&b, "// New%s constructs the %s container, wiring each resource by\n// calling its per-resource constructor (base paths baked in per §4).\n", ci.structName, ci.structName)
		fmt.Fprintf(&b, "func New%s(client HTTPClient) *%s {\n\treturn &%s{\n", ci.structName, ci.structName, ci.structName)
		for _, m := range members {
			fmt.Fprintf(&b, "\t\t%s: New%s(client),\n", m.field, m.goStruct)
		}
		b.WriteString("\t}\n}\n\n")
	}

	namespacesFile = b.String()

	// --- SECOND FILE (package rest): rest_tree_generated.go ---
	//
	// The _GeneratedResourceTree struct (fields typed as namespaces.* — the tree
	// lives in package rest so the hand RestClient can embed it) + its
	// wireGeneratedTree method, which takes a namespaces.HTTPClient and calls the
	// per-resource/container constructors (all in package namespaces, so qualified).
	var r strings.Builder
	fmt.Fprintf(&r, genHeaderPkg,
		"The generated REST resource tree (§8): the _GeneratedResourceTree the hand\n// RestClient embeds + its wireGeneratedTree method. Lives in package rest (not\n// namespaces) so the hand RestClient can embed the underscore-unexported tree —\n// Go forbids embedding a cross-package underscore-unexported type. The leading\n// underscore keeps it off the client's public API surface.",
		"rest")
	r.WriteString("\nimport \"github.com/signalwire/signalwire-go/pkg/rest/namespaces\"\n\n")

	// Build lookup: field -> namespaces-qualified Go type expression.
	fieldType := map[string]string{}
	for _, f := range flats {
		fieldType[f.field] = "*namespaces." + f.goStruct
	}
	for _, attr := range containerAttrs {
		ci := containers[attr]
		fieldType[ci.clientAttr] = "*namespaces." + ci.structName
	}

	r.WriteString("// _GeneratedResourceTree holds every flat REST resource plus the namespace\n")
	r.WriteString("// containers. The hand RestClient embeds it and calls wireGeneratedTree; the\n")
	r.WriteString("// leading underscore keeps it off the client's public API surface.\n")
	r.WriteString("type _GeneratedResourceTree struct {\n")
	// Emit in clientFieldOrder; fail loud if any generated field is not ordered
	// and any ordered field is missing (keeps the two lists in lockstep).
	ordered := map[string]bool{}
	for _, f := range clientFieldOrder {
		ordered[f] = true
		t, ok := fieldType[f]
		if !ok {
			panic(fmt.Sprintf("clientFieldOrder lists %q but no generated resource/container produces it", f))
		}
		fmt.Fprintf(&r, "\t%s %s\n", f, t)
	}
	for f := range fieldType {
		if !ordered[f] {
			panic(fmt.Sprintf("generated field %q is not in clientFieldOrder (add it)", f))
		}
	}
	r.WriteString("}\n\n")

	// Emit the wire method (constructors are in package namespaces, so qualified).
	r.WriteString("// wireGeneratedTree constructs every flat resource + container from the given\n")
	r.WriteString("// HTTPClient. The hand RestClient calls this after building its HTTP layer.\n")
	r.WriteString("func (t *_GeneratedResourceTree) wireGeneratedTree(client namespaces.HTTPClient) {\n")
	flatCtor := map[string]string{} // field -> constructor call
	for _, f := range flats {
		flatCtor[f.field] = "namespaces.New" + f.goStruct + "(client)"
	}
	containerCtor := map[string]string{}
	for _, attr := range containerAttrs {
		ci := containers[attr]
		containerCtor[ci.clientAttr] = "namespaces.New" + ci.structName + "(client)"
	}
	for _, f := range clientFieldOrder {
		if c, ok := flatCtor[f]; ok {
			fmt.Fprintf(&r, "\tt.%s = %s\n", f, c)
		} else if c, ok := containerCtor[f]; ok {
			fmt.Fprintf(&r, "\tt.%s = %s\n", f, c)
		}
	}
	r.WriteString("}\n")
	restFile = r.String()
	return namespacesFile, restFile
}

// ---------------------------------------------------------------------------
// Generated StructTable slice — the namespaces.* entries for internal/surface.
//
// Reproduces the exact Class-name spelling + goMethod->pyMethod maps currently
// in tables.go for the REST resources, derived from the markup: the Python
// class name is per the pyClassName table (Python's actual class names, which
// differ from the Go struct names), the module from the spec, and each method
// pair from the resolved Go method name + the markup snake method name.
// ---------------------------------------------------------------------------

// The Python class name is the spec x-sdk-resource.name VERBATIM. Python's fully
// generated REST layer names each subclass exactly after the markup name
// (AiAgents, CallFlows, CxmlApplications, SipEndpoints, Subscribers,
// FabricAddresses, GenericResources, …). Python does NOT apply Go's initialism
// casing — so the Class: string is DISTINCT from the Go struct name (AIAgents,
// SIPEndpoints). PORT_PHILOSOPHY_GO.md: do not touch the Class:/Module: strings.
// Hence there is no pyClassName table: Class == rm.name.

// pyModule returns the Python module string for a resource. The Python REST
// surface is now FULLY GENERATED: every resource lives in a per-spec-namespace
// generated module `signalwire.rest.namespaces.<ns>_resources_generated`, where
// <ns> is the spec directory name with "relay-rest" folded to "relay_rest"
// (a Python identifier). Registry resources live in the relay-rest spec, so
// they too are in relay_rest_resources_generated — the container/attr markup
// does NOT change the module. This is the Python oracle target
// (python_signatures.json), NOT the old per-hand-file modules.
func pyModule(specName, specDir string) string {
	return "signalwire.rest.namespaces." + pyModuleNS(specDir) + "_resources_generated"
}

// pyModuleNS folds a spec directory name to its Python module namespace stem.
func pyModuleNS(specDir string) string {
	return strings.ReplaceAll(specDir, "-", "_")
}

// implicitBaseMethods returns the typed-override method subset a base contributes
// to a generated Python subclass (the methods the oracle records BEYOND the
// resource's own declared markup methods). Derived from and verified against the
// Python oracle (python_surface.json): both a FabricResource and a plain
// CrudResource subclass override exactly create/update; a ReadResource
// contributes NOTHING implicit (its list/get are untyped inherited base
// methods); a BaseResource contributes nothing implicit.
// The untyped inherited base methods (ReadResource.list/get, CrudResource.list/
// get/delete, FabricResource.list/get/delete/list_addresses) are intentionally
// NOT included — the oracle does not record them on the subclass. (Verified: not
// one of the 19 CrudResource/CrudWithAddresses-embedding subclasses exposes
// `delete` in the reference; each records exactly create+update. `delete` in the
// reference belongs only to resources that DECLARE it in their markup methods,
// which flow in separately below.)
func implicitBaseMethods(base string) []string {
	switch base {
	case "CrudResource":
		return []string{"create", "update"}
	case "FabricResource":
		return []string{"create", "update"}
	default: // ReadResource, BaseResource (and command-dispatch, handled separately)
		// ReadResource contributes NOTHING implicit: its list/get are untyped
		// inherited base methods that the Python oracle records on the ReadResource
		// BASE (see the internal/surface/tables.go rest.CrudResource->ReadResource
		// adapter), NOT re-declared on the concrete subclass. Attributing them to
		// the subclass over-counts vs the reference (which lists only __init__ on a
		// pure ReadResource subclass) — same reason CrudResource.list/get are
		// excluded above. A resource that genuinely re-surfaces get/list carries
		// them in its explicit markup methods:, which flow in separately below.
		return nil
	}
}

// emitStructTableSlice writes a Go source file with the generated REST
// StructTable entries (a map fragment a later turn folds into tables.go).
func emitStructTableSlice(specs []*specDoc, bases map[string]*baseSpec) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, genHeaderPkg,
		"Generated REST StructTable entries (the namespaces.* keys) — reproduces\n// the Class-name spelling + goMethod->pyMethod maps for the REST resources\n// currently hand-maintained in tables.go, derived from the x-sdk-* markup.",
		"surface")
	b.WriteString("\n// GeneratedRESTStructTable holds the REST namespace entries.\n")
	b.WriteString("var GeneratedRESTStructTable = map[string][]ClassTarget{\n")

	type entry struct {
		key     string
		module  string
		class   string
		methods [][2]string // goName, pyName ordered
	}
	var entries []entry
	seen := map[string]bool{}
	for _, sd := range specs {
		for _, pi := range sd.paths {
			rm := pi.res
			if rm == nil || rm.exclude || rm.name == "" {
				continue
			}
			goStruct, ok := goStructName[rm.name]
			if !ok {
				return "", fmt.Errorf("StructTable: no Go struct for %q", rm.name)
			}
			key := "namespaces." + goStruct
			if seen[key] {
				continue
			}
			seen[key] = true
			// Module = <ns>_resources_generated (from the spec dir); Class = the
			// x-sdk-resource.name VERBATIM (Python's actual subclass name).
			e := entry{key: key, module: pyModule(rm.name, sd.name), class: rm.name}
			// Every generated Python subclass defines __init__.
			e.methods = append(e.methods, [2]string{"New" + goStruct, "__init__"})
			if rm.kind == "command-dispatch" {
				// command-dispatch (calling): methods from the discriminator mapping.
				mapping, err := loadDiscriminatorMapping(sd, rm.request)
				if err != nil {
					return "", err
				}
				for _, cmd := range mapping {
					e.methods = append(e.methods, [2]string{callingMethodName[cmd], commandPyName(cmd)})
				}
			} else {
				// The generated Python subclass carries, in addition to its declared
				// markup methods, the typed-override subset that its base contributes
				// (create/update for FabricResource; create/update/delete for
				// CrudResource; list/get for ReadResource; nothing for BaseResource).
				// Inherited-but-untyped base methods (e.g. CrudResource.list/get,
				// FabricResource.list/get/delete/list_addresses) are NOT emitted as
				// subclass methods — matching the Python oracle exactly. Any such
				// method a resource genuinely exposes appears in its markup methods:.
				emitted := map[string]bool{}
				addPy := func(goName, pyName string) {
					if emitted[pyName] {
						return
					}
					emitted[pyName] = true
					e.methods = append(e.methods, [2]string{goName, pyName})
				}
				for _, py := range implicitBaseMethods(rm.base) {
					gm, err := resolveMethodName(rm.name, py)
					if err != nil {
						return "", err
					}
					addPy(gm, py)
				}
				for _, mm := range rm.methods {
					gm, err := resolveMethodName(rm.name, mm.name)
					if err != nil {
						return "", err
					}
					addPy(gm, mm.name)
				}
				for _, sm := range rm.setMethods {
					addPy(setMethodGoName(sm.name), sm.name)
				}
			}
			entries = append(entries, e)
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].key < entries[j].key })
	for _, e := range entries {
		fmt.Fprintf(&b, "\t%q: {{\n", e.key)
		fmt.Fprintf(&b, "\t\tModule: %q, Class: %q,\n", e.module, e.class)
		b.WriteString("\t\tMethods: map[string]string{\n")
		// stable order by Go name
		ms := make([][2]string, len(e.methods))
		copy(ms, e.methods)
		sort.SliceStable(ms, func(i, j int) bool { return ms[i][0] < ms[j][0] })
		for _, m := range ms {
			fmt.Fprintf(&b, "\t\t\t%q: %q,\n", m[0], m[1])
		}
		b.WriteString("\t\t},\n\t}},\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

// commandPyName derives the Python method name for a command (strip leading
// "calling." domain prefix, dots -> underscores; dial/update stay bare).
func commandPyName(cmd string) string {
	s := strings.TrimPrefix(cmd, "calling.")
	return strings.ReplaceAll(s, ".", "_")
}

const genHeaderPkg = `// Code generated by cmd/generate-rest; DO NOT EDIT.
//
// AUTO-GENERATED from porting-sdk/rest-apis/ (x-sdk-* markup) — regenerate with:
//   go run ./cmd/generate-rest
//
// %s

package %s
`

// ---------------------------------------------------------------------------
// Driver.
// ---------------------------------------------------------------------------

// actualSpecDirs are the 12 real REST spec directories (registry has no own
// dir — its resources live inside relay-rest via namespace: registry).
var actualSpecDirs = []string{
	"relay-rest", "fabric", "calling", "video", "datasphere",
	"logs", "message", "voice", "fax", "project", "chat", "pubsub",
}

func gofmtSrc(src string) ([]byte, error) {
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return nil, fmt.Errorf("gofmt: %w\n---\n%s", err, src)
	}
	return formatted, nil
}

func findRepoRoot(start string) (string, error) {
	cur := start
	for {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("no go.mod above %s", start)
		}
		cur = parent
	}
}

func resolvePortingSDK(repoRoot string) (string, error) {
	if p := os.Getenv("PORTING_SDK"); p != "" {
		if _, err := os.Stat(filepath.Join(p, "rest-apis")); err == nil {
			return p, nil
		}
	}
	cand := filepath.Join(repoRoot, "..", "porting-sdk")
	if _, err := os.Stat(filepath.Join(cand, "rest-apis")); err == nil {
		return filepath.Abs(cand)
	}
	return "", fmt.Errorf("porting-sdk not found (set $PORTING_SDK or clone adjacent)")
}

// emitSpecFile emits the generated resource file for one spec.
func emitSpecFile(sd *specDoc, bases map[string]*baseSpec) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, genHeader,
		fmt.Sprintf("Generated REST resources for the %q namespace spec.", sd.name))
	b.WriteString("\n")
	emitted := 0
	for _, pi := range sd.paths {
		rm := pi.res
		if rm == nil || rm.exclude || rm.name == "" {
			continue
		}
		if err := emitResource(&b, rm, sd, bases); err != nil {
			return "", err
		}
		emitted++
	}
	if emitted == 0 {
		return "", nil
	}
	return b.String(), nil
}

func run() error {
	check := flag.Bool("check", false, "GEN-FRESH: exit non-zero if any generated file is stale")
	out := flag.String("out", "", "emit to this directory instead of the repo tree (scratch mode)")
	flag.Parse()

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	repoRoot, err := findRepoRoot(cwd)
	if err != nil {
		return err
	}
	psdk, err := resolvePortingSDK(repoRoot)
	if err != nil {
		if *check {
			return fmt.Errorf("generate-rest --check: %w", err)
		}
		fmt.Fprintf(os.Stderr, "generate-rest: %v — skipping (committed files kept)\n", err)
		return nil
	}

	restOverlay, err = overlay.Load(psdk)
	if err != nil {
		return err
	}

	bases, err := loadBases(psdk)
	if err != nil {
		return err
	}

	var specs []*specDoc
	for _, ns := range actualSpecDirs {
		sd, err := loadSpec(psdk, ns)
		if err != nil {
			return err
		}
		specs = append(specs, sd)
	}

	// Build outputs: one file per spec (that has resources) + client tree + table.
	// Each generated file has a real home in the repo tree:
	//   pkg/rest/namespaces/<ns>_resources_generated.go   (package namespaces)
	//   pkg/rest/namespaces/client_tree_generated.go       (package namespaces)
	//   pkg/rest/rest_tree_generated.go                    (package rest)
	//   internal/surface/struct_table_generated.go         (package surface)
	// In scratch mode (--out DIR) every file is written flat into DIR (for the
	// GEN-FRESH-independent scratch join verify); the base names stay distinct.
	type outFile struct {
		path string
		src  string
	}
	nsDir := filepath.Join(repoRoot, "pkg", "rest", "namespaces")
	restDir := filepath.Join(repoRoot, "pkg", "rest")
	surfaceDir := filepath.Join(repoRoot, "internal", "surface")
	// dir picks the real-tree directory for a base name, or the scratch --out dir.
	dir := func(realDir string) string {
		if *out != "" {
			return *out
		}
		return realDir
	}
	var outs []outFile
	for _, sd := range specs {
		src, err := emitSpecFile(sd, bases)
		if err != nil {
			return err
		}
		if src != "" {
			fn := strings.ReplaceAll(sd.name, "-", "_") + "_resources_generated.go"
			outs = append(outs, outFile{path: filepath.Join(dir(nsDir), fn), src: src})
		}
		// Per-spec typed wire types module (<ns>_types_generated.go): one Go type
		// per components/schemas entry. Emitted for every spec that has schemas.
		typesSrc, err := emitTypesFile(sd)
		if err != nil {
			return err
		}
		if typesSrc != "" {
			tfn := strings.ReplaceAll(sd.name, "-", "_") + "_types_generated.go"
			outs = append(outs, outFile{path: filepath.Join(dir(nsDir), tfn), src: typesSrc})
		}
	}
	// swml-webhooks: a types-ONLY namespace (its openapi.yaml declares no servers
	// and paths:{}, so it has no x-sdk-resource resources — but it DOES carry
	// components/schemas the reference emits as swml_webhooks_types_generated types).
	// The full loadSpec path requires servers/paths, so emit its types module via a
	// minimal types-only spec doc (emitTypesFile only needs name + rawPath). Matches
	// the reference (python swml_webhooks_types_generated.py, TS PlatformContracts).
	swmlWebhooksSpec := &specDoc{
		name:    "swml-webhooks",
		rawPath: filepath.Join(psdk, "rest-apis", "swml-webhooks", "openapi.yaml"),
	}
	if _, err := os.Stat(swmlWebhooksSpec.rawPath); err == nil {
		swSrc, err := emitTypesFile(swmlWebhooksSpec)
		if err != nil {
			return err
		}
		if swSrc != "" {
			tfn := strings.ReplaceAll(swmlWebhooksSpec.name, "-", "_") + "_types_generated.go"
			outs = append(outs, outFile{path: filepath.Join(dir(nsDir), tfn), src: swSrc})
		}
	}

	// NOTE: the RELAY WS protocol types (pkg/relay/protocol_types_generated.go) are
	// no longer emitted here — they moved to the standalone cmd/generate-relay-protocol
	// command (one of the fixed 5 cross-port generators). This generator emits only
	// the REST resource/types/client-tree/struct-table surface.

	placed := resolvePlacement(specs)
	nsTree, restTree := emitClientTree(placed)
	outs = append(outs, outFile{path: filepath.Join(dir(nsDir), "client_tree_generated.go"), src: nsTree})
	outs = append(outs, outFile{path: filepath.Join(dir(restDir), "rest_tree_generated.go"), src: restTree})
	tbl, err := emitStructTableSlice(specs, bases)
	if err != nil {
		return err
	}
	outs = append(outs, outFile{path: filepath.Join(dir(surfaceDir), "struct_table_generated.go"), src: tbl})

	var stale []string
	for _, o := range outs {
		formatted, err := gofmtSrc(o.src)
		if err != nil {
			return fmt.Errorf("%s: %w", o.path, err)
		}
		if *check {
			existing, err := os.ReadFile(o.path)
			if err != nil || !bytes.Equal(existing, formatted) {
				stale = append(stale, o.path)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(o.path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(o.path, formatted, 0o644); err != nil {
			return err
		}
		fmt.Printf("generated %s\n", o.path)
	}
	if *check && len(stale) > 0 {
		fmt.Fprintf(os.Stderr, "\nGEN-FRESH FAIL: %d generated REST file(s) stale — run `go run ./cmd/generate-rest` and commit:\n", len(stale))
		for _, f := range stale {
			fmt.Fprintf(os.Stderr, "  - %s\n", f)
		}
		return fmt.Errorf("stale generated files")
	}
	if *check {
		fmt.Println("GEN-FRESH: generated REST files match the canonical specs.")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
