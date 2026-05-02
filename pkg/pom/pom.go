// Package pom provides a typed Prompt Object Model — a structured tree of
// sections that can be rendered to Markdown, XML, JSON, or YAML.  The
// rendered output matches the Python reference at
// signalwire/signalwire/pom/pom.py byte-for-byte for the canonical
// scenarios covered by the cross-port parity tests in
// tests/unit/pom/test_pom_render_parity.py.
//
// Two types make up the API:
//
//   - Section: one node in the tree (title, body, bullets, subsections,
//     numbered, numberedBullets).
//   - PromptObjectModel: the root container that holds the top-level
//     sections and provides JSON / YAML round-trip helpers plus the
//     Markdown / XML renderers.
//
// Both types are exported so callers can build a POM imperatively
// (NewPromptObjectModel + AddSection + AddSubsection) or by parsing a
// JSON/YAML document (FromJSON / FromYAML).  The rendered output is the
// canonical wire format; user-facing helpers like AgentBase.Pom() return
// a *PromptObjectModel value to keep mutations off the agent's internal
// state.
package pom

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// Section is one node in the POM tree.
//
// Python equivalent: signalwire.pom.pom.Section.  The exported field
// names match the JSON / YAML schema:
//
//	{"title": "...", "body": "...", "bullets": [...],
//	 "subsections": [...], "numbered": true, "numberedBullets": true}
//
// Title is a *string (not string) to faithfully model Python's
// "title may be None" semantics for the optional first top-level section.
// All other fields use zero values to mean "absent".
type Section struct {
	// Title is the section heading.  nil means untitled (only legal for
	// the first top-level section in a PromptObjectModel).
	Title *string
	// Body is a paragraph of free text (rendered before any bullets).
	Body string
	// Bullets is the list of bullet points.  Empty means no bullet list.
	Bullets []string
	// Subsections is the list of nested child sections.
	Subsections []*Section
	// Numbered, when non-nil, opts the section into (or out of) numeric
	// section numbering.  nil means "default" (inherit sibling behavior).
	Numbered *bool
	// NumberedBullets renders bullets as "1." "2." instead of "-".
	NumberedBullets bool
}

// NewSection returns a new Section with the supplied title (which may be
// empty to indicate untitled).  Body, bullets, and subsections start
// empty; populate them via AddBody / AddBullets / AddSubsection.
//
// Python equivalent: Section.__init__
func NewSection(title string) *Section {
	t := title
	return &Section{Title: &t}
}

// AddBody sets (or replaces) the section body text.
//
// Python equivalent: Section.add_body — the docstring says "Add OR
// REPLACE the body text"; this is a setter, not an appender.
func (s *Section) AddBody(body string) {
	s.Body = body
}

// AddBullets appends bullet points to the section.
//
// Python equivalent: Section.add_bullets — the Python contract is to
// extend (not replace) the existing bullet list.
func (s *Section) AddBullets(bullets []string) {
	s.Bullets = append(s.Bullets, bullets...)
}

// AddSubsection creates and appends a subsection under this section.
// title must be non-empty (subsections always require a title).
// Returns the new *Section so callers can keep building.
//
// Python equivalent: Section.add_subsection
func (s *Section) AddSubsection(title string, opts ...SectionOption) (*Section, error) {
	if title == "" {
		return nil, errors.New("pom: subsections must have a non-empty title")
	}
	sub := NewSection(title)
	for _, o := range opts {
		o(sub)
	}
	s.Subsections = append(s.Subsections, sub)
	return sub, nil
}

// SectionOption configures a Section at construction (used by
// AddSubsection and PromptObjectModel.AddSection).
type SectionOption func(*Section)

// WithBody sets the section body.
func WithBody(body string) SectionOption { return func(s *Section) { s.Body = body } }

// WithBullets sets the section bullets (replaces, not appends).
func WithBullets(bullets []string) SectionOption {
	return func(s *Section) { s.Bullets = append([]string(nil), bullets...) }
}

