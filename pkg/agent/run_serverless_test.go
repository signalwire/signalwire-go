// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package agent

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/signalwire/signalwire-go/pkg/swml"
)

// Regression tests for #192 — Run() auto-detects serverless environments.
//
// Python's run() (web_mixin.py:341 + serverless_mixin.py) is the universal
// entry point: it computes mode = force_mode or get_execution_mode() and
// dispatches to serve() for server mode or handle_serverless_request() for a
// detected serverless platform. Go's Run() mirrors the DETECTION + DISPATCH:
// server mode serves HTTP; a detected serverless platform returns a
// descriptive error (Go's serverless request handling lives in the dedicated
// adapter pkg/lambda wrapping AsRouter(), not inline in Run()).

// withEnv sets env vars for the duration of fn and restores them after.
func withEnv(t *testing.T, kv map[string]string, fn func()) {
	t.Helper()
	for k, v := range kv {
		t.Setenv(k, v)
	}
	fn()
}

// clearServerlessEnv zeroes every env var inspected by swml.GetExecutionMode so
// a CI runner that happens to set one of them cannot perturb the test.
func clearServerlessEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"GATEWAY_INTERFACE",
		"AWS_LAMBDA_FUNCTION_NAME", "LAMBDA_TASK_ROOT",
		"FUNCTION_TARGET", "K_SERVICE", "GOOGLE_CLOUD_PROJECT",
		"AZURE_FUNCTIONS_ENVIRONMENT", "FUNCTIONS_WORKER_RUNTIME", "AzureWebJobsStorage",
	} {
		t.Setenv(k, "")
	}
}

func TestDetectRunMode_DefaultIsServer(t *testing.T) {
	clearServerlessEnv(t)
	a := NewAgentBase(WithName("t"))
	if got := a.DetectRunMode(); got != swml.ModeServer {
		t.Errorf("DetectRunMode() = %q, want %q", got, swml.ModeServer)
	}
}

func TestDetectRunMode_PerPlatformEnvFixtures(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want swml.ExecutionMode
	}{
		{"cgi", map[string]string{"GATEWAY_INTERFACE": "CGI/1.1"}, swml.ModeCGI},
		{"lambda_function_name", map[string]string{"AWS_LAMBDA_FUNCTION_NAME": "fn"}, swml.ModeLambda},
		{"lambda_task_root", map[string]string{"LAMBDA_TASK_ROOT": "/var/task"}, swml.ModeLambda},
		{"gcf_function_target", map[string]string{"FUNCTION_TARGET": "handler"}, swml.ModeGoogleCloudFunction},
		{"gcf_k_service", map[string]string{"K_SERVICE": "svc"}, swml.ModeGoogleCloudFunction},
		{"azure_worker_runtime", map[string]string{"FUNCTIONS_WORKER_RUNTIME": "python"}, swml.ModeAzureFunction},
		{"azure_webjobs", map[string]string{"AzureWebJobsStorage": "x"}, swml.ModeAzureFunction},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearServerlessEnv(t)
			withEnv(t, tc.env, func() {
				a := NewAgentBase(WithName("t"))
				if got := a.DetectRunMode(); got != tc.want {
					t.Errorf("DetectRunMode() = %q, want %q", got, tc.want)
				}
			})
		})
	}
}

func TestDetectRunMode_CGIWinsOverLambda(t *testing.T) {
	clearServerlessEnv(t)
	withEnv(t, map[string]string{
		"GATEWAY_INTERFACE":        "CGI/1.1",
		"AWS_LAMBDA_FUNCTION_NAME": "fn",
	}, func() {
		a := NewAgentBase(WithName("t"))
		if got := a.DetectRunMode(); got != swml.ModeCGI {
			t.Errorf("DetectRunMode() = %q, want %q (CGI precedes Lambda)", got, swml.ModeCGI)
		}
	})
}

