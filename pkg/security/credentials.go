package security

// This file carries the two inbound-credential value types the reference's
// authentication surface is written against:
// signalwire.core.auth_handler.BasicCredentials (username, password) and
// signalwire.core.auth_handler.BearerCredentials (scheme, credentials).
//
// In the reference these are FastAPI's HTTPBasicCredentials /
// HTTPAuthorizationCredentials, re-exported through signalwire.core.auth_handler.
// Until porting-sdk dcff742 griffe could not resolve those FastAPI names into the
// `signalwire.` tree, so the oracle emitted DANGLING class refs — names with no
// definition anywhere — and a port could neither match them nor coherently miss
// them. dcff742 filled them in as the real two-field classes they always were.
//
// The pydantic model is Python idiom; the CONTRACT is two strings each, and that
// is what parity is about. Go expresses it as plain structs with exported fields —
// no framework, no validation layer, no constructor ceremony. A composite literal
// (`security.BasicCredentials{Username: u, Password: p}`) is the Go analogue of
// the reference's generated dataclass `__init__`.
//
// Deliberately transport-free: nothing here imports net/http. A caller holding an
// *http.Request fills these from `r.BasicAuth()` / the Authorization header at the
// call site; a caller that parsed the header elsewhere (a gateway, a queue
// consumer, a test) builds one directly. That is the same split the reference gets
// from FastAPI's dependency injection, without binding the value type to a
// transport.

// BasicCredentials carries an HTTP Basic username/password pair decoded from an
// Authorization header (reference: signalwire.core.auth_handler.BasicCredentials,
// FastAPI's HTTPBasicCredentials).
type BasicCredentials struct {
	// Username is the user-id half of the decoded `Basic` credential.
	Username string
	// Password is the password half of the decoded `Basic` credential.
	Password string
}

// BearerCredentials carries a parsed Authorization header's scheme and credential
// token (reference: signalwire.core.auth_handler.BearerCredentials, FastAPI's
// HTTPAuthorizationCredentials).
//
// Scheme is retained separately from Credentials because the reference does: the
// header `Authorization: Bearer abc123` yields Scheme="Bearer",
// Credentials="abc123", so a verifier that only reads Credentials still lets a
// caller inspect the scheme the token arrived under.
type BearerCredentials struct {
	// Scheme is the authorization scheme exactly as it appeared on the wire
	// (canonically "Bearer") — preserved verbatim, not normalized.
	Scheme string
	// Credentials is the token following the scheme.
	Credentials string
}

// NO `NewBasicCredentials` / `NewBearerCredentials` FACTORY — deliberate, and the
// reason is an ORACLE INCONSISTENCY, not a port choice. Recorded here because the
// obvious next edit is to add one.
//
// The reference declares all four fields REQUIRED, and `required` is contract. Go's
// composite literal cannot express that (an omitted field zero-values), so the
// construction contract reads all four as optional and the signature differ emits a
// `construction-required-flip` on each. A `New<Struct>` factory is exactly the Go
// mechanism that carries a requirement, and the enumerator already reads a factory's
// params as required — so adding one resolves all four flips.
//
// It cannot be added, because the two oracles disagree about these two classes:
//
//	python_signatures.json  BasicCredentials -> [__init__, password, username]
//	python_surface.json     BasicCredentials -> [password, username]
//
// Mapping a factory to `__init__` satisfies the signatures oracle and immediately
// violates the surface oracle, which lists no `__init__` for these classes — the
// emitted `__init__` lands as an unexcused SURFACE-DIFF extra. Measured both ways:
// with the factory, excused falls by 4 (a REPORT-ONLY bucket) and SURFACE-DIFF gains
// 2 hard extras (an ENFORCING gate). Without it, SURFACE-DIFF is clean and the 4
// flips sit in the report-only construction bucket alongside go's 24 pre-existing
// ones.
//
// This asymmetry is specific to the two classes porting-sdk dcff742 hand-filled;
// the hand-written AuthHandler in the same module carries `__init__` in BOTH
// oracles. The fix belongs in the oracle (add `__init__` to python_surface.json's
// entry for these two classes, as griffe records for every other dataclass), not in
// a port working around it. Once the oracles agree, add the two factories and map
// them to `__init__`.