// WithNumbered marks the section as numbered (or explicitly un-numbered).
func WithNumbered(v bool) SectionOption { return func(s *Section) { s.Numbered = &v } }

// WithNumberedBullets switches bullet rendering to numbered form.
func WithNumberedBullets(v bool) SectionOption {
	return func(s *Section) { s.NumberedBullets = v }
}

// ToMap returns the section as a map[string]any with keys in canonical
// order (title, body, bullets, subsections, numbered, numberedBullets).
// Empty-or-zero fields are omitted to match Python's to_dict behavior.
//
// The returned value is intended for JSON / YAML serialization; callers
// that need a plain Go map (and don't care about key order) can
// type-assert each value.
//
// Python equivalent: Section.to_dict
func (s *Section) ToMap() map[string]any {
	m := make(map[string]any)
	if s.Title != nil {
		m["title"] = *s.Title
	}
	if s.Body != "" {
		m["body"] = s.Body
	}
	if len(s.Bullets) > 0 {
		m["bullets"] = append([]string(nil), s.Bullets...)
	}
	if len(s.Subsections) > 0 {
		subs := make([]any, len(s.Subsections))
		for i, sub := range s.Subsections {
			subs[i] = sub.ToMap()
		}
		m["subsections"] = subs
	}
	if s.Numbered != nil && *s.Numbered {
		m["numbered"] = true
	}
	if s.NumberedBullets {
		m["numberedBullets"] = true
	}
	return m
}

// orderedKeys returns the section's fields in the canonical Python
// to_dict order: title, body, bullets, subsections, numbered,
// numberedBullets.  Used by JSON / YAML renderers that must preserve
// key order.
func (s *Section) orderedKeys() []string {
	keys := make([]string, 0, 6)
	if s.Title != nil {
		keys = append(keys, "title")
	}
	if s.Body != "" {
		keys = append(keys, "body")
	}
	if len(s.Bullets) > 0 {
		keys = append(keys, "bullets")
	}
	if len(s.Subsections) > 0 {
		keys = append(keys, "subsections")
	}
	if s.Numbered != nil && *s.Numbered {
		keys = append(keys, "numbered")
	}
	if s.NumberedBullets {
		keys = append(keys, "numberedBullets")
	}
	return keys
}

// RenderMarkdown returns this section (and its subsections) as a
// Markdown string.  level controls the starting heading level (default
// 2 == "##"); sectionNumber is the optional dotted prefix the section
// inherits when its parent is numbered.
//
// Python equivalent: Section.render_markdown
func (s *Section) RenderMarkdown(level int, sectionNumber []int) string {
	if level == 0 {
		level = 2
	}
	var md []string

	if s.Title != nil {
		prefix := ""
		if len(sectionNumber) > 0 {
			parts := make([]string, len(sectionNumber))
			for i, n := range sectionNumber {
				parts[i] = strconv.Itoa(n)
			}
			prefix = strings.Join(parts, ".") + ". "
		}
		md = append(md, strings.Repeat("#", level)+" "+prefix+*s.Title+"\n")
	}

	if s.Body != "" {
		md = append(md, s.Body+"\n")
	}

	for i, b := range s.Bullets {
		if s.NumberedBullets {
			md = append(md, strconv.Itoa(i+1)+". "+b)
		} else {
			md = append(md, "- "+b)
		}
	}
	if len(s.Bullets) > 0 {
		md = append(md, "")
	}

	anyNumbered := false
	for _, sub := range s.Subsections {
		if sub.Numbered != nil && *sub.Numbered {
			anyNumbered = true
			break
		}
	}

	for i, sub := range s.Subsections {
		var newNumber []int
		var nextLevel int
		if s.Title != nil || len(sectionNumber) > 0 {
			if anyNumbered && !(sub.Numbered != nil && !*sub.Numbered) {
				newNumber = append(append([]int(nil), sectionNumber...), i+1)
			} else {
				newNumber = sectionNumber
			}
			nextLevel = level + 1
		} else {
			newNumber = sectionNumber
			nextLevel = level
		}
		md = append(md, sub.RenderMarkdown(nextLevel, newNumber))
	}

	return strings.Join(md, "\n")
}

