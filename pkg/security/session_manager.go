// Package security provides session management and token-based authentication
// for SWAIG function calls, preventing unauthorized tool execution.
package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/signalwire/signalwire-go/pkg/logging"
)

// SessionManager creates and validates HMAC-SHA256 tokens for SWAIG function
// calls. The secret can be supplied at construction time (for cross-process
// validation) or auto-generated (default, single-process use).
type SessionManager struct {
	secret          []byte
	tokenExpirySecs int
	debugMode       bool
	logger          *logging.Logger
}

// Option is a functional option for NewSessionManager.
type Option func(*SessionManager)

// WithSecret injects a fixed secret key into the SessionManager. Use this when
// you need multiple processes or instances to validate each other's tokens.
// Pass nil to keep the default behaviour (auto-generate a random 32-byte secret).
func WithSecret(key []byte) Option {
	return func(sm *SessionManager) {
		if key != nil {
			sm.secret = key
		}
	}
}

// WithDebugMode enables the DebugToken method. Off by default to prevent
// accidental token introspection in production.
func WithDebugMode(enabled bool) Option {
	return func(sm *SessionManager) {
		sm.debugMode = enabled
	}
}

// NewSessionManager creates a new SessionManager. If tokenExpirySecs is <= 0,
// a default of 900 seconds (15 minutes) is used, matching the Python SDK
// default. Provide functional options (e.g. WithSecret) to customise behaviour.
func NewSessionManager(tokenExpirySecs int, opts ...Option) *SessionManager {
	if tokenExpirySecs <= 0 {
		tokenExpirySecs = 900
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		// This should never happen on a properly configured system.
		panic("security: failed to generate random secret: " + err.Error())
	}

	sm := &SessionManager{
		secret:          secret,
		tokenExpirySecs: tokenExpirySecs,
		debugMode:       false,
		logger:          logging.New("SessionManager"),
	}

	for _, opt := range opts {
		opt(sm)
	}

	return sm
}

// CreateSession returns callID unchanged if it is non-empty; otherwise it
// generates a cryptographically random URL-safe string (matches Python
// secrets.token_urlsafe(16) — 16 bytes of entropy, base64url-encoded without
// padding).
func (sm *SessionManager) CreateSession(callID string) string {
	if callID != "" {
		return callID
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("security: failed to generate session ID: " + err.Error())
	}
	// RawURLEncoding omits padding, matching Python's token_urlsafe behaviour.
	return base64.RawURLEncoding.EncodeToString(b)
}

// CreateToken generates an HMAC-SHA256 signed token for the given function
// name and call ID. The token embeds an expiry timestamp and is returned as a
// base64url-encoded string.
func (sm *SessionManager) CreateToken(functionName string, callID string) string {
	expiry := time.Now().Unix() + int64(sm.tokenExpirySecs)
	message := functionName + ":" + callID + ":" + strconv.FormatInt(expiry, 10)

	mac := hmac.New(sha256.New, sm.secret)
	mac.Write([]byte(message))
	signature := mac.Sum(nil)

	payload := message + "." + hex.EncodeToString(signature)
	return base64.URLEncoding.EncodeToString([]byte(payload))
}

