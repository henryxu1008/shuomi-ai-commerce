package supplychainhttp

import (
	"bytes"
	"encoding/json"
	"reflect"
	sigjson "sigs.k8s.io/json"
	"strconv"
	"strings"
	supplyapp "task-processor/internal/app/supplychain"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/collection"
)

// The browser DTO encodes TEMU integers as decimal strings. The fixed typed
// owners and official wire protocol keep native integers. This is serialization
// only: no business-field defaults, aliases or unknown-field forwarding.
type supplyTargetBody struct {
	record.TargetInput
	Temu      json.RawMessage `json:"temu,omitempty"`
	TemuRules json.RawMessage `json:"temuRules,omitempty"`
}

func (b supplyTargetBody) domain() (record.TargetInput, error) {
	value := b.TargetInput
	if len(b.Temu) > 0 {
		var input temugoods.DraftInput
		if decodeTemuBrowser(b.Temu, &input) != nil {
			return record.TargetInput{}, preparation.ErrInvalid
		}
		value.Temu = &input
	}
	if len(b.TemuRules) > 0 {
		var selection model.RuleSelection
		if decodeTemuBrowser(b.TemuRules, &selection) != nil {
			return record.TargetInput{}, preparation.ErrInvalid
		}
		value.TemuRules = &selection
	}
	return value, nil
}

func decodeTemuBrowser(raw []byte, destination any) error {
	if len(raw) == 0 || len(raw) > supplyMaxBytes || bytes.Equal(raw, []byte("null")) {
		return preparation.ErrInvalid
	}
	var checked any
	strict, err := sigjson.UnmarshalStrict(raw, &checked, sigjson.DisallowDuplicateFields)
	if err != nil || len(strict) > 0 {
		return preparation.ErrInvalid
	}
	var tree any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&tree) != nil {
		return preparation.ErrInvalid
	}
	typed, err := temuBrowserIntegerInput(tree, reflect.TypeOf(destination).Elem())
	if err != nil {
		return err
	}
	normalized, err := json.Marshal(typed)
	if err != nil {
		return preparation.ErrInvalid
	}
	strict, err = sigjson.UnmarshalStrict(normalized, destination, sigjson.DisallowDuplicateFields, sigjson.DisallowUnknownFields)
	if err != nil || len(strict) > 0 {
		return preparation.ErrInvalid
	}
	return nil
}

func temuBrowserIntegerInput(value any, typ reflect.Type) (any, error) {
	if value == nil {
		return nil, nil
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Int, reflect.Int64, reflect.Int32:
		text, ok := value.(string)
		if !ok {
			number, ok := value.(json.Number)
			if !ok {
				return nil, preparation.ErrInvalid
			}
			text = number.String()
		}
		number, err := strconv.ParseInt(text, 10, typ.Bits())
		if err != nil || strconv.FormatInt(number, 10) != text {
			return nil, preparation.ErrInvalid
		}
		return json.Number(text), nil
	case reflect.Struct:
		object, ok := value.(map[string]any)
		if !ok {
			return nil, preparation.ErrInvalid
		}
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if field.Anonymous && name == "" {
				if _, err := temuBrowserIntegerInput(object, field.Type); err != nil {
					return nil, err
				}
				continue
			}
			if name == "" || name == "-" {
				continue
			}
			if member, exists := object[name]; exists {
				converted, err := temuBrowserIntegerInput(member, field.Type)
				if err != nil {
					return nil, err
				}
				object[name] = converted
			}
		}
	case reflect.Slice, reflect.Array:
		values, ok := value.([]any)
		if !ok {
			return nil, preparation.ErrInvalid
		}
		if typ.Kind() == reflect.Array && len(values) != typ.Len() {
			return nil, preparation.ErrInvalid
		}
		for i, v := range values {
			converted, err := temuBrowserIntegerInput(v, typ.Elem())
			if err != nil {
				return nil, err
			}
			values[i] = converted
		}
	}
	return value, nil
}

type publicTemuValue struct{ value any }