// RenderXML returns this section (and its subsections) as a chunk of
// XML.  indent is the starting indent level (each level == 2 spaces).
//
// Python equivalent: Section.render_xml
func (s *Section) RenderXML(indent int, sectionNumber []int) string {
	indentStr := strings.Repeat("  ", indent)
	var xml []string

	xml = append(xml, indentStr+"<section>")

	if s.Title != nil {
		prefix := ""
		if len(sectionNumber) > 0 {
			parts := make([]string, len(sectionNumber))
			for i, n := range sectionNumber {
				parts[i] = strconv.Itoa(n)
			}
			prefix = strings.Join(parts, ".") + ". "
		}
		xml = append(xml, indentStr+"  <title>"+prefix+*s.Title+"</title>")
	}

	if s.Body != "" {
		xml = append(xml, indentStr+"  <body>"+s.Body+"</body>")
	}

	if len(s.Bullets) > 0 {
		xml = append(xml, indentStr+"  <bullets>")
		for i, b := range s.Bullets {
			if s.NumberedBullets {
				xml = append(xml, indentStr+`    <bullet id="`+strconv.Itoa(i+1)+`">`+b+"</bullet>")
			} else {
				xml = append(xml, indentStr+"    <bullet>"+b+"</bullet>")
			}
		}
		xml = append(xml, indentStr+"  </bullets>")
	}

	if len(s.Subsections) > 0 {
		xml = append(xml, indentStr+"  <subsections>")
		anyNumbered := false
		for _, sub := range s.Subsections {
			if sub.Numbered != nil && *sub.Numbered {
				anyNumbered = true
				break
			}
		}
		for i, sub := range s.Subsections {
			var newNumber []int
			if s.Title != nil || len(sectionNumber) > 0 {
				if anyNumbered && !(sub.Numbered != nil && !*sub.Numbered) {
					newNumber = append(append([]int(nil), sectionNumber...), i+1)
				} else {
					newNumber = sectionNumber
				}
			} else {
				newNumber = sectionNumber
			}
			xml = append(xml, sub.RenderXML(indent+2, newNumber))
		}
		xml = append(xml, indentStr+"  </subsections>")
	}

	xml = append(xml, indentStr+"</section>")
	return strings.Join(xml, "\n")
}

// PromptObjectModel is the root container — a list of top-level Sections
// plus serialization / rendering helpers.  Use NewPromptObjectModel() to
// construct one, or FromJSON / FromYAML to parse one.
//
// Python equivalent: signalwire.pom.pom.PromptObjectModel
type PromptObjectModel struct {
	// Sections is the ordered list of top-level sections.  Only the
	// first section may have a nil Title.
	Sections []*Section
	// Debug, when true, prints rendering decisions to stderr (matches
	// the Python flag).  Off by default.
	Debug bool
}

// NewPromptObjectModel returns an empty POM ready for AddSection calls.
//
// Python equivalent: PromptObjectModel.__init__
func NewPromptObjectModel() *PromptObjectModel {
	return &PromptObjectModel{Sections: nil}
}

// AddSection appends a top-level section.  title may be empty only for
// the first section (Python contract: "Only the first section can have
// no title").  The returned *Section can be configured further (for
// example, by calling AddSubsection on it).
//
// Python equivalent: PromptObjectModel.add_section
func (p *PromptObjectModel) AddSection(title string, opts ...SectionOption) (*Section, error) {
	if title == "" && len(p.Sections) > 0 {
		return nil, errors.New("pom: only the first section can have no title")
	}
	var s *Section
	if title == "" {
		s = &Section{}
	} else {
		s = NewSection(title)
	}
	for _, o := range opts {
		o(s)
	}
	p.Sections = append(p.Sections, s)
	return s, nil
}

