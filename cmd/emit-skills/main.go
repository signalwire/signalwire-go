// Command emit-skills is the Go port's SKILL-DUMP program for the cross-port
// SKILL-CONTRACT differ (porting-sdk/scripts/diff_skill_contracts.py).
//
// The sibling of cmd/emit-corpus, for built-in SKILLS rather than
// FunctionResult. For each covered skill it looks up the skill's factory in the
// global registry (populated by the builtin packages' init()), instantiates it
// with the canonical config from the shared corpus
// (porting-sdk/scripts/skill_contract_corpus.py — the single source of truth),
// runs Setup() + RegisterTools(), and prints ONE JSON object mapping
//
//	skill-id -> [ { "name": ..., "parameters": {...} }, ... ]
//
// to stdout. The differ runs this, parses it, and structurally compares each
// skill's tool contract against the Python reference (which registers the same
// tools). The differ normalises both sides (flat vs wrapped params, required
// list, enum order); this program just emits each tool's Name + Parameters
// verbatim. DESCRIPTIONS are not part of the compared contract.
//
// CONTRACT (mirrors the per-port dump contract in the differ's --help):
//   - The id set MUST equal corpus_ids() (the differ rejects a mismatch).
//   - Only stdout carries the JSON object; logs go to stderr.
//
// Run from the signalwire-go repo root:
//
//	go run ./cmd/emit-skills
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/signalwire/signalwire-go/pkg/skills"
	_ "github.com/signalwire/signalwire-go/pkg/skills/all" // register all built-in skills via init()
)

// corpusEntry mirrors one entry of skill_contract_corpus.py's CORPUS.
type corpusEntry struct {
	ID     string         `json:"id"`
	Skill  string         `json:"skill"`
	Config map[string]any `json:"config"`
}

// toolContract is the per-tool shape the differ reads. Parameters is emitted
// verbatim (the wrapped JSON-Schema map the skill registered); the differ peels
// off type/properties/required and folds enum/required order.
type toolContract struct {
	Name       string         `json:"name"`
	Parameters map[string]any `json:"parameters"`
}

// loadCorpus runs the shared corpus script and returns its CORPUS entries.
// porting-sdk is resolved via $PORTING_SDK / $PORTING_SDK_PATH or the sibling
// ../porting-sdk (the adjacency convention).
func loadCorpus() ([]corpusEntry, error) {
	var bases []string
	for _, env := range []string{os.Getenv("PORTING_SDK"), os.Getenv("PORTING_SDK_PATH")} {
		if env != "" {
			bases = append(bases, env)
		}
	}
	if wd, err := os.Getwd(); err == nil {
		bases = append(bases, filepath.Join(wd, "..", "porting-sdk"))
	}
	for _, base := range bases {
		script := filepath.Join(base, "scripts", "skill_contract_corpus.py")
		if _, err := os.Stat(script); err != nil {
			continue
		}
		out, err := exec.Command("python3", script).Output()
		if err != nil {
			return nil, fmt.Errorf("running %s: %w", script, err)
		}
		var parsed struct {
			Corpus []corpusEntry `json:"corpus"`
		}
		if err := json.Unmarshal(out, &parsed); err != nil {
			return nil, fmt.Errorf("parsing corpus JSON: %w", err)
		}
		return parsed.Corpus, nil
	}
	return nil, fmt.Errorf("cannot locate porting-sdk/scripts/skill_contract_corpus.py " +
		"(set PORTING_SDK / PORTING_SDK_PATH or clone porting-sdk adjacent)")
}

func main() {
	corpus, err := loadCorpus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "emit-skills: %v\n", err)
		os.Exit(1)
	}

	result := make(map[string][]toolContract, len(corpus))
	for _, entry := range corpus {
		factory := skills.GetSkillFactory(entry.Skill)
		if factory == nil {
			fmt.Fprintf(os.Stderr, "emit-skills: no registered factory for covered skill %q\n", entry.Skill)
			os.Exit(1)
		}
		skill := factory(entry.Config)
		if !skill.Setup() {
			fmt.Fprintf(os.Stderr, "emit-skills: skill %q Setup() returned false with the "+
				"corpus config — config drift between the corpus and the port.\n", entry.Skill)
			os.Exit(1)
		}
		tools := skill.RegisterTools()
		contracts := make([]toolContract, 0, len(tools))
		for _, t := range tools {
			contracts = append(contracts, toolContract{Name: t.Name, Parameters: t.Parameters})
		}
		result[entry.ID] = contracts
	}

	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "emit-skills: encoding result: %v\n", err)
		os.Exit(1)
	}
}
