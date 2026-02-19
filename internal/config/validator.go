package config

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// schemaBytes holds the embedded JSON Schema.
// It is set by the schemas package init or by SetSchema() for testing.
var schemaBytes []byte

// SetSchema sets the JSON Schema bytes used for validation.
// This is called by the schemas package init() or can be called in tests.
func SetSchema(data []byte) {
	schemaBytes = data
}

// GetSchema returns the embedded JSON Schema bytes.
func GetSchema() []byte {
	return schemaBytes
}

// ValidationError represents a single validation failure.
type ValidationError struct {
	Field       string `json:"field"`
	Description string `json:"description"`
}

// ValidationResult holds the outcome of a config validation.
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// Validate validates an LZConfig against the embedded JSON Schema.
func Validate(cfg *LZConfig) (*ValidationResult, error) {
	// Convert Go struct → JSON for schema validation
	jsonBytes, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshaling config to JSON: %w", err)
	}

	if len(schemaBytes) == 0 {
		return nil, fmt.Errorf("JSON schema not loaded; call config.SetSchema() or import the schemas package")
	}

	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
	documentLoader := gojsonschema.NewBytesLoader(jsonBytes)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("running schema validation: %w", err)
	}

	vr := &ValidationResult{Valid: result.Valid()}
	for _, e := range result.Errors() {
		vr.Errors = append(vr.Errors, ValidationError{
			Field:       e.Field(),
			Description: e.Description(),
		})
	}
	return vr, nil
}

// ValidateYAML validates raw YAML bytes against the schema.
// It parses YAML → JSON first (since JSON Schema operates on JSON).
func ValidateYAML(data []byte) (*ValidationResult, error) {
	// Parse YAML to generic map, then re-marshal as JSON
	var raw interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}
	// yaml.v3 produces map[string]interface{} for mappings, which json.Marshal handles
	jsonBytes, err := json.Marshal(convertYAMLToJSON(raw))
	if err != nil {
		return nil, fmt.Errorf("converting YAML to JSON: %w", err)
	}

	if len(schemaBytes) == 0 {
		return nil, fmt.Errorf("JSON schema not loaded; call config.SetSchema() or import the schemas package")
	}

	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
	documentLoader := gojsonschema.NewBytesLoader(jsonBytes)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("running schema validation: %w", err)
	}

	vr := &ValidationResult{Valid: result.Valid()}
	for _, e := range result.Errors() {
		vr.Errors = append(vr.Errors, ValidationError{
			Field:       e.Field(),
			Description: e.Description(),
		})
	}
	return vr, nil
}

// convertYAMLToJSON recursively converts yaml-parsed maps (map[string]interface{})
// to a format compatible with JSON marshaling. yaml.v3 already uses
// map[string]interface{} so this is mostly a passthrough, but handles
// edge cases like map[interface{}]interface{} from older YAML libs.
func convertYAMLToJSON(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, v2 := range val {
			result[k] = convertYAMLToJSON(v2)
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, v2 := range val {
			result[fmt.Sprintf("%v", k)] = convertYAMLToJSON(v2)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, v2 := range val {
			result[i] = convertYAMLToJSON(v2)
		}
		return result
	default:
		return v
	}
}
