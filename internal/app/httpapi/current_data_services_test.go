package httpapi

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/core/config"
	datahttp "github.com/henryxu1008/shuomi-ai-commerce/internal/dataservice/httpapi"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestDataServicesDisabledAdmissionNeedsOnlyCurrentReadOwners(t *testing.T) {
	cfg := &config.Config{}
	cfg.ListingKit.Zitadel.TenantDirectoryToken = "fixture-only"
	opts := currentApplicationOptions{dataServicesConfigs: 1, dataServices: &DataServicesDependencies{ProductDB: &gorm.DB{}}, productAcquisitionDB: &gorm.DB{}, productCollections: 1, commercialOwnerDB: &gorm.DB{}}
	require.NoError(t, validateDataServicesDependencies(opts, &gorm.DB{}, cfg))
}

func TestDataServicesRejectsMissingFormalOrganizationAuthority(t *testing.T) {
	a, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	_, _, err = buildDataServices(context.Background(), currentApplicationOptions{dataServices: &DataServicesDependencies{}}, routeAuthDependencies{}, a, &config.Config{}, storecenter.RuntimeCapabilities{})
	require.ErrorContains(t, err, "formal organization authority")
}

func TestDataServicesNativeAdmissionPreservesAllThreeAuthBoundaries(t *testing.T) {
	var base []httproute.Descriptor
	for _, r := range currentWorkbenchApplicationRoutes {
		base = append(base, httproute.Descriptor{Method: r.Method, Path: r.Path})
	}
	routes := datahttp.BuildRoutes(datahttp.Dependencies{})
	all := append(append([]httproute.Descriptor{}, base...), routes...)
	validate := func(r []httproute.Descriptor, on bool) error {
		return validateCurrentApplicationRoutesInternal(r, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{DataServices: on})
	}
	require.NoError(t, validate(all, true))
	require.Error(t, validate(all, false))
	require.Error(t, validate(base, true))
	for i := range routes {
		for _, mutate := range []func(*httproute.Descriptor){
			func(r *httproute.Descriptor) { r.AuthPolicy = httproute.AuthPolicyCurrentIdentity },
			func(r *httproute.Descriptor) {
				r.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyContextRead
			},
			func(r *httproute.Descriptor) { r.Permission = "workbench.chat.read" },
			func(r *httproute.Descriptor) { r.Module = "ai-workbench" },
			func(r *httproute.Descriptor) { r.RequestTimeout = time.Minute },
			func(r *httproute.Descriptor) { r.RejectUnreadRequestBody = !r.RejectUnreadRequestBody },
		} {
			changed := append([]httproute.Descriptor{}, all...)
			mutate(&changed[len(base)+i])
			require.Error(t, validate(changed, true), routes[i].Method+" "+routes[i].Path)
		}
	}
}