// FindSection performs a recursive depth-first search for a section
// whose Title matches.  Returns nil if no match is found.
//
// Python equivalent: PromptObjectModel.find_section
func (p *PromptObjectModel) FindSection(title string) *Section {
	return findSection(p.Sections, title)
}

func findSection(sections []*Section, title string) *Section {
	for _, s := range sections {
		if s.Title != nil && *s.Title == title {
			return s
		}
		if found := findSection(s.Subsections, title); found != nil {
			return found
		}
	}
	return nil
}

// ToList returns the POM as []map[string]any (one entry per top-level
// section), matching Python's to_dict.
//
// Python equivalent: PromptObjectModel.to_dict
func (p *PromptObjectModel) ToList() []map[string]any {
	out := make([]map[string]any, len(p.Sections))
	for i, s := range p.Sections {
		out[i] = s.ToMap()
	}
	return out
}

// ToJSON serializes the POM to a JSON string.  Matches Python's
// json.dumps(..., indent=2) byte-for-byte for the canonical fixtures.
//
// Python equivalent: PromptObjectModel.to_json
func (p *PromptObjectModel) ToJSON() (string, error) {
	if len(p.Sections) == 0 {
		return "[]", nil
	}
	var buf bytes.Buffer
	buf.WriteString("[\n")
	for i, s := range p.Sections {
		if err := writeJSONSection(&buf, s, 1); err != nil {
			return "", err
		}
		if i < len(p.Sections)-1 {
			buf.WriteString(",\n")
		} else {
			buf.WriteString("\n")
		}
	}
	buf.WriteString("]")
	return buf.String(), nil
}

func writeJSONSection(buf *bytes.Buffer, s *Section, indent int) error {
	pad := strings.Repeat("  ", indent)
	pad2 := strings.Repeat("  ", indent+1)
	buf.WriteString(pad + "{\n")
	keys := s.orderedKeys()
	for i, k := range keys {
		buf.WriteString(pad2)
		buf.WriteString(`"` + k + `": `)
		switch k {
		case "title":
			b, err := json.Marshal(*s.Title)
			if err != nil {
				return err
			}
			buf.Write(b)
		case "body":
			b, err := json.Marshal(s.Body)
			if err != nil {
				return err
			}
			buf.Write(b)
		case "bullets":
			writeJSONStringList(buf, s.Bullets, indent+1)
		case "subsections":
			buf.WriteString("[\n")
			for j, sub := range s.Subsections {
				if err := writeJSONSection(buf, sub, indent+2); err != nil {
					return err
				}
				if j < len(s.Subsections)-1 {
					buf.WriteString(",\n")
				} else {
					buf.WriteString("\n")
				}
			}
			buf.WriteString(strings.Repeat("  ", indent+1) + "]")
		case "numbered":
			buf.WriteString("true")
		case "numberedBullets":
			buf.WriteString("true")
		}
		if i < len(keys)-1 {
			buf.WriteString(",\n")
		} else {
			buf.WriteString("\n")
		}
	}
	buf.WriteString(pad + "}")
	return nil
}

func writeJSONStringList(buf *bytes.Buffer, items []string, indent int) {
	pad := strings.Repeat("  ", indent)
	pad2 := strings.Repeat("  ", indent+1)
	buf.WriteString("[\n")
	for i, it := range items {
		b, _ := json.Marshal(it)
		buf.WriteString(pad2)
		buf.Write(b)
		if i < len(items)-1 {
			buf.WriteString(",\n")
		} else {
			buf.WriteString("\n")
		}
	}
	buf.WriteString(pad + "]")
}

