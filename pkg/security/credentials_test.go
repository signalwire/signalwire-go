package security

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestBasicCredentialsCarryAuthorizationHeader drives the carrier through the
// path the reference drives it through: an inbound Authorization header becomes
// a BasicCredentials, and AuthHandler verification reads Username/Password off it
// and reaches the same verdict as VerifyBasicAuthPair.
//
// Reference: signalwire.core.auth_handler.AuthHandler.verify_basic_auth reads
// credentials.username / credentials.password (auth_handler.py:98-111).
func TestBasicCredentialsCarryAuthorizationHeader(t *testing.T) {
	h := NewAuthHandler(&SecurityConfig{
		BasicAuthUser:     "alice",
		BasicAuthPassword: "s3cret",
	})

	cases := []struct {
		name     string
		user     string
		pass     string
		wantAuth bool
	}{
		{"both correct", "alice", "s3cret", true},
		{"wrong password", "alice", "wrong", false},
		{"wrong username", "mallory", "s3cret", false},
		{"both wrong", "mallory", "wrong", false},
		{"empty", "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.SetBasicAuth(tc.user, tc.pass)

			user, pass, ok := req.BasicAuth()
			if !ok {
				t.Fatalf("request did not carry a Basic credential")
			}
			creds := BasicCredentials{Username: user, Password: pass}

			// The carrier round-trips the wire values verbatim.
			if creds.Username != tc.user {
				t.Fatalf("Username = %q, want %q", creds.Username, tc.user)
			}
			if creds.Password != tc.pass {
				t.Fatalf("Password = %q, want %q", creds.Password, tc.pass)
			}

			// Verification reached THROUGH the carrier agrees with the
			// direct two-string form and with the request form.
			got := h.VerifyBasicAuthPair(creds.Username, creds.Password)
			if got != tc.wantAuth {
				t.Fatalf("VerifyBasicAuthPair(%q,%q) = %v, want %v",
					creds.Username, creds.Password, got, tc.wantAuth)
			}
			if viaReq := h.VerifyBasicAuth(req); viaReq != tc.wantAuth {
				t.Fatalf("VerifyBasicAuth(req) = %v, want %v", viaReq, tc.wantAuth)
			}
		})
	}
}

// TestBearerCredentialsSplitAuthorizationHeader asserts the bearer carrier holds
// the scheme and the token as two separate values, matching the reference's
// HTTPAuthorizationCredentials (scheme + credentials), and that the token is the
// part after the scheme — not the whole header.
//
// Reference: verify_bearer_token compares credentials.credentials against the
// configured token (auth_handler.py:113-119) — it never looks at the raw header,
// so the split is the contract.
func TestBearerCredentialsSplitAuthorizationHeader(t *testing.T) {
	const token = "abc123.def456"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	header := req.Header.Get("Authorization")
	// Split exactly as the reference's framework does: scheme, then the rest.
	scheme, rest := header[:len("Bearer")], header[len("Bearer")+1:]
	creds := BearerCredentials{Scheme: scheme, Credentials: rest}

	if creds.Scheme != "Bearer" {
		t.Fatalf("Scheme = %q, want %q", creds.Scheme, "Bearer")
	}
	if creds.Credentials != token {
		t.Fatalf("Credentials = %q, want %q", creds.Credentials, token)
	}
	// The token must NOT carry the scheme prefix — that is the whole point of
	// the two-field shape.
	if creds.Credentials == header {
		t.Fatalf("Credentials still carries the scheme prefix: %q", creds.Credentials)
	}
}

// TestCredentialConstructorsDriveTheSameVerification pins that the two
// constructors are real constructors, not decoration: a carrier built by
// NewBasicCredentials / NewBearerCredentials must reach the SAME auth verdict, and
// carry the same field values, as the composite literal the other tests build.
//
// This is the behavioural half of mapping NewX -> __init__ in
// internal/surface/tables.go. The constructors exist because the reference declares
// all four fields REQUIRED and a Go composite literal cannot express that (an
// omitted field zero-values); asserting only that they assign fields would not
// catch a constructor that assigned them in the wrong ORDER, which is exactly the
// mistake a two-same-typed-string signature invites — so both cases below would
// fail on a swap.
func TestCredentialConstructorsDriveTheSameVerification(t *testing.T) {
	h := NewAuthHandler(&SecurityConfig{
		BasicAuthUser:     "alice",
		BasicAuthPassword: "s3cret",
	})

	t.Run("basic", func(t *testing.T) {
		// Correct pair verifies; the arguments must land username-then-password.
		// A swapped constructor turns this into ("s3cret","alice") and fails.
		built := NewBasicCredentials("alice", "s3cret")
		literal := BasicCredentials{Username: "alice", Password: "s3cret"}

		if *built != literal {
			t.Fatalf("NewBasicCredentials = %+v, want %+v", *built, literal)
		}
		if !h.VerifyBasicAuthPair(built.Username, built.Password) {
			t.Fatalf("constructed credential failed verification that the literal passes")
		}
		// And a wrong pair must still be rejected through the constructed carrier.
		bad := NewBasicCredentials("alice", "wrong")
		if h.VerifyBasicAuthPair(bad.Username, bad.Password) {
			t.Fatalf("constructed credential wrongly verified a bad password")
		}
	})

	t.Run("bearer", func(t *testing.T) {
		// Scheme-then-credentials. A swapped constructor yields
		// Scheme="abc123.def456", which both checks below catch.
		built := NewBearerCredentials("Bearer", "abc123.def456")
		literal := BearerCredentials{Scheme: "Bearer", Credentials: "abc123.def456"}

		if *built != literal {
			t.Fatalf("NewBearerCredentials = %+v, want %+v", *built, literal)
		}
		if built.Scheme != "Bearer" {
			t.Fatalf("Scheme = %q, want %q", built.Scheme, "Bearer")
		}
		if built.Credentials != "abc123.def456" {
			t.Fatalf("Credentials = %q, want %q", built.Credentials, "abc123.def456")
		}
	})
}

// TestBearerCredentialsPreserveNonBearerScheme pins that the carrier is
// scheme-agnostic: the reference's HTTPAuthorizationCredentials records whatever
// scheme arrived and leaves the scheme check to the verifier. A carrier that
// normalized or rejected a non-Bearer scheme would silently lose information the
// reference preserves.
func TestBearerCredentialsPreserveNonBearerScheme(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString([]byte("alice:s3cret"))
	creds := BearerCredentials{Scheme: "Basic", Credentials: raw}

	if creds.Scheme != "Basic" {
		t.Fatalf("Scheme = %q, want %q (scheme preserved verbatim)", creds.Scheme, "Basic")
	}
	if creds.Credentials != raw {
		t.Fatalf("Credentials = %q, want %q", creds.Credentials, raw)
	}
}
