package target

import (
	"strings"
	"testing"
	"time"

	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/storecenter"
)

func TestTikTokTargetIsAnExplicitVariantBoundToItsCurrentRuleSnapshot(t *testing.T) {
	snapshot := marketplace.RuleSnapshot{Selection: marketplace.RuleSelection{Method: marketplace.GlobalPublishing, CategoryID: "cat"}}
	value := TargetRecord{Merchant: storecenter.ProductMerchantBinding{Platform: storecenter.PlatformTikTokShop, Site: "tiktokshop-us", ApplicationID: "app", ApplicationRevision: "r1", SupplierIdentityHash: strings.Repeat("a", 64), ScopesHash: strings.Repeat("b", 64), CredentialExpiresAt: time.Now().Add(time.Hour), TikTok: storecenter.TikTokMerchantBinding{GrantID: "00000000-0000-4000-8000-000000000002", ShopID: "shop", GrantGeneration: 1, GrantRevision: 1, BindingRevision: 1}}, Input: TargetInput{TikTok: &marketplace.DraftInput{Selection: snapshot.Selection}}, TikTok: &marketplace.Draft{ReadyForUpload: true}, TikTokRuleSnapshot: &snapshot, RulesHash: marketplace.RuleHash(snapshot)}
	if !ValidPlatformRecord(value) || !PublicationReady(value) {
		t.Fatal("qualified TikTok target rejected")
	}
	value.Input.Draft.Product.SourceSystem = "OpenAPI"
	if ValidPlatformRecord(value) {
		t.Fatal("mixed SHEIN/TikTok target accepted")
	}
	value.Input.Draft.Product.SourceSystem = ""
	snapshot.Selection.CategoryID = "another-category"
	if ValidPlatformRecord(value) {
		t.Fatal("drifted rule snapshot accepted")
	}
}
