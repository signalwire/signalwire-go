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

	parts := strings.SplitN(string(decoded), ".", 2)
	if len(parts) != 2 {
		t.Fatal("unexpected token format")
	}

	// Change the function name in the message.
	tampered := strings.Replace(parts[0], "get_weather", "evil_func", 1) + "." + parts[1]
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

	parts := strings.SplitN(string(decoded), ".", 2)
	if len(parts) != 2 {
		t.Fatal("unexpected token format")
	}

	// Replace the signature with a bogus one.
	bogus := parts[0] + "." + hex.EncodeToString(make([]byte, 32))
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

func TestValidateToken_MissingSeparator(t *testing.T) {
	sm := NewSessionManager(3600)
	noSep := base64.URLEncoding.EncodeToString([]byte("nodothere"))
	if sm.ValidateToken("get_weather", noSep, "call-123") {
		t.Error("expected validation to fail for token without separator")
	}
}

func TestValidateToken_InvalidHexSignature(t *testing.T) {
	sm := NewSessionManager(3600)
	bad := base64.URLEncoding.EncodeToString([]byte("a:b:999999999999.not_hex!!!"))
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

	// Should contain a dot separator.
	parts := strings.SplitN(string(decoded), ".", 2)
	if len(parts) != 2 {
		t.Fatal("decoded token should have message.signature format")
	}

	// Message should be functionName:callID:expiry.
	msgParts := strings.SplitN(parts[0], ":", 3)
	if len(msgParts) != 3 {
		t.Fatal("message should have three colon-separated parts")
	}

	if msgParts[0] != "my_func" {
		t.Errorf("expected function name 'my_func', got %q", msgParts[0])
	}
	if msgParts[1] != "call-xyz" {
		t.Errorf("expected call ID 'call-xyz', got %q", msgParts[1])
	}

	expiry, err := strconv.ParseInt(msgParts[2], 10, 64)
	if err != nil {
		t.Fatalf("expiry should be a valid integer: %v", err)
	}

	now := time.Now().Unix()
	if expiry < now || expiry > now+3601 {
		t.Errorf("expiry %d is not within expected range [%d, %d]", expiry, now, now+3601)
	}

	// Signature should be valid hex-encoded HMAC-SHA256.
	sigBytes, err := hex.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("signature should be valid hex: %v", err)
	}

	mac := hmac.New(sha256.New, sm.secret)
	mac.Write([]byte(parts[0]))
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(sigBytes, expectedSig) {
		t.Error("signature does not match expected HMAC-SHA256")
	}
}
