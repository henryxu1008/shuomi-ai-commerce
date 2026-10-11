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

func TestLocalAttributesBindShippingTemplateAndPreserveConditionalRules(t *testing.T) {
	calls := 0
	c := identityClient(t, storecenter.PartitionEU, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "openapi-b-eu.temu.com", r.URL.Host)
		var body map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		if calls == 1 {
			return jsonResponse(`{"success":true,"result":{"mallId":123,"mallType":100,"regionId":20,"expiredTime":1900000000,"apiScopeList":["temu.local.product.attributes.get"]}}`), nil
		}
		require.JSONEq(t, `"temu.local.product.attributes.get"`, string(body["type"]))
		require.Equal(t, "9007199254740993", string(body["catId"]))
		require.JSONEq(t, `"template-37"`, string(body["costTemplateId"]))
		require.JSONEq(t, `"en"`, string(body["language"]))
		require.NotContains(t, body, "request")
		require.NotContains(t, body, "siteId")
		return jsonResponse(`{"success":true,"result":{"catId":9007199254740993,"language":"en","attributeList":[{"refPid":11,"attributeName":"Material","required":false,"attributeRules":{"controlType":16,"chooseMaxNum":4,"transnationalAttribute":true,"attributeValueRules":{"valuePrecision":0,"minValue":"0","maxValue":"100","valueRule":1}},"attributeValueDetail":[{"groupId":"g1","attributeValueList":[{"vid":22,"value":"Cotton","childAttribute":[{"childAttributeId":33,"childAttributeValueGroupId":"g2"}]}]}],"attributeValueUnitList":[{"valueUnitName":"Percentage","valueUnitId":44}]},{"refPid":12,"attributeName":"Unknown requirement"}]}}`), nil
	}))
	result, err := c.QueryLocalAttributes(context.Background(), tokenCredential(), model.LocalAttributeQuery{CatID: 9007199254740993, Language: "en", CostTemplateID: "template-37"})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, int64(9007199254740993), *result.CatID)
	require.False(t, *result.Attributes[0].Required)
	require.True(t, *result.Attributes[0].Rules.Transnational)
	require.Equal(t, int64(0), *result.Attributes[0].Rules.Values.Precision)
	require.Equal(t, int64(33), *result.Attributes[0].ValueGroups[0].Values[0].ChildAttributes[0].RefPID)
	require.Equal(t, "g2", result.Attributes[0].ValueGroups[0].Values[0].ChildAttributes[0].ValueGroupID)
	require.Nil(t, result.Attributes[1].Required)
	require.Nil(t, result.Attributes[1].Rules)
}

func TestLocalAttributeResponseMustMatchRequestedCategoryAndLanguage(t *testing.T) {
	for _, result := range []string{
		`{"catId":34,"language":"en"}`,
		`{"catId":34,"language":"en","attributeList":null}`,
		`{"catId":35,"language":"en","attributeList":[]}`,
		`{"catId":34,"language":"fr","attributeList":[]}`,
		`{"catId":34,"language":"en","attributeList":[{"refPid":11,"attributeName":"A"},{"refPid":11,"attributeName":"B"}]}`,
	} {
		t.Run(result, func(t *testing.T) {
			calls := 0
			c := identityClient(t, storecenter.PartitionGlobal, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return jsonResponse(`{"success":true,"result":{"mallId":123,"mallType":100,"regionId":37,"expiredTime":1900000000,"apiScopeList":["temu.local.product.attributes.get"]}}`), nil
				}
				return jsonResponse(`{"success":true,"result":` + result + `}`), nil
			}))
			_, err := c.QueryLocalAttributes(context.Background(), tokenCredential(), model.LocalAttributeQuery{CatID: 34, Language: "en"})
			require.ErrorIs(t, err, storecenter.ErrOfficialConnectionUnavailable)
		})
	}
}

func TestLocalAttributesCannotUseOtherScopeOrInvalidInputs(t *testing.T) {
	for _, query := range []model.LocalAttributeQuery{{CatID: 0}, {CatID: 34, CostTemplateID: "template\n"}, {CatID: 34, Language: " en"}} {
		calls := 0
		c := identityClient(t, storecenter.PartitionGlobal, roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return jsonResponse(`{}`), nil }))
		_, err := c.QueryLocalAttributes(context.Background(), tokenCredential(), query)
		require.ErrorIs(t, err, storecenter.ErrOfficialConnectionUnavailable)
		require.Zero(t, calls)
	}
	calls := 0
	c := identityClient(t, storecenter.PartitionGlobal, roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return jsonResponse(`{"success":true,"result":{"mallId":123,"mallType":100,"regionId":37,"expiredTime":1900000000,"apiScopeList":["temu.local.goods.cats.rules.get"]}}`), nil
	}))
	_, err := c.QueryLocalAttributes(context.Background(), tokenCredential(), model.LocalAttributeQuery{CatID: 34})
	require.ErrorIs(t, err, storecenter.ErrOfficialAuthorizationRejected)
	require.Equal(t, 1, calls, "field rules scope cannot grant attribute metadata")
}
