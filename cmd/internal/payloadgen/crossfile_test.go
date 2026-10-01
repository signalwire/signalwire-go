package payloadgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The cross-file $ref resolver replaced last-path-segment-wins, which discarded the
// FILE part of `swaig-response.yaml#/components/schemas/SwaigResponse` and so could not
// tell a sibling-file schema from a same-file one. That silently emitted a Go type name
// nothing declared (`undefined: SwaigResponse`) or, worse, bound to an unrelated
// same-named local schema.
//
// These are NEGATIVE CONTROLS: each asserts that a specific way of getting the ref
// wrong FAILS LOUD and names both the file and the schema, rather than degrading to an
// opaque map (which looks exactly like successful resolution while erasing the type).
// A resolver only ever exercised on inputs it resolves proves nothing.

// writeSpecTree lays out a minimal porting-sdk-shaped spec root: <root>/swaig-specs/
// with the given file name -> contents. Returns the root.
func writeSpecTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "swaig-specs")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// responseSpec is a swaig-response.yaml declaring exactly the two envelope schemas.
const responseSpec = `openapi: 3.1.0
components:
  schemas:
    SwaigAction:
      type: object
      properties:
        stop: {type: boolean}
    SwaigResponse:
      type: object
      properties:
        response: {type: string}
`

// postPromptSpecFmt is a post-prompt.yaml whose single cross-file ref is a format
// placeholder, so each control can point it somewhere different.
func postPromptSpec(ref string) string {
	return `openapi: 3.1.0
components:
  schemas:
    PostPrompt:
      type: object
      properties:
        post_response:
          "$ref": "` + ref + `"
`
}

func TestCrossFileRefResolvesWhenFileAndSchemaBothExist(t *testing.T) {
	root := writeSpecTree(t, map[string]string{"swaig-response.yaml": responseSpec})
	src, err := EmitPostPrompt([]byte(postPromptSpec("swaig-response.yaml#/components/schemas/SwaigResponse")), root)
	if err != nil {
		t.Fatalf("resolve of a valid cross-file ref failed: %v", err)
	}
	// The Go FIELD names the sibling type (same package, so no import needed)...
	if !strings.Contains(src, "*SwaigResponse") {
		t.Errorf("emitted field does not name SwaigResponse:\n%s", src)
	}
	// ...and the canonical AUDIT tag names the module that DECLARES it, which is the
	// swaig_actions module (crossFileModules), NOT the emitting post_prompt one. This
	// is the assertion that catches a resolver that returns the right name from the
	// wrong file.
	want := "class:" + canonModule("swaig_actions") + ".SwaigResponse"
	if !strings.Contains(src, want) {
		t.Errorf("audit tag does not name the declaring module %q:\n%s", want, src)
	}
	if strings.Contains(src, "class:"+canonModule("post_prompt")+".SwaigResponse") {
		t.Errorf("audit tag claims the EMITTING module declares SwaigResponse:\n%s", src)
	}
}

func TestCrossFileRefNegativeControls(t *testing.T) {
	cases := []struct {
		name string
		ref  string
		// files overrides the spec tree; nil means "the normal one".
		files map[string]string
		// wantIn are substrings the error MUST contain — at minimum the file and the
		// schema, so a human reading CI knows what to fix without reproducing it.
		wantIn []string
	}{
		{
			name:   "schema missing from an existing registered file",
			ref:    "swaig-response.yaml#/components/schemas/NoSuchSchema",
			wantIn: []string{"NoSuchSchema", "swaig-response.yaml", "does not exist", "SwaigResponse"},
		},
		{
			name:   "file is not registered in crossFileModules",
			ref:    "some-other-spec.yaml#/components/schemas/SwaigResponse",
			wantIn: []string{"some-other-spec.yaml", "SwaigResponse", "unregistered"},
		},
		{
			name:   "registered file is absent from the spec tree",
			ref:    "swaig-request.yaml#/components/schemas/SwaigRequest",
			files:  map[string]string{"swaig-response.yaml": responseSpec},
			wantIn: []string{"swaig-request.yaml", "not found", "swaig-specs"},
		},
		{
			name:   "pointer is not a components/schemas pointer",
			ref:    "swaig-response.yaml#/definitions/SwaigResponse",
			wantIn: []string{"/definitions/SwaigResponse", "components/schemas"},
		},
		{
			name:   "pointer is empty",
			ref:    "swaig-response.yaml#/components/schemas/",
			wantIn: []string{"components/schemas"},
		},
		{
			name:   "pointer digs deeper than a schema name",
			ref:    "swaig-response.yaml#/components/schemas/SwaigResponse/properties/response",
			wantIn: []string{"components/schemas"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := tc.files
			if files == nil {
				files = map[string]string{"swaig-response.yaml": responseSpec}
			}
			root := writeSpecTree(t, files)
			src, err := EmitPostPrompt([]byte(postPromptSpec(tc.ref)), root)
			if err == nil {
				t.Fatalf("BAD REF RESOLVED SILENTLY — expected an error, got source:\n%s", src)
			}
			if src != "" {
				t.Errorf("source was returned alongside the error (it must be discarded): %q", src)
			}
			for _, want := range tc.wantIn {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error does not mention %q: %v", want, err)
				}
			}
		})
	}
}

