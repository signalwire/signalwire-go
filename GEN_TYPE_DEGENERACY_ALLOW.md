<!-- ══════════════════════════════════════════════════════════════════════════
BEFORE YOU ADD AN ENTRY TO THIS FILE — READ THIS.

Every entry here is a place the parity checker STOPS comparing. That is a real cost:
a divergence you list is a divergence no gate will ever catch again. So entries must
be RARE, and each one must earn its place. Default to skepticism: assume the entry is
NOT needed and make the case that it is.

The order of preference, always:
  1. FIX THE PORT so it matches the reference (add the missing member; make the
     signature match).
  2. FIX THE EMISSION so idiom folds onto the reference shape — the enumerator/emitter
     canonicalizes your language's spelling onto the oracle's (builder → __init__,
     getters → attributes, Result<T,E> → the plain return, CamelCase → the reference
     name, options-object/kwargs → the expanded param list, RAII/dispose → close).
     MOST divergences are idiom and belong here, not in this file.
  3. FIX THE REFERENCE if the oracle itself is wrong or stale (a Python-only symbol
     that leaked into the contract, a param the reference added and the oracle never
     re-enumerated). Fix Python / the oracle, then re-drift — do not paper over a
     broken reference with a per-port entry.
  4. Only when 1–3 genuinely cannot apply does an entry here become justified.

An entry is JUSTIFIED ONLY IF it is irreducible after correct emission — i.e. the
divergence survives because the two languages genuinely cannot express the same thing,
not because the emitter hasn't folded the idiom yet. If emission COULD fold it, the
entry is a bug in this file; go fix the emitter.

Each entry MUST state WHY, concretely, in one of these forms:
  • ADDITION — this symbol exists in the port but not the reference. Answer: is it
    genuine port-only surface with NO reference twin (say what it is and why the
    reference has no equivalent), or is it IDIOM the emitter should have folded (then
    it does not belong here — fold it)? A convenience/alias/back-compat wrapper is NOT
    a justification.
  • OMISSION — this reference symbol has no port member. Answer: WHY can it not exist
    here — what specific language feature is absent (e.g. no async-context-manager
    protocol, no __init__ method protocol)? "impossible:" means the construct cannot
    be expressed at all; if it merely LOOKS different, that's idiom → fold it, don't
    omit it. Cite a precedent when one exists (e.g. RelayClient omits the same dunder).
  • SIGNATURE — the symbol matches by name but its parameters differ. Answer: is the
    difference a foldable idiom collapse (options-object, leading context/self,
    builder) — then EXPAND it in the signature emitter so names+count match, don't list
    it — or a genuine reference-only parameter with no cross-language analogue?

If you cannot write a crisp, specific WHY that survives the "could emission fold this?"
test, the entry is not ready. Prove it's needed before you add it.
═══════════════════════════════════════════════════════════════════════════════ -->

# GEN_TYPE_DEGENERACY_ALLOW.md — justified loose-type exceptions (Go)

This file lists the generated-type findings the `gen_type_degeneracy` gate excuses,
each with a concrete reason. It is NOT a place to silence the gate — every entry is a
`oneOf`/`anyOf` **union** schema for which Go has no native union type.

## Why these are legitimate, not degenerate

Go has no sum/union type. A `oneOf`/`anyOf` schema (a value that is one of several
distinct object variants) therefore cannot be a single struct. The idiomatic Go
representation is the empty interface `any` — the same choice the generator's typing
rules document (`cmd/generate-rest/types.go`: "oneOf/anyOf = `any` (Go has no union
type)"). Crucially, PARITY IS PRESERVED: the Python oracle records each of these as a
named union class (e.g. `swml_verbs_generated.AIPrompt = AIPromptText | AIPromptPom`),
and the Go enumerator (`genTypeModule`) folds a field/return referencing the alias to
that same union class leaf — so signature/surface drift stays clean against Python.
Ports with a native union construct (TypeScript `A | B`, Rust tagged enums) type these
directly and are gate-clean without an allowlist; Go structurally cannot, and this is
the language-idiom exception the gate anticipates.

