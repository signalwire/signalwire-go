# SignalWire AI Agents (Go) - Serverless Deployment Guide

This guide covers deploying SignalWire AI Agents built with the Go SDK to serverless
platforms.

## Platform Support

The Go SDK ships a serverless adapter for **AWS Lambda** only (package
`github.com/signalwire/signalwire-go/pkg/lambda`).

- **AWS Lambda** — Supported via `pkg/lambda` (Function URLs and API Gateway HTTP API v2).
- **Google Cloud Functions** — Not yet implemented in the Go SDK. There is no GCF
  adapter package. (`swaig-test --simulate-serverless cloud_function` returns a
  "not implemented in this port" error.)
- **Azure Functions** — Not yet implemented in the Go SDK. There is no Azure adapter
  package. (`swaig-test --simulate-serverless azure_function` returns a "not
  implemented in this port" error.)

> The Python SDK supports GCF and Azure deployment; the Go port does not yet. If you
> need GCF or Azure today, run the agent as a normal HTTP server (see
> [web_service.md](web_service.md)) behind the platform's HTTP trigger, or use AWS
> Lambda with the adapter below.

## AWS Lambda

The Lambda adapter wraps the `http.Handler` produced by `agent.AsRouter()` (any
`http.Handler` works) and translates Lambda invocation events into synthetic
`*http.Request` values. Two event shapes are supported:

- **Lambda Function URLs** (`HandleFunctionURL`) — the simplest deployment; no API
  Gateway required. This is the recommended path.
- **API Gateway HTTP API v2** (`HandleAPIGatewayV2`) — for deployments that front the
  function with an API Gateway HTTP API.

### Function URL deployment (recommended)

```go
package main

import (
	awslambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/signalwire/signalwire-go/pkg/agent"
	swlambda "github.com/signalwire/signalwire-go/pkg/lambda"
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
