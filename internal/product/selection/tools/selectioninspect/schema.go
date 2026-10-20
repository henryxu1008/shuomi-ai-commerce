package selectioninspect

import (
	"encoding/json"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
)

// Fact values use current Catalog's closed DTO schemas. No arbitrary object
// can bypass the original Tool authority-field schema and output checks.
func OutputSchema() json.RawMessage {
	var document, canonical map[string]any
	if json.Unmarshal(outputSchema, &document) != nil || json.Unmarshal(canonicalinspect.OutputSchema(), &canonical) != nil {
		return nil
	}
	document["$defs"] = canonical["$defs"]
	object := func(properties map[string]any) map[string]any {
		return map[string]any{"type": "object", "additionalProperties": false, "properties": properties}
	}
	array := func(items any) map[string]any { return map[string]any{"type": "array", "items": items} }
	text := map[string]any{"type": "string"}
	attribute := object(map[string]any{"Name": text, "Value": text})
	price := object(map[string]any{"amount": map[string]any{"type": "number"}, "currency": text})
	pair := array(text)
	pair["minItems"] = 2
	pair["maxItems"] = 2
	value := map[string]any{"anyOf": []any{
		text, map[string]any{"type": "number"}, map[string]any{"type": "boolean"},
		array(text), array(pair), attribute, price,
		map[string]any{"$ref": "#/$defs/specifications"},
		map[string]any{"$ref": "#/$defs/review"},
		array(map[string]any{"$ref": "#/$defs/warning"}),
	}}
	properties := document["properties"].(map[string]any)
	products := properties["products"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
	facts := products["facts"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
	facts["value"] = value
	raw, err := json.Marshal(document)
	if err != nil {
		return nil
	}
	return raw
}
