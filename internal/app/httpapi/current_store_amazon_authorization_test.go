package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	zitadelruntime "github.com/henryxu1008/shuomi-ai-commerce/internal/authruntime/zitadel"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	storehttp "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/httpapi"
	"github.com/stretchr/testify/require"
)

func TestAmazonAuthorizationFingerprintUsesVerifiedBearer(t *testing.T) {
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: "actor"})
	require.Equal(t, [32]byte{}, currentAmazonSessionProof(ctx))
	session := zitadelruntime.WithBearerToken(ctx, "synthetic-workbench-bearer")
	fingerprint := currentAmazonSessionProof(session)
	require.NotEqual(t, [32]byte{}, fingerprint)
	require.Equal(t, fingerprint, currentAmazonSessionProof(session))
	require.NotEqual(t, fingerprint, currentAmazonSessionProof(zitadelruntime.WithBearerToken(ctx, "new-bearer")))
}

func TestAmazonAuthorizationStoreRoutesRequireLiveWrite(t *testing.T) {
	for _, path := range storehttp.AmazonAuthorizationPaths() {
		route := httproute.Descriptor{
			Method: http.MethodPost, Path: path, Module: storehttp.ModuleName,
			Permission: authz.PermissionWorkbenchStoreUpdate, AuthPolicy: httproute.AuthPolicyCurrentIdentity,
			OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite,
			RequestTimeout:           35 * time.Second, RejectUnreadRequestBody: true,
			Handler: func(*gin.Context) {},
		}
		if path == storehttp.AmazonAuthorizationStatusPath {
			route.Method, route.Permission, route.RequestTimeout = http.MethodGet, authz.PermissionWorkbenchStoreRead, 5*time.Second
		}
		require.NoError(t, validateCurrentStoreDescriptor(route))
		route.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyContextRead
		require.Error(t, validateCurrentStoreDescriptor(route))
	}
}
