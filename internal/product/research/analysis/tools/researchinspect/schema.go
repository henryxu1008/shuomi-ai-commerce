package researchinspect

import (
	"encoding/json"
	"github.com/invopop/jsonschema"
	"task-processor/internal/product/research/analysis"
	"task-processor/internal/product/selection/tools/selectioninspect"
)

// Use the existing closed Catalog fact-value schema, including its definitions.
func OutputSchema() json.RawMessage {
	r := jsonschema.Reflector{Anonymous: true, ExpandedStruct: true}
	raw, e := json.Marshal(r.Reflect(analysis.Inspection{}))
	if e != nil {
		return nil
	}
	var doc, original map[string]any
	if json.Unmarshal(raw, &doc) != nil || json.Unmarshal(selectioninspect.OutputSchema(), &original) != nil {
		return nil
	}
	defs, _ := doc["$defs"].(map[string]any)
	if defs == nil {
		defs = map[string]any{}
	}
	for k, v := range original["$defs"].(map[string]any) {
		defs[k] = v
	}
	doc["$defs"] = defs
	products := original["properties"].(map[string]any)["products"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
	fact := products["facts"].(map[string]any)
	defs["Fact"] = fact["items"]
	// The only arbitrary RawMessage is replaced by the qualified fact schema.
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if props, ok := x["properties"].(map[string]any); ok {
				if _, found := props["facts"]; found {
					props["facts"] = fact
				}
			}
			for _, child := range x {
				walk(child)
			}
		case []any:
			for _, child := range x {
				walk(child)
			}
		}
	}
	walk(doc)
	out, _ := json.Marshal(doc)
	return out
}
