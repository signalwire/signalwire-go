package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewSessionManager_DefaultExpiry(t *testing.T) {
	sm := NewSessionManager(0)
	if sm.tokenExpirySecs != 900 {
		t.Errorf("expected default expiry 900, got %d", sm.tokenExpirySecs)
	}
}

func TestNewSessionManager_NegativeExpiry(t *testing.T) {
	sm := NewSessionManager(-10)
	if sm.tokenExpirySecs != 900 {
		t.Errorf("expected default expiry 900 for negative input, got %d", sm.tokenExpirySecs)
	}
}

func TestNewSessionManager_CustomExpiry(t *testing.T) {
	sm := NewSessionManager(7200)
	if sm.tokenExpirySecs != 7200 {
		t.Errorf("expected expiry 7200, got %d", sm.tokenExpirySecs)
	}
}

func TestNewSessionManager_SecretLength(t *testing.T) {
	sm := NewSessionManager(3600)
	if len(sm.secret) != 32 {
		t.Errorf("expected 32-byte secret, got %d bytes", len(sm.secret))
	}
}

func TestNewSessionManager_UniqueSecrets(t *testing.T) {
	sm1 := NewSessionManager(3600)
	sm2 := NewSessionManager(3600)
	if string(sm1.secret) == string(sm2.secret) {
		t.Error("two managers should not share the same secret")
	}
}

func TestCreateTokenAndValidate_HappyPath(t *testing.T) {
	sm := NewSessionManager(3600)
	functionName := "get_weather"
	callID := "call-abc-123"

	token := sm.CreateToken(functionName, callID)
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	if !sm.ValidateToken(functionName, token, callID) {
		t.Error("expected valid token to pass validation")
	}
}

func TestValidateToken_WrongFunctionName(t *testing.T) {
	sm := NewSessionManager(3600)
	token := sm.CreateToken("get_weather", "call-123")

	if sm.ValidateToken("set_weather", token, "call-123") {
		t.Error("expected validation to fail for wrong function name")
	}
}

func TestValidateToken_WrongCallID(t *testing.T) {
	sm := NewSessionManager(3600)
	token := sm.CreateToken("get_weather", "call-123")

	if sm.ValidateToken("get_weather", token, "call-456") {
		t.Error("expected validation to fail for wrong call ID")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Use a manager with 0-second expiry (will default to 3600), so we
	// construct one with 1-second expiry and sleep past it.
	sm := NewSessionManager(1)
	token := sm.CreateToken("get_weather", "call-123")

	// Wait for the token to expire.
	time.Sleep(2 * time.Second)

	if sm.ValidateToken("get_weather", token, "call-123") {
		t.Error("expected validation to fail for expired token")
	}
}

func TestValidateToken_TamperedToken(t *testing.T) {
	sm := NewSessionManager(3600)
	token := sm.CreateToken("get_weather", "call-123")

	// Decode, tamper with the message, re-encode.
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("failed to decode token: %v", err)
	}

	parts := strings.Split(string(decoded), ".")
	if len(parts) != 5 {
		t.Fatal("unexpected token format")
	}

	// Change the function name field (parts[1]) without re-signing.
	parts[1] = "evil_func"
	tampered := strings.Join(parts, ".")
	tamperedToken := base64.URLEncoding.EncodeToString([]byte(tampered))

	if sm.ValidateToken("evil_func", tamperedToken, "call-123") {
		t.Error("expected validation to fail for tampered token")
	}
}

func TestValidateToken_TamperedSignature(t *testing.T) {
	sm := NewSessionManager(3600)
	token := sm.CreateToken("get_weather", "call-123")

	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("failed to decode token: %v", err)
	}

	parts := strings.Split(string(decoded), ".")
	if len(parts) != 5 {
		t.Fatal("unexpected token format")
	}

	// Replace the signature field (parts[4]) with a bogus one.
	parts[4] = hex.EncodeToString(make([]byte, 32))
	bogus := strings.Join(parts, ".")
	bogusToken := base64.URLEncoding.EncodeToString([]byte(bogus))

	if sm.ValidateToken("get_weather", bogusToken, "call-123") {
		t.Error("expected validation to fail for tampered signature")
	}
}

