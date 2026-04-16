// Command swaig-test is a CLI tool for testing SWAIG agents by exercising
// their HTTP endpoints. Unlike the Python SDK's swaig-test which loads agent
// files dynamically, this tool operates against a running agent server.
//
// Usage:
//
//	swaig-test --url http://user:pass@localhost:3000/ --dump-swml
//	swaig-test --url http://user:pass@localhost:3000/ --list-tools
//	swaig-test --url http://user:pass@localhost:3000/ --exec get_weather --param location=London
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// paramList collects repeatable --param flags.
type paramList []string

func (p *paramList) String() string { return strings.Join(*p, ", ") }
func (p *paramList) Set(value string) error {
	*p = append(*p, value)
	return nil
}

// config holds the parsed CLI flags.
type config struct {
	url                string
	dumpSWML           bool
	listTools          bool
	exec               string
	params             paramList
	raw                bool
	verbose            bool
	simulateServerless string
}

func main() {
	cfg := parseFlags(os.Args[1:])

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

// parseFlags parses command-line arguments into a config struct.
func parseFlags(args []string) config {
	var cfg config
	fs := flag.NewFlagSet("swaig-test", flag.ExitOnError)

	fs.StringVar(&cfg.url, "url", "", "Agent URL (e.g. http://user:pass@localhost:3000/)")
	fs.BoolVar(&cfg.dumpSWML, "dump-swml", false, "Dump the SWML document from the agent")
	fs.BoolVar(&cfg.listTools, "list-tools", false, "List available SWAIG tools")
	fs.StringVar(&cfg.exec, "exec", "", "Execute a SWAIG tool by name")
	fs.Var(&cfg.params, "param", "Parameter as key=value (repeatable)")
	fs.BoolVar(&cfg.raw, "raw", false, "Output compact JSON instead of pretty-printed")
	fs.BoolVar(&cfg.verbose, "verbose", false, "Show request/response details")
	fs.StringVar(&cfg.simulateServerless, "simulate-serverless", "",
		"Simulate a serverless environment (currently supported: lambda). "+
			"Sets mode-detection env vars and clears SWML_PROXY_URL_BASE so "+
			"platform-specific URL generation is exercised.")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: swaig-test --url <agent-url> [options]\n\n")
		fmt.Fprintf(os.Stderr, "A CLI tool for testing SWAIG agents via their HTTP endpoints.\n\n")
		fmt.Fprintf(os.Stderr, "Modes:\n")
		fmt.Fprintf(os.Stderr, "  --dump-swml          Dump the SWML document (GET)\n")
		fmt.Fprintf(os.Stderr, "  --list-tools         List available SWAIG functions\n")
		fmt.Fprintf(os.Stderr, "  --exec <name>        Execute a SWAIG function (POST)\n\n")
		fmt.Fprintf(os.Stderr, "Serverless simulation:\n")
		fmt.Fprintf(os.Stderr, "  --simulate-serverless lambda\n")
		fmt.Fprintf(os.Stderr, "                       Apply Lambda mode-detection env vars around\n")
		fmt.Fprintf(os.Stderr, "                       the invocation; clears SWML_PROXY_URL_BASE\n")
		fmt.Fprintf(os.Stderr, "                       for the duration. Combine with --dump-swml\n")
		fmt.Fprintf(os.Stderr, "                       or --exec as normal.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	fs.Parse(args)
	return cfg
}

// run executes the CLI command based on the parsed config.
func run(cfg config) error {
	// Platform validation always runs first — if the user asked for
	// a platform we don't implement, we want to fail before touching
	// the environment or making any HTTP request.
	if cfg.simulateServerless != "" {
		if err := validateSimulatePlatform(cfg.simulateServerless); err != nil {
			return err
		}
	}

	if cfg.url == "" {
		// --simulate-serverless without --url can't fully exercise the
		// adapter from a Go CLI (Go agents are compiled binaries, not
		// dynamically loadable files), so the CLI requires --url and
		// documents the library API for in-process use.
		if cfg.simulateServerless != "" {
			return fmt.Errorf(
				"--simulate-serverless %s: requires --url <agent-url>. "+
					"Go agents are compiled binaries, so this CLI simulates by "+
					"running the agent URL with the platform env vars applied. "+
					"For true in-process adapter dispatch, use "+
					"SimulateDumpSWMLViaLambda / SimulateExecToolViaLambda "+
					"from package main directly (see cmd/swaig-test/simulate.go).",
				cfg.simulateServerless,
			)
		}
		return fmt.Errorf("--url is required")
	}

	if !cfg.dumpSWML && !cfg.listTools && cfg.exec == "" {
		// In simulate mode without a sub-action, default to dumping
		// SWML — mirrors the Python CLI's "bare --simulate-serverless"
		// mode. This also makes the "render SWML and exit" combination
		// work intuitively.
		if cfg.simulateServerless != "" {
			cfg.dumpSWML = true
		} else {
			return fmt.Errorf("one of --dump-swml, --list-tools, or --exec is required")
		}
	}

	baseURL, user, pass, err := parseAuthURL(cfg.url)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// If the user requested serverless simulation, set the mode-
	// detection env vars for the chosen platform for the duration of
	// this run and unconditionally restore them on exit. The
	// simulation ALSO clears SWML_PROXY_URL_BASE (matches Python's
	// mock_env.py behaviour) so platform-specific URL generation is
	// actually exercised.
	if cfg.simulateServerless != "" {
		snap, err := activateSimulation(cfg.simulateServerless, cfg.verbose)
		if err != nil {
			return err
		}
		defer snap.restore()
	}

	switch {
	case cfg.dumpSWML:
		return doDumpSWML(baseURL, user, pass, cfg)
	case cfg.listTools:
		return doListTools(baseURL, user, pass, cfg)
	case cfg.exec != "":
		return doExec(baseURL, user, pass, cfg)
	}

	return nil
}

// activateSimulation sets env vars for the chosen platform and returns
// a snapshot that restores the original values when restore() is
// called. The caller is responsible for deferring restore(); this
// separation lets the CLI's run() function apply the change around
// whatever sub-action was selected.
//
// Only "lambda" is supported today. Unsupported platforms are rejected
// earlier in run(); if we somehow reach this function with an unknown
// platform we fall through to an explicit error so the bug is easy
// to spot.
func activateSimulation(platform string, verbose bool) (envSnapshot, error) {
	switch platform {
	case "lambda":
		logger := (func(format string, args ...any))(nil)
		if verbose {
			logger = func(format string, args ...any) {
				fmt.Fprintf(os.Stderr, "simulate-serverless: "+format+"\n", args...)
			}
		}
		return activateLambdaEnv(SimulateLambdaOptions{Logger: logger}), nil
	default:
		return envSnapshot{}, fmt.Errorf(
			"activateSimulation: internal error — platform %q passed validation but has no activator",
			platform,
		)
	}
}

// parseAuthURL extracts basic auth credentials from a URL.
// Returns the clean URL (without credentials), username, and password.
func parseAuthURL(rawURL string) (cleanURL, user, pass string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse URL: %w", err)
	}

	if u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
		u.User = nil
	}

	cleanURL = u.String()
	return cleanURL, user, pass, nil
}

