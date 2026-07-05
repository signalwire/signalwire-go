// Command generate-swml-verbs emits the typed SWML verb CONFIG surface from the
// authoritative vendored schema in porting-sdk:
//
//	schema.json ($defs)  -> pkg/swml/swml_verbs_generated.go
//
// One Go type per schema.json $defs entry (object -> struct; non-object ->
// defined-type alias) plus the flattened <Verb>Config payload shapes the SWML
// builder verb methods accept. It mirrors the Python reference emitter
// generate_swml_verbs and the TypeScript generateSwmlVerbs.
//
// It is one of the fixed 5 cross-port generator commands (generate_rest,
// generate_rest_tests, generate_relay_protocol, generate_swaig_payloads,
// generate_swml_verbs); the SWAIG payloads it used to co-emit live in the sibling
// cmd/generate-swaig-payloads command. The shared schema/emission machinery lives
// in cmd/internal/payloadgen, so both split commands emit byte-identically.
//
// GEN-FRESH-gated: `--check` reproduces the committed *_generated.go and exits
// non-zero if it differs. Resolves porting-sdk via $PORTING_SDK or sibling.
//
// Usage:
//
//	go run ./cmd/generate-swml-verbs          # (re)write swml_verbs_generated.go
//	go run ./cmd/generate-swml-verbs --check  # GEN-FRESH: fail if it is stale
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
	check := flag.Bool("check", false, "GEN-FRESH: exit non-zero if the generated file is stale")
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
			return fmt.Errorf("generate-swml-verbs --check: %w", err)
		}
		fmt.Fprintf(os.Stderr, "generate-swml-verbs: %v — skipping (committed files kept)\n", err)
		return nil
	}

	raw, err := os.ReadFile(filepath.Join(psdk, "schema.json"))
	if err != nil {
		return err
	}
	src, err := payloadgen.EmitSwmlVerbs(raw)
	if err != nil {
		return err
	}
	out := gen.Output{
		Path: filepath.Join(repoRoot, "pkg", "swml", "swml_verbs_generated.go"),
		Src:  src,
	}
	return gen.Run(*check, "generate-swml-verbs", "SWML verb", []gen.Output{out})
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
