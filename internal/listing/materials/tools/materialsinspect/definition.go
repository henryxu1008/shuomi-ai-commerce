// Package materialsinspect binds the platform materials read to the existing
// Commerce Tool registry, live authorization and durable call audit.
package materialsinspect

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/invopop/jsonschema"

	"task-processor/internal/authz"
	"task-processor/internal/commercetool"
	"task-processor/internal/listing/materials"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
)

func AgentDefinition() commercetool.AgentDefinition {
	return commercetool.AgentDefinition{ID: materials.AgentID, Version: materials.AgentVersion, AllowedTools: []commercetool.ToolRef{materials.InspectToolRef()}}
}

func Definition() commercetool.Definition {
	return commercetool.Definition{
		Ref: materials.InspectToolRef(), Capability: "listing.materials", Owner: "listing.materials",
		Description: "Read exact saved local SHEIN draft, standard product, current merchant rules and approved assets. No publication or mutation.",
		InputSchema: InputSchema(), OutputSchema: OutputSchema(), Risk: commercetool.RiskRead,
		Permission:  commercetool.PermissionRequirement{Permission: authz.PermissionLocalAgentWrite},
		SideEffects: commercetool.SideEffectPolicy{Mode: commercetool.SideEffectNone},
		Idempotency: commercetool.IdempotencyPolicy{Mode: commercetool.IdempotencyDeterministic},
		Timeout:     commercetool.TimeoutPolicy{Duration: 10 * time.Second},
		Retry:       commercetool.RetryPolicy{Owner: commercetool.RetryOwnerCaller},
		Usage:       commercetool.UsagePolicy{Owner: commercetool.UsageOwnerUnmetered},
	}
}

func InputSchema() json.RawMessage {
	return json.RawMessage(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,"required":["binding"],"properties":{"binding":{"type":"object","additionalProperties":false,"required":["ContextKind","ContextID","ProductKey","CatalogVersion","PublicationID","TargetPlatform"],"properties":{"ContextKind":{"const":"listing-target-record"},"ContextID":{"type":"string","minLength":1,"maxLength":128},"ProductKey":{"type":"string","minLength":1,"maxLength":128},"CatalogVersion":{"type":"string","pattern":"^[1-9][0-9]{0,18}$"},"PublicationID":{"type":"string","minLength":1,"maxLength":128},"TargetPlatform":{"const":"shein"}}}}}`)
}

func OutputSchema() json.RawMessage {
	// Reuse the installed open-source reflector for the current official
	// DTOs, and Catalog's exact published schema for Product facts. Every
	// object remains closed; no opaque JSON/string or permissive fallback.
	r := jsonschema.Reflector{Anonymous: true, ExpandedStruct: true, Mapper: func(t reflect.Type) *jsonschema.Schema {
		if t == reflect.TypeOf(catalog.ProductSnapshot{}) {
			return &jsonschema.Schema{Ref: "#/$defs/product_snapshot"}
		}
		return nil
	}}
	raw, err := json.Marshal(r.Reflect(materials.Disclosure{}))
	if err != nil {
		return nil
	}
	var document, canonical map[string]any
	if json.Unmarshal(raw, &document) != nil || json.Unmarshal(canonicalinspect.OutputSchema(), &canonical) != nil {
		return nil
	}
	// Current owner DTOs distinguish nil arrays from absent scalar flags.
	// Accept JSON null for those arrays without relaxing their item schema.
	allowNilArrays(document)
	allowUnknownOwnerFlags(document, reflect.TypeOf(materials.Disclosure{}))
	defs, ok := document["$defs"].(map[string]any)
	if !ok {
		defs = map[string]any{}
		document["$defs"] = defs
	}
	for name, schema := range canonical["$defs"].(map[string]any) {
		defs[name] = schema
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		return nil
	}
	return encoded
}

// The reflector's nullable tag cannot be added to another owner's DTO. Keep
// its actual pointer flags nullable in this local read schema; nil means the
// official rule is unknown. Catalog retains its existing schema unchanged.
func allowUnknownOwnerFlags(document map[string]any, root reflect.Type) {
	defs, _ := document["$defs"].(map[string]any)
	seen := map[reflect.Type]bool{}
	var visit func(reflect.Type)
	visit = func(t reflect.Type) {
		for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct || seen[t] || t == reflect.TypeOf(catalog.ProductSnapshot{}) {
			return
		}
		seen[t] = true
		schema, _ := defs[t.Name()].(map[string]any)
		if t == root {
			schema = document
		}
		properties, _ := schema["properties"].(map[string]any)
		for n := 0; n < t.NumField(); n++ {
			field := t.Field(n)
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "" {
				name = field.Name
			}
			if value, ok := properties[name]; ok && field.Type.Kind() == reflect.Pointer {
				properties[name] = map[string]any{"anyOf": []any{value, map[string]any{"type": "null"}}}
			}
			visit(field.Type)
		}
	}
	visit(root)
}

func allowNilArrays(v any) {
	switch value := v.(type) {
	case map[string]any:
		if value["type"] == "array" {
			value["type"] = []string{"array", "null"}
		}
		for _, child := range value {
			allowNilArrays(child)
		}
	case []any:
		for _, child := range value {
			allowNilArrays(child)
		}
	}
}
