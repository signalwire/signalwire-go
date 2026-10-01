// Command token-interop-mint is the Go port's TOKEN-INTEROP mint fixture for the
// cross-port checker (porting-sdk/scripts/diff_port_token_interop.py).
//
// The contract being proven is property 3 of the SWAIG tool-token contract: a token
// this port MINTS must validate under the REFERENCE's own decoder. The other two
// properties (that a token is minted at all; that the HMAC is keyed with the
// secret_key STRING's bytes) already had coverage — this one did not, and a port can
// pass both and still emit a token no other implementation accepts, in which case
// every secure tool call fails authentication in production.
//
// Protocol: read the FIXED mint inputs from the environment (the checker owns them, so
// this fixture cannot drift from the values it is verified against), construct a
// SessionManager with that secret key, mint ONE token, and print JUST the token on
// stdout. Anything else belongs on stderr.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/token-interop-mint
package main

import (
	"fmt"
	"os"

	"github.com/signalwire/signalwire-go/v3/pkg/security"
)

// required reads a fixed mint input from the environment, or fails loud.
func required(name string) string {
	value := os.Getenv(name)
	if value == "" {
		fmt.Fprintf(os.Stderr,
			"%s is not set — the TOKEN-INTEROP checker supplies the fixed mint inputs "+
				"in the environment; run this via diff_port_token_interop.py --mint-cmd.\n",
			name)
		os.Exit(1)
	}
	return value
}

func main() {
	secretKey := required("SW_TOKEN_INTEROP_SECRET_KEY")
	callID := required("SW_TOKEN_INTEROP_CALL_ID")
	functionName := required("SW_TOKEN_INTEROP_FUNCTION_NAME")

	// Default expiry — the token must carry a FUTURE expiry, which the checker verifies.
	// WithSecretKey takes the secret as the reference's secret_key STRING (whose bytes
	// key the HMAC), not as 32 raw bytes.
	sm := security.NewSessionManager(900, security.WithSecretKey(secretKey))
	fmt.Println(sm.GenerateToken(functionName, callID))
}
