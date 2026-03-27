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
	"time"

	"github.com/signalwire/signalwire-go/pkg/logging"
)

// SessionManager creates and validates HMAC-SHA256 tokens for SWAIG function
// calls. Each manager instance generates its own random secret on creation.
type SessionManager struct {
	secret          []byte
	tokenExpirySecs int
	logger          *logging.Logger
}

// NewSessionManager creates a new SessionManager with a randomly generated
// 32-byte secret. If tokenExpirySecs is <= 0, a default of 3600 seconds
// (1 hour) is used.
func NewSessionManager(tokenExpirySecs int) *SessionManager {
	if tokenExpirySecs <= 0 {
		tokenExpirySecs = 3600
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		// This should never happen on a properly configured system.
		panic("security: failed to generate random secret: " + err.Error())
	}

	return &SessionManager{
		secret:          secret,
		tokenExpirySecs: tokenExpirySecs,
		logger:          logging.New("SessionManager"),
	}
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