// ToYAML serializes the POM to a YAML string in the same shape as
// Python's yaml.dump(..., default_flow_style=False, sort_keys=False).
// PyYAML uses block-sequence-with-indent-0 by default (the leading "-"
// of each list item aligns with the parent's mapping key, not after
// it); gopkg.in/yaml.v3 cannot be configured to do the same, so this
// renderer writes the YAML structure manually for byte-for-byte parity.
//
// Python equivalent: PromptObjectModel.to_yaml
func (p *PromptObjectModel) ToYAML() string {
	if len(p.Sections) == 0 {
		return "[]\n"
	}
	var buf bytes.Buffer
	for _, s := range p.Sections {
		writeYAMLSection(&buf, s, 0)
	}
	return buf.String()
}

func writeYAMLSection(buf *bytes.Buffer, s *Section, indent int) {
	pad := strings.Repeat("  ", indent)
	keys := s.orderedKeys()
	first := true
	for _, k := range keys {
		var prefix string
		if first {
			prefix = pad + "- "
			first = false
		} else {
			prefix = pad + "  "
		}
		switch k {
		case "title":
			buf.WriteString(prefix + "title: " + yamlScalar(*s.Title) + "\n")
		case "body":
			buf.WriteString(prefix + "body: " + yamlScalar(s.Body) + "\n")
		case "bullets":
			buf.WriteString(prefix + "bullets:\n")
			for _, b := range s.Bullets {
				buf.WriteString(pad + "  - " + yamlScalar(b) + "\n")
			}
		case "subsections":
			buf.WriteString(prefix + "subsections:\n")
			for _, sub := range s.Subsections {
				writeYAMLSection(buf, sub, indent+1)
			}
		case "numbered":
			buf.WriteString(prefix + "numbered: true\n")
		case "numberedBullets":
			buf.WriteString(prefix + "numberedBullets: true\n")
		}
	}
}

// yamlScalar emits a YAML scalar with PyYAML-compatible quoting.  It
// passes through plain identifiers unchanged and single-quotes anything
// containing characters that would otherwise be interpreted as YAML
// syntax (colons, leading dashes, etc.).
func yamlScalar(v string) string {
	if v == "" {
		return "''"
	}
	// Keep the safe set small: alphanumerics, spaces, and a handful of
	// punctuation that PyYAML's default style emits as plain scalars.
	for _, c := range v {
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') &&
			!(c >= '0' && c <= '9') && c != ' ' && c != '_' && c != '-' &&
			c != '.' && c != '/' && c != '(' && c != ')' && c != ',' &&
			c != ';' && c != '?' && c != '!' {
			return "'" + strings.ReplaceAll(v, "'", "''") + "'"
		}
	}
	// Strings that look like YAML reserved words still need quoting.
	switch strings.ToLower(v) {
	case "true", "false", "yes", "no", "null", "~", "on", "off":
		return "'" + v + "'"
	}
	return v
}

// FromJSON parses a JSON string (an array of section maps) and returns
// a populated *PromptObjectModel.  Subsections are validated to require
// a title; any section without body/bullets/subsections is rejected.
//
// Python equivalent: PromptObjectModel.from_json
func FromJSON(jsonStr string) (*PromptObjectModel, error) {
	var data []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("pom: invalid JSON: %w", err)
	}
	return fromList(data)
}

// FromYAML parses a YAML string (an array of section maps) and returns
// a populated *PromptObjectModel.
//
// Python equivalent: PromptObjectModel.from_yaml
func FromYAML(yamlStr string) (*PromptObjectModel, error) {
	var data []map[string]any
	if err := yaml.Unmarshal([]byte(yamlStr), &data); err != nil {
		return nil, fmt.Errorf("pom: invalid YAML: %w", err)
	}
	return fromList(data)
}

// FromList builds a POM from a pre-parsed []map[string]any (callers can
// use this when they already have the dict form, e.g. from a database
// row or another config source).
//
// Python equivalent: PromptObjectModel._from_dict (the internal helper
// shared by from_json / from_yaml).
func FromList(data []map[string]any) (*PromptObjectModel, error) {
	return fromList(data)
}

