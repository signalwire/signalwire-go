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

	"gopkg.in/yaml.v3"
)

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

// pathParams returns the ordered {brace} param names in a path.
func pathParams(p string) []string {
	var out []string
	for {
		i := strings.Index(p, "{")
		if i < 0 {
			break
		}
		j := strings.Index(p[i:], "}")
		if j < 0 {
			break
		}
		out = append(out, p[i+1:i+j])
		p = p[i+j+1:]
	}
	return out
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
	var tail string
	switch {
	case writeVerb:
		if op.hasBody {
			params = append(params, "data map[string]any")
			tail = "data"
		}
	case verb == "get":
		// The hand loose surface takes query params on LIST/collection-style GETs
		// (path targets a collection: last segment is a literal, not a {param}),
		// and NOT on single-item Get(id) (path ends in {param}). A handful of
		// singleton/action GETs (SipProfile.Get, GetNextMember) target a literal
		// tail but take no query — opted out via noQueryGets.
		lastLiteral := len(segs) == 0 || !isBrace(segs[len(segs)-1])
		if lastLiteral && !noQueryGets[rm.name+"."+mm.name] {
			params = append(params, "params map[string]string")
			tail = "params"
		}
	}

	// Path expression.
	var pathCode string
	if sibling {
		// Absolute server-rooted path (§4 sibling): build the literal + args.
		pathCode = absolutePath(serverPath, op.path, idArgs)
	} else if len(pathExpr) == 0 {
		pathCode = "r.Base"
	} else {
		pathCode = "r.Path(" + strings.Join(pathExpr, ", ") + ")"
	}

	fmt.Fprintf(b, "func (r *%s) %s(%s) (map[string]any, error) {\n", recv, goName, strings.Join(params, ", "))
	switch verb {
	case "get":
		if tail == "params" {
			fmt.Fprintf(b, "\treturn r.HTTP.Get(%s, params)\n", pathCode)
		} else {
			fmt.Fprintf(b, "\treturn r.HTTP.Get(%s, nil)\n", pathCode)
		}
	case "post":
		if tail == "data" {
			fmt.Fprintf(b, "\treturn r.HTTP.Post(%s, data, nil)\n", pathCode)
		} else {
			fmt.Fprintf(b, "\treturn r.HTTP.Post(%s, nil, nil)\n", pathCode)
		}
	case "put":
		if tail == "data" {
			fmt.Fprintf(b, "\treturn r.HTTP.Put(%s, data)\n", pathCode)
		} else {
			fmt.Fprintf(b, "\treturn r.HTTP.Put(%s, nil)\n", pathCode)
		}
	case "patch":
		if tail == "data" {
			fmt.Fprintf(b, "\treturn r.HTTP.Patch(%s, data)\n", pathCode)
		} else {
			fmt.Fprintf(b, "\treturn r.HTTP.Patch(%s, nil)\n", pathCode)
		}
	case "delete":
		fmt.Fprintf(b, "\treturn r.HTTP.Delete(%s)\n", pathCode)
	}
	b.WriteString("}\n\n")
	return nil
}