// ValidateToken verifies that a token is authentic, unexpired, and matches
// the expected function name and call ID. All comparisons are performed in
// constant time where possible to prevent timing attacks. Returns true only
// if every check passes.
func (sm *SessionManager) ValidateToken(functionName string, token string, callID string) bool {
	if token == "" {
		sm.logger.Debug("token validation failed: empty token")
		return false
	}

	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		sm.logger.Debug("token validation failed: base64 decode error: %v", err)
		return false
	}

	parts := strings.SplitN(string(decoded), ".", 2)
	if len(parts) != 2 {
		sm.logger.Debug("token validation failed: invalid token format (missing separator)")
		return false
	}

	message := parts[0]
	sigHex := parts[1]

	// Verify HMAC-SHA256 signature using constant-time comparison.
	providedSig, err := hex.DecodeString(sigHex)
	if err != nil {
		sm.logger.Debug("token validation failed: hex decode error: %v", err)
		return false
	}

	mac := hmac.New(sha256.New, sm.secret)
	mac.Write([]byte(message))
	expectedSig := mac.Sum(nil)

	if subtle.ConstantTimeCompare(providedSig, expectedSig) != 1 {
		sm.logger.Debug("token validation failed: signature mismatch")
		return false
	}

	// Parse the message: "functionName:callID:expiryTimestamp"
	msgParts := strings.SplitN(message, ":", 3)
	if len(msgParts) != 3 {
		sm.logger.Debug("token validation failed: invalid message format")
		return false
	}

	tokenFunctionName := msgParts[0]
	tokenCallID := msgParts[1]
	tokenExpiryStr := msgParts[2]

	// Check function name matches.
	if subtle.ConstantTimeCompare([]byte(tokenFunctionName), []byte(functionName)) != 1 {
		sm.logger.Debug("token validation failed: function name mismatch (expected %q)", functionName)
		return false
	}

	// Check call ID matches.
	if subtle.ConstantTimeCompare([]byte(tokenCallID), []byte(callID)) != 1 {
		sm.logger.Debug("token validation failed: call ID mismatch (expected %q)", callID)
		return false
	}

	// Check expiry.
	expiryUnix, err := strconv.ParseInt(tokenExpiryStr, 10, 64)
	if err != nil {
		sm.logger.Debug("token validation failed: invalid expiry timestamp: %v", err)
		return false
	}

	if expiryUnix <= time.Now().Unix() {
		sm.logger.Debug("token validation failed: token expired at %d", expiryUnix)
		return false
	}

	return true
}

// DebugToken decodes a token and returns a map of its components and status
// without performing signature validation. This is intended for development
// and debugging only. It requires debug mode to be enabled via WithDebugMode;
// if not, it returns map["error": "debug mode not enabled"].
func (sm *SessionManager) DebugToken(token string) map[string]any {
	if !sm.debugMode {
		return map[string]any{"error": "debug mode not enabled"}
	}

	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return map[string]any{
			"valid_format": false,
			"error":        err.Error(),
			"token_length": len(token),
		}
	}

	// Go token format: "functionName:callID:expiry.hexsig"
	// Split on the single dot separator between message and signature.
	dotParts := strings.SplitN(string(decoded), ".", 2)
	if len(dotParts) != 2 {
		return map[string]any{
			"valid_format": false,
			"parts_count":  len(dotParts),
			"token_length": len(token),
		}
	}

	message := dotParts[0]
	tokenSignature := dotParts[1]

	// Parse "functionName:callID:expiry"
	msgParts := strings.SplitN(message, ":", 3)
	if len(msgParts) != 3 {
		return map[string]any{
			"valid_format": false,
			"parts_count":  len(msgParts),
			"token_length": len(token),
		}
	}

	tokenFunction := msgParts[0]
	tokenCallID := msgParts[1]
	tokenExpiryStr := msgParts[2]

	currentTime := time.Now().Unix()

	var isExpired any
	var expiresIn any
	var expiryDate any
	var expiryRaw any = tokenExpiryStr

	expiryUnix, parseErr := strconv.ParseInt(tokenExpiryStr, 10, 64)
	if parseErr == nil {
		expired := expiryUnix < currentTime
		isExpired = expired
		if !expired {
			expiresIn = expiryUnix - currentTime
		} else {
			expiresIn = int64(0)
		}
		expiryDate = fmt.Sprintf("%s", time.Unix(expiryUnix, 0).Format(time.RFC3339))
		expiryRaw = tokenExpiryStr
	}

	// Truncate call_id and signature for safety, matching Python behaviour.
	callIDDisplay := tokenCallID
	if len(callIDDisplay) > 8 {
		callIDDisplay = callIDDisplay[:8] + "..."
	}
	sigDisplay := tokenSignature
	if len(sigDisplay) > 8 {
		sigDisplay = sigDisplay[:8] + "..."
	}

	return map[string]any{
		"valid_format": true,
		"components": map[string]any{
			"call_id":     callIDDisplay,
			"function":    tokenFunction,
			"expiry":      expiryRaw,
			"expiry_date": expiryDate,
			"nonce":       nil, // Go token format has no nonce field
			"signature":   sigDisplay,
		},
		"status": map[string]any{
			"current_time":       currentTime,
			"is_expired":         isExpired,
			"expires_in_seconds": expiresIn,
		},
	}
}
