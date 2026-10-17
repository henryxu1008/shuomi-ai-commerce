package publishing

import (
	"reflect"
	"slices"
	"testing"
)

func TestValidateSPUStructure(t *testing.T) {
	valid := submissionSpec{parentSpecID: "color", specID: "blue", name: "Blue"}
	tests := []struct {
		name    string
		title   string
		skcs    [][][]submissionSpec
		wantErr string
	}{
		{name: "complete structure", title: "Product", skcs: [][][]submissionSpec{{{valid}}}},
		{name: "name checked before structure", wantErr: "商品名称不能为空"},
		{name: "nil SKCs", title: "Product", wantErr: "SKC列表不能为空"},
		{name: "empty SKCs", title: "Product", skcs: [][][]submissionSpec{}, wantErr: "SKC列表不能为空"},
		{name: "nil SKU list", title: "Product", skcs: [][][]submissionSpec{nil}, wantErr: "SKC[0]的SKU列表不能为空"},
		{name: "empty SKU list", title: "Product", skcs: [][][]submissionSpec{{}}, wantErr: "SKC[0]的SKU列表不能为空"},
		{name: "nil specifications", title: "Product", skcs: [][][]submissionSpec{{nil}}, wantErr: "SKC[0]的SKU[0]的规格列表不能为空，TEMU要求每个SKU必须有规格"},
		{name: "empty specifications", title: "Product", skcs: [][][]submissionSpec{{{}}}, wantErr: "SKC[0]的SKU[0]的规格列表不能为空，TEMU要求每个SKU必须有规格"},
		{name: "spec ID checked before parent and name", title: "Product", skcs: [][][]submissionSpec{{{{}}}}, wantErr: "SKC[0]的SKU[0]的规格[0]的SpecID不能为空"},
		{name: "parent checked before name", title: "Product", skcs: [][][]submissionSpec{{{{specID: "blue"}}}}, wantErr: "SKC[0]的SKU[0]的规格[0]的ParentSpecID不能为空"},
		{name: "spec name required", title: "Product", skcs: [][][]submissionSpec{{{{parentSpecID: "color", specID: "blue"}}}}, wantErr: "SKC[0]的SKU[0]的规格[0]的SpecName不能为空"},
		{name: "later empty SKC reports its index", title: "Product", skcs: [][][]submissionSpec{{{valid}}, nil}, wantErr: "SKC[1]的SKU列表不能为空"},
		{name: "later empty SKU reports both indices", title: "Product", skcs: [][][]submissionSpec{{{valid}}, {{valid}, nil}}, wantErr: "SKC[1]的SKU[1]的规格列表不能为空，TEMU要求每个SKU必须有规格"},
		{name: "later spec reports every index", title: "Product", skcs: [][][]submissionSpec{{{valid}}, {{valid}, {valid, {parentSpecID: "color", specID: "blue"}}}}, wantErr: "SKC[1]的SKU[1]的规格[1]的SpecName不能为空"},
		{name: "first spec error precedes later empty SKC", title: "Product", skcs: [][][]submissionSpec{{{valid, {}}}, nil}, wantErr: "SKC[0]的SKU[0]的规格[1]的SpecID不能为空"},
		{name: "whitespace values keep exact nonempty semantics", title: " ", skcs: [][][]submissionSpec{{{{parentSpecID: " ", specID: " ", name: " "}}}}},
		{name: "nonempty zero strings remain valid", title: "0", skcs: [][][]submissionSpec{{{{parentSpecID: "0", specID: "0", name: "0"}}}}},
		{name: "temporary IDs are handled by the separate spec owner", title: "Product", skcs: [][][]submissionSpec{{{{parentSpecID: "TEMP_color", specID: "TEMP_blue", name: "Blue"}}}}},
		{name: "all SKU entries are checked", title: "Product", skcs: [][][]submissionSpec{{{valid}, {valid}, {{parentSpecID: "color", specID: "blue"}}}}, wantErr: "SKC[0]的SKU[2]的规格[0]的SpecName不能为空"},
		{name: "multiple SKCs and SKUs", title: "Product", skcs: [][][]submissionSpec{{{valid}, {valid, valid}}, {{valid}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := slices.Clone(tt.skcs)
			for i, skc := range tt.skcs {
				before[i] = slices.Clone(skc)
				for j, sku := range skc {
					before[i][j] = slices.Clone(sku)
				}
			}
			err := validateSPUStructure(tt.title, tt.skcs)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.skcs, before) {
				t.Fatal("validation modified input specifications")
			}
		})
	}
}
