package pom

// PomBuilder is a thin fluent wrapper around PromptObjectModel that mirrors
// signalwire.core.pom_builder.PomBuilder: a convenience builder for assembling a
// Prompt Object Model section-by-section and rendering it to markdown/XML/dict/
// JSON. It delegates to the same PromptObjectModel/Section primitives that back
// signalwire.pom.pom, so its output is byte-for-byte compatible.
type PomBuilder struct {
	pom *PromptObjectModel
}

// NewPomBuilder creates an empty PomBuilder.
func NewPomBuilder() *PomBuilder {
	return &PomBuilder{pom: NewPromptObjectModel()}
}

// FromSections builds a PomBuilder from a list of section dicts (the Python
// PomBuilder.from_sections classmethod). Invalid input yields an empty builder.
func FromSections(sections []map[string]any) *PomBuilder {
	p, err := FromList(sections)
	if err != nil || p == nil {
		return NewPomBuilder()
	}
	return &PomBuilder{pom: p}
}

// AddSection adds a top-level section and returns the builder for chaining.
func (b *PomBuilder) AddSection(title, body string, bullets []string, numbered, numberedBullets bool) *PomBuilder {
	opts := []SectionOption{}
	if body != "" {
		opts = append(opts, WithBody(body))
	}
	if len(bullets) > 0 {
		opts = append(opts, WithBullets(bullets))
	}
	if numbered {
		opts = append(opts, WithNumbered(true))
	}
	if numberedBullets {
		opts = append(opts, WithNumberedBullets(true))
	}
	_, _ = b.pom.AddSection(title, opts...)
	return b
}

// AddToSection appends body and/or bullets to an existing section (creating it
// if absent) and returns the builder for chaining.
func (b *PomBuilder) AddToSection(title, body string, bullets []string) *PomBuilder {
	section := b.pom.FindSection(title)
	if section == nil {
		section, _ = b.pom.AddSection(title)
	}
	if section == nil {
		return b
	}
	if body != "" {
		section.AddBody(body)
	}
	if len(bullets) > 0 {
		section.AddBullets(bullets)
	}
	return b
}

// AddSubsection adds a subsection under the named parent section (creating the
// parent if absent) and returns the builder for chaining.
func (b *PomBuilder) AddSubsection(parentTitle, title, body string, bullets []string) *PomBuilder {
	parent := b.pom.FindSection(parentTitle)
	if parent == nil {
		parent, _ = b.pom.AddSection(parentTitle)
	}
	if parent == nil {
		return b
	}
	opts := []SectionOption{}
	if body != "" {
		opts = append(opts, WithBody(body))
	}
	if len(bullets) > 0 {
		opts = append(opts, WithBullets(bullets))
	}
	_, _ = parent.AddSubsection(title, opts...)
	return b
}

// HasSection reports whether a top-level or nested section with the title exists.
func (b *PomBuilder) HasSection(title string) bool {
	return b.pom.FindSection(title) != nil
}

// GetSection returns the section with the given title, or nil.
func (b *PomBuilder) GetSection(title string) *Section {
	return b.pom.FindSection(title)
}

// RenderMarkdown renders the built POM to markdown.
func (b *PomBuilder) RenderMarkdown() string { return b.pom.RenderMarkdown() }

// RenderXML renders the built POM to XML.
func (b *PomBuilder) RenderXML() string { return b.pom.RenderXML() }

// ToDict returns the POM as a list of section maps (the Python to_dict shape).
func (b *PomBuilder) ToDict() []map[string]any { return b.pom.ToList() }

// ToJSON returns the POM serialised as a JSON string.
func (b *PomBuilder) ToJSON() (string, error) { return b.pom.ToJSON() }
