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
// reference spelling for each. Four kinds:
//
//   - INITIALISM PLURALS — GoNameToSnake breaks at the internal uppercase-run
//     boundary, so `URLs` becomes "ur_ls" and `FAQs` becomes "fa_qs".
//   - INITIALISM RUNS the reference spells as one lowercase word (`MFA` -> "mfa",
//     `SWMLScripts` -> "swml_scripts").
//   - WIRE KEYS that are genuinely camelCase in the reference and must NOT be
//     snake-cased. `numberedBullets` round-trips through the POM dict verbatim
//     (pom.py:345,361,371), so converting it would break the wire. Only four such
//     members exist in the whole reference surface — this one plus JSON-Schema's
//     allOf/anyOf/oneOf.
//   - PROTOCOL-SPELLING RENAMES — the reference names a member after the legacy
//     protocol ("ssl") where Go names it after the current one ("TLS"). These are
//     the SAME caller-observable attribute under two spellings, so they are folded
//     HERE (the adapter rename table), never carried as an omission. Renaming the
//     Go member instead would be a gratuitous break of an established public API
//     to paper over a pure naming difference. Each is unambiguous: `TLSEnabled` is
//     declared exactly once in the SDK (pkg/swml/service.go, *Service).
var nameCorrections = map[string]string{
	"URLs": "urls",
	"FAQs": "faqs",
	"MFA":  "mfa",

	// swml.Service TLS accessors <-> reference SWMLService.ssl_* attributes.
	"TLSEnabled":  "ssl_enabled",
	"TLSCertPath": "ssl_cert_path",
	"TLSKeyPath":  "ssl_key_path",

	"PubSub":               "pubsub",
	"FreeSwitchConnectors": "freeswitch_connectors",
	"SIPEndpoints":         "sip_endpoints",
	"SIPGateways":          "sip_gateways",
	"SWMLScripts":          "swml_scripts",
	"SWMLWebhooks":         "swml_webhooks",
	"CXMLScripts":          "cxml_scripts",
	"CXMLApplications":     "cxml_applications",
	"CXMLWebhooks":         "cxml_webhooks",

	// `XPath` is one word in the reference's spelling (`remove_xpaths`), but
	// GoNameToSnake breaks at the X->P uppercase-run boundary ("remove_x_paths").
	"RemoveXPaths": "remove_xpaths",

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

// promotedFieldCarriers are the UNEXPORTED struct types whose exported fields Go
// promotes onto an exported embedder, making those FIELDS public API even though
// the carrier TYPE is unexported.
//
// `_GeneratedResourceTree` is the only one. `rest.RestClient` embeds it, and its
// 22 fields (`Fabric`, `Calling`, `Video`, …) are the client's namespace
// accessors — the exact members the reference exposes as `client.fabric`,
// `client.calling`, `client.video`. The leading underscore keeps the tree TYPE
// off the public surface (and is required: Go forbids embedding a cross-package
// underscore-unexported type, which is why the tree lives in package `rest`); it
// does not make the promoted fields private, and `client.Fabric` resolves
// through the embed exactly as the reference's `client.fabric` does.
//
// Both enumerators skip unexported types when walking type declarations, so
// without this exemption the tree is never recorded, the embed resolves to
// nothing, and all 22 accessors read as "missing-port" on BOTH the signature and
// surface axes — a blind spot in the walkers, not an absent capability (the
// accessors are proven live by the mock-backed TestResourceTreeAccessors_*
// tests in pkg/rest/namespaces).
//
// Shared here, alongside the name-fold table, for the reason that file documents:
// the two enumerators previously each carried their own copy of a fold table and
// DIVERGED, making the two gates mutually unsatisfiable. One table means an
// exemption cannot land in half the pipeline.
var promotedFieldCarriers = map[string]bool{
	"_GeneratedResourceTree": true,
}

// IsPromotedFieldCarrier reports whether an unexported type must still be walked
// because an exported struct embeds it and promotes its fields onto the public
// surface. See promotedFieldCarriers.
func IsPromotedFieldCarrier(name string) bool { return promotedFieldCarriers[name] }

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isLower(r rune) bool { return r >= 'a' && r <= 'z' }
func toLowerRune(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}
