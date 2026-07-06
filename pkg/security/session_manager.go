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
	"strconv"
	"strings"
	"sync"
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

	mu             sync.RWMutex
	activeSessions map[string]struct{}
	sessionMeta    map[string]map[string]any
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
		activeSessions:  map[string]struct{}{},
		sessionMeta:     map[string]map[string]any{},
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

// newNonce returns 16 hex characters of cryptographic randomness, matching
// Python's secrets.token_hex(8) (8 random bytes → 16 hex chars).
func newNonce() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic("security: failed to generate token nonce: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// CreateToken generates an HMAC-SHA256 signed token for the given function
// name and call ID. The token embeds an expiry timestamp and a per-mint nonce
// and is returned as a base64url-encoded string. The DECODED token is the
// 5-field dot-joined form matching the Python oracle:
// {call_id}.{function_name}.{expiry}.{nonce}.{signature}, where the signed
// message is {call_id}:{function_name}:{expiry}:{nonce}.
func (sm *SessionManager) CreateToken(functionName string, callID string) string {
	expiry := time.Now().Unix() + int64(sm.tokenExpirySecs)
	nonce := newNonce()
	message := callID + ":" + functionName + ":" + strconv.FormatInt(expiry, 10) + ":" + nonce

	mac := hmac.New(sha256.New, sm.secret)
	mac.Write([]byte(message))
	signature := hex.EncodeToString(mac.Sum(nil))

	token := callID + "." + functionName + "." + strconv.FormatInt(expiry, 10) + "." + nonce + "." + signature
	return base64.URLEncoding.EncodeToString([]byte(token))
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

	// Decoded token: "{call_id}.{function_name}.{expiry}.{nonce}.{signature}"
	parts := strings.Split(string(decoded), ".")
	if len(parts) != 5 {
		sm.logger.Debug("token validation failed: invalid token format (expected 5 dot-fields, got %d)", len(parts))
		return false
	}

	tokenCallID := parts[0]
	tokenFunctionName := parts[1]
	tokenExpiryStr := parts[2]
	tokenNonce := parts[3]
	sigHex := parts[4]

	// Verify HMAC-SHA256 signature using constant-time comparison.
	// Signed message: "{call_id}:{function_name}:{expiry}:{nonce}".
	message := tokenCallID + ":" + tokenFunctionName + ":" + tokenExpiryStr + ":" + tokenNonce
	mac := hmac.New(sha256.New, sm.secret)
	mac.Write([]byte(message))
	expectedSigHex := hex.EncodeToString(mac.Sum(nil))

	// Constant-time compare over the hex-encoded signatures (matches Python's
	// hmac.compare_digest on the hexdigest strings; no first-mismatch early return).
	if subtle.ConstantTimeCompare([]byte(sigHex), []byte(expectedSigHex)) != 1 {
		sm.logger.Debug("token validation failed: signature mismatch")
		return false
	}

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

	// Decoded token: "{call_id}.{function_name}.{expiry}.{nonce}.{signature}".
	parts := strings.Split(string(decoded), ".")
	if len(parts) != 5 {
		return map[string]any{
			"valid_format": false,
			"parts_count":  len(parts),
			"token_length": len(token),
		}
	}

	tokenCallID := parts[0]
	tokenFunction := parts[1]
	tokenExpiryStr := parts[2]
	tokenNonce := parts[3]
	tokenSignature := parts[4]

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
		expiryDate = time.Unix(expiryUnix, 0).Format(time.RFC3339)
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
			"nonce":       tokenNonce,
			"signature":   sigDisplay,
		},
		"status": map[string]any{
			"current_time":       currentTime,
			"is_expired":         isExpired,
			"expires_in_seconds": expiresIn,
		},
	}
}

// GenerateToken is the session-scoped token generator (Python
// SessionManager.generate_token). It is the underlying HMAC token primitive that
// CreateToken (create_tool_token) wraps for the SWAIG tool-call use case; the two
// share the same signing scheme so a tool token IS a session token for the
// (functionName, callID) pair.
func (sm *SessionManager) GenerateToken(functionName string, callID string) string {
	return sm.CreateToken(functionName, callID)
}

// ValidateSessionToken is the session-scoped token validator (Python
// SessionManager.validate_token), the counterpart to GenerateToken. It shares the
// verification path with ValidateToken (validate_tool_token).
func (sm *SessionManager) ValidateSessionToken(functionName string, token string, callID string) bool {
	return sm.ValidateToken(functionName, token, callID)
}

// ActivateSession marks a session id active (Python
// SessionManager.activate_session), returning false if it was already active.
func (sm *SessionManager) ActivateSession(sessionID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if _, ok := sm.activeSessions[sessionID]; ok {
		return false
	}
	sm.activeSessions[sessionID] = struct{}{}
	return true
}

// EndSession removes a session and its metadata (Python
// SessionManager.end_session), returning false if it was not active.
func (sm *SessionManager) EndSession(sessionID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if _, ok := sm.activeSessions[sessionID]; !ok {
		return false
	}
	delete(sm.activeSessions, sessionID)
	delete(sm.sessionMeta, sessionID)
	return true
}

// GetSessionMetadata returns the metadata map for a session (Python
// SessionManager.get_session_metadata), or nil if none is stored.
func (sm *SessionManager) GetSessionMetadata(sessionID string) map[string]any {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	meta, ok := sm.sessionMeta[sessionID]
	if !ok {
		return nil
	}
	out := make(map[string]any, len(meta))
	for k, v := range meta {
		out[k] = v
	}
	return out
}

// SetSessionMetadata stores metadata for a session (Python
// SessionManager.set_session_metadata).
func (sm *SessionManager) SetSessionMetadata(sessionID string, metadata map[string]any) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	copied := make(map[string]any, len(metadata))
	for k, v := range metadata {
		copied[k] = v
	}
	sm.sessionMeta[sessionID] = copied
}
