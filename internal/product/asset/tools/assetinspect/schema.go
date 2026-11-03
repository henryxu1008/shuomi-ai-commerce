package assetinspect

import (
	"encoding/json"
	"github.com/invopop/jsonschema"
	"task-processor/internal/product/asset"

	"task-processor/internal/product/catalog/tools/canonicalinspect"
)

func InputSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","additionalProperties":false,"required":["product_key","catalog_version","target_platform"],"properties":{"product_key":{"type":"string","minLength":1,"maxLength":128},"catalog_version":{"type":"string","pattern":"^[1-9][0-9]{0,18}$"},"target_platform":{"type":"string","minLength":1,"maxLength":128}}}`)
}

// Reuse the existing approved Catalog projection schema; this tool adds only
// exact Asset facts. Each call returns a detached schema document.
func OutputSchema() json.RawMessage {
	var schema map[string]any
	if err := json.Unmarshal(canonicalinspect.OutputSchema(), &schema); err != nil {
		panic(err)
	}
	properties := schema["properties"].(map[string]any)
	properties["asset_binding"] = map[string]any{
		"type": "object", "additionalProperties": false,
		"required": []string{"target_platform", "source_snapshot_version"},
		"properties": map[string]any{
			"target_platform":         map[string]any{"type": "string", "minLength": 1, "maxLength": asset.MaxIdentityLength},
			"source_snapshot_version": map[string]any{"type": "string", "pattern": "^[1-9][0-9]{0,18}$"},
		},
	}
	// Domain projection validates the exact current origin before marshaling.
	// Reflect its existing fact fields rather than maintaining a generated-only
	// duplicate DTO that rejects source approvals and immutable evidence.
	reflector := jsonschema.Reflector{Anonymous: true, ExpandedStruct: true}
	var approved map[string]any
	encoded, err := json.Marshal(reflector.Reflect(asset.ApprovedAsset{}))
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(encoded, &approved); err != nil {
		panic(err)
	}
	delete(approved, "$schema")
	if definitions, ok := approved["$defs"].(map[string]any); ok {
		original, ok := schema["$defs"].(map[string]any)
		if !ok {
			original = map[string]any{}
		}
		for key, value := range definitions {
			if _, exists := original[key]; exists {
				panic("asset and Catalog schema definition collision")
			}
			original[key] = value
		}
		schema["$defs"] = original
		delete(approved, "$defs")
	}
	fields := approved["properties"].(map[string]any)
	fields["role"] = map[string]any{"type": "string", "enum": []string{"design", "main", "white_background", "gallery"}}
	for _, key := range []string{"id", "source_asset_id"} {
		fields[key] = map[string]any{"type": "string", "minLength": 1, "maxLength": asset.MaxIdentityLength}
	}
	fields["url"] = map[string]any{"type": "string", "minLength": 1}
	generatedFields, sourceFields := map[string]any{}, map[string]any{}
	for key, value := range fields {
		if key != "source_approval" {
			generatedFields[key] = value
		}
		if key != "generation_evidence" && key != "selection_receipt" {
			sourceFields[key] = value
		}
	}
	for _, key := range []string{"run_id", "slot_id"} {
		generatedFields[key] = map[string]any{"type": "string", "minLength": 1, "maxLength": asset.MaxIdentityLength}
		sourceFields[key] = map[string]any{"type": "string", "const": ""}
	}
	for _, key := range []string{"plan_revision", "attempt"} {
		generatedFields[key] = map[string]any{"type": "integer", "minimum": 1}
		sourceFields[key] = map[string]any{"type": "integer", "const": 0}
	}
	required := approved["required"].([]any)
	generated := map[string]any{"type": "object", "additionalProperties": false, "properties": generatedFields, "required": required}
	source := map[string]any{"type": "object", "additionalProperties": false, "properties": sourceFields, "required": append(append([]any{}, required...), "source_approval", "source_asset_id")}
	properties["approved_assets"] = map[string]any{"type": "array", "minItems": 1, "items": map[string]any{"oneOf": []any{generated, source}}}
	schema["required"] = append(schema["required"].([]any), "asset_binding", "approved_assets")
	raw, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return raw
}
