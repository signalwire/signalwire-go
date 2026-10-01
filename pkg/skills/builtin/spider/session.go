package spider

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/util"
)

// publicTransport is the RoundTripper behind the spider's session: it checks the
// URL of every request it sends — the first one and every redirect hop, which
// net/http also sends through the transport — and, when it connects directly,
// refuses a connection whose actual peer is a private or internal address (DNS
// rebinding would otherwise pass a resolve-time check). SWML_ALLOW_PRIVATE_URLS
// turns both checks off, as it does for util.ValidateURL. Proxies are ignored
// unless SWML_URL_FETCH_USE_PROXY is set, because through a proxy the peer check
// cannot apply.
type publicTransport struct {
	base *http.Transport
}

// RoundTrip validates the request URL, then sends it.
func (t *publicTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !util.ValidateURL(req.URL.String(), false) {
		return nil, fmt.Errorf("URL rejected: %s://%s is private, internal or invalid", req.URL.Scheme, req.URL.Host)
	}
	return t.base.RoundTrip(req)
}

// envTruthy reports whether an environment variable is set to 1/true/yes.
func envTruthy(name string) bool {
	switch strings.ToLower(os.Getenv(name)) {
	case "1", "true", "yes":
		return true
	}
	return false
}

// newPublicSession builds the HTTP client the spider fetches user-supplied URLs
// with (the reference's _PublicSession).
func newPublicSession(timeout time.Duration) *http.Client {
	useProxy := envTruthy("SWML_URL_FETCH_USE_PROXY")
	dialer := &net.Dialer{Timeout: timeout}
	base := &http.Transport{
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: timeout,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			if useProxy || envTruthy("SWML_ALLOW_PRIVATE_URLS") {
				return conn, nil
			}
			peer, _, splitErr := net.SplitHostPort(conn.RemoteAddr().String())
			if splitErr != nil || !util.ValidateURL("http://"+net.JoinHostPort(peer, "80"), false) {
				_ = conn.Close()
				return nil, fmt.Errorf("refused to connect to %s: %s is a private or internal address", addr, peer)
			}
			return conn, nil
		},
	}
	if useProxy {
		base.Proxy = http.ProxyFromEnvironment
	}
	return &http.Client{Timeout: timeout, Transport: &publicTransport{base: base}}
}

// Session returns the HTTP client the skill fetches pages with. Every request it
// sends — redirects included — is checked against private and internal
// addresses (see SWML_ALLOW_PRIVATE_URLS / SWML_URL_FETCH_USE_PROXY).
func (s *SpiderSkill) Session() *http.Client {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if s.session == nil {
		timeout := s.timeout
		if timeout <= 0 {
			timeout = 5
		}
		s.session = newPublicSession(time.Duration(timeout) * time.Second)
	}
	return s.session
}
