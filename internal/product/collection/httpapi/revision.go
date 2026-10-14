package httpapi

import (
	"encoding/json"
	"math"
	"strconv"
	"task-processor/internal/product/collection"
)

func revisionVersionDeclared(raw []byte, input collection.Mutation) bool {
	if input.Action != "revise_product" {
		return true
	}
	if input.ExpectedSourceVersion == 0 || input.ExpectedSourceVersion > math.MaxInt64 {
		return false
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return false
	}
	var declared string
	return json.Unmarshal(fields["expectedSourceVersion"], &declared) == nil && declared == strconv.FormatUint(input.ExpectedSourceVersion, 10)
}
