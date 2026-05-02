// Tests for pkg/pom — every assertion is paired with the Python
// reference test in:
//
//	signalwire-python/tests/unit/pom/test_pom_render_parity.py
//	signalwire-python/tests/unit/pom/test_pom_object_model.py
//
// Each test uses exact-string assertions to guarantee byte-for-byte
// parity with the Python renderer (no substring shortcuts).
package pom

import (
	"strings"
	"testing"
)

// helper: assert two strings match, with a clear diff if they don't.
func assertEqual(t *testing.T, got, want, label string) {
	t.Helper()
	if got != want {
		t.Errorf("%s mismatch.\n--- got %d bytes ---\n%q\n--- want %d bytes ---\n%q",
			label, len(got), got, len(want), want)
	}
}

// =====================================================================
// Construction / basic invariants
// =====================================================================

func TestNewPromptObjectModelIsEmpty(t *testing.T) {
	p := NewPromptObjectModel()
	if p == nil {
		t.Fatal("NewPromptObjectModel returned nil")
	}
	if len(p.Sections) != 0 {
		t.Errorf("expected 0 sections, got %d", len(p.Sections))
	}
}

func TestAddSectionReturnsSection(t *testing.T) {
	p := NewPromptObjectModel()
	s, err := p.AddSection("Greeting")
	if err != nil {
		t.Fatalf("AddSection error: %v", err)
	}
	if s == nil || s.Title == nil || *s.Title != "Greeting" {
		t.Errorf("section title not set; got %+v", s)
	}
	if len(p.Sections) != 1 {
		t.Errorf("expected 1 section, got %d", len(p.Sections))
	}
}

func TestAddSectionAppendsInOrder(t *testing.T) {
	p := NewPromptObjectModel()
	_, _ = p.AddSection("A")
	_, _ = p.AddSection("B")
	if len(p.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(p.Sections))
	}
	if *p.Sections[0].Title != "A" || *p.Sections[1].Title != "B" {
		t.Errorf("section order wrong: %v / %v", p.Sections[0].Title, p.Sections[1].Title)
	}
}

func TestAddSectionRejectsUntitledAfterFirst(t *testing.T) {
	p := NewPromptObjectModel()
	_, _ = p.AddSection("First")
	_, err := p.AddSection("")
	if err == nil {
		t.Error("expected error when adding untitled section after the first")
	}
}

func TestAddSubsectionRequiresTitle(t *testing.T) {
	s := NewSection("Parent")
	_, err := s.AddSubsection("")
	if err == nil {
		t.Error("expected error when adding subsection with empty title")
	}
}

func TestSectionAddBodyReplaces(t *testing.T) {
	s := NewSection("X")
	s.AddBody("first")
	s.AddBody("second")
	if s.Body != "second" {
		t.Errorf("AddBody should replace, got %q", s.Body)
	}
}

func TestSectionAddBulletsAppends(t *testing.T) {
	s := NewSection("X")
	s.AddBullets([]string{"a", "b"})
	s.AddBullets([]string{"c"})
	if strings.Join(s.Bullets, ",") != "a,b,c" {
		t.Errorf("AddBullets should append, got %v", s.Bullets)
	}
}

// =====================================================================
// Empty POM (parity with TestEmptyPom)
// =====================================================================

func TestEmptyRenderMarkdownIsEmpty(t *testing.T) {
	p := NewPromptObjectModel()
	assertEqual(t, p.RenderMarkdown(), "", "empty markdown")
}

func TestEmptyRenderXMLIsJustPromptTags(t *testing.T) {
	p := NewPromptObjectModel()
	want := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<prompt>\n</prompt>"
	assertEqual(t, p.RenderXML(), want, "empty xml")
}

func TestEmptyToJSONIsEmptyArray(t *testing.T) {
	p := NewPromptObjectModel()
	got, err := p.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, got, "[]", "empty json")
}

func TestEmptyToYAML(t *testing.T) {
	p := NewPromptObjectModel()
	assertEqual(t, p.ToYAML(), "[]\n", "empty yaml")
}

