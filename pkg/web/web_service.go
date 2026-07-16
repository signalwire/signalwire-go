// Package web provides a standalone static-file HTTP service, mirroring
// signalwire.web.web_service. It serves one or more directories under
// configurable routes, with optional basic auth, extension filtering and CORS.
package web

import (
	"context"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/signalwire/signalwire-go/v3/pkg/security"
)

// WebService serves static files from mounted directories over HTTP.
type WebService struct {
	port                    int
	enableDirectoryBrowsing bool
	allowedExtensions       []string
	blockedExtensions       []string
	maxFileSize             int64
	enableCORS              bool
	basicAuthUser           string
	basicAuthPassword       string
	securityConfig          *security.SecurityConfig

	mu          sync.RWMutex
	directories map[string]string // route -> directory
	server      *http.Server
}

// Options configures a WebService.
type Options struct {
	Port                    int
	Directories             map[string]string
	BasicAuthUser           string
	BasicAuthPassword       string
	ConfigFile              string
	EnableDirectoryBrowsing bool
	AllowedExtensions       []string
	BlockedExtensions       []string
	MaxFileSize             int64
	EnableCORS              bool
}

// NewWebService creates a WebService. Zero-valued options fall back to the
// reference defaults (port 8002, 100 MiB max file size, CORS enabled).
func NewWebService(opts Options) *WebService {
	port := opts.Port
	if port == 0 {
		port = 8002
	}
	maxFileSize := opts.MaxFileSize
	if maxFileSize == 0 {
		maxFileSize = 100 * 1024 * 1024
	}
	ws := &WebService{
		port:                    port,
		enableDirectoryBrowsing: opts.EnableDirectoryBrowsing,
		allowedExtensions:       opts.AllowedExtensions,
		blockedExtensions:       opts.BlockedExtensions,
		maxFileSize:             maxFileSize,
		enableCORS:              opts.EnableCORS,
		basicAuthUser:           opts.BasicAuthUser,
		basicAuthPassword:       opts.BasicAuthPassword,
		securityConfig:          security.NewSecurityConfig(),
		directories:             map[string]string{},
	}
	for route, dir := range opts.Directories {
		ws.directories[ws.normalizeRoute(route)] = dir
	}
	return ws
}

func (ws *WebService) normalizeRoute(route string) string {
	if route == "" {
		return "/"
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	return path.Clean(route)
}

// AddDirectory mounts a directory at the given route.
func (ws *WebService) AddDirectory(route, directory string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.directories[ws.normalizeRoute(route)] = directory
}

// RemoveDirectory unmounts the directory at the given route.
func (ws *WebService) RemoveDirectory(route string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	delete(ws.directories, ws.normalizeRoute(route))
}

// Security returns the WebService's SecurityConfig.
func (ws *WebService) Security() *security.SecurityConfig { return ws.securityConfig }

// handler builds the http.Handler serving all mounted directories.
func (ws *WebService) handler() http.Handler {
	mux := http.NewServeMux()
	ws.mu.RLock()
	for route, dir := range ws.directories {
		prefix := route
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		fs := http.StripPrefix(prefix, ws.fileServer(dir))
		mux.Handle(prefix, ws.wrap(fs))
	}
	ws.mu.RUnlock()
	return mux
}

func (ws *WebService) fileServer(dir string) http.Handler {
	base := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ws.isFileAllowed(r.URL.Path) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		base.ServeHTTP(w, r)
	})
}

func (ws *WebService) isFileAllowed(p string) bool {
	ext := strings.ToLower(path.Ext(p))
	for _, b := range ws.blockedExtensions {
		if strings.EqualFold(ext, b) {
			return false
		}
	}
	if len(ws.allowedExtensions) == 0 {
		return true
	}
	if ext == "" {
		// A directory or extension-less path is allowed only with browsing on.
		return ws.enableDirectoryBrowsing
	}
	for _, a := range ws.allowedExtensions {
		if strings.EqualFold(ext, a) {
			return true
		}
	}
	return false
}

// wrap applies CORS and basic-auth middleware around a handler.
func (ws *WebService) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ws.enableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		if ws.basicAuthPassword != "" {
			user, pass, ok := r.BasicAuth()
			if !ok || user != ws.basicAuthUser || pass != ws.basicAuthPassword {
				w.Header().Set("WWW-Authenticate", `Basic realm="web_service"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Start begins serving on host:port (port overrides the constructor port when
// non-zero). It blocks until Stop is called or the server errors. ssl paths, if
// both non-empty, enable TLS.
func (ws *WebService) Start(host string, port int, sslCert, sslKey string) error {
	if host == "" {
		host = "0.0.0.0"
	}
	if port == 0 {
		port = ws.port
	}
	addr := host + ":" + strconv.Itoa(port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           ws.handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	ws.mu.Lock()
	ws.server = srv
	ws.mu.Unlock()

	if sslCert != "" && sslKey != "" {
		return srv.ListenAndServeTLS(sslCert, sslKey)
	}
	return srv.ListenAndServe()
}

// Stop gracefully shuts the server down.
func (ws *WebService) Stop() error {
	ws.mu.RLock()
	srv := ws.server
	ws.mu.RUnlock()
	if srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
