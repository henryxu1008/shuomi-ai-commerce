package httpapi

import (
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/httproute"
	reporthttp "task-processor/internal/reportcenter/httpapi"
	"testing"
)

func TestOfflineRecordNativeAdmissionAndPrivateOwners(t *testing.T) {
	source, products, assets, stores, reports := &gorm.DB{}, &gorm.DB{}, &gorm.DB{}, &gorm.DB{}, &gorm.DB{}
	var o currentApplicationOptions
	WithReportCenter(reports)(&o)
	WithSheinOfflineRecords(assets)(&o)
	require.Error(t, validateReportCenterPool(o, source))
	o.productAcquisitionDB = products
	o.productCollections = 1
	o.storeCenterDB = stores
	o.storeCenters = 1
	require.NoError(t, validateReportCenterPool(o, source))
	saved := o
	o.offlineRecordAssets = products
	require.Error(t, validateReportCenterPool(o, source))
	o = saved
	o.localTrials = 1
	require.Error(t, validateReportCenterPool(o, source))
	o = saved
	WithSheinOfflineRecords(assets)(&o)
	require.Error(t, validateReportCenterPool(o, source))
	var routes []httproute.Descriptor
	for _, r := range currentWorkbenchApplicationRoutes {
		routes = append(routes, httproute.Descriptor{Method: r.Method, Path: r.Path})
	}
	routes = append(routes, reporthttp.Routes(nil)...)
	normal := offlineRecordRoutes(nil)
	routes = append(routes, normal...)
	validate := func(value []httproute.Descriptor, on bool) error {
		return validateCurrentApplicationRoutesInternal(value, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{ReportCenter: true, OfflineRecords: on})
	}
	require.NoError(t, validate(routes, true))
	require.Error(t, validate(routes, false))
	for _, r := range normal {
		for _, mutate := range []func(*httproute.Descriptor){func(d *httproute.Descriptor) { d.AuthPolicy = httproute.AuthPolicyPublic }, func(d *httproute.Descriptor) { d.Permission = "workbench.report.read" }, func(d *httproute.Descriptor) {
			d.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
		}, func(d *httproute.Descriptor) { d.RequestTimeout++ }} {
			changed := r
			mutate(&changed)
			require.Error(t, validateOfflineRecordDescriptor(changed))
		}
	}
}

func TestOfflineWireVersionIsExactAndCanonical(t *testing.T) {
	dto := offlineInputDTO{ProductKey: "p", SnapshotVersion: "9223372036854775807", StoreID: "11111111-1111-4111-8111-111111111111", Country: "US", Language: "en", Action: "save_draft"}
	input, err := dto.domain()
	require.NoError(t, err)
	require.Equal(t, dto, offlineInput(input))
	for _, value := range []string{"0", "01", "-1", "9223372036854775808", "not-a-version"} {
		dto.SnapshotVersion = value
		_, err := dto.domain()
		require.Error(t, err)
	}
}

func TestOfflineRecordCapabilityIsExplicitForEveryProductConsumer(t *testing.T) {
	cap, _, err := collectionAcquisitionOptions([]bool{true, false, false, false, false, false, false, true})
	require.NoError(t, err)
	require.True(t, cap.OfflineRecords)
	require.True(t, cap.Collections)
	require.False(t, cap.SupplyChain)
	_, _, err = collectionAcquisitionOptions([]bool{false, false, false, false, false, false, false, true})
	require.Error(t, err)
	_, _, err = collectionAcquisitionOptions(make([]bool, 9))
	require.Error(t, err)
}