// =====================================================================
// Simple section: title + body (parity with TestSimpleSection)
// =====================================================================

func TestSimpleSectionRenderMarkdown(t *testing.T) {
	p := NewPromptObjectModel()
	_, _ = p.AddSection("Greeting", WithBody("Hello world"))
	want := "## Greeting\n\nHello world\n"
	assertEqual(t, p.RenderMarkdown(), want, "simple markdown")
}

func TestSimpleSectionRenderXML(t *testing.T) {
	p := NewPromptObjectModel()
	_, _ = p.AddSection("Greeting", WithBody("Hello world"))
	want := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		"<prompt>\n" +
		"  <section>\n" +
		"    <title>Greeting</title>\n" +
		"    <body>Hello world</body>\n" +
		"  </section>\n" +
		"</prompt>"
	assertEqual(t, p.RenderXML(), want, "simple xml")
}

// =====================================================================
// Bullets (parity with TestBullets)
// =====================================================================

func TestRenderMarkdownWithBullets(t *testing.T) {
	p := NewPromptObjectModel()
	_, _ = p.AddSection("Goals",
		WithBody("Be helpful"),
		WithBullets([]string{"Be concise", "Be clear"}))
	want := "## Goals\n\nBe helpful\n\n- Be concise\n- Be clear\n"
	assertEqual(t, p.RenderMarkdown(), want, "bullets markdown")
}

func TestRenderXMLWithBullets(t *testing.T) {
	p := NewPromptObjectModel()
	_, _ = p.AddSection("Goals",
		WithBody("Be helpful"),
		WithBullets([]string{"Be concise", "Be clear"}))
	want := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		"<prompt>\n" +
		"  <section>\n" +
		"    <title>Goals</title>\n" +
		"    <body>Be helpful</body>\n" +
		"    <bullets>\n" +
		"      <bullet>Be concise</bullet>\n" +
		"      <bullet>Be clear</bullet>\n" +
		"    </bullets>\n" +
		"  </section>\n" +
		"</prompt>"
	assertEqual(t, p.RenderXML(), want, "bullets xml")
}

// =====================================================================
// Subsections (parity with TestSubsections)
// =====================================================================

func TestRenderMarkdownWithSubsection(t *testing.T) {
	p := NewPromptObjectModel()
	s, _ := p.AddSection("Top", WithBody("Top body"))
	_, _ = s.AddSubsection("Sub1",
		WithBody("Sub1 body"),
		WithBullets([]string{"a", "b"}))
	want := "## Top\n\nTop body\n\n### Sub1\n\nSub1 body\n\n- a\n- b\n"
	assertEqual(t, p.RenderMarkdown(), want, "subsection markdown")
}

func TestRenderXMLWithSubsection(t *testing.T) {
	p := NewPromptObjectModel()
	s, _ := p.AddSection("Top", WithBody("Top body"))
	_, _ = s.AddSubsection("Sub1",
		WithBody("Sub1 body"),
		WithBullets([]string{"a", "b"}))
	want := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		"<prompt>\n" +
		"  <section>\n" +
		"    <title>Top</title>\n" +
		"    <body>Top body</body>\n" +
		"    <subsections>\n" +
		"      <section>\n" +
		"        <title>Sub1</title>\n" +
		"        <body>Sub1 body</body>\n" +
		"        <bullets>\n" +
		"          <bullet>a</bullet>\n" +
		"          <bullet>b</bullet>\n" +
		"        </bullets>\n" +
		"      </section>\n" +
		"    </subsections>\n" +
		"  </section>\n" +
		"</prompt>"
	assertEqual(t, p.RenderXML(), want, "subsection xml")
}

// =====================================================================
// Numbered sections (parity with TestNumberedSections)
// =====================================================================

func TestRenderMarkdownNumberedPropagates(t *testing.T) {
	p := NewPromptObjectModel()
	_, _ = p.AddSection("S1", WithBody("b1"), WithNumbered(true))
	_, _ = p.AddSection("S2", WithBody("b2"))
	want := "## 1. S1\n\nb1\n\n## 2. S2\n\nb2\n"
	assertEqual(t, p.RenderMarkdown(), want, "numbered markdown")
}

