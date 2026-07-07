# WebService Documentation

The `WebService` type (package `github.com/signalwire/signalwire-go/pkg/web`) provides static file serving capabilities for the SignalWire AI Agents Go SDK. It can run as a standalone service or alongside your AI agents.

## Table of Contents
- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Security Features](#security-features)
- [HTTPS/SSL Support](#httpsssl-support)
- [API Endpoints](#api-endpoints)
- [Usage Examples](#usage-examples)
- [Deployment Patterns](#deployment-patterns)

## Overview

WebService is designed to serve static files with configurable security features. It's perfect for:
- Serving agent documentation and API specs
- Hosting static assets (images, CSS, JavaScript)
- Serving generated reports and exports
- Providing configuration files and templates
- Hosting agent UI components

### Key Features
- **Multiple directory mounting** - Serve different directories at different URL paths
- **Security-first design** - Authentication, CORS, security headers, file filtering
- **HTTPS support** - Full SSL/TLS support with PEM files
- **Directory browsing** - Optional HTML directory listings
- **MIME type handling** - Automatic content-type detection
- **Path traversal protection** - Prevents access outside designated directories
- **File filtering** - Allow/block specific file extensions

## Installation

WebService is included in the core SignalWire AI Agents Go SDK. Add it to your module:

```bash
go get github.com/signalwire/signalwire-go
```

Then import the `web` package:

```go
import "github.com/signalwire/signalwire-go/pkg/web"
```

## Quick Start

```go
package main

import "github.com/signalwire/signalwire-go/pkg/web"

func main() {
	// Create a service to serve files
	service := web.NewWebService(web.Options{
		Port: 8002,
		Directories: map[string]string{
			"/docs":   "./documentation",
			"/assets": "./static/assets",
		},
	})

	// Start the service (blocks). host="" defaults to 0.0.0.0, port 0 uses
	// the constructor port, and empty ssl paths serve plain HTTP.
	// Service available at http://localhost:8002
	if err := service.Start("", 0, "", ""); err != nil {
		panic(err)
	}
}
```

## Configuration

WebService can be configured through multiple methods (in order of priority):

### 1. Constructor Options

The `web.Options` struct configures the service:

```go
service := web.NewWebService(web.Options{
	Port: 8002, // Port to bind to
	Directories: map[string]string{ // URL path to directory mappings
		"/docs":   "./documentation",
		"/assets": "./static",
	},
	BasicAuthUser:           "admin", // Custom authentication
	BasicAuthPassword:       "secret",
	EnableDirectoryBrowsing: true,                             // Allow directory listings
	AllowedExtensions:       []string{".html", ".css", ".js"}, // Whitelist extensions
	BlockedExtensions:       []string{".env", ".key"},         // Blacklist extensions
	MaxFileSize:             100 * 1024 * 1024,                // Max file size (100MB)
	EnableCORS:              true,                             // Enable CORS headers
})
```

### 2. Environment Variables

```bash
# Basic authentication
export SWML_BASIC_AUTH_USER="admin"
export SWML_BASIC_AUTH_PASS="secretpassword"

# SSL/HTTPS configuration
export SWML_SSL_ENABLED=true
export SWML_SSL_CERT="/path/to/cert.pem"
export SWML_SSL_KEY="/path/to/key.pem"

# Security settings
export SWML_ALLOWED_HOSTS="example.com,*.example.com"
export SWML_CORS_ORIGINS="https://app.example.com"
```

### 3. Configuration File

Create a `web.json` or `swml_web.json` file:

```json
{
    "service": {
        "port": 8002,
        "directories": {
            "/docs": "./documentation",
            "/api": "./api-specs",
            "/reports": "./generated/reports"
        },
        "enable_directory_browsing": true,
        "max_file_size": 52428800,
        "allowed_extensions": [".html", ".css", ".js", ".json", ".pdf"],
        "blocked_extensions": [".env", ".key", ".pem"]
    },
    "security": {
        "basic_auth": {
            "username": "admin",
            "password": "secure123"
        },
        "ssl_enabled": true,
        "ssl_cert": "/etc/ssl/certs/server.crt",
        "ssl_key": "/etc/ssl/private/server.key",
        "allowed_hosts": ["*"],
        "cors_origins": ["*"]
    }
}
```

## Security Features

### Basic Authentication

WebService implements HTTP Basic Authentication. Credentials can be set via:

1. **Constructor**: `basic_auth=("username", "password")`
2. **Environment**: `SWML_BASIC_AUTH_USER` and `SWML_BASIC_AUTH_PASS`
3. **Config file**: `security.basic_auth` section
4. **Auto-generated**: If not specified, generates random credentials

### File Security

#### Default Blocked Extensions/Files
- `.env`, `.git`, `.gitignore`
- `.key`, `.pem`, `.crt`
- `.pyc`, `__pycache__`
- `.DS_Store`, `.swp`

#### Path Traversal Protection
WebService prevents access outside designated directories:
```text
These attempts will be blocked:
  GET /docs/../../../etc/passwd
  GET /docs/./././../config.json
```

#### File Size Limits
Default maximum file size is 100MB. Configure with:
```go
service := web.NewWebService(web.Options{MaxFileSize: 50 * 1024 * 1024}) // 50MB
```

### Security Headers

Automatically adds security headers to all responses:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Strict-Transport-Security` (when HTTPS is enabled)

## HTTPS/SSL Support

WebService provides multiple ways to enable HTTPS:

### Method 1: Environment Variables

```bash
# Using file paths
export SWML_SSL_CERT="/path/to/cert.pem"
export SWML_SSL_KEY="/path/to/key.pem"

# Or using inline PEM content
export SWML_SSL_CERT_INLINE="-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKLdQVPy...
-----END CERTIFICATE-----"
export SWML_SSL_KEY_INLINE="-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQE...
-----END PRIVATE KEY-----"
```

### Method 2: Direct Parameters

Pass the certificate and key paths as the last two arguments to `Start`:

```go
service := web.NewWebService(web.Options{
	Directories: map[string]string{"/docs": "./docs"},
})
// host, port, sslCert, sslKey — non-empty cert+key enables TLS.
service.Start("", 0, "/path/to/cert.pem", "/path/to/key.pem")
// Service available at https://localhost:8002
```

### Method 3: Configuration File

```json
{
    "security": {
        "ssl_enabled": true,
        "ssl_cert": "/etc/ssl/certs/server.crt",
        "ssl_key": "/etc/ssl/private/server.key"
    }
}
```

### Generating Self-Signed Certificates

For development/testing:

```bash
# Generate a self-signed certificate
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem \
    -days 365 -nodes -subj "/CN=localhost"

# Use with WebService
export SWML_SSL_CERT="cert.pem"
export SWML_SSL_KEY="key.pem"
```

## API Endpoints

### GET /health
Health check endpoint (no authentication required)

**Response:**
```json
{
    "status": "healthy",
    "directories": ["/docs", "/assets"],
    "ssl_enabled": false,
    "auth_required": true,
    "directory_browsing": true
}
```

### GET /
Root endpoint showing available directories

**Response:** HTML page listing all mounted directories

### GET /{route}/{file_path}
Serve files from mounted directories

**Parameters:**
- `route`: The mounted directory route (e.g., `/docs`)
- `file_path`: Path to file within the directory

**Response:**
- File content with appropriate MIME type
- 404 if file not found
- 403 if file type blocked or directory browsing disabled

## Usage Examples

### Basic File Serving

```go
package main

import "github.com/signalwire/signalwire-go/pkg/web"

func main() {
	// Serve documentation
	service := web.NewWebService(web.Options{
		Directories: map[string]string{
			"/docs": "./documentation",
			"/api":  "./api-specs",
		},
	})
	service.Start("", 0, "", "")

	// Files accessible at:
	//   http://localhost:8002/docs/index.html
	//   http://localhost:8002/api/swagger.json
}
```

### With Directory Browsing

```go
service := web.NewWebService(web.Options{
	Directories:             map[string]string{"/files": "./public"},
	EnableDirectoryBrowsing: true, // Allow browsing directories
})
service.Start("", 0, "", "")

// Browse files at: http://localhost:8002/files/
```

### Restricted File Types

```go
// Only serve web assets
service := web.NewWebService(web.Options{
	Directories:             map[string]string{"/web": "./www"},
	AllowedExtensions:       []string{".html", ".css", ".js", ".png", ".jpg", ".woff2"},
	EnableDirectoryBrowsing: false,
})
```

### Dynamic Directory Management

```go
service := web.NewWebService(web.Options{})

// Add directories after initialization
service.AddDirectory("/docs", "./documentation")
service.AddDirectory("/reports", "./generated/reports")

// Remove a directory
service.RemoveDirectory("/reports")

service.Start("", 0, "", "")
```

### With Custom Authentication

```go
service := web.NewWebService(web.Options{
	Directories:       map[string]string{"/private": "./sensitive-docs"},
	BasicAuthUser:     "admin",
	BasicAuthPassword: "super-secret-password",
})
service.Start("", 0, "", "")
```

### HTTPS with Let's Encrypt

```go
// Assuming you have Let's Encrypt certificates
service := web.NewWebService(web.Options{
	Directories: map[string]string{"/secure": "./secure-files"},
})
service.Start(
	"", 0,
	"/etc/letsencrypt/live/example.com/fullchain.pem",
	"/etc/letsencrypt/live/example.com/privkey.pem",
)
// Service available at https://example.com:8002
```

### Multi-Environment Configuration

```go
import "os"

// Development vs Production
if os.Getenv("ENVIRONMENT") == "production" {
	service := web.NewWebService(web.Options{
		Port:                    443,
		Directories:             map[string]string{"/": "./dist"},
		EnableDirectoryBrowsing: false,
	})
	service.Start(
		"0.0.0.0", 0,
		"/etc/ssl/certs/production.crt",
		"/etc/ssl/private/production.key",
	)
} else {
	service := web.NewWebService(web.Options{
		Port:                    8002,
		Directories:             map[string]string{"/": "./src"},
		EnableDirectoryBrowsing: true,
	})
	service.Start("", 0, "", "")
}
```

## Deployment Patterns

### Standalone Service

Run WebService as a dedicated static file server:

```go
// web_server.go
package main

import "github.com/signalwire/signalwire-go/pkg/web"

func main() {
	service := web.NewWebService(web.Options{
		Port: 8002,
		Directories: map[string]string{
			"/docs":      "/var/www/docs",
			"/assets":    "/var/www/assets",
			"/downloads": "/var/www/downloads",
		},
	})
	service.Start("", 0, "", "")
}
```

### Alongside AI Agents

Run WebService alongside your AI agents on different ports. Because `Start`
blocks, run the WebService in its own goroutine:

```go
// main.go
package main

import (
	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/web"
)

func main() {
	// Start WebService in the background (Start blocks).
	go func() {
		ws := web.NewWebService(web.Options{
			Port:        8002,
			Directories: map[string]string{"/docs": "./agent-docs"},
		})
		ws.Start("", 0, "", "")
	}()

	// Run your agent on a different port.
	a := agent.NewAgentBase(
		agent.WithName("My Agent"),
		agent.WithPort(3000),
	)
	a.Run() // Agent on port 3000, WebService on 8002
}
```

### Docker Deployment

```dockerfile
# Build stage
FROM golang:1.22 AS build
WORKDIR /src
COPY . .
# Build your web_server.go (see the Standalone Service example above)
RUN CGO_ENABLED=0 go build -o /out/web-server ./cmd/web-server

# Runtime stage
FROM gcr.io/distroless/static-debian12
WORKDIR /app

# Copy the compiled binary and static files
COPY --from=build /out/web-server /app/web-server
COPY ./static /app/static

# Expose port
EXPOSE 8002

# Run WebService
CMD ["/app/web-server"]
```

### Systemd Service

Create `/etc/systemd/system/signalwire-web.service`:

```ini
[Unit]
Description=SignalWire Web Service
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/signalwire
Environment="SWML_SSL_CERT=/etc/ssl/certs/server.crt"
Environment="SWML_SSL_KEY=/etc/ssl/private/server.key"
# Run your compiled Go web server binary (see the Standalone Service example).
ExecStart=/opt/signalwire/web-server
Restart=always

[Install]
WantedBy=multi-user.target
```

### Nginx Reverse Proxy

For production, use Nginx as a reverse proxy:

```nginx
server {
    listen 80;
    server_name static.example.com;
    
    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name static.example.com;
    
    ssl_certificate /etc/ssl/certs/example.com.crt;
    ssl_certificate_key /etc/ssl/private/example.com.key;
    
    location / {
        proxy_pass http://localhost:8002;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Cache static assets
        location ~* \.(jpg|jpeg|png|gif|ico|css|js)$ {
            proxy_pass http://localhost:8002;
            expires 1h;
            add_header Cache-Control "public, immutable";
        }
    }
}
```

## Best Practices

### Security
1. **Always use HTTPS in production** - Protect data in transit
2. **Change default credentials** - Never use auto-generated auth in production
3. **Restrict file types** - Use `allowed_extensions` to whitelist safe files
4. **Disable directory browsing** - Turn off in production environments
5. **Use reverse proxy** - Put Nginx/Apache in front for additional security

### Performance
1. **Set appropriate cache headers** - WebService adds 1-hour cache by default
2. **Limit file sizes** - Adjust `max_file_size` based on your needs
3. **Use CDN for static assets** - Offload traffic for better performance
4. **Compress large files** - Use gzip/brotli at reverse proxy level

### Organization
1. **Separate content types** - Use different routes for different file types
2. **Version your assets** - Include version in path (e.g., `/assets/v1/`)
3. **Use index.html** - Provide default files for directories
4. **Document your structure** - Maintain clear directory organization

## Troubleshooting

### Common Issues

**Issue: Build errors — module not found**
```bash
# Ensure the SDK is in your go.mod
go get github.com/signalwire/signalwire-go
go mod tidy
```

**Issue: SSL certificate errors**
```go
// Check certificate paths
import "os"

func certExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
// certExists("/path/to/cert.pem") should return true
// certExists("/path/to/key.pem")  should return true
```

**Issue: Permission denied**
```bash
# Ensure read permissions on directories
chmod -R 755 /path/to/static/files
```

**Issue: Directory not found**
```go
// Use absolute paths
import "path/filepath"

docs, _ := filepath.Abs("./documentation")
service := web.NewWebService(web.Options{
	Directories: map[string]string{"/docs": docs},
})
```

### Debug Logging

The SDK's structured logger honors the `SIGNALWIRE_LOG_LEVEL` environment
variable. Set it to `debug` to troubleshoot issues:

```bash
export SIGNALWIRE_LOG_LEVEL=debug
```

```go
service := web.NewWebService(web.Options{
	Directories: map[string]string{"/test": "./test"},
})
service.Start("", 0, "", "")
```

## API Reference

### WebService Type

The constructor takes a `web.Options` struct:

```go
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

func NewWebService(opts Options) *WebService
```

#### Options fields
- `Port`: Port to bind to (default: 8002)
- `Directories`: Map of URL paths to local directories
- `BasicAuthUser` / `BasicAuthPassword`: Credentials for HTTP Basic Auth
- `ConfigFile`: Path to JSON configuration file
- `EnableDirectoryBrowsing`: Allow directory listing (default: false)
- `AllowedExtensions`: List of allowed file extensions
- `BlockedExtensions`: List of blocked file extensions
- `MaxFileSize`: Maximum file size in bytes (default: 100MB)
- `EnableCORS`: Enable CORS headers (default: false unless set)

#### Methods

##### Start
```go
func (ws *WebService) Start(host string, port int, sslCert, sslKey string) error
```
Start the web service. `host=""` defaults to `0.0.0.0`; `port=0` uses the
constructor port; non-empty `sslCert` and `sslKey` enable TLS. `Start` blocks
until `Stop` is called or the server errors.

##### Stop
```go
func (ws *WebService) Stop() error
```
Gracefully shut the server down.

##### AddDirectory
```go
func (ws *WebService) AddDirectory(route, directory string)
```
Add a new directory to serve.

##### RemoveDirectory
```go
func (ws *WebService) RemoveDirectory(route string)
```
Remove a directory from being served.

## Integration with SignalWire Agents

WebService complements AI agents by providing static file serving:

```go
package main

import (
	"fmt"

	"github.com/signalwire/signalwire-go/pkg/agent"
	"github.com/signalwire/signalwire-go/pkg/swaig"
	"github.com/signalwire/signalwire-go/pkg/web"
)

func main() {
	a := agent.NewAgentBase(
		agent.WithName("Documentation Assistant"),
		agent.WithPort(3000),
	)

	// Reference documentation served by WebService.
	a.PromptAddSection(
		"Documentation",
		"User documentation is available at https://example.com:8002/docs/",
		nil,
	)

	a.DefineTool(agent.ToolDefinition{
		Name:        "get_doc_link",
		Description: "Get link to a documentation page",
		Parameters: map[string]any{
			"doc_name": map[string]any{
				"type":        "string",
				"description": "Name of the documentation page",
			},
		},
		Handler: func(args map[string]any, rawData map[string]any) *swaig.FunctionResult {
			docName, _ := args["doc_name"].(string)
			return swaig.NewFunctionResult(
				fmt.Sprintf("Documentation available at: https://example.com:8002/docs/%s.html", docName),
			)
		},
	})

	// Start WebService for documentation in the background (Start blocks).
	go func() {
		ws := web.NewWebService(web.Options{
			Port:        8002,
			Directories: map[string]string{"/docs": "./documentation"},
		})
		ws.Start("", 0, "", "")
	}()

	// Run the agent on port 3000.
	a.Run()
}
```

## Summary

WebService provides a secure, configurable static file server that integrates with the SignalWire AI Agents SDK. It follows the same architectural patterns as other SDK services, making it familiar and easy to use while providing configurable security features and flexible deployment options.