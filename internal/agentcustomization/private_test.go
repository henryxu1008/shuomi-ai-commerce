package agentcustomization

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestPrivateDeliveryRequiresDeliveredStageAndConfirmedProposal(t *testing.T) {
	for _, r := range []Request{{Stage: Developing}, {Stage: Developing, Proposal: "方案"}, {Stage: Proposed, Proposal: "方案", OfflineConfirmation: "确认"}} {
		u := Update{Stage: r.Stage, Note: "发布", DeliverQualityAgent: true}
		require.Error(t, ApplyUpdate(&r, u, true))
	}
	r := Request{Stage: Developing, Proposal: "方案", OfflineConfirmation: "确认"}
	require.NoError(t, ApplyUpdate(&r, Update{Stage: Delivered, Note: "交付质检智能体", DeliverQualityAgent: true}, true))
	r.Stage = Delivered
	require.NoError(t, ApplyUpdate(&r, Update{Stage: Delivered, Note: "追加使用说明", DeliverQualityAgent: true}, true))
}

func TestQualityDraftAcceptsQualifiedNativeSitesAndRejectsMixedPlatformSites(t *testing.T) {
	id := uuid.NewString()
	hash := strings.Repeat("a", 64)
	v := DraftSnapshot{DraftBinding: DraftBinding{RecordID: id, Revision: 1, SourceID: id, PreparationID: id, StoreID: id, Platform: "temu", Site: "temu-local", ProductKey: "product-a", ProductVersion: "1", Title: "Current native title", RecordHash: hash, ProductHash: hash, RulesHash: hash, InventoryHash: hash, SavedAt: time.Now()}, Issues: []DraftIssue{{Field: "price", Code: "price_missing", Message: "填写实际价格"}}}
	require.True(t, ValidDraft(v))
	v.Site = "temu-supplier"
	require.True(t, ValidDraft(v))
	v.Site = "shein-us"
	require.False(t, ValidDraft(v))
	v.Platform = "shein"
	v.Site = "temu-supplier"
	require.False(t, ValidDraft(v))
}