func fromList(data []map[string]any) (*PromptObjectModel, error) {
	pom := NewPromptObjectModel()
	for i, d := range data {
		if i > 0 {
			if _, ok := d["title"]; !ok {
				d["title"] = "Untitled Section"
			}
		}
		s, err := buildSection(d, false)
		if err != nil {
			return nil, err
		}
		pom.Sections = append(pom.Sections, s)
	}
	return pom, nil
}

func buildSection(d map[string]any, isSubsection bool) (*Section, error) {
	if d == nil {
		return nil, errors.New("pom: each section must be a map")
	}
	if t, ok := d["title"]; ok {
		if _, isStr := t.(string); !isStr {
			return nil, errors.New("pom: 'title' must be a string if present")
		}
	}
	if subs, ok := d["subsections"]; ok {
		if !isList(subs) {
			return nil, errors.New("pom: 'subsections' must be a list if provided")
		}
	}
	if bs, ok := d["bullets"]; ok {
		if !isList(bs) {
			return nil, errors.New("pom: 'bullets' must be a list if provided")
		}
	}
	if n, ok := d["numbered"]; ok {
		if _, isBool := n.(bool); !isBool {
			return nil, errors.New("pom: 'numbered' must be a boolean if provided")
		}
	}
	if nb, ok := d["numberedBullets"]; ok {
		if _, isBool := nb.(bool); !isBool {
			return nil, errors.New("pom: 'numberedBullets' must be a boolean if provided")
		}
	}

	hasBody := false
	if b, ok := d["body"].(string); ok && b != "" {
		hasBody = true
	}
	hasBullets := false
	if bs, ok := d["bullets"]; ok {
		hasBullets = listLen(bs) > 0
	}
	hasSubs := false
	if subs, ok := d["subsections"]; ok {
		hasSubs = listLen(subs) > 0
	}
	if !hasBody && !hasBullets && !hasSubs {
		return nil, errors.New("pom: all sections must have either a non-empty body, non-empty bullets, or subsections")
	}
	if isSubsection {
		if _, ok := d["title"]; !ok {
			return nil, errors.New("pom: all subsections must have a title")
		}
	}

	s := &Section{}
	if t, ok := d["title"].(string); ok {
		s.Title = &t
	}
	if b, ok := d["body"].(string); ok {
		s.Body = b
	}
	if bs, ok := d["bullets"]; ok {
		s.Bullets = listToStrings(bs)
	}
	if n, ok := d["numbered"].(bool); ok {
		s.Numbered = &n
	}
	if nb, ok := d["numberedBullets"].(bool); ok {
		s.NumberedBullets = nb
	}

	if subs, ok := d["subsections"]; ok {
		for _, raw := range listToMaps(subs) {
			child, err := buildSection(raw, true)
			if err != nil {
				return nil, err
			}
			s.Subsections = append(s.Subsections, child)
		}
	}
	return s, nil
}

// isList accepts either []any or []map[string]any or []string — all
// shapes that the JSON/YAML decoders may produce.
func isList(v any) bool {
	switch v.(type) {
	case []any, []map[string]any, []string:
		return true
	}
	return false
}

func listLen(v any) int {
	switch t := v.(type) {
	case []any:
		return len(t)
	case []map[string]any:
		return len(t)
	case []string:
		return len(t)
	}
	return 0
}