func (v publicTemuValue) MarshalJSON() ([]byte, error) {
	raw, err := json.Marshal(v.value)
	if err != nil {
		return nil, err
	}
	var tree any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err = decoder.Decode(&tree); err != nil {
		return nil, err
	}
	return json.Marshal(temuBrowserIntegerOutput(tree))
}
func temuBrowserIntegerOutput(value any) any {
	switch v := value.(type) {
	case json.Number:
		return v.String()
	case map[string]any:
		for k, item := range v {
			v[k] = temuBrowserIntegerOutput(item)
		}
	case []any:
		for i, item := range v {
			v[i] = temuBrowserIntegerOutput(item)
		}
	}
	return value
}
func temuPublic(value any) *publicTemuValue {
	if value == nil || (reflect.ValueOf(value).Kind() == reflect.Pointer || reflect.ValueOf(value).Kind() == reflect.Slice) && reflect.ValueOf(value).IsNil() {
		return nil
	}
	return &publicTemuValue{value}
}

type supplyTargetInputResponse struct {
	record.TargetInput
	Draft     *struct{}        `json:"draft,omitempty"`
	Temu      *publicTemuValue `json:"temu,omitempty"`
	TemuRules *publicTemuValue `json:"temuRules,omitempty"`
}
type supplyTargetResponse struct {
	record.TargetRecord
	Result                *struct{}                 `json:"result,omitempty"`
	Input                 supplyTargetInputResponse `json:"input"`
	Temu                  *publicTemuValue          `json:"temu,omitempty"`
	TemuRuleSnapshot      *publicTemuValue          `json:"temuRuleSnapshot,omitempty"`
	TemuImageObservations *publicTemuValue          `json:"temuImageObservations,omitempty"`
}

func publicTarget(value record.TargetRecord) supplyTargetResponse {
	return supplyTargetResponse{TargetRecord: value, Input: supplyTargetInputResponse{TargetInput: value.Input, Temu: temuPublic(value.Input.Temu), TemuRules: temuPublic(value.Input.TemuRules)}, Temu: temuPublic(value.Temu), TemuRuleSnapshot: temuPublic(value.TemuRuleSnapshot), TemuImageObservations: temuPublic(value.TemuImageObservations)}
}
func supplyPublicResponse(value any) any {
	switch v := value.(type) {
	case record.TargetRecord:
		if v.Input.Temu != nil {
			return publicTarget(v)
		}
	case record.TargetReceipt:
		if v.Record.Input.Temu != nil {
			return struct {
				Record   supplyTargetResponse `json:"record"`
				Replayed bool                 `json:"replayed"`
			}{publicTarget(v.Record), v.Replayed}
		}
	case supplyapp.TargetRules:
		if v.Temu != nil {
			return struct {
				supplyapp.TargetRules
				Temu *publicTemuValue `json:"temu"`
			}{v, temuPublic(v.Temu)}
		}
	case supplyapp.PublicationView:
		if v.TemuProduct != nil {
			return struct {
				supplyapp.PublicationView
				TemuProduct *publicTemuValue `json:"temuProduct,omitempty"`
			}{v, temuPublic(v.TemuProduct)}
		}
	case supplyapp.UploadResult:
		if v.TemuProduct != nil {
			return struct {
				supplyapp.UploadResult
				Product     *struct{}        `json:"product,omitempty"`
				TemuProduct *publicTemuValue `json:"temuProduct,omitempty"`
			}{v, nil, temuPublic(v.TemuProduct)}
		}
	case collection.Page[supplyapp.OperationItemView]:
		type item struct {
			supplyapp.OperationItemView
			ConfirmedTemuProduct *publicTemuValue `json:"confirmedTemuProduct,omitempty"`
		}
		items := make([]item, 0, len(v.Items))
		for _, value := range v.Items {
			items = append(items, item{value, temuPublic(value.ConfirmedTemuProduct)})
		}
		return struct {
			Items      []item `json:"items"`
			Total      int64  `json:"total"`
			NextCursor string `json:"nextCursor,omitempty"`
		}{items, v.Total, v.NextCursor}
	}
	return value
}
