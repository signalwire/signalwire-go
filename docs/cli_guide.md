# CLI Guide

This guide covers `swaig-test`, the command-line tool included with the
SignalWire Agents Go SDK for testing agents.

## Overview

The Go `swaig-test` CLI tests an agent by exercising its **HTTP endpoints**.
Unlike the Python SDK's `swaig-test` (which dynamically loads agent *source
files*), the Go tool operates against a **running agent server** via `--url`, or
introspects a compiled example binary via `--example`. It can:

- **Dump SWML** — fetch the agent's rendered SWML document (`--dump-swml`).
- **List tools** — list the agent's registered SWAIG functions (`--list-tools`).
- **Execute a function** — invoke a SWAIG function by name with `--param
  key=value` arguments (`--exec`).

SWAIG (SignalWire AI Gateway) is the platform's AI tool-calling system; SWML
(SignalWire Markup Language) is the JSON document format that defines agent
behavior during calls.

## Key Features

- **HTTP-based testing** — tests a running agent server over HTTP via `--url`.
- **Binary introspection** — lists tools from a compiled example via `--example`
  (no HTTP).
- **SWML dumping** — fetch and pretty-print (or `--raw` for compact) the rendered
  SWML document.
- **Function execution** — invoke SWAIG functions with repeatable
  `--param key=value` arguments.
- **Lambda serverless simulation** — `--simulate-serverless lambda` applies
  Lambda mode-detection env vars around the invocation. Only `lambda` is
  implemented in Go.
- **Verbose debugging** — `--verbose` shows request/response details.
- **Parse-only validation** — `--parse-only` / `--dry-run` validates the
  arguments and exits without loading the agent or making any network call.

## Installation

`swaig-test` is a Go command in this repository (`cmd/swaig-test`). Build or run
it with the Go toolchain (Go 1.25 or later):

```bash
# Run directly from the repo
go run ./cmd/swaig-test --help

# Or install the binary onto your PATH
go install github.com/signalwire/signalwire-go/cmd/swaig-test@latest
swaig-test --help
```

## Command Line Options

The Go `swaig-test` accepts the following flags (from `cmd/swaig-test`). It
targets a **running agent server** over HTTP via `--url`, or introspects a
compiled example binary via `--example`. Exactly one of `--dump-swml`,
`--list-tools`, or `--exec` selects the action.

| Option | Description |
|--------|-------------|
| `--url URL` | Agent URL, e.g. `http://user:pass@localhost:3000/`. Basic-auth credentials may be embedded in the URL. |
| `--example NAME` | Introspect a binary in `./examples/<NAME>/` via the `SWAIG_LIST_TOOLS` env var (no HTTP, no port binding). Mutually exclusive with `--url`; currently only supports `--list-tools`. |
| `--dump-swml` | Dump the SWML document from the agent (HTTP GET). |
| `--list-tools` | List available SWAIG tools. |
| `--exec NAME` | Execute a SWAIG tool by name (HTTP POST). |
| `--param key=value` | Parameter for `--exec`, as `key=value` (repeatable). Values that parse as JSON numbers/booleans/null are converted; everything else is kept as a string. |
| `--raw` | Output compact JSON instead of pretty-printed. |
| `--verbose` | Show request/response details. |
| `--simulate-serverless PLATFORM` | Simulate a serverless environment around the invocation. Currently only `lambda` is supported; it sets Lambda mode-detection env vars and clears `SWML_PROXY_URL_BASE` so platform-specific URL generation is exercised. Requires `--url`. |
| `--parse-only`, `--dry-run` | Validate the arguments and exit (prints `parse OK`); loads no agent and makes no network call. |

> **Go vs Python:** The Go tool does **not** have the Python CLI's file-loading /
> agent-discovery flags (`--agent-class`, `--route`, `--list-agents`),
> fake-data/override flags (`--call-type`, `--call-direction`, `--user-vars`,
> `--override`, …), mock-request flags (`--header`, `--method`, `--body`), or the
> non-Lambda serverless platforms (CGI, Cloud Functions, Azure). For function
> arguments the Go tool uses repeatable `--param key=value` flags rather than
> JSON-string positional arguments. Run `swaig-test --help` for the authoritative
> flag set.

## Quick Start

The Go CLI drives a **running** agent, so start the agent first, then point
`swaig-test` at its URL.

```bash
# Terminal 1: run the agent (this example serves on :3000/)
go run ./cmd/my_agent

# Terminal 2: introspect and exercise it
swaig-test --url http://localhost:3000/ --list-tools
swaig-test --url http://localhost:3000/ --dump-swml
swaig-test --url http://localhost:3000/ --exec get_weather --param location="New York"
```

If the agent uses basic auth, embed the credentials in the URL:

```bash
swaig-test --url http://user:pass@localhost:3000/ --list-tools
```

### Introspecting a Compiled Example

To list an example binary's tool registry without HTTP, use `--example NAME`
(list-tools only):

```bash
swaig-test --example my_example --list-tools
```

## List Available Functions

`--list-tools` prints every SWAIG function the running agent has registered,
including DataMap (serverless) functions and external-webhook functions:

```bash
swaig-test --url http://localhost:3000/ --list-tools
```

**Example output:**

```
Available SWAIG functions:
  search_knowledge - DataMap function (serverless)
    Config: {"webhooks": [...], "output": {...}}
  calculate - Perform mathematical calculations and return the result
```

## Test SWML Generation

`--dump-swml` fetches the agent's rendered SWML document over HTTP. Add `--raw`
for compact JSON suitable for piping into `jq`, and `--verbose` to see the
request/response details:

```bash
# Pretty-printed SWML
swaig-test --url http://localhost:3000/ --dump-swml

# Compact JSON, piped into jq
swaig-test --url http://localhost:3000/ --dump-swml --raw | jq '.'

# Verbose (shows the HTTP request/response)
swaig-test --url http://localhost:3000/ --dump-swml --verbose
```

## Execute SWAIG Functions

`--exec NAME` invokes a SWAIG function by name (HTTP POST). Pass arguments with
repeatable `--param key=value` flags. Values that parse as JSON numbers,
booleans, or null are converted; everything else is kept as a string:

```bash
# Execute with a single argument
swaig-test --url http://localhost:3000/ --exec get_weather --param location="New York"

# Multiple arguments
swaig-test --url http://localhost:3000/ --exec search --param query="SignalWire" --param limit=5

# Verbose execution (shows the request body and response)
swaig-test --url http://localhost:3000/ --verbose --exec get_joke --param type=dadjokes
```

External-webhook functions and DataMap functions execute exactly the same way —
the running agent forwards the call as it would in production, so the CLI needs
no special flags for them.

## Logging and Output Control

By default the agent's own logs are suppressed so the CLI's output (the SWML
document or the function result) is clean and pipeable:

```bash
# Default: agent logs suppressed, clean output
swaig-test --url http://localhost:3000/ --dump-swml

# --verbose surfaces request/response details for debugging
swaig-test --url http://localhost:3000/ --dump-swml --verbose
```

Set `SIGNALWIRE_LOG_MODE=off` in the environment to force-suppress logs from any
subprocess the CLI spawns (e.g. under `--example`).

## Serverless Environment Simulation

The Go CLI implements **Lambda simulation only**. `--simulate-serverless lambda`
applies the Lambda mode-detection env vars around the invocation (and clears
`SWML_PROXY_URL_BASE`) so platform-specific webhook-URL generation is exercised.
Because Go agents are compiled binaries rather than dynamically loadable files,
the simulation runs against the live agent URL — so it **requires `--url`**:

```bash
# Lambda simulation while dumping SWML
swaig-test --url http://localhost:3000/ --simulate-serverless lambda --dump-swml

# Lambda simulation while executing a function
swaig-test --url http://localhost:3000/ --simulate-serverless lambda \
  --exec get_weather --param location="Miami"
```

Other platforms (CGI, Google Cloud Functions, Azure Functions) are **not**
implemented by the Go CLI; passing them returns a clear "not implemented" error.

### In-Process Simulation (no running server)

For true in-process adapter dispatch without a running server, call the library
helpers `SimulateDumpSWMLViaLambda` / `SimulateExecToolViaLambda` from
`package main` directly — see `cmd/swaig-test/simulate.go`.

## Validating Invocations (`--parse-only`)

`--parse-only` (alias `--dry-run`) validates the arguments and exits `0` with
`parse OK`, loading no agent and making no network call. It is position-
independent — it works whether it precedes or trails the other arguments:

```bash
swaig-test --url http://localhost:3000/ --exec my_func --param x=1 --parse-only
# prints: parse OK
```

Invalid arguments (an unknown flag, a missing action, a malformed URL) exit with
status `2` and an error message, which makes `--parse-only` useful in scripts and
CI to check that documented invocations still match the CLI.

## Troubleshooting

- **`--url is required`** — the Go CLI cannot load an agent from a source file;
  start the agent and pass its `--url`, or use `--example NAME` for list-tools.
- **`--example currently only supports --list-tools`** — `--dump-swml` and
  `--exec` require a running server (`--url`).
- **`--example and --url are mutually exclusive`** — pick one input mode.
- **`--simulate-serverless lambda: requires --url`** — see
  [Serverless Environment Simulation](#serverless-environment-simulation) for why,
  and the in-process helpers for the no-server path.
- **Connection refused** — make sure the agent is running and the `--url` host,
  port, and route path match what the agent serves.