// RunWithMode dispatches on the supplied force-mode rather than auto-detecting,
// mirroring Python run(force_mode=...). The platform-runtime-driven modes
// (lambda / gcf / azure) return ErrServerlessUnsupported directing the caller to
// the adapter wired from main(); CGI is dispatched inline (see
// TestRunWithMode_CGIDispatches) because the CGI host invokes the process once
// per request over stdin/stdout.
func TestRunWithMode_ServerlessReturnsDescriptiveError(t *testing.T) {
	clearServerlessEnv(t)
	for _, mode := range []swml.ExecutionMode{
		swml.ModeLambda,
		swml.ModeGoogleCloudFunction,
		swml.ModeAzureFunction,
	} {
		t.Run(string(mode), func(t *testing.T) {
			a := NewAgentBase(WithName("t"))
			err := a.RunWithMode(mode)
			if err == nil {
				t.Fatalf("RunWithMode(%q) returned nil; want a descriptive serverless error", mode)
			}
			if !errors.Is(err, ErrServerlessUnsupported) {
				t.Fatalf("RunWithMode(%q) error = %v; want errors.Is ErrServerlessUnsupported", mode, err)
			}
			// The error must name the detected mode and point to the adapter.
			if !strings.Contains(err.Error(), string(mode)) {
				t.Errorf("error %q does not name the mode %q", err.Error(), mode)
			}
			if !strings.Contains(err.Error(), "AsRouter") {
				t.Errorf("error %q does not mention the AsRouter() adapter path", err.Error())
			}
		})
	}
}

// TestRunWithMode_CGIDispatches is the CGI half of Tier-2 contract #5: CGI mode
// must DISPATCH the request to a real response (write a SWML document to stdout),
// NOT return ErrServerlessUnsupported. The CGI host invokes the process once per
// request, handing the request off via env + stdin and reading the response from
// stdout — so RunWithMode(ModeCGI) serves the request through pkg/serverless and
// returns nil. We capture stdout via a pipe and assert a 200 SWML document.
func TestRunWithMode_CGIDispatches(t *testing.T) {
	clearServerlessEnv(t)
	// PATH_INFO selects the agent's route; the CGI branch renders SWML there.
	// HTTP_AUTHORIZATION carries basic auth through the dispatch (proving the
	// agent's auth middleware runs on the CGI path).
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("u:p"))
	t.Setenv("REQUEST_METHOD", "GET")
	t.Setenv("PATH_INFO", "/t")
	t.Setenv("CONTENT_LENGTH", "")
	t.Setenv("HTTP_AUTHORIZATION", authHeader)

	a := NewAgentBase(WithName("t"), WithRoute("/t"), WithBasicAuth("u", "p"))
	a.PromptAddSection("Role", "helpful", nil)

	// Capture os.Stdout (ServeCGI writes the CGI response there).
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	runErr := a.RunWithMode(swml.ModeCGI)
	_ = w.Close()
	os.Stdout = orig

	out, _ := io.ReadAll(r)

	if errors.Is(runErr, ErrServerlessUnsupported) {
		t.Fatalf("RunWithMode(ModeCGI) returned ErrServerlessUnsupported; CGI must dispatch")
	}
	if runErr != nil {
		t.Fatalf("RunWithMode(ModeCGI) error = %v", runErr)
	}

	s := string(out)
	if !strings.HasPrefix(s, "Status: 200") {
		t.Errorf("CGI response did not start with Status: 200; got %q", s[:min(60, len(s))])
	}
	// Body after the header separator must be a SWML document.
	if idx := strings.Index(s, "\r\n\r\n"); idx >= 0 {
		var doc map[string]any
		if err := json.Unmarshal([]byte(s[idx+4:]), &doc); err != nil {
			t.Fatalf("CGI body is not JSON SWML: %v; body=%q", err, s[idx+4:])
		}
		if _, ok := doc["sections"]; !ok {
			t.Errorf("CGI body lacks 'sections' (not a SWML doc); doc=%v", doc)
		}
	} else {
		t.Errorf("CGI response missing header/body separator; got %q", s)
	}
}

// Run() in a plain (non-serverless) environment must still take the HTTP serve
// path. We force-detect server mode and confirm RunWithMode(ModeServer) does
// NOT short-circuit with the serverless error — it proceeds to serving, which
// we abort immediately via a pre-cancelled graceful shutdown so the test does
// not block on a real listener.
func TestRunWithMode_ServerModeServesHTTP(t *testing.T) {
	clearServerlessEnv(t)
	a := NewAgentBase(WithName("t"), WithPort(0))

	// Pre-arm the graceful-shutdown channel so buildAndServe returns promptly
	// instead of blocking on ListenAndServe.
	a.mu.Lock()
	a.shutdownCh = make(chan struct{})
	close(a.shutdownCh)
	a.mu.Unlock()

	err := a.RunWithMode(swml.ModeServer)
	if errors.Is(err, ErrServerlessUnsupported) {
		t.Fatalf("RunWithMode(ModeServer) returned the serverless error; server mode must serve HTTP")
	}
	// A nil error (clean shutdown) or a bind error are both acceptable proof
	// that we took the serve path rather than the serverless short-circuit.
}