The `uuid`/`docid`/`jwt`/`play_url` scalar-format aliases are NOT here: those were a
real defect (an unexported type leaking into an exported field) and were fixed at the
template — the generator now exports them (`Uuid`/`Docid`/`Jwt`/`Play_url`) and the
enumerator folds the exported name back to the reference's lowercase canonical leaf.

## Entries

- `pkg/rest/namespaces/calling_types_generated.go:loose-alias:CallRequest` — CallRequest is an OpenAPI `oneOf of 37 variants`; Go has no union type, so the generator emits a `type CallRequest any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/calling_types_generated.go:loose-alias:CallResponse` — CallResponse is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type CallResponse any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/datasphere_types_generated.go:loose-alias:DocumentCreateRequest` — DocumentCreateRequest is an OpenAPI `oneOf of 4 variants`; Go has no union type, so the generator emits a `type DocumentCreateRequest any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:AIPostPrompt` — AIPostPrompt is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type AIPostPrompt any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:AIPostPromptUpdate` — AIPostPromptUpdate is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type AIPostPromptUpdate any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:AIPrompt` — AIPrompt is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type AIPrompt any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:AIPromptUpdate` — AIPromptUpdate is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type AIPromptUpdate any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:Action` — Action is an OpenAPI `anyOf of 16 variants`; Go has no union type, so the generator emits a `type Action any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:AddressChannel` — AddressChannel is an OpenAPI `anyOf of 3 variants`; Go has no union type, so the generator emits a `type AddressChannel any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:BedrockPostPrompt` — BedrockPostPrompt is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type BedrockPostPrompt any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:BedrockPrompt` — BedrockPrompt is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type BedrockPrompt any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:BedrockSWAIGFunction` — BedrockSWAIGFunction is an OpenAPI `anyOf of 4 variants`; Go has no union type, so the generator emits a `type BedrockSWAIGFunction any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:CallFlowVersionDeployRequest` — CallFlowVersionDeployRequest is an OpenAPI `oneOf of 2 variants`; Go has no union type, so the generator emits a `type CallFlowVersionDeployRequest any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:CondParams` — CondParams is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type CondParams any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:ContextSteps` — ContextSteps is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type ContextSteps any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:ContextsObject` — ContextsObject is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type ContextsObject any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:ContextsObjectUpdate` — ContextsObjectUpdate is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type ContextsObjectUpdate any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:FunctionFillers` — FunctionFillers is an OpenAPI `anyOf of 55 variants`; Go has no union type, so the generator emits a `type FunctionFillers any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:FunctionFillersUpdate` — FunctionFillersUpdate is an OpenAPI `anyOf of 55 variants`; Go has no union type, so the generator emits a `type FunctionFillersUpdate any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:Languages` — Languages is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type Languages any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:POM` — POM is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type POM any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:PayPromptAction` — PayPromptAction is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type PayPromptAction any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:ResourceResponse` — ResourceResponse is an OpenAPI `oneOf of 14 variants`; Go has no union type, so the generator emits a `type ResourceResponse any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:SWAIGFunction` — SWAIGFunction is an OpenAPI `anyOf of 4 variants`; Go has no union type, so the generator emits a `type SWAIGFunction any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:SWMLMethod` — SWMLMethod is an OpenAPI `anyOf of 38 variants`; Go has no union type, so the generator emits a `type SWMLMethod any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:SchemaType` — SchemaType is an OpenAPI `anyOf of 11 variants`; Go has no union type, so the generator emits a `type SchemaType any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:SummarizeActionUnion` — SummarizeActionUnion is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type SummarizeActionUnion any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:TranscribeAction` — TranscribeAction is an OpenAPI `anyOf of 3 variants`; Go has no union type, so the generator emits a `type TranscribeAction any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:TranscribeSummarizeActionUnion` — TranscribeSummarizeActionUnion is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type TranscribeSummarizeActionUnion any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:TranslateAction` — TranslateAction is an OpenAPI `anyOf of 4 variants`; Go has no union type, so the generator emits a `type TranslateAction any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/fabric_types_generated.go:loose-alias:ValidConfirmMethods` — ValidConfirmMethods is an OpenAPI `anyOf of 15 variants`; Go has no union type, so the generator emits a `type ValidConfirmMethods any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/relay_rest_types_generated.go:loose-alias:Recording` — Recording is an OpenAPI `anyOf of 3 variants`; Go has no union type, so the generator emits a `type Recording any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/video_types_generated.go:loose-alias:VideoLog` — VideoLog is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type VideoLog any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/rest/namespaces/voice_types_generated.go:loose-alias:VoiceLog` — VoiceLog is an OpenAPI `anyOf of 5 variants`; Go has no union type, so the generator emits a `type VoiceLog any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swaig/post_prompt_generated.go:loose-alias:PostPromptCallLogEntry` — PostPromptCallLogEntry is an OpenAPI `oneOf of 6 variants`; Go has no union type, so the generator emits a `type PostPromptCallLogEntry any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:AIPostPrompt` — AIPostPrompt is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type AIPostPrompt any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:AIPrompt` — AIPrompt is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type AIPrompt any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:Action` — Action is an OpenAPI `anyOf of 16 variants`; Go has no union type, so the generator emits a `type Action any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:BedrockPostPrompt` — BedrockPostPrompt is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type BedrockPostPrompt any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:BedrockPrompt` — BedrockPrompt is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type BedrockPrompt any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:BedrockSWAIGFunction` — BedrockSWAIGFunction is an OpenAPI `anyOf of 4 variants`; Go has no union type, so the generator emits a `type BedrockSWAIGFunction any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:CondParams` — CondParams is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type CondParams any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:ContextSteps` — ContextSteps is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type ContextSteps any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:ContextsObject` — ContextsObject is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type ContextsObject any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:FunctionFillers` — FunctionFillers is an OpenAPI `anyOf of 55 variants`; Go has no union type, so the generator emits a `type FunctionFillers any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:Languages` — Languages is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type Languages any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:POM` — POM is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type POM any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:PayPromptAction` — PayPromptAction is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type PayPromptAction any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:SWAIGFunction` — SWAIGFunction is an OpenAPI `anyOf of 4 variants`; Go has no union type, so the generator emits a `type SWAIGFunction any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:SWMLMethod` — SWMLMethod is an OpenAPI `anyOf of 38 variants`; Go has no union type, so the generator emits a `type SWMLMethod any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:SchemaType` — SchemaType is an OpenAPI `anyOf of 11 variants`; Go has no union type, so the generator emits a `type SchemaType any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:SummarizeActionUnion` — SummarizeActionUnion is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type SummarizeActionUnion any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:TranscribeAction` — TranscribeAction is an OpenAPI `anyOf of 3 variants`; Go has no union type, so the generator emits a `type TranscribeAction any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:TranscribeSummarizeActionUnion` — TranscribeSummarizeActionUnion is an OpenAPI `anyOf of 2 variants`; Go has no union type, so the generator emits a `type TranscribeSummarizeActionUnion any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:TranslateAction` — TranslateAction is an OpenAPI `anyOf of 4 variants`; Go has no union type, so the generator emits a `type TranslateAction any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)
- `pkg/swml/swml_verbs_generated.go:loose-alias:ValidConfirmMethods` — ValidConfirmMethods is an OpenAPI `anyOf of 15 variants`; Go has no union type, so the generator emits a `type ValidConfirmMethods any` alias. The Python oracle records the same wire shape as a union and the enumerator folds the Go leaf to that union class, so drift stays parity-clean. (burn-go, 2026-07-07)

<!-- user-approved 2026-07-07: all entries reviewed + approved for enforcing flip -->