func TestRenderXMLNumberedPropagates(t *testing.T) {
	p := NewPromptObjectModel()
	_, _ = p.AddSection("S1", WithBody("b1"), WithNumbered(true))
	_, _ = p.AddSection("S2", WithBody("b2"))
	want := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		"<prompt>\n" +
		"  <section>\n" +
		"    <title>1. S1</title>\n" +
		"    <body>b1</body>\n" +
		"  </section>\n" +
		"  <section>\n" +
		"    <title>2. S2</title>\n" +
		"    <body>b2</body>\n" +
		"  </section>\n" +
		"</prompt>"
	assertEqual(t, p.RenderXML(), want, "numbered xml")
}

func TestRenderMarkdownNestedNumbered(t *testing.T) {
	// Nested numbering (parity with the manual probe in the porting log).
	p := NewPromptObjectModel()
	s, _ := p.AddSection("S1", WithBody("b1"), WithNumbered(true))
	_, _ = s.AddSubsection("Sub1", WithBody("sb1"), WithNumbered(true))
	want := "## 1. S1\n\nb1\n\n### 1.1. Sub1\n\nsb1\n"
	assertEqual(t, p.RenderMarkdown(), want, "nested numbered markdown")
}

// =====================================================================
// Numbered bullets (parity with TestNumberedBullets)
// =====================================================================

func TestRenderMarkdownNumberedBullets(t *testing.T) {
	p := NewPromptObjectModel()
	_, _ = p.AddSection("X",
		WithBullets([]string{"one", "two"}),
		WithNumberedBullets(true))
	want := "## X\n\n1. one\n2. two\n"
	assertEqual(t, p.RenderMarkdown(), want, "numbered bullets markdown")
}

func TestRenderXMLNumberedBulletsUseIDAttr(t *testing.T) {
	p := NewPromptObjectModel()
	_, _ = p.AddSection("X",
		WithBullets([]string{"one", "two"}),
		WithNumberedBullets(true))
	want := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		"<prompt>\n" +
		"  <section>\n" +
		"    <title>X</title>\n" +
		"    <bullets>\n" +
		"      <bullet id=\"1\">one</bullet>\n" +
		"      <bullet id=\"2\">two</bullet>\n" +
		"    </bullets>\n" +
		"  </section>\n" +
		"</prompt>"
	assertEqual(t, p.RenderXML(), want, "numbered bullets xml")
}

// =====================================================================
// Serialization (parity with TestSerialization)
// =====================================================================

func TestToJSONExactShape(t *testing.T) {
	p := NewPromptObjectModel()
	s, _ := p.AddSection("A", WithBody("ab"))
	_, _ = s.AddSubsection("A1", WithBody("a1b"), WithBullets([]string{"x"}))
	want := "[\n" +
		"  {\n" +
		"    \"title\": \"A\",\n" +
		"    \"body\": \"ab\",\n" +
		"    \"subsections\": [\n" +
		"      {\n" +
		"        \"title\": \"A1\",\n" +
		"        \"body\": \"a1b\",\n" +
		"        \"bullets\": [\n" +
		"          \"x\"\n" +
		"        ]\n" +
		"      }\n" +
		"    ]\n" +
		"  }\n" +
		"]"
	got, err := p.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, got, want, "to_json shape")
}

func TestToYAMLExactShape(t *testing.T) {
	p := NewPromptObjectModel()
	s, _ := p.AddSection("A", WithBody("ab"))
	_, _ = s.AddSubsection("A1", WithBody("a1b"), WithBullets([]string{"x"}))
	want := "- title: A\n" +
		"  body: ab\n" +
		"  subsections:\n" +
		"  - title: A1\n" +
		"    body: a1b\n" +
		"    bullets:\n" +
		"    - x\n"
	assertEqual(t, p.ToYAML(), want, "to_yaml shape")
}

