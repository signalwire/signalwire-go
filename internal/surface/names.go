package surface

import "strings"

// This file is the SINGLE source for folding an exported Go identifier to its
// Python-canonical name. Both cmd/enumerate-surface and cmd/enumerate-signatures
// consume it.
//
// It is shared rather than duplicated on purpose. The two enumerators previously
// each carried their own copy of the correction table, and they had DIVERGED: the
// signature side knew `FAQs` -> "faqs" while the surface side did not, so
// `FAQBotAgent.faqs` folded in SIGNATURES-DIFF and reported missing in
// SURFACE-DIFF. A member that folds in one gate and reds in the other is the same
// failure class as an oracle exclusion applied to one enumerator and not its twin —
// the two gates become mutually exclusive and no commit can satisfy both. Keeping
// one table means a correction cannot land in half the pipeline.

// GoNameToSnake folds an exported Go PascalCase identifier to snake_case.
// Initialism runs (CallID->call_id, SIPReferTo->sip_refer_to) fold correctly via
// the uppercase->Aa boundary rule (a `_` before an uppercase that begins a new
// word). It does NOT handle initialism PLURALS or wire keys — use GoNameToPython,
// which applies the correction table first.
func GoNameToSnake(s string) string {
	var out strings.Builder
	for i, r := range s {
		if i > 0 {
			prev := rune(s[i-1])
			if (isUpper(r) && isLower(prev)) ||
				(isUpper(r) && i+1 < len(s) && isLower(rune(s[i+1])) && isUpper(prev)) {
				out.WriteByte('_')
			}
		}
		out.WriteRune(toLowerRune(r))
	}
	return out.String()
}

// nameCorrections are the identifiers GoNameToSnake gets wrong, and the canonical
// reference spelling for each. Three kinds:
//
//   - INITIALISM PLURALS — GoNameToSnake breaks at the internal uppercase-run
//     boundary, so `URLs` becomes "ur_ls" and `FAQs` becomes "fa_qs".
//   - INITIALISM RUNS the reference spells as one lowercase word (`MFA` -> "mfa",
//     `SWMLScripts` -> "swml_scripts").
//   - WIRE KEYS that are genuinely camelCase in the reference and must NOT be
//     snake-cased. `numberedBullets` round-trips through the POM dict verbatim
//     (pom.py:345,361,371), so converting it would break the wire. Only four such
//     members exist in the whole oracle — this one plus JSON-Schema's
//     allOf/anyOf/oneOf.
var nameCorrections = map[string]string{
	"URLs": "urls",
	"FAQs": "faqs",
	"MFA":  "mfa",

	"PubSub":               "pubsub",
	"FreeSwitchConnectors": "freeswitch_connectors",
	"SIPEndpoints":         "sip_endpoints",
	"SIPGateways":          "sip_gateways",
	"SWMLScripts":          "swml_scripts",
	"SWMLWebhooks":         "swml_webhooks",
	"CXMLScripts":          "cxml_scripts",
	"CXMLApplications":     "cxml_applications",
	"CXMLWebhooks":         "cxml_webhooks",

	"NumberedBullets": "numberedBullets",
}

// GoNameToPython folds an exported Go identifier to its Python-canonical name,
// applying nameCorrections before falling back to GoNameToSnake.
func GoNameToPython(s string) string {
	if fixed, ok := nameCorrections[s]; ok {
		return fixed
	}
	return GoNameToSnake(s)
}

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isLower(r rune) bool { return r >= 'a' && r <= 'z' }
func toLowerRune(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}
