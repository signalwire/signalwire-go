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

// The reference declares all four fields REQUIRED, and `required` is contract. A Go
// composite literal cannot carry that (an omitted field zero-values), so on its own
// the construction contract reads all four as optional and the signature differ
// emits a `construction-required-flip` on each. A `New<Struct>` factory IS the Go
// mechanism that carries a requirement — the enumerator reads a factory's params as
// required (cmd/enumerate-signatures/main.go, `factoryRequired`) — so the two
// factories below resolve all four flips.
//
// An earlier turn wrote these, measured them, and correctly REVERTED them, because
// the two oracles disagreed about these two classes at the time:
//
//	python_signatures.json  BasicCredentials -> [__init__, password, username]
//	python_surface.json     BasicCredentials -> [password, username]
//
// Mapping a factory to `__init__` satisfied the signatures oracle and immediately
// violated the surface one, landing the emitted `__init__` as an unexcused
// SURFACE-DIFF extra — trading a clean ENFORCING gate for a REPORT-ONLY gain, which
// is the wrong trade. porting-sdk 8828dd2 fixed the oracle (python_surface.json now
// records the synthesized dataclass `__init__` for all 30 affected classes, these
// two among them), so that trade no longer exists and the factories are back.
//
// They are constructors, not convenience helpers: each takes exactly the reference
// ctor's parameters, in the reference's order, and is mapped to `__init__` in
// internal/surface/tables.go. Composite-literal construction remains available and
// is what the transport-free note above describes.

// NewBasicCredentials builds a BasicCredentials from a decoded `Basic` credential
// pair. Mirrors the reference constructor
// signalwire.core.auth_handler.BasicCredentials(username, password), whose two
// fields are both required.
func NewBasicCredentials(username, password string) *BasicCredentials {
	return &BasicCredentials{Username: username, Password: password}
}

// NewBearerCredentials builds a BearerCredentials from a parsed Authorization
// header. Mirrors the reference constructor
// signalwire.core.auth_handler.BearerCredentials(scheme, credentials), whose two
// fields are both required. Scheme is preserved verbatim, not normalized.
func NewBearerCredentials(scheme, credentials string) *BearerCredentials {
	return &BearerCredentials{Scheme: scheme, Credentials: credentials}
}