func TestValidateToken_EmptyToken(t *testing.T) {
	sm := NewSessionManager(3600)
	if sm.ValidateToken("get_weather", "", "call-123") {
		t.Error("expected validation to fail for empty token")
	}
}

func TestValidateToken_EmptyFunctionName(t *testing.T) {
	sm := NewSessionManager(3600)
	token := sm.CreateToken("", "call-123")

	// Should validate when the function name is also empty.
	if !sm.ValidateToken("", token, "call-123") {
		t.Error("expected token with empty function name to validate with empty function name")
	}

	// Should not validate against a non-empty function name.
	if sm.ValidateToken("get_weather", token, "call-123") {
		t.Error("expected validation to fail when function names differ")
	}
}

func TestValidateToken_EmptyCallID(t *testing.T) {
	sm := NewSessionManager(3600)
	token := sm.CreateToken("get_weather", "")

	if !sm.ValidateToken("get_weather", token, "") {
		t.Error("expected token with empty call ID to validate with empty call ID")
	}

	if sm.ValidateToken("get_weather", token, "call-123") {
		t.Error("expected validation to fail when call IDs differ")
	}
}

func TestValidateToken_InvalidBase64(t *testing.T) {
	sm := NewSessionManager(3600)
	if sm.ValidateToken("get_weather", "not-valid-base64!!!", "call-123") {
		t.Error("expected validation to fail for invalid base64")
	}
}

func TestValidateToken_WrongFieldCount(t *testing.T) {
	sm := NewSessionManager(3600)
	// A 3-field / no-nonce token (the OLD go format) must be rejected.
	threeField := base64.URLEncoding.EncodeToString([]byte("call.get_weather.999999999999"))
	if sm.ValidateToken("get_weather", threeField, "call") {
		t.Error("expected validation to fail for a 3-field token")
	}
	noSep := base64.URLEncoding.EncodeToString([]byte("nodothere"))
	if sm.ValidateToken("get_weather", noSep, "call-123") {
		t.Error("expected validation to fail for token without separators")
	}
}

func TestValidateToken_InvalidHexSignature(t *testing.T) {
	sm := NewSessionManager(3600)
	// 5 dot-fields but a non-hex signature — the constant-time hex-string
	// compare rejects it (it can never equal a valid hex HMAC digest).
	bad := base64.URLEncoding.EncodeToString([]byte("b.a.999999999999.deadbeefdeadbeef.not_hex!!!"))
	if sm.ValidateToken("a", bad, "b") {
		t.Error("expected validation to fail for invalid hex signature")
	}
}

func TestValidateToken_CrossManagerFails(t *testing.T) {
	sm1 := NewSessionManager(3600)
	sm2 := NewSessionManager(3600)

	token := sm1.CreateToken("get_weather", "call-123")

	if sm2.ValidateToken("get_weather", token, "call-123") {
		t.Error("expected validation to fail when using a different manager's token")
	}
}

func TestCreateToken_Format(t *testing.T) {
	sm := NewSessionManager(3600)
	token := sm.CreateToken("my_func", "call-xyz")

	// Should be valid base64url.
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("token is not valid base64url: %v", err)
	}

	// Decoded token: call_id.function_name.expiry.nonce.signature (5 fields).
	parts := strings.Split(string(decoded), ".")
	if len(parts) != 5 {
		t.Fatalf("decoded token should have 5 dot-fields, got %d", len(parts))
	}

	callID, functionName, expiryStr, nonce, sig := parts[0], parts[1], parts[2], parts[3], parts[4]

	if callID != "call-xyz" {
		t.Errorf("expected call ID 'call-xyz', got %q", callID)
	}
	if functionName != "my_func" {
		t.Errorf("expected function name 'my_func', got %q", functionName)
	}

	expiry, err := strconv.ParseInt(expiryStr, 10, 64)
	if err != nil {
		t.Fatalf("expiry should be a valid integer: %v", err)
	}

	now := time.Now().Unix()
	if expiry < now || expiry > now+3601 {
		t.Errorf("expiry %d is not within expected range [%d, %d]", expiry, now, now+3601)
	}

	// Nonce must be 16 hex chars (token_hex(8)).
	if len(nonce) != 16 {
		t.Errorf("expected 16-char nonce, got %d chars (%q)", len(nonce), nonce)
	}
	if _, err := hex.DecodeString(nonce); err != nil {
		t.Errorf("nonce should be hex: %v", err)
	}

	// Signature should be valid hex-encoded HMAC-SHA256 over
	// call_id:function_name:expiry:nonce.
	sigBytes, err := hex.DecodeString(sig)
	if err != nil {
		t.Fatalf("signature should be valid hex: %v", err)
	}

	message := callID + ":" + functionName + ":" + expiryStr + ":" + nonce
	mac := hmac.New(sha256.New, sm.secret)
	mac.Write([]byte(message))
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(sigBytes, expectedSig) {
		t.Error("signature does not match expected HMAC-SHA256")
	}
}