func isBrace(s string) bool { return strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") }

// noQueryGets are singleton/action GET methods whose op path ends in a literal
// segment (so the list-style heuristic would add query params) but the hand
// surface takes NONE — keyed by "<specName>.<markupMethod>".
var noQueryGets = map[string]bool{
	"SipProfile.get":         true, // singleton project SIP profile
	"Queues.get_next_member": true, // single-member action (…/members/next)
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

func embedFor(rm *resourceMarkup) (embedInfo, error) {
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
	emb, err := embedFor(rm)
	if err != nil {
		return err
	}
	fmt.Fprintf(b, "// %s is generated from x-sdk-resource %q in the %s spec.\n", goName, rm.name, sd.name)
	fmt.Fprintf(b, "type %s struct {\n\t%s\n}\n\n", goName, emb.field)

	// ReadResource maps to the method-less Go `Resource` embed, so the base's
	// list+get are synthesized here (the hand code writes them out explicitly).
	if rm.base == "ReadResource" {
		base := rm.basePath(sd.serverPath)
		fmt.Fprintf(b, "func (r *%s) List(params map[string]string) (map[string]any, error) {\n\treturn r.HTTP.Get(r.Base, params)\n}\n\n", goName)
		fmt.Fprintf(b, "func (r *%s) Get(id string) (map[string]any, error) {\n\treturn r.HTTP.Get(r.Path(id), nil)\n}\n\n", goName)
		_ = base
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
	// required args become positional string params; optional args are noted.
	var params []string
	params = append(params, "sid string")
	var bodyLines []string
	bodyLines = append(bodyLines, fmt.Sprintf("\t\t%q: %q,", "call_handler", sm.handler))
	for _, a := range sm.args {
		if a.field == "" {
			return fmt.Errorf("%s.%s: arg %q missing field", rm.name, sm.name, a.name)
		}
		if a.required {
			params = append(params, escapeIdent(a.name)+" string")
			bodyLines = append(bodyLines, fmt.Sprintf("\t\t%q: %s,", a.field, escapeIdent(a.name)))
		}
	}
	params = append(params, "extra ...map[string]any")
	fmt.Fprintf(b, "func (r *%s) %s(%s) (map[string]any, error) {\n", recv, goName, strings.Join(params, ", "))
	b.WriteString("\tbody := map[string]any{\n")
	for _, l := range bodyLines {
		b.WriteString(l + "\n")
	}
	b.WriteString("\t}\n")
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
	mapping, err := loadDiscriminatorMapping(sd, rm.request)
	if err != nil {
		return err
	}
	op, ok := sd.opIndex["call-commands"]
	base := joinPath(sd.serverPath, "calls")
	if ok {
		base = joinPath(sd.serverPath, strings.TrimPrefix(op.path, "/"))
	}
	fmt.Fprintf(b, "// %s is generated from the command-dispatch x-sdk-resource %q (%s spec).\n", goName, rm.name, sd.name)
	fmt.Fprintf(b, "type %s struct {\n\tResource\n}\n\n", goName)
	fmt.Fprintf(b, "// basePath for %s is %q (baked into the constructor).\n\n", goName, base)
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
		if commandsWithoutID[cmd] {
			fmt.Fprintf(b, "func (c *%s) %s(params map[string]any) (map[string]any, error) {\n", goName, mName)
			fmt.Fprintf(b, "\treturn c.execute(%q, \"\", params)\n}\n\n", cmd)
		} else {
			fmt.Fprintf(b, "func (c *%s) %s(callID string, params map[string]any) (map[string]any, error) {\n", goName, mName)
			fmt.Fprintf(b, "\treturn c.execute(%q, callID, params)\n}\n\n", cmd)
		}
	}
	return nil
}

// loadDiscriminatorMapping returns the ordered command strings from the request
// schema's discriminator.mapping. Fail loud if absent (§9).
func loadDiscriminatorMapping(sd *specDoc, schemaName string) ([]string, error) {
	// Re-read the raw spec doc for components/schemas access.
	raw, err := os.ReadFile(sd.rawPath)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	root := rootOf(&doc)
	comps := mapChild(root, "components")
	schemas := mapChild(comps, "schemas")
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

// emitClientTree writes the placement report: the container groups and the flat
// resources. Because the hand FabricNamespace/VideoNamespace/etc. carry
// irregular wrapper types + field names + baked sub-paths, this emits a
// SUMMARY tree (container -> [accessor: goStruct]) that a later turn reconciles
// against rest_client.go, rather than re-deriving the hand constructors (which
// embed hand-only abstractions like AutoMaterializedWebhookResource).
func emitClientTree(placed []placedResource) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(genHeader,
		"The generated REST client-namespace placement tree (§8): which resources\n// are flat on the client vs grouped under a container, resolved from\n// x-sdk-namespace.attr + per-resource x-sdk-resource.namespace/attr."))
	b.WriteString("\n// ClientPlacement is one resolved resource placement.\n")
	b.WriteString("type ClientPlacement struct {\n\tSpecName  string\n\tGoStruct  string\n\tContainer string // \"\" = flat on the client\n\tAccessor  string\n}\n\n")
	b.WriteString("// GeneratedPlacements is the resolved REST client tree.\n")
	b.WriteString("var GeneratedPlacements = []ClientPlacement{\n")
	sorted := make([]placedResource, len(placed))
	copy(sorted, placed)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].container != sorted[j].container {
			return sorted[i].container < sorted[j].container
		}
		return sorted[i].specName < sorted[j].specName
	})
	for _, p := range sorted {
		fmt.Fprintf(&b, "\t{SpecName: %q, GoStruct: %q, Container: %q, Accessor: %q},\n",
			p.specName, p.goStruct, p.container, p.accessor)
	}
	b.WriteString("}\n")
	return b.String()
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
// Python oracle (python_signatures.json): a FabricResource subclass overrides
// create/update; a plain CrudResource overrides create/update/delete; a
// ReadResource surfaces list/get; a BaseResource contributes nothing implicit.
// The untyped inherited base methods (CrudResource.list/get,
// FabricResource.list/get/delete/list_addresses) are intentionally NOT included —
// the oracle does not record them on the subclass.
func implicitBaseMethods(base string) []string {
	switch base {
	case "ReadResource":
		return []string{"list", "get"}
	case "CrudResource":
		return []string{"create", "update", "delete"}
	case "FabricResource":
		return []string{"create", "update"}
	default: // BaseResource (and command-dispatch, handled separately)
		return nil
	}
}

