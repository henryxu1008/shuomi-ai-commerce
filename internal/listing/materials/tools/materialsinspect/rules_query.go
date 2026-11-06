package materialsinspect

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/invopop/jsonschema"
	sigjson "sigs.k8s.io/json"
	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
	"task-processor/internal/listing/materials"
)

func RulesDefinition() commercetool.Definition {
	d := Definition()
	d.Ref = materials.RulesToolRef()
	d.Description = "Read the user-selected SHEIN category base rules for the frozen saved Target. Conditional attribute combinations remain not_evaluated. No save/adopt/publication."
	d.InputSchema, d.OutputSchema = RulesInputSchema(), RulesOutputSchema()
	return d
}

func RulesInputSchema() json.RawMessage {
	var doc map[string]any
	if json.Unmarshal(InputSchema(), &doc) != nil {
		return nil
	}
	b := doc["properties"].(map[string]any)["binding"].(map[string]any)
	p := b["properties"].(map[string]any)
	p["ContextKind"] = map[string]any{"const": agent.CandidateRulesContextKind}
	fields := map[string]any{}
	for _, name := range []string{"BaseRevision", "CandidateCategoryID"} {
		fields[name] = map[string]any{"type": "string", "pattern": "^[1-9][0-9]{0,18}$"}
	}
	for _, name := range []string{"BaseDigest", "MerchantDigest", "SelectorDigest"} {
		fields[name] = map[string]any{"type": "string", "pattern": "^[0-9a-f]{64}$"}
	}
	fields["StoreID"] = map[string]any{"type": "string", "minLength": 36, "maxLength": 36}
	fields["Site"] = map[string]any{"const": "shein-us"}
	p["CandidateRules"] = map[string]any{"type": "object", "additionalProperties": false, "required": []string{"BaseRevision", "BaseDigest", "StoreID", "Site", "MerchantDigest", "CandidateCategoryID", "SelectorDigest"}, "properties": fields}
	b["required"] = []string{"ContextKind", "ContextID", "ProductKey", "CatalogVersion", "PublicationID", "TargetPlatform", "CandidateRules"}
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil
	}
	return raw
}

func RulesOutputSchema() json.RawMessage {
	r := jsonschema.Reflector{Anonymous: true, ExpandedStruct: true}
	raw, err := json.Marshal(r.Reflect(materials.CandidateRulesDisclosure{}))
	if err != nil {
		return nil
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	allowNilArrays(doc)
	allowUnknownOwnerFlags(doc, reflect.TypeOf(materials.CandidateRulesDisclosure{}))
	setBindingSchema(doc, RulesInputSchema())
	raw, err = json.Marshal(doc)
	if err != nil {
		return nil
	}
	return raw
}

func setBindingSchema(doc map[string]any, input json.RawMessage) {
	var in map[string]any
	if json.Unmarshal(input, &in) != nil {
		return
	}
	if defs, ok := doc["$defs"].(map[string]any); ok {
		defs["Binding"] = in["properties"].(map[string]any)["binding"]
	}
}

type rulesExecutor struct {
	reader materials.CandidateRulesReader
}

func (e rulesExecutor) Execute(ctx context.Context, envelope commercetool.ExecutionEnvelope, raw json.RawMessage) (commercetool.ExecutionResult, error) {
	var in input
	strict, err := sigjson.UnmarshalStrict(raw, &in)
	if err != nil || len(strict) != 0 || !in.Binding.Valid() || in.Binding.ContextKind != agent.CandidateRulesContextKind {
		return commercetool.ExecutionResult{}, commercetool.NewError(commercetool.ErrorInvalidInput, "invalid candidate rules binding", nil)
	}
	value, err := e.reader.ReadCandidateRules(ctx, in.Binding)
	if err != nil {
		return commercetool.ExecutionResult{}, readError(err)
	}
	if value.Binding != in.Binding || value.OrganizationID != envelope.Principal().TenantID {
		return commercetool.ExecutionResult{}, commercetool.NewError(commercetool.ErrorPermissionDenied, "candidate rules owner differs from current organization", nil)
	}
	value.Evidence.CallID = envelope.Metadata().CallID
	output, err := json.Marshal(value)
	if err != nil || len(output) > materials.MaxInspectionBytes {
		return commercetool.ExecutionResult{}, readError(materials.ErrInspectionTooLarge)
	}
	if _, err = materials.ParseCandidateRulesDisclosure(output); err != nil {
		return commercetool.ExecutionResult{}, readError(err)
	}
	if err = ctx.Err(); err != nil {
		return commercetool.ExecutionResult{}, readError(err)
	}
	return commercetool.ExecutionResult{Output: output}, nil
}