// parseParams converts a list of "key=value" strings into a map.
func parseParams(params []string) (map[string]interface{}, error) {
	result := make(map[string]interface{}, len(params))
	for _, p := range params {
		idx := strings.Index(p, "=")
		if idx < 0 {
			return nil, fmt.Errorf("invalid param %q: expected key=value", p)
		}
		key := p[:idx]
		value := p[idx+1:]
		if key == "" {
			return nil, fmt.Errorf("invalid param %q: empty key", p)
		}

		// Try to parse as JSON value for numbers, booleans, etc.
		var jsonVal interface{}
		if err := json.Unmarshal([]byte(value), &jsonVal); err == nil {
			// Only use JSON-parsed value for non-string types (numbers, bools, null).
			// If it parsed as a string, just keep the original string.
			switch jsonVal.(type) {
			case string:
				result[key] = value
			default:
				result[key] = jsonVal
			}
		} else {
			result[key] = value
		}
	}
	return result, nil
}

// doRequest makes an HTTP request with optional basic auth and returns the body.
func doRequest(method, reqURL, user, pass string, body io.Reader, cfg config) ([]byte, int, error) {
	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	if user != "" || pass != "" {
		req.SetBasicAuth(user, pass)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if cfg.verbose {
		fmt.Fprintf(os.Stderr, ">> %s %s\n", method, reqURL)
		if user != "" {
			fmt.Fprintf(os.Stderr, ">> Auth: %s:****\n", user)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	if cfg.verbose {
		fmt.Fprintf(os.Stderr, "<< HTTP %d %s\n", resp.StatusCode, resp.Status)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, resp.StatusCode, fmt.Errorf("authentication failed (HTTP 401). Check your credentials in the URL")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, resp.StatusCode, nil
}

// formatJSON pretty-prints or compacts JSON depending on the raw flag.
func formatJSON(data []byte, raw bool) (string, error) {
	if raw {
		// Compact the JSON (remove any existing whitespace)
		var buf bytes.Buffer
		if err := json.Compact(&buf, data); err != nil {
			return "", fmt.Errorf("invalid JSON: %w", err)
		}
		return buf.String(), nil
	}

	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	pretty, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}
	return string(pretty), nil
}

// doDumpSWML fetches and prints the SWML document from the agent.
func doDumpSWML(baseURL, user, pass string, cfg config) error {
	body, _, err := doRequest("GET", baseURL, user, pass, nil, cfg)
	if err != nil {
		return fmt.Errorf("failed to fetch SWML: %w", err)
	}

	output, err := formatJSON(body, cfg.raw)
	if err != nil {
		return err
	}

	fmt.Println(output)
	return nil
}

// extractFunctions parses the SWML JSON to find SWAIG function definitions.
// It looks for: sections.main[].ai.SWAIG.functions[]
func extractFunctions(data []byte) ([]map[string]interface{}, error) {
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid SWML JSON: %w", err)
	}

	sections, ok := doc["sections"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("SWML document missing 'sections'")
	}

	mainSection, ok := sections["main"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("SWML document missing 'sections.main'")
	}

	for _, verb := range mainSection {
		verbMap, ok := verb.(map[string]interface{})
		if !ok {
			continue
		}

		aiConfig, ok := verbMap["ai"].(map[string]interface{})
		if !ok {
			continue
		}

		swaig, ok := aiConfig["SWAIG"].(map[string]interface{})
		if !ok {
			continue
		}

		functionsRaw, ok := swaig["functions"].([]interface{})
		if !ok {
			return nil, nil // No functions defined
		}

		functions := make([]map[string]interface{}, 0, len(functionsRaw))
		for _, f := range functionsRaw {
			if fm, ok := f.(map[string]interface{}); ok {
				functions = append(functions, fm)
			}
		}
		return functions, nil
	}

	return nil, nil
}

// doListTools fetches the SWML and lists all SWAIG functions.
func doListTools(baseURL, user, pass string, cfg config) error {
	body, _, err := doRequest("GET", baseURL, user, pass, nil, cfg)
	if err != nil {
		return fmt.Errorf("failed to fetch SWML: %w", err)
	}

	functions, err := extractFunctions(body)
	if err != nil {
		return err
	}

	if len(functions) == 0 {
		fmt.Println("No SWAIG functions found.")
		return nil
	}

	fmt.Printf("Available SWAIG functions (%d):\n\n", len(functions))
	for _, fn := range functions {
		name, _ := fn["function"].(string)
		purpose, _ := fn["purpose"].(string)
		if name == "" {
			continue
		}

		fmt.Printf("  %s\n", name)
		if purpose != "" {
			fmt.Printf("    %s\n", purpose)
		}

		// Show parameters if present
		if arg, ok := fn["argument"].(map[string]interface{}); ok {
			if props, ok := arg["properties"].(map[string]interface{}); ok && len(props) > 0 {
				fmt.Printf("    Parameters:\n")
				for pName, pDef := range props {
					pMap, _ := pDef.(map[string]interface{})
					pType, _ := pMap["type"].(string)
					pDesc, _ := pMap["description"].(string)
					if pType != "" {
						fmt.Printf("      --%s (%s)", pName, pType)
					} else {
						fmt.Printf("      --%s", pName)
					}
					if pDesc != "" {
						fmt.Printf(": %s", pDesc)
					}
					fmt.Println()
				}
			}
		}
		fmt.Println()
	}

	return nil
}

// doExec executes a SWAIG function by POSTing to the /swaig endpoint.
func doExec(baseURL, user, pass string, cfg config) error {
	args, err := parseParams(cfg.params)
	if err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	// Build the SWAIG function call payload in the standard SignalWire format.
	payload := map[string]interface{}{
		"function": cfg.exec,
		"argument": map[string]interface{}{
			"parsed": []interface{}{args},
		},
		"call_id": "test-call-id",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	if cfg.verbose {
		formatted, _ := formatJSON(payloadBytes, false)
		fmt.Fprintf(os.Stderr, ">> Body: %s\n", formatted)
	}

	// Build the SWAIG URL by appending /swaig to the base URL.
	swaigURL := strings.TrimRight(baseURL, "/") + "/swaig"

	body, _, err := doRequest("POST", swaigURL, user, pass, strings.NewReader(string(payloadBytes)), cfg)
	if err != nil {
		return fmt.Errorf("function execution failed: %w", err)
	}

	output, err := formatJSON(body, cfg.raw)
	if err != nil {
		// If JSON formatting fails, print the raw response
		fmt.Println(string(body))
		return nil
	}

	fmt.Println(output)
	return nil
}
