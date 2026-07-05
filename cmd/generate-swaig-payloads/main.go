// Command generate-swaig-payloads emits the typed READ-side SWAIG payloads from
// the authoritative vendored swaig-specs in porting-sdk:
//
//	swaig-specs/swaig-request.yaml   -> pkg/swaig/swaig_request_generated.go
//	swaig-specs/post-prompt.yaml     -> pkg/swaig/post_prompt_generated.go
//	swaig-specs/swaig-response.yaml  -> pkg/swaig/swaig_actions_generated.go
//
// These are the payloads the agent RECEIVES (function-webhook request body,
// post-prompt / onSummary callback summary) plus the SWAIG response-action CONFIG
// shapes the FunctionResult builder accepts. It mirrors the Python reference
// emitters generate_swaig_request / generate_post_prompt / generate_swaig_actions
// and the TypeScript generateSwaigContracts / generateSwaigActions.
//
// It is one of the fixed 5 cross-port generator commands (generate_rest,
// generate_rest_tests, generate_relay_protocol, generate_swaig_payloads,
// generate_swml_verbs); the SWML-verb surface it used to co-emit lives in the
// sibling cmd/generate-swml-verbs command. The shared schema/emission machinery
// lives in cmd/internal/payloadgen, so both split commands emit byte-identically.
//
// GEN-FRESH-gated: `--check` reproduces the committed *_generated.go and exits
// non-zero if any differs. Resolves porting-sdk via $PORTING_SDK or sibling.
//
// Usage:
//
//	go run ./cmd/generate-swaig-payloads          # (re)write the *_generated.go files
//	go run ./cmd/generate-swaig-payloads --check  # GEN-FRESH: fail if any is stale
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/signalwire/signalwire-go/cmd/internal/gen"
	"github.com/signalwire/signalwire-go/cmd/internal/payloadgen"
)

func run() error {
	check := flag.Bool("check", false, "GEN-FRESH: exit non-zero if any generated file is stale")
	flag.Parse()

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	repoRoot, err := gen.FindRepoRoot(cwd)
	if err != nil {
		return err
	}
	psdk, err := gen.ResolvePortingSDK(repoRoot, "swaig-specs")
	if err != nil {
		if *check {
			return fmt.Errorf("generate-swaig-payloads --check: %w", err)
		}
		fmt.Fprintf(os.Stderr, "generate-swaig-payloads: %v — skipping (committed files kept)\n", err)
		return nil
	}

	type job struct {
		out  []string // repo-relative output path parts
		spec []string // porting-sdk-relative spec path parts
		emit func([]byte) (string, error)
	}
	jobs := []job{
		{out: []string{"pkg", "swaig", "swaig_request_generated.go"}, spec: []string{"swaig-specs", "swaig-request.yaml"}, emit: payloadgen.EmitSwaigRequest},
		{out: []string{"pkg", "swaig", "post_prompt_generated.go"}, spec: []string{"swaig-specs", "post-prompt.yaml"}, emit: payloadgen.EmitPostPrompt},
		{out: []string{"pkg", "swaig", "swaig_actions_generated.go"}, spec: []string{"swaig-specs", "swaig-response.yaml"}, emit: payloadgen.EmitSwaigActions},
	}

	var outs []gen.Output
	for _, j := range jobs {
		raw, err := os.ReadFile(filepath.Join(append([]string{psdk}, j.spec...)...))
		if err != nil {
			return err
		}
		src, err := j.emit(raw)
		if err != nil {
			return err
		}
		outs = append(outs, gen.Output{
			Path: filepath.Join(append([]string{repoRoot}, j.out...)...),
			Src:  src,
		})
	}
	return gen.Run(*check, "generate-swaig-payloads", "SWAIG payload", outs)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