// A cross-file ref with NO spec tree available must fail rather than pass unverified.
// The generator mains have a path (porting-sdk unresolvable, not --check) that keeps the
// committed files and skips; nothing may resolve a ref it cannot check.
func TestCrossFileRefWithoutSpecRootIsAnError(t *testing.T) {
	_, err := EmitPostPrompt([]byte(postPromptSpec("swaig-response.yaml#/components/schemas/SwaigResponse")), "")
	if err == nil {
		t.Fatal("a cross-file ref resolved with no spec root to verify it against")
	}
	if !strings.Contains(err.Error(), "swaig-response.yaml") {
		t.Errorf("error does not name the unverifiable file: %v", err)
	}
}

// A SAME-FILE ref keeps resolving locally: the resolver must not capture refs that were
// never cross-file. `#/components/schemas/X` has no file part.
func TestSameFileRefIsUnaffected(t *testing.T) {
	spec := `openapi: 3.1.0
components:
  schemas:
    Inner:
      type: object
      properties:
        a: {type: string}
    Outer:
      type: object
      properties:
        inner:
          "$ref": "#/components/schemas/Inner"
`
	src, err := EmitPostPrompt([]byte(spec), "")
	if err != nil {
		t.Fatalf("same-file ref failed with no spec root (it needs none): %v", err)
	}
	if !strings.Contains(src, "*Inner") {
		t.Errorf("same-file ref did not resolve to the local type:\n%s", src)
	}
	// The audit tag must name the EMITTING module for a local schema.
	if !strings.Contains(src, "class:"+canonModule("post_prompt")+".Inner") {
		t.Errorf("local schema's audit tag does not name the emitting module:\n%s", src)
	}
}

// A whole-file JSON ref (no `#` pointer) names a document this generator emits no types
// for, so an opaque map is correct and must NOT be routed to the resolver. But a JSON
// ref WITH a pointer is a schema reference — the reference generator's guard has a
// `"#" not in ref` clause that go's copy had dropped, which would have sent
// `x.json#/components/schemas/Y` to the opaque fallback instead of the resolver.
func TestWholeFileJSONRefIsOpaqueButPointedJSONRefIsNot(t *testing.T) {
	if !isWholeFileJSONRef("SWMLObject.json") {
		t.Error("a bare whole-file .json ref should be opaque")
	}
	if isWholeFileJSONRef("SWMLObject.json#/components/schemas/Thing") {
		t.Error("a .json ref WITH a pointer is a schema reference, not an opaque document")
	}
	if isWholeFileJSONRef("#/components/schemas/Thing") {
		t.Error("a same-file ref is not a whole-file json ref")
	}
	if !isCrossFileRef("SWMLObject.json#/components/schemas/Thing") {
		t.Error("a .json ref with a file part AND a pointer is cross-file")
	}
	if isCrossFileRef("#/components/schemas/Thing") {
		t.Error("a same-file ref is not cross-file")
	}
	if isCrossFileRef("SWMLObject.json") {
		t.Error("a ref with no pointer is not cross-file")
	}
	if isCrossFileRef("swaig-response.yaml#") {
		t.Error("a ref with an EMPTY pointer is not a resolvable cross-file ref")
	}
}

// post-prompt.yaml declares 15 schemas and none is named SwaigRequest, so the
// `if name == "SwaigRequest" { continue }` skip EmitPostPrompt used to carry was already
// dead. It was removed rather than generalised alongside the resolver (the reference
// generator's identical dead skip was removed in porting-sdk 4ddda70). If a future spec
// revision DID add a SwaigRequest schema to post-prompt.yaml, it would now be emitted
// into post_prompt_generated.go and collide with swaig_request_generated.go's copy in
// the same package — a compile error, which is the correct loud outcome, but this test
// pins the assumption so the failure is explained rather than discovered.
func TestPostPromptSpecDeclaresNoSwaigRequestSchema(t *testing.T) {
	// The sibling-checkout path, a constant so gosec sees no tainted input. $PORTING_SDK
	// is deliberately NOT consulted: this asserts a fact about the VENDORED spec, and the
	// skip below covers the checkout where it is elsewhere.
	const spec = "../../../../porting-sdk/swaig-specs/post-prompt.yaml"
	raw, err := os.ReadFile(spec)
	if err != nil {
		t.Skipf("porting-sdk spec tree unavailable at %s: %v", spec, err)
	}
	schemas, _, err := loadYAMLSchemas(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := schemas["SwaigRequest"]; ok {
		t.Error("post-prompt.yaml now declares SwaigRequest; it will be emitted into " +
			"post_prompt_generated.go and collide with swaig_request_generated.go. Decide " +
			"which file owns it instead of re-adding a name-based skip.")
	}
}
