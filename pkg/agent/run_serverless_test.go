// Copyright (c) 2025 SignalWire
//
// This file is part of the SignalWire SDK.
//
// Licensed under the MIT License.
// See LICENSE file in the project root for full license information.

package agent

import (
	"errors"
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
// mirroring Python run(force_mode=...). A serverless platform returns
// ErrServerlessUnsupported (Go handles those via pkg/lambda, not inline) rather
// than silently serving HTTP.
func TestRunWithMode_ServerlessReturnsDescriptiveError(t *testing.T) {
	clearServerlessEnv(t)
	for _, mode := range []swml.ExecutionMode{
		swml.ModeLambda,
		swml.ModeGoogleCloudFunction,
		swml.ModeAzureFunction,
		swml.ModeCGI,
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