// emitStructTableSlice writes a Go source file with the generated REST
// StructTable entries (a map fragment a later turn folds into tables.go).
func emitStructTableSlice(specs []*specDoc, bases map[string]*baseSpec) (string, error) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(genHeaderPkg,
		"Generated REST StructTable entries (the namespaces.* keys) — reproduces\n// the Class-name spelling + goMethod->pyMethod maps for the REST resources\n// currently hand-maintained in tables.go, derived from the x-sdk-* markup.",
		"surface"))
	b.WriteString("\n// GeneratedRESTStructTable holds the REST namespace entries.\n")
	b.WriteString("var GeneratedRESTStructTable = map[string][]ClassTarget{\n")

	type entry struct {
		key     string
		module  string
		class   string
		methods [][2]string // goName, pyName ordered
		newFn   string      // constructor Go name if flat (-> __init__)
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
	b.WriteString(fmt.Sprintf(genHeader,
		fmt.Sprintf("Generated REST resources for the %q namespace spec.", sd.name)))
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
	type outFile struct {
		path string
		src  string
	}
	outDir := filepath.Join(repoRoot, "pkg", "rest", "namespaces", "generated")
	if *out != "" {
		outDir = *out
	}
	var outs []outFile
	for _, sd := range specs {
		src, err := emitSpecFile(sd, bases)
		if err != nil {
			return err
		}
		if src == "" {
			continue
		}
		fn := strings.ReplaceAll(sd.name, "-", "_") + "_generated.go"
		outs = append(outs, outFile{path: filepath.Join(outDir, fn), src: src})
	}
	placed := resolvePlacement(specs)
	outs = append(outs, outFile{path: filepath.Join(outDir, "client_tree_generated.go"), src: emitClientTree(placed)})
	tbl, err := emitStructTableSlice(specs, bases)
	if err != nil {
		return err
	}
	outs = append(outs, outFile{path: filepath.Join(outDir, "struct_table_generated.go"), src: tbl})

	var stale []string
	if !*check {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return err
		}
	}
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
