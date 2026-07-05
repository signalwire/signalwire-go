// Package gen holds the small shared driver the split code-generator commands
// (generate-relay-protocol, generate-swaig-payloads, generate-swml-verbs) use to
// resolve the repo root + porting-sdk, gofmt each emitted source, and run the
// GEN-FRESH `--check` / write loop. It factors the byte-for-byte-identical
// scaffolding out of the individual commands so each command holds only its own
// spec-reading + emission logic.
package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
)

// Output is one generated file: its absolute path and its (pre-gofmt) source.
type Output struct {
	Path string
	Src  string
}

// GofmtSrc formats Go source, wrapping a parse error with the offending source
// for a legible failure.
func GofmtSrc(src string) ([]byte, error) {
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return nil, fmt.Errorf("gofmt: %w\n---\n%s", err, src)
	}
	return formatted, nil
}

// FindRepoRoot walks up from start to the directory containing go.mod.
func FindRepoRoot(start string) (string, error) {
	cur := start
	for {
		if _, err := os.Stat(filepath.Join(cur, "go.mod")); err == nil {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("no go.mod above %s", start)
		}
		cur = parent
	}
}

// ResolvePortingSDK locates the sibling porting-sdk checkout, preferring
// $PORTING_SDK when it points at a tree containing the given marker subdir
// (rest-apis / relay-protocol / swaig-specs), else falling back to ../porting-sdk.
func ResolvePortingSDK(repoRoot, marker string) (string, error) {
	if p := os.Getenv("PORTING_SDK"); p != "" {
		if _, err := os.Stat(filepath.Join(p, marker)); err == nil {
			return p, nil
		}
	}
	cand := filepath.Join(repoRoot, "..", "porting-sdk")
	if _, err := os.Stat(filepath.Join(cand, marker)); err == nil {
		return filepath.Abs(cand)
	}
	return "", fmt.Errorf("porting-sdk not found (set $PORTING_SDK or clone adjacent)")
}

// Run gofmts every Output then, in --check mode, byte-compares each against the
// on-disk file and reports the stale set non-zero (GEN-FRESH); otherwise writes
// each file. cmdName / label appear in the log + failure messages (e.g.
// cmdName="generate-relay-protocol", label="relay protocol").
func Run(check bool, cmdName, label string, outs []Output) error {
	var stale []string
	for _, o := range outs {
		formatted, err := GofmtSrc(o.Src)
		if err != nil {
			return fmt.Errorf("%s: %w", o.Path, err)
		}
		if check {
			existing, err := os.ReadFile(o.Path)
			if err != nil || !bytes.Equal(existing, formatted) {
				stale = append(stale, o.Path)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(o.Path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(o.Path, formatted, 0o644); err != nil {
			return err
		}
		fmt.Printf("generated %s\n", o.Path)
	}
	if check && len(stale) > 0 {
		fmt.Fprintf(os.Stderr, "\nGEN-FRESH FAIL: %d generated %s file(s) stale — run `go run ./cmd/%s` and commit:\n", len(stale), label, cmdName)
		for _, f := range stale {
			fmt.Fprintf(os.Stderr, "  - %s\n", f)
		}
		return fmt.Errorf("stale generated files")
	}
	if check {
		fmt.Printf("GEN-FRESH: generated %s files match the canonical specs.\n", label)
	}
	return nil
}