func TestFromJSONRoundTrip(t *testing.T) {
	p := NewPromptObjectModel()
	s, _ := p.AddSection("A", WithBody("ab"))
	_, _ = s.AddSubsection("A1", WithBody("a1b"), WithBullets([]string{"x", "y"}))
	jsonStr, err := p.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := FromJSON(jsonStr)
	if err != nil {
		t.Fatal(err)
	}
	got, err := restored.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, got, jsonStr, "round-trip json")
}

func TestFromYAMLRoundTrip(t *testing.T) {
	p := NewPromptObjectModel()
	s, _ := p.AddSection("A", WithBody("ab"))
	_, _ = s.AddSubsection("A1", WithBody("a1b"), WithBullets([]string{"x", "y"}))
	yamlStr := p.ToYAML()
	restored, err := FromYAML(yamlStr)
	if err != nil {
		t.Fatal(err)
	}
	assertEqual(t, restored.ToYAML(), yamlStr, "round-trip yaml")
}

func TestFromJSONInvalidReturnsError(t *testing.T) {
	_, err := FromJSON("not valid json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFromYAMLInvalidReturnsError(t *testing.T) {
	_, err := FromYAML(":\n :\n  : invalid")
	if err == nil {
		t.Error("expected error for malformed YAML")
	}
}

func TestFromListRejectsSectionWithoutContent(t *testing.T) {
	_, err := FromList([]map[string]any{
		{"title": "Empty"},
	})
	if err == nil {
		t.Error("expected error for section with no body/bullets/subsections")
	}
}

func TestFromListRejectsNonStringTitle(t *testing.T) {
	_, err := FromList([]map[string]any{
		{"title": 123, "body": "b"},
	})
	if err == nil {
		t.Error("expected error for non-string title")
	}
}

func TestFromListAcceptsValidNestedSchema(t *testing.T) {
	pom, err := FromList([]map[string]any{
		{
			"title": "Outer",
			"body":  "ob",
			"subsections": []map[string]any{
				{"title": "Inner", "body": "ib"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pom.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(pom.Sections))
	}
	if len(pom.Sections[0].Subsections) != 1 {
		t.Fatalf("expected 1 subsection, got %d", len(pom.Sections[0].Subsections))
	}
}

// =====================================================================
// find_section (parity with TestFindSection)
// =====================================================================

func TestFindSectionTopLevel(t *testing.T) {
	p := NewPromptObjectModel()
	_, _ = p.AddSection("One", WithBody("b1"))
	_, _ = p.AddSection("Two", WithBody("b2"))
	got := p.FindSection("Two")
	if got == nil {
		t.Fatal("expected to find section 'Two'")
	}
	if got.Body != "b2" {
		t.Errorf("expected body 'b2', got %q", got.Body)
	}
}

func TestFindSectionRecurses(t *testing.T) {
	p := NewPromptObjectModel()
	s, _ := p.AddSection("Outer", WithBody("ob"))
	_, _ = s.AddSubsection("Inner", WithBody("ib"))
	got := p.FindSection("Inner")
	if got == nil {
		t.Fatal("expected to find subsection 'Inner'")
	}
	if got.Body != "ib" {
		t.Errorf("expected body 'ib', got %q", got.Body)
	}
}

func TestFindSectionReturnsNilWhenAbsent(t *testing.T) {
	p := NewPromptObjectModel()
	_, _ = p.AddSection("Only", WithBody("b"))
	if got := p.FindSection("Missing"); got != nil {
		t.Errorf("expected nil for missing section, got %+v", got)
	}
}

// =====================================================================
// add_pom_as_subsection (parity with TestAddPomAsSubsection)
// =====================================================================

func TestAddPomAsSubsectionByTitle(t *testing.T) {
	host := NewPromptObjectModel()
	_, _ = host.AddSection("Host", WithBody("hb"))

	guest := NewPromptObjectModel()
	_, _ = guest.AddSection("Guest", WithBody("gb"))

	if err := host.AddPomAsSubsection("Host", guest); err != nil {
		t.Fatal(err)
	}
	hostSec := host.FindSection("Host")
	if hostSec == nil {
		t.Fatal("expected host section")
	}
	if len(hostSec.Subsections) != 1 {
		t.Fatalf("expected 1 subsection, got %d", len(hostSec.Subsections))
	}
	if *hostSec.Subsections[0].Title != "Guest" {
		t.Errorf("expected subsection title 'Guest', got %q", *hostSec.Subsections[0].Title)
	}
}

func TestAddPomAsSubsectionBySectionPointer(t *testing.T) {
	host := NewPromptObjectModel()
	target, _ := host.AddSection("Host", WithBody("hb"))

	guest := NewPromptObjectModel()
	_, _ = guest.AddSection("GuestA", WithBody("ab"))
	_, _ = guest.AddSection("GuestB", WithBody("bb"))

	if err := host.AddPomAsSubsection(target, guest); err != nil {
		t.Fatal(err)
	}
	if len(target.Subsections) != 2 {
		t.Fatalf("expected 2 subsections, got %d", len(target.Subsections))
	}
	titles := []string{*target.Subsections[0].Title, *target.Subsections[1].Title}
	if titles[0] != "GuestA" || titles[1] != "GuestB" {
		t.Errorf("expected [GuestA GuestB], got %v", titles)
	}
}

func TestAddPomAsSubsectionUnknownTitleErrors(t *testing.T) {
	host := NewPromptObjectModel()
	_, _ = host.AddSection("Host", WithBody("hb"))
	guest := NewPromptObjectModel()
	_, _ = guest.AddSection("Guest", WithBody("gb"))
	if err := host.AddPomAsSubsection("Missing", guest); err == nil {
		t.Error("expected error for missing host title")
	}
}

func TestAddPomAsSubsectionInvalidTargetType(t *testing.T) {
	host := NewPromptObjectModel()
	_, _ = host.AddSection("Host", WithBody("hb"))
	guest := NewPromptObjectModel()
	_, _ = guest.AddSection("Guest", WithBody("gb"))
	if err := host.AddPomAsSubsection(42, guest); err == nil {
		t.Error("expected error for non-string non-Section target")
	}
}

// =====================================================================
// Section ToMap key order (covers Python to_dict canonical order)
// =====================================================================

func TestSectionToMapHasCanonicalKeys(t *testing.T) {
	s := NewSection("X")
	s.AddBody("body")
	s.AddBullets([]string{"a"})
	m := s.ToMap()
	wantKeys := []string{"body", "bullets", "title"}
	gotKeys := sortKeys(m)
	if strings.Join(gotKeys, ",") != strings.Join(wantKeys, ",") {
		t.Errorf("expected sorted keys %v, got %v", wantKeys, gotKeys)
	}
	if m["title"] != "X" {
		t.Errorf("expected title 'X', got %v", m["title"])
	}
}

func TestSectionToMapOmitsEmptyFields(t *testing.T) {
	s := NewSection("Bare")
	m := s.ToMap()
	if _, has := m["body"]; has {
		t.Error("body should be omitted when empty")
	}
	if _, has := m["bullets"]; has {
		t.Error("bullets should be omitted when empty")
	}
	if _, has := m["subsections"]; has {
		t.Error("subsections should be omitted when empty")
	}
}

// =====================================================================
// Clone preserves all fields and is deep
// =====================================================================

func TestCloneIsDeepCopy(t *testing.T) {
	p := NewPromptObjectModel()
	s, _ := p.AddSection("A", WithBody("ab"), WithBullets([]string{"x"}))
	_, _ = s.AddSubsection("A1", WithBody("a1b"))

	c := p.Clone()
	// Mutate the clone — original should be unaffected.
	c.Sections[0].Body = "MUTATED"
	c.Sections[0].Bullets[0] = "MUTATED"
	c.Sections[0].Subsections[0].Body = "MUTATED"

	if p.Sections[0].Body != "ab" {
		t.Errorf("clone mutation leaked to original body")
	}
	if p.Sections[0].Bullets[0] != "x" {
		t.Errorf("clone mutation leaked to original bullets")
	}
	if p.Sections[0].Subsections[0].Body != "a1b" {
		t.Errorf("clone mutation leaked to original subsection body")
	}
}
