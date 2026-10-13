package temu

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"task-processor/internal/storecenter"
	"testing"
)

func TestSupplierSizeClassReadsPAAndDistinguishesExplicitEmptyFromMissing(t *testing.T) {
	for _, test := range []struct {
		name, body      string
		valid, required bool
	}{
		{"explicit-null", `{"success":true,"result":null}`, true, false},
		{"explicit-empty-mapping", `{"success":true,"result":{"sizeSpecClassCat":null}}`, true, false},
		{"missing", `{"success":true,"result":{}}`, false, false},
		{"required", `{"success":true,"result":{"sizeSpecClassCat":{"catId":34,"classId":7,"parentClassId":0,"relatedClassIds":[],"classType":0}}}`, true, true},
		{"wrong-category", `{"success":true,"result":{"sizeSpecClassCat":{"catId":99,"classId":7,"classType":0}}}`, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := identityClient(t, storecenter.PartitionCN, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				fields := map[string]json.RawMessage{}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&fields))
				var api string
				require.NoError(t, json.Unmarshal(fields["type"], &api))
				switch api {
				case supplierInfoAPI:
					return jsonResponse(`{"success":true,"result":{"uniqueId":"pa-mall-1","expiredTime":1900000000,"apiScopeList":["bg.glo.goods.sizecharts.class.get"]}}`), nil
				case supplierModeAPI:
					return jsonResponse(`{"success":true,"result":{"semiManagedMall":false}}`), nil
				case supplierSizeClassAPI:
					require.Equal(t, supplierAPIOrigin+"/openapi/router", r.URL.String())
					require.JSONEq(t, `34`, string(fields["catId"]))
					require.NotContains(t, fields, "classId")
					return jsonResponse(test.body), nil
				default:
					t.Fatalf("unexpected %s", api)
					return nil, nil
				}
			}))
			result, err := c.QuerySupplierSizeClass(context.Background(), tokenCredential(), 34)
			if !test.valid {
				require.Error(t, err)
				require.False(t, result.Observed)
				return
			}
			require.NoError(t, err)
			require.True(t, result.Observed)
			require.Equal(t, test.required, result.Class != nil)
		})
	}
}
