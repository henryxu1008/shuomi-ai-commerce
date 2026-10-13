package httpapi

import (
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
)

func TestCatalogObservationEligibilityKeepsTemuProductsAndOrdersSeparate(t *testing.T) {
	for _, tc := range []struct {
		platform storecenter.Platform
		status   storecenter.RecordStatus
		kind     o.Kind
		want     bool
	}{
		{storecenter.PlatformTemu, storecenter.RecordStatusActive, o.Products, true},
		{storecenter.PlatformTemu, storecenter.RecordStatusActive, o.Orders, false},
		{storecenter.PlatformTemu, storecenter.RecordStatusDisabled, o.Products, false},
		{storecenter.PlatformTemu, storecenter.RecordStatusDeleting, o.Products, false},
		{storecenter.PlatformShein, storecenter.RecordStatusActive, o.Products, true},
		{storecenter.PlatformShein, storecenter.RecordStatusActive, o.Orders, true},
		{"unsupported", storecenter.RecordStatusActive, o.Products, false},
		{storecenter.PlatformTemu, storecenter.RecordStatusActive, "unsupported", false},
	} {
		require.Equal(t, tc.want, catalogObservationsEligible(tc.platform, tc.status, tc.kind), "%s/%s/%s", tc.platform, tc.status, tc.kind)
	}
}

func TestCatalogSyncHealthUsesOnlySupportedKinds(t *testing.T) {
	complete := o.StoreSummary{Complete: true, SyncStatus: "completed"}
	require.True(t, catalogSyncHealthy(complete, o.StoreSummary{}, false), "TEMU complete products must not require an unsupported order summary")
	require.False(t, catalogSyncHealthy(complete, o.StoreSummary{}, true), "SHEIN still requires orders")
	require.True(t, catalogSyncHealthy(complete, complete, true))
	for _, status := range []string{"pending", "running", "failed", "partial", "suspended"} {
		require.False(t, catalogSyncHealthy(o.StoreSummary{Complete: true, SyncStatus: status}, o.StoreSummary{}, false))
	}
	require.False(t, catalogSyncHealthy(o.StoreSummary{SyncStatus: "completed"}, o.StoreSummary{}, false))
	require.False(t, catalogSyncHealthy(o.StoreSummary{}, o.StoreSummary{}, false))
}

func TestCatalogTemuModeUsesVerifiedMerchantAndCompoundApplication(t *testing.T) {
	choices := []storecenter.OfficialApplicationChoice{
		{Platform: storecenter.PlatformShein, AppID: "same-id", Revision: "same-revision", AuthorizationMethod: storecenter.AuthorizationCallback, Type: storecenter.ApplicationSelfOperated},
		{Platform: storecenter.PlatformTemu, IdentityPartition: storecenter.PartitionCN, AppID: "same-id", Revision: "same-revision", AuthorizationMethod: storecenter.AuthorizationManualToken, SupportedModes: storecenter.ModesFullyManaged | storecenter.ModesSemiManaged},
		{Platform: storecenter.PlatformTemu, IdentityPartition: storecenter.PartitionUS, AppID: "same-id", Revision: "same-revision", AuthorizationMethod: storecenter.AuthorizationManualToken, SupportedModes: storecenter.ModesLocal},
	}
	view := storecenter.OfficialConnectionView{Platform: storecenter.PlatformTemu, IdentityPartition: storecenter.PartitionCN, AppID: "same-id", AppRevision: "same-revision", MerchantIdentityHash: "verified-merchant", Mode: storecenter.MerchantFullyManaged}
	require.Equal(t, storecenter.ApplicationFullyManaged, catalogApplicationType(view, choices))
	view.Mode = storecenter.MerchantSemiManaged
	require.Equal(t, storecenter.ApplicationSemiManaged, catalogApplicationType(view, choices))
	view.Mode = ""
	require.Empty(t, catalogApplicationType(view, choices), "dual-mode application does not prove merchant mode")
	view.Mode = storecenter.MerchantFullyManaged
	view.MerchantIdentityHash = ""
	require.Empty(t, catalogApplicationType(view, choices), "a configured application is not a verified merchant")
	view.MerchantIdentityHash = "verified-merchant"
	view.IdentityPartition = storecenter.PartitionUS
	require.Empty(t, catalogApplicationType(view, choices), "regional authorization cannot be projected as a supplier")
	view.Mode = storecenter.MerchantLocal
	require.Equal(t, storecenter.ApplicationSelfOperated, catalogApplicationType(view, choices))
	view.IdentityPartition = storecenter.PartitionEU
	require.Empty(t, catalogApplicationType(view, choices), "another partition's app with the same raw ID cannot match")
	view.IdentityPartition = storecenter.PartitionUS
	view.AppRevision = "stale-revision"
	require.Empty(t, catalogApplicationType(view, choices))
	require.Equal(t, storecenter.ApplicationSelfOperated, catalogApplicationType(storecenter.OfficialConnectionView{Platform: storecenter.PlatformShein, AppID: "same-id", AppRevision: "same-revision"}, choices))
}
