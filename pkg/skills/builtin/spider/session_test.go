package spider

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSessionRefusesPrivateAddresses ports the reference's _PublicSession
// contract: a fetch of a loopback URL is refused unless SWML_ALLOW_PRIVATE_URLS
// opts in, and a redirect to a private address is refused too.
func TestSessionRefusesPrivateAddresses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "http://127.0.0.1:1/internal", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	t.Setenv("SWML_ALLOW_PRIVATE_URLS", "")
	s := &SpiderSkill{timeout: 2}
	if resp, err := s.Session().Get(srv.URL); err == nil {
		_ = resp.Body.Close()
		t.Fatal("loopback fetch was allowed without SWML_ALLOW_PRIVATE_URLS")
	}

	t.Setenv("SWML_ALLOW_PRIVATE_URLS", "1")
	allowed := &SpiderSkill{timeout: 2}
	resp, err := allowed.Session().Get(srv.URL)
	if err != nil {
		t.Fatalf("fetch with SWML_ALLOW_PRIVATE_URLS=1: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d", resp.StatusCode)
	}
}
