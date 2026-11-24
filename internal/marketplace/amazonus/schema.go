package amazonus

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/message"
)

const (
	metaSchemaURL = "https://schemas.amazon.com/selling-partners/definitions/product-types/meta-schema/v1"
	vocabularyURL = "https://schemas.amazon.com/selling-partners/definitions/product-types/vocabulary/v1"
	rulesResource = "urn:shuomi:amazon-product-type-definition"
)

func validateSchema(source, instance []byte) error {
	if len(source) == 0 || len(source) > 1<<20 {
		return ErrInvalidRules
	}
	raw, err := jsonschema.UnmarshalJSON(bytes.NewReader(source))
	doc, ok := raw.(map[string]any)
	if err != nil || !ok || doc["$schema"] != metaSchemaURL {
		return ErrInvalidRules
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2019)
	// The published Amazon dialect URI identifies an offline schema.
	compiler.UseLoader(jsonschema.SchemeURLLoader{})
	compiler.AssertVocabs()
	compiler.RegisterVocabulary(&jsonschema.Vocabulary{URL: vocabularyURL, Compile: compileAmazonConstraints})
	meta := map[string]any{
		"$schema": "https://json-schema.org/draft/2019-09/schema",
		"$id":     metaSchemaURL,
		"allOf":   []any{map[string]any{"$ref": "https://json-schema.org/draft/2019-09/schema"}},
	}
	if compiler.AddResource(metaSchemaURL, meta) != nil || compiler.AddResource(rulesResource, doc) != nil {
		return ErrInvalidRules
	}
	schema, err := compiler.Compile(rulesResource)
	if err != nil {
		return ErrInvalidRules
	}
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(instance))
	if err != nil {
		return ErrInvalidDraft
	}
	if schema.Validate(value) != nil {
		return ErrUnmetRules
	}
	return nil
}

type amazonConstraints struct {
	minBytes, maxBytes, minUnique, maxUnique int
	selectors                                []string
}

func compileAmazonConstraints(_ *jsonschema.CompilerContext, keywords map[string]any) (jsonschema.SchemaExt, error) {
	constraints := amazonConstraints{minBytes: -1, maxBytes: -1, minUnique: -1, maxUnique: -1}
	for name, target := range map[string]*int{
		"minUtf8ByteLength": &constraints.minBytes, "maxUtf8ByteLength": &constraints.maxBytes,
		"minUniqueItems": &constraints.minUnique, "maxUniqueItems": &constraints.maxUnique,
	} {
		raw, present := keywords[name]
		if !present {
			continue
		}
		number, ok := raw.(json.Number)
		if !ok {
			return nil, ErrInvalidRules
		}
		parsed, err := strconv.Atoi(string(number))
		if err != nil || parsed < 0 {
			return nil, ErrInvalidRules
		}
		*target = parsed
	}
	if raw, present := keywords["selectors"]; present {
		list, ok := raw.([]any)
		if !ok {
			return nil, ErrInvalidRules
		}
		for _, entry := range list {
			name, ok := entry.(string)
			if !ok || name == "" {
				return nil, ErrInvalidRules
			}
			constraints.selectors = append(constraints.selectors, name)
		}
	}
	if constraints.minBytes < 0 && constraints.maxBytes < 0 && constraints.minUnique < 0 && constraints.maxUnique < 0 {
		return nil, nil
	}
	return constraints, nil
}

func (a amazonConstraints) Validate(ctx *jsonschema.ValidatorContext, value any) {
	switch data := value.(type) {
	case string:
		checkRange(ctx, len(data), a.minBytes, a.maxBytes, "minUtf8ByteLength", "maxUtf8ByteLength")
	case []any:
		if a.minUnique < 0 && a.maxUnique < 0 {
			return
		}
		unique := make([]any, 0, len(data))
		for _, row := range data {
			key := row
			if object, ok := row.(map[string]any); ok && len(a.selectors) > 0 {
				selected := make(map[string]any, len(a.selectors))
				for _, name := range a.selectors {
					if field, present := object[name]; present {
						selected[name] = field
					}
				}
				key = selected
			}
			found := false
			for _, prior := range unique {
				equal, err := ctx.Equals(prior, key)
				if err != nil {
					ctx.AddErr(err)
					return
				}
				if equal {
					found = true
					break
				}
			}
			if !found {
				unique = append(unique, key)
			}
		}
		checkRange(ctx, len(unique), a.minUnique, a.maxUnique, "minUniqueItems", "maxUniqueItems")
	}
}

func checkRange(ctx *jsonschema.ValidatorContext, value, min, max int, minName, maxName string) {
	if min >= 0 && value < min {
		ctx.AddError(amazonConstraintError(minName))
	}
	if max >= 0 && value > max {
		ctx.AddError(amazonConstraintError(maxName))
	}
}

type amazonConstraintError string

func (e amazonConstraintError) KeywordPath() []string { return []string{string(e)} }
func (e amazonConstraintError) LocalizedString(p *message.Printer) string {
	return p.Sprintf("Amazon listing requirement %s is not met", string(e))
}