func listToStrings(v any) []string {
	switch t := v.(type) {
	case []string:
		return append([]string(nil), t...)
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func listToMaps(v any) []map[string]any {
	switch t := v.(type) {
	case []map[string]any:
		return t
	case []any:
		out := make([]map[string]any, 0, len(t))
		for _, x := range t {
			if m, ok := x.(map[string]any); ok {
				out = append(out, m)
			} else if m, ok := x.(map[any]any); ok {
				// gopkg.in/yaml.v3 sometimes emits map[any]any for mappings.
				normalized := make(map[string]any, len(m))
				for k, v := range m {
					if ks, ok := k.(string); ok {
						normalized[ks] = v
					}
				}
				out = append(out, normalized)
			}
		}
		return out
	}
	return nil
}

// RenderMarkdown renders the entire POM as a Markdown document.
//
// Python equivalent: PromptObjectModel.render_markdown
func (p *PromptObjectModel) RenderMarkdown() string {
	anyNumbered := false
	for _, s := range p.Sections {
		if s.Numbered != nil && *s.Numbered {
			anyNumbered = true
			break
		}
	}
	var md []string
	counter := 0
	for _, s := range p.Sections {
		var sectionNumber []int
		if s.Title != nil {
			counter++
			if anyNumbered && !(s.Numbered != nil && !*s.Numbered) {
				sectionNumber = []int{counter}
			}
		}
		md = append(md, s.RenderMarkdown(2, sectionNumber))
	}
	return strings.Join(md, "\n")
}

// RenderXML renders the entire POM as an XML document with the
// canonical ``<?xml ...?><prompt> ... </prompt>`` envelope.
//
// Python equivalent: PromptObjectModel.render_xml
func (p *PromptObjectModel) RenderXML() string {
	var xml []string
	xml = append(xml, `<?xml version="1.0" encoding="UTF-8"?>`)
	xml = append(xml, "<prompt>")
	anyNumbered := false
	for _, s := range p.Sections {
		if s.Numbered != nil && *s.Numbered {
			anyNumbered = true
			break
		}
	}
	counter := 0
	for _, s := range p.Sections {
		var sectionNumber []int
		if s.Title != nil {
			counter++
			if anyNumbered && !(s.Numbered != nil && !*s.Numbered) {
				sectionNumber = []int{counter}
			}
		}
		xml = append(xml, s.RenderXML(1, sectionNumber))
	}
	xml = append(xml, "</prompt>")
	return strings.Join(xml, "\n")
}

// AddPomAsSubsection attaches every top-level section of pomToAdd
// underneath the section identified by target — either the title of an
// existing section in this POM, or a *Section pointer.
//
// Python equivalent: PromptObjectModel.add_pom_as_subsection
func (p *PromptObjectModel) AddPomAsSubsection(target any, pomToAdd *PromptObjectModel) error {
	var host *Section
	switch t := target.(type) {
	case string:
		host = p.FindSection(t)
		if host == nil {
			return fmt.Errorf("pom: no section with title %q found", t)
		}
	case *Section:
		host = t
	default:
		return errors.New("pom: target must be a string or *Section")
	}
	for _, s := range pomToAdd.Sections {
		host.Subsections = append(host.Subsections, s)
	}
	return nil
}

// Clone returns a deep copy of the POM.  Useful when an agent wants to
// hand callers a snapshot without exposing internal mutable state.
func (p *PromptObjectModel) Clone() *PromptObjectModel {
	out := &PromptObjectModel{Debug: p.Debug}
	out.Sections = make([]*Section, len(p.Sections))
	for i, s := range p.Sections {
		out.Sections[i] = s.clone()
	}
	return out
}

func (s *Section) clone() *Section {
	c := &Section{Body: s.Body, NumberedBullets: s.NumberedBullets}
	if s.Title != nil {
		t := *s.Title
		c.Title = &t
	}
	if s.Numbered != nil {
		n := *s.Numbered
		c.Numbered = &n
	}
	if len(s.Bullets) > 0 {
		c.Bullets = append([]string(nil), s.Bullets...)
	}
	if len(s.Subsections) > 0 {
		c.Subsections = make([]*Section, len(s.Subsections))
		for i, sub := range s.Subsections {
			c.Subsections[i] = sub.clone()
		}
	}
	return c
}

// sortKeys is a small helper used only by tests that compare key sets.
// Exported because external callers occasionally need it for debugging.
func sortKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
