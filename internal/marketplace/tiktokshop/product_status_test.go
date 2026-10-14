package tiktokshop

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestProductStatusCorrelatesOriginalProductAndSeparatesAudit(t *testing.T) {
	expected := ProductStatusExpectation{ProductID: "local-1", LocalSKUIDs: []string{"sku-1"}, GlobalProductID: "global-1", Mappings: []PublishedSKU{{LocalSKUID: "sku-1", GlobalSKUID: "global-sku-1"}}}
	raw := `{"code":0,"data":{"id":"local-1","status":"PENDING","skus":[{"id":"sku-1"}],"audit":{"status":"PRE_APPROVED","pre_approved_reasons":["KYC_PENDING"],"audit_failed_reasons":[{"reason":"private arbitrary message"}]},"global_product_association":{"global_product_id":"global-1","sku_mappings":[{"local_sku_id":"sku-1","global_sku_id":"global-sku-1"}]}}}`
	value, err := DecodeProductStatus([]byte(raw), expected)
	require.NoError(t, err)
	require.Equal(t, "PENDING", value.Status)
	require.Equal(t, "PRE_APPROVED", value.AuditStatus)
	require.Equal(t, []string{"KYC_PENDING"}, value.Reasons)
	for _, change := range [][2]string{{"local-1", "other"}, {"sku-1", "other-sku"}, {"global-1", "other-global"}, {"global-sku-1", "wrong-global-sku"}, {"PENDING", "UNKNOWN"}, {"PRE_APPROVED", "made-up"}} {
		_, err := DecodeProductStatus([]byte(strings.ReplaceAll(raw, change[0], change[1])), expected)
		require.Error(t, err, change[0])
	}
}

func TestProductStatusAcceptsOfficialLifecycleWithoutInferringLive(t *testing.T) {
	for _, status := range []string{"DRAFT", "PENDING", "FAILED", "ACTIVATE", "SELLER_DEACTIVATED", "PLATFORM_DEACTIVATED", "FREEZE", "DELETED", "SCHEDULED"} {
		value, err := DecodeProductStatus([]byte(`{"code":0,"data":{"id":"local","status":"`+status+`","skus":[{"id":"sku"}],"audit":{"status":"NONE","not_submitted_reasons":["SHOP_INACTIVE","future_private_reason"]}}}`), ProductStatusExpectation{ProductID: "local", LocalSKUIDs: []string{"sku"}})
		require.NoError(t, err)
		require.Equal(t, status, value.Status)
		require.Equal(t, []string{"SHOP_INACTIVE"}, value.Reasons)
	}
}
