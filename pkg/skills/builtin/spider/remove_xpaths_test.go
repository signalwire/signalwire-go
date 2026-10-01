package spider

import (
	"strings"
	"testing"
)

// newSpiderSkill builds a configured SpiderSkill for the extraction tests. The
// registry factory returns skills.SkillBase (the interface), which does not carry
// the concrete accessors, so the tests construct the concrete type directly.
func newSpiderSkill(t *testing.T) *SpiderSkill {
	t.Helper()
	s, ok := NewSpider(nil).(*SpiderSkill)
	if !ok {
		t.Fatal("NewSpider did not return *SpiderSkill")
	}
	if !s.Setup() {
		t.Fatal("Setup() returned false")
	}
	return s
}

// TestRemoveXPathsPrefilledDefault pins the reference's default list: the
// attribute is PREFILLED at construction, not empty. Order matters — it is the
// reference's order (signalwire/skills/spider/skill.py).
func TestRemoveXPathsPrefilledDefault(t *testing.T) {
	s := newSpiderSkill(t)
	got := s.RemoveXPaths()
	want := []string{"//script", "//style", "//nav", "//header", "//footer", "//aside", "//noscript"}
	if len(got) != len(want) {
		t.Fatalf("RemoveXPaths() = %v (len %d), want len %d", got, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("RemoveXPaths()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestRemoveXPathsReturnsCopy proves the accessor hands back a copy: mutating the
// returned slice must not reconfigure the skill.
func TestRemoveXPathsReturnsCopy(t *testing.T) {
	s := newSpiderSkill(t)
	got := s.RemoveXPaths()
	got[0] = "//mutated"
	if s.RemoveXPaths()[0] != "//script" {
		t.Errorf("mutating the returned slice changed the skill: %q", s.RemoveXPaths()[0])
	}
}

// TestSetRemoveXPaths covers the assignment half of the attribute (the reference
// exposes remove_xpaths as a plain writable attribute).
func TestSetRemoveXPaths(t *testing.T) {
	s := newSpiderSkill(t)
	s.SetRemoveXPaths([]string{"//aside"})
	got := s.RemoveXPaths()
	if len(got) != 1 || got[0] != "//aside" {
		t.Fatalf("after SetRemoveXPaths, RemoveXPaths() = %v, want [//aside]", got)
	}
}

const chromeHTML = `<html><body>
<nav>NAVLINK</nav>
<header>HEADERTEXT</header>
<aside>SIDEBARTEXT</aside>
<script>SCRIPTCODE</script>
<style>STYLECODE</style>
<noscript>NOSCRIPTTEXT</noscript>
<p>REALCONTENT</p>
<footer>FOOTERTEXT</footer>
</body></html>`

// TestExtractTextDropsChromeElements is the BEHAVIORAL half: the default
// remove_xpaths set must actually strip navigation/header/footer/aside/noscript
// chrome from extracted text, not merely exist as surface. Before this was
// wired, only <script>/<style> were removed and the rest leaked into the text.
func TestExtractTextDropsChromeElements(t *testing.T) {
	s := newSpiderSkill(t)
	text := s.extractText([]byte(chromeHTML))

	if !strings.Contains(text, "REALCONTENT") {
		t.Fatalf("extractText dropped the real content: %q", text)
	}
	for _, chrome := range []string{
		"NAVLINK", "HEADERTEXT", "SIDEBARTEXT",
		"SCRIPTCODE", "STYLECODE", "NOSCRIPTTEXT", "FOOTERTEXT",
	} {
		if strings.Contains(text, chrome) {
			t.Errorf("extractText leaked %q into the text: %q", chrome, text)
		}
	}
}

// TestExtractTextHonorsCustomXPaths proves extraction reads the CONFIGURED value
// rather than a hardcoded list: narrowing the set must let previously-stripped
// chrome through, and still strip what remains configured.
func TestExtractTextHonorsCustomXPaths(t *testing.T) {
	s := newSpiderSkill(t)
	s.SetRemoveXPaths([]string{"//footer"})
	text := s.extractText([]byte(chromeHTML))

	if strings.Contains(text, "FOOTERTEXT") {
		t.Errorf("configured //footer was not stripped: %q", text)
	}
	if !strings.Contains(text, "NAVLINK") {
		t.Errorf("//nav was stripped though it is no longer configured: %q", text)
	}
}

// TestExtractTextEmptyXPathsStripsNothingStructurally guards the degenerate
// case: an empty set skips the XPath pass entirely, leaving the regex tag strip
// (which still removes script/style bodies) as the only cleanup.
func TestExtractTextEmptyXPathsStripsNothingStructurally(t *testing.T) {
	s := newSpiderSkill(t)
	s.SetRemoveXPaths(nil)
	text := s.extractText([]byte(chromeHTML))

	if !strings.Contains(text, "NAVLINK") {
		t.Errorf("nav removed with an empty remove_xpaths set: %q", text)
	}
	if strings.Contains(text, "SCRIPTCODE") {
		t.Errorf("stripHTMLTags should still drop script bodies: %q", text)
	}
}

// TestApplyRemoveXPathsMalformedInput proves the pass degrades instead of
// panicking on input that is not well-formed HTML or uses a bad expression.
func TestApplyRemoveXPathsMalformedInput(t *testing.T) {
	if got := applyRemoveXPaths([]byte("<p>hi"), []string{"//nav"}); !strings.Contains(got, "hi") {
		t.Errorf("malformed HTML lost its content: %q", got)
	}
	if got := applyRemoveXPaths([]byte(chromeHTML), []string{"//["}); !strings.Contains(got, "REALCONTENT") {
		t.Errorf("invalid xpath lost the document: %q", got)
	}
}
