package temu

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/storecenter"
)

func TestLocalCategoryRuleReadRetainsOfficialFieldSpellingRequirementsAndWarnings(t *testing.T) {
	calls := 0
	c := identityClient(t, storecenter.PartitionEU, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "openapi-b-eu.temu.com", r.URL.Host)
		var body map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		if calls == 1 {
			return jsonResponse(`{"success":true,"result":{"mallId":123,"mallType":100,"regionId":20,"expiredTime":1900000000,"apiScopeList":["temu.local.goods.cats.rules.get"]}}`), nil
		}
		require.JSONEq(t, `"temu.local.goods.cats.rules.get"`, string(body["type"]))
		require.JSONEq(t, `34`, string(body["catId"]))
		require.JSONEq(t, `"en"`, string(body["language"]))
		require.NotContains(t, body, "siteId")
		return jsonResponse(`{"success":true,"result":{"fieldRules":[{"filedName":"goodsBasic.goodsName","requirementType":1},{"filedName":"bookInfo","requirementType":3},{"filedName":"newOfficialField","requirementType":9},{"filedName":"missingRequirement"}],"expandCatType":2,"warnings":[{"message":"Bookstore qualification required"}],"secondHand":[{"businessScope":2,"condition":[{"type":2,"agency":[{"insName":"Example Agency","grade":["A"]}]}]}]}}`), nil
	}))
	rules, err := c.QueryLocalCategoryRules(context.Background(), tokenCredential(), model.CategoryRuleQuery{CatID: 34, Language: "en"})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, "goodsBasic.goodsName", rules.FieldRules[0].FieldName)
	require.Equal(t, int64(1), *rules.FieldRules[0].RequirementType)
	require.Equal(t, int64(3), *rules.FieldRules[1].RequirementType)
	require.Equal(t, int64(9), *rules.FieldRules[2].RequirementType, "new constraints are preserved for a fail-closed consumer")
	require.Nil(t, rules.FieldRules[3].RequirementType)
	require.Equal(t, "Bookstore qualification required", rules.Warnings[0].Message)
	require.Equal(t, "Example Agency", rules.SecondHand[0].Conditions[0].Agencies[0].Name)
}

func TestSupplierAttributeTemplateRetainsParentConditionsAndZeroPrecision(t *testing.T) {
	calls := 0
	c := identityClient(t, storecenter.PartitionCN, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "openapi.kuajingmaihuo.com", r.URL.Host)
		var body map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		if calls == 1 {
			return jsonResponse(`{"success":true,"result":{"mallId":123,"expiredTime":1900000000,"apiScopeList":["bg.goods.attrs.get"]}}`), nil
		}
		require.JSONEq(t, `"bg.goods.attrs.get"`, string(body["type"]))
		require.JSONEq(t, `34`, string(body["catId"]))
		require.NotContains(t, body, "supportedType", "never override the authorized mall's channel")
		return jsonResponse(`{"success":true,"result":{"inputMaxSpecNum":0,"chooseAllQualifySpec":false,"singleSpecValueNum":4,"properties":[{"refPid":11,"templatePid":12,"name":"Material percentage","required":true,"isSale":false,"valuePrecision":0,"inputMaxNum":1,"controlType":16,"minValue":"0","maxValue":"100","valueRule":1,"showCondition":[{"parentRefPid":21,"parentVids":[31]}],"templatePropertyValueParentList":[{"vidList":[41],"parentVidList":[31]}],"values":[{"vid":41,"value":"Cotton","lang2Value":{"en":"Cotton"}}]},{"refPid":13,"templatePid":14,"name":"Unknown required flag"}]}}`), nil
	}))
	rules, err := c.QuerySupplierAttributes(context.Background(), tokenCredential(), int64(34))
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, int64(0), *rules.InputMaxSpecNum)
	require.Equal(t, int64(0), *rules.Properties[0].ValuePrecision)
	require.Equal(t, []int64{31}, rules.Properties[0].ShowConditions[0].ParentVIDs)
	require.Equal(t, []int64{41}, rules.Properties[0].ValueParents[0].VIDs)
	require.Nil(t, rules.Properties[1].Required, "missing is not optional")
}

func TestRuleTemplatesMissingAndDuplicateFieldsAreNotEmptyRules(t *testing.T) {
	for _, body := range []string{`{}`, `{"fieldRules":null}`, `{"fieldRules":[{"filedName":"a","requirementType":1},{"filedName":"a","requirementType":2}]}`} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			c := identityClient(t, storecenter.PartitionGlobal, roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return jsonResponse(`{"success":true,"result":{"mallId":123,"mallType":100,"regionId":211,"expiredTime":1900000000,"apiScopeList":["temu.local.goods.cats.rules.get"]}}`), nil
				}
				return jsonResponse(`{"success":true,"result":` + body + `}`), nil
			}))
			_, err := c.QueryLocalCategoryRules(context.Background(), tokenCredential(), model.CategoryRuleQuery{CatID: 34})
			require.ErrorIs(t, err, storecenter.ErrOfficialConnectionUnavailable)
		})
	}
}
