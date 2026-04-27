package swml

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/signalwire/signalwire-go/pkg/logging"
)

//go:embed schema.json
var schemaFS embed.FS

var log = logging.New("swml")

// VerbInfo holds metadata about a SWML verb extracted from the schema.
type VerbInfo struct {
	// Name is the actual SWML verb name (e.g., "sip_refer", "ai", "play")
	Name string
	// SchemaName is the PascalCase name from the schema definition (e.g., "SIPRefer", "AI", "Play")
	SchemaName string
	// Definition is the raw schema definition for this verb
	Definition map[string]any
}

// Schema holds the parsed SWML schema and provides verb metadata.
type Schema struct {
	mu    sync.RWMutex
	raw   map[string]any
	verbs map[string]*VerbInfo // keyed by actual verb name (e.g., "sip_refer")
}

var (
	globalSchema     *Schema
	globalSchemaOnce sync.Once
	globalSchemaErr  error
)

// GetSchema returns the global singleton Schema loaded from the embedded schema.json.
func GetSchema() (*Schema, error) {
	globalSchemaOnce.Do(func() {
		globalSchema, globalSchemaErr = loadEmbeddedSchema()
	})
	return globalSchema, globalSchemaErr
}

func loadEmbeddedSchema() (*Schema, error) {
	data, err := schemaFS.ReadFile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded schema.json: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse schema.json: %w", err)
	}

	s := &Schema{
		raw:   raw,
		verbs: make(map[string]*VerbInfo),
	}

	s.extractVerbDefinitions()
	log.Debug("schema loaded with %d verbs", len(s.verbs))
	return s, nil
}

// extractVerbDefinitions parses the schema to discover all SWML verbs.
// The actual verb name is extracted from the first property key of each definition,
// matching Python's behavior. For example, schema def "SIPRefer" has property "sip_refer".
func (s *Schema) extractVerbDefinitions() {
	defs, ok := s.raw["$defs"].(map[string]any)
	if !ok {
		log.Warn("schema missing $defs")
		return
	}

	swmlMethod, ok := defs["SWMLMethod"].(map[string]any)
	if !ok {
		log.Warn("schema missing SWMLMethod definition")
		return
	}

	anyOf, ok := swmlMethod["anyOf"].([]any)
	if !ok {
		log.Warn("SWMLMethod missing anyOf")
		return
	}

	for _, ref := range anyOf {
		refMap, ok := ref.(map[string]any)
		if !ok {
			continue
		}
		refStr, ok := refMap["$ref"].(string)
		if !ok {
			continue
		}

		// Extract schema name from $ref like "#/$defs/SIPRefer"
		schemaName := refStr[len("#/$defs/"):]

		defn, ok := defs[schemaName].(map[string]any)
		if !ok {
			continue
		}

		// The actual verb name is the first property key in the definition
		props, ok := defn["properties"].(map[string]any)
		if !ok {
			continue
		}

		for actualVerb := range props {
			s.verbs[actualVerb] = &VerbInfo{
				Name:       actualVerb,
				SchemaName: schemaName,
				Definition: defn,
			}
			break // only the first property key
		}
	}
}

// GetVerb returns metadata for a verb by its actual name (e.g., "sip_refer").
func (s *Schema) GetVerb(name string) (*VerbInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.verbs[name]
	return v, ok
}

// GetAllVerbNames returns all known verb names (the actual SWML names, not schema names).
func (s *Schema) GetAllVerbNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.verbs))
	for name := range s.verbs {
		names = append(names, name)
	}
	return names
}

// IsValidVerb returns whether a name is a recognized SWML verb.
func (s *Schema) IsValidVerb(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.verbs[name]
	return ok
}

// VerbCount returns the number of verbs in the schema.
func (s *Schema) VerbCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.verbs)
}

// LoadSchemaFromFile loads a SWML schema from the given file path instead of
// the embedded schema.json. Mirrors Python's schema_path constructor param.
func LoadSchemaFromFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file %q: %w", path, err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse schema file %q: %w", path, err)
	}
	s := &Schema{
		raw:   raw,
		verbs: make(map[string]*VerbInfo),
	}
	s.extractVerbDefinitions()
	log.Debug("schema loaded from file %q with %d verbs", path, len(s.verbs))
	return s, nil
}
