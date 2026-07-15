# SignalWire AI Agents (Go) - Serverless Deployment Guide

This guide covers deploying SignalWire AI Agents built with the Go SDK to serverless
platforms.

## Platform Support

The Go SDK ships serverless adapters for **AWS Lambda** (package
`github.com/signalwire/signalwire-go/v3/pkg/lambda`), **Google Cloud Functions** and
**CGI** (both via package `github.com/signalwire/signalwire-go/v3/pkg/serverless`).

- **AWS Lambda** — Supported via `pkg/lambda` (Function URLs and API Gateway HTTP API v2).
- **Google Cloud Functions** — Supported via `pkg/serverless`: wrap
  `agent.AsRouter()` in `serverless.NewHandler(...)` and register its `ServeHTTP`
  as the GCF HTTP function (2nd-gen / Cloud Run functions). See
  [Google Cloud Functions](#google-cloud-functions) below.
- **CGI** — Supported via `pkg/serverless` and auto-detected: when the process runs
  under a CGI environment, `agent.Run()` dispatches through `serverless.ServeCGI`
  automatically (no adapter wiring required).
- **Azure Functions** — Not yet implemented in the Go SDK. There is no Azure adapter
  package. (`swaig-test --simulate-serverless azure_function` returns a "not
  implemented in this port" error.)

> The `swaig-test --simulate-serverless` harness currently simulates the **lambda**
> platform end-to-end; GCF/CGI/Azure are not yet wired into the CLI simulator even
> where an adapter ships. If you need Azure today, run the agent as a normal HTTP
> server (see [web_service.md](web_service.md)) behind the platform's HTTP trigger.

## AWS Lambda

The Lambda adapter wraps the `http.Handler` produced by `agent.AsRouter()` (any
`http.Handler` works) and translates Lambda invocation events into synthetic
`*http.Request` values. Two event shapes are supported:

- **Lambda Function URLs** (`HandleFunctionURL`) — the simplest deployment; no API
  Gateway required. This is the recommended path.
- **API Gateway HTTP API v2** (`HandleAPIGatewayV2`) — for deployments that front the
  function with an API Gateway HTTP API.

### Function URL deployment (recommended)

<!-- snippet: no-compile requires third-party module github.com/aws/aws-lambda-go/lambda (not part of the SDK-linked snippet module) -->
```go
package main

import (
	awslambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/signalwire/signalwire-go/v3/pkg/agent"
	swlambda "github.com/signalwire/signalwire-go/v3/pkg/lambda"
)

var a = agent.NewAgentBase(
	agent.WithName("MyAgent"),
	agent.WithRoute("/my-agent"),
)

var handler = swlambda.NewHandler(a.AsRouter())

func main() {
	awslambda.Start(handler.HandleFunctionURL)
}
```

Because `agent.AsRouter()` installs routes relative to the agent's route (e.g.
`/my-agent`, `/my-agent/swaig`), the Lambda event's request path must line up with that
route. Lambda Function URLs preserve the full request path unchanged, so no rewriting
is needed in the common case.

### API Gateway HTTP API v2 deployment

<!-- snippet: no-compile requires third-party github.com/aws/aws-lambda-go/lambda and the handler var from the prior example -->
```go
func main() {
	awslambda.Start(handler.HandleAPIGatewayV2)
}
```

The classic REST API (v1) payload is intentionally not supported as a first-class path.
Users who need v1 can wrap it via `github.com/awslabs/aws-lambda-go-api-proxy`.

### Building and deploying

Build a Linux binary named `bootstrap` (required by the `provided.al2`/`al2023`
runtimes) and package it for upload:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap ./cmd/my-agent
zip function.zip bootstrap

aws lambda create-function \
  --function-name my-agent \
  --runtime provided.al2023 \
  --handler bootstrap \
  --zip-file fileb://function.zip \
  --role arn:aws:iam::<account-id>:role/<lambda-role>
```

### Environment detection and URL generation

The SDK detects the Lambda environment from standard Lambda environment variables
(e.g. `AWS_LAMBDA_FUNCTION_NAME`, `AWS_REGION`, and `AWS_LAMBDA_FUNCTION_URL` when using
Function URLs) and generates webhook URLs accordingly. Setting `SWML_PROXY_URL_BASE`
overrides URL generation with a fixed base; clear it if you want the platform-derived
URLs.

## Google Cloud Functions

The GCF adapter lives in `pkg/serverless`. Wrap the `http.Handler` from
`agent.AsRouter()` in `serverless.NewHandler(...)` and register the resulting
`ServeHTTP` as your 2nd-gen (Cloud Run) HTTP function's entry point:

<!-- snippet: no-compile requires third-party module github.com/GoogleCloudPlatform/functions-framework-go/functions (not part of the SDK-linked snippet module) -->
```go
package function

import (
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/signalwire/signalwire-go/v3/pkg/agent"
	"github.com/signalwire/signalwire-go/v3/pkg/serverless"
)

var a = agent.NewAgentBase(
	agent.WithName("MyAgent"),
	agent.WithRoute("/my-agent"),
)

func init() {
	functions.HTTP("Agent", serverless.NewHandler(a.AsRouter()).ServeHTTP)
}
```

`ServeHTTP` mounts the agent at the request root, dispatches SWAIG at `/swaig`, and
enforces the agent's auth — the same request-handling core the Lambda and CGI
adapters use.

## CGI

The CGI adapter is auto-detected: when the process runs under a CGI environment
`agent.Run()` dispatches through `serverless.ServeCGI` without any adapter wiring.
To dispatch CGI explicitly:

<!-- snippet: no-compile illustrative one-line fragment (a / serverless / context are from the surrounding program, not in scope here) -->
```go
serverless.NewHandler(a.AsRouter()).ServeCGI(context.Background())
```

## Testing Lambda locally

Use the `swaig-test` CLI to exercise an agent with the Lambda mode-detection
environment applied. The Go tool requires a running agent server (`--url`):

```bash
# Apply Lambda env vars around a SWML dump against a running agent
swaig-test --url http://user:pass@localhost:3000/my-agent \
  --simulate-serverless lambda --dump-swml

# Execute a function under the simulated Lambda environment
swaig-test --url http://user:pass@localhost:3000/my-agent \
  --simulate-serverless lambda \
  --exec search_knowledge --param query=test
```

For true in-process Lambda adapter dispatch (constructing the agent *after* the Lambda
environment is active, so env-captured state reflects the simulated environment), call
the library functions `SimulateDumpSWMLViaLambda` / `SimulateExecToolViaLambda` in
`cmd/swaig-test/simulate.go` directly from a Go test.

See the [CLI Guide](cli_guide.md) for the full `swaig-test` flag set.

## Authentication

The agent validates HTTP Basic Auth credentials configured on the agent. Configure
them with the relevant `AgentOption`s when constructing the agent, then send an
`Authorization: Basic <credentials>` header (or embed `user:pass@` in the URL when
testing). An unauthenticated request to a protected agent returns 401 with a
`WWW-Authenticate` header.

```bash
# Should return 401
curl https://<function-url>/my-agent

# With valid credentials
curl -u username:password https://<function-url>/my-agent
```

## Best Practices

- Minimize cold-start time by keeping the deployment package small and using
  `CGO_ENABLED=0` static binaries.
- Always use HTTPS endpoints.
- Use environment variables / a secret manager for credentials; do not hard-code them.
- Enable platform logging and set sensible timeouts.
