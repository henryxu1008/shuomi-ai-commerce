package database

import (
	"encoding/json"
	"reflect"
	"strings"

	pgquery "github.com/wasilibs/go-pgquery/parser"
)

// SchemaConstraintEqual compares complete constraint definitions, retaining
// constants, operators and types. Only PostgreSQL's logical-restore rewrite of
// an unbounded varchar literal array cast to text[] is admitted. Catalog checks
// for identity, validation, deferrability, ownership and grants remain with the
// owning verifier. This function never executes SQL or repairs a schema.
func SchemaConstraintEqual(actual, expected string) bool {
	return schemaSQLEqual("CREATE TABLE __schema_contract ("+actual+")", "CREATE TABLE __schema_contract ("+expected+")")
}

// SchemaIndexEqual applies the same bounded comparison to full index DDL.
func SchemaIndexEqual(actual, expected string) bool { return schemaSQLEqual(actual, expected) }

func schemaSQLEqual(actual, expected string) bool {
	if actual == expected {
		return true
	}
	// All callers pass fixed schema contracts and bounded pg_catalog output.
	if len(actual) > 65536 || len(expected) > 65536 {
		return false
	}
	parse := func(sql string) (any, error) {
		raw, err := pgquery.ParseToJSON(sql)
		if err != nil {
			return nil, err
		}
		var tree any
		err = json.Unmarshal([]byte(raw), &tree)
		return normalizeRestoreTree(tree), err
	}
	a, err := parse(actual)
	if err != nil {
		return false
	}
	b, err := parse(expected)
	return err == nil && reflect.DeepEqual(a, b)
}

func normalizeRestoreTree(value any) any {
	switch node := value.(type) {
	case []any:
		for i := range node {
			node[i] = normalizeRestoreTree(node[i])
		}
	case map[string]any:
		for key, child := range node {
			switch key {
			case "location", "stmt_location", "stmt_len":
				delete(node, key)
			default:
				node[key] = normalizeRestoreTree(child)
			}
		}
		cast, ok := node["TypeCast"].(map[string]any)
		if !ok {
			return node
		}
		if schemaCastType(cast, "text", false) {
			if literal, ok := varcharStringLiteral(cast["arg"]); ok {
				cast["arg"] = literal
			}
		}
		if schemaCastType(cast, "text", true) {
			arg, _ := cast["arg"].(map[string]any)
			array, _ := arg["A_ArrayExpr"].(map[string]any)
			elements, ok := array["elements"].([]any)
			if !ok || len(elements) == 0 || len(array) != 1 {
				return node
			}
			literals := make([]any, len(elements))
			for i, element := range elements {
				literal, valid := varcharStringLiteral(element)
				if !valid {
					return node
				}
				literals[i] = map[string]any{"TypeCast": map[string]any{"arg": literal, "typeName": map[string]any{"names": []any{map[string]any{"String": map[string]any{"sval": "text"}}}, "typemod": float64(-1)}}}
			}
			return map[string]any{"A_ArrayExpr": map[string]any{"elements": literals}}
		}
	}
	return value
}

func varcharStringLiteral(value any) (any, bool) {
	node, _ := value.(map[string]any)
	cast, _ := node["TypeCast"].(map[string]any)
	if !schemaCastType(cast, "varchar", false) {
		return nil, false
	}
	arg, _ := cast["arg"].(map[string]any)
	constant, _ := arg["A_Const"].(map[string]any)
	_, isString := constant["sval"]
	return arg, isString && len(constant) == 1
}

func schemaCastType(cast map[string]any, name string, array bool) bool {
	typeName, _ := cast["typeName"].(map[string]any)
	if typeName == nil || len(cast) != 2 {
		return false
	}
	for key := range typeName {
		if key != "names" && key != "typemod" && key != "arrayBounds" {
			return false
		}
	}
	if typeName["typemod"] != float64(-1) {
		return false
	}
	names, _ := typeName["names"].([]any)
	parts := make([]string, len(names))
	for i, n := range names {
		m, _ := n.(map[string]any)
		str, _ := m["String"].(map[string]any)
		parts[i], _ = str["sval"].(string)
	}
	qualified := strings.Join(parts, ".")
	if qualified != name && qualified != "pg_catalog."+name {
		return false
	}
	bounds, _ := typeName["arrayBounds"].([]any)
	if !array {
		return len(bounds) == 0
	}
	return reflect.DeepEqual(bounds, []any{map[string]any{"Integer": map[string]any{"ival": float64(-1)}}})
}