// TestContract7_TokenWireFormat is the contract-7 lock-in: python-equivalent 5-field
// wire format, non-empty per-mint nonce, nonce uniqueness across mints, and
// python-reference-format interop.
func TestContract7_TokenWireFormat(t *testing.T) {
	sm := NewSessionManager(3600)

	// (1) A freshly minted token, decoded, has exactly 5 dot-fields with a
	// NON-EMPTY nonce.
	tok := sm.CreateToken("get_weather", "call-abc-123")
	decoded, err := base64.URLEncoding.DecodeString(tok)
	if err != nil {
		t.Fatalf("token is not valid base64url: %v", err)
	}
	fields := strings.Split(string(decoded), ".")
	if len(fields) != 5 {
		t.Fatalf("expected 5 dot-fields, got %d: %q", len(fields), string(decoded))
	}
	if fields[0] != "call-abc-123" {
		t.Errorf("field 0 should be call_id, got %q", fields[0])
	}
	if fields[1] != "get_weather" {
		t.Errorf("field 1 should be function_name, got %q", fields[1])
	}
	if fields[3] == "" {
		t.Error("nonce (field 3) must be non-empty")
	}

	// (2) Two mints for the SAME (function_name, call_id, expiry) produce
	// DIFFERENT nonces.
	nonceOf := func(token string) string {
		d, derr := base64.URLEncoding.DecodeString(token)
		if derr != nil {
			t.Fatalf("decode: %v", derr)
		}
		p := strings.Split(string(d), ".")
		if len(p) != 5 {
			t.Fatalf("expected 5 fields, got %d", len(p))
		}
		return p[3]
	}
	n1 := nonceOf(sm.CreateToken("f", "c"))
	n2 := nonceOf(sm.CreateToken("f", "c"))
	if n1 == "" || n2 == "" {
		t.Fatal("nonces must be non-empty")
	}
	if n1 == n2 {
		t.Errorf("two mints must have different nonces, both were %q", n1)
	}

	// (3) A token constructed in the python-oracle format validates in-port
	// (cross-port interop). Build it exactly as Python does:
	//   message  = call_id:function_name:expiry:nonce
	//   sig      = hex(HMAC-SHA256(secret, message))
	//   token    = call_id.function_name.expiry.nonce.sig
	//   wire     = base64url(token)
	callID := "oracle-call-777"
	fn := "lookup"
	expiry := strconv.FormatInt(time.Now().Unix()+300, 10)
	nonce := "0123456789abcdef"
	msg := callID + ":" + fn + ":" + expiry + ":" + nonce
	mac := hmac.New(sha256.New, sm.secret)
	mac.Write([]byte(msg))
	oracleSig := hex.EncodeToString(mac.Sum(nil))
	oracleToken := callID + "." + fn + "." + expiry + "." + nonce + "." + oracleSig
	oracleWire := base64.URLEncoding.EncodeToString([]byte(oracleToken))

	if !sm.ValidateToken(fn, oracleWire, callID) {
		t.Error("a python-oracle-format token must validate in-port (contract-7 interop)")
	}

	// (4) Flip one byte of the signature → validation fails.
	badSig := []byte(oracleSig)
	if badSig[0] == 'a' {
		badSig[0] = 'b'
	} else {
		badSig[0] = 'a'
	}
	tamperedToken := callID + "." + fn + "." + expiry + "." + nonce + "." + string(badSig)
	tamperedWire := base64.URLEncoding.EncodeToString([]byte(tamperedToken))
	if sm.ValidateToken(fn, tamperedWire, callID) {
		t.Error("a signature-flipped token must fail validation")
	}
}
