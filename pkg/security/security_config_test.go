package security

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetSSLContextKwargsPrimitives asserts that the TLS config surfaces the
// cert/key paths as primitive path strings under the same keys the Python
// reference's get_ssl_context_kwargs returns ({ssl_certfile, ssl_keyfile}).
func TestGetSSLContextKwargsPrimitives(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	if err := os.WriteFile(certPath, []byte("cert"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}

	c := &SecurityConfig{
		SSLEnabled:  true,
		SSLCertPath: certPath,
		SSLKeyPath:  keyPath,
	}
	kwargs := c.GetSSLContextKwargs()

	cert, ok := kwargs["ssl_certfile"].(string)
	if !ok || cert != certPath {
		t.Fatalf("ssl_certfile = %v, want %s", kwargs["ssl_certfile"], certPath)
	}
	key, ok := kwargs["ssl_keyfile"].(string)
	if !ok || key != keyPath {
		t.Fatalf("ssl_keyfile = %v, want %s", kwargs["ssl_keyfile"], keyPath)
	}
}

// TestGetSSLContextKwargsDisabled mirrors Python: an empty map when SSL is off.
func TestGetSSLContextKwargsDisabled(t *testing.T) {
	c := &SecurityConfig{SSLEnabled: false}
	if got := c.GetSSLContextKwargs(); len(got) != 0 {
		t.Fatalf("GetSSLContextKwargs() with SSL disabled = %v, want empty", got)
	}
}
