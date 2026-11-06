package routeconfiguration

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"testing"
)

func TestFlareSharedImageAdmissionUsesOnlyFrozenReferenceAndOriginalCredential(t *testing.T) {
	c, m, ref := sharedCatalogImageFixture(t)
	now := c.now()
	price := m.prices.Prices[0]
	price.Version = ""
	price.Provider = "openai"
	price.Region = "global"
	price.Model = aicapability.FlareImageModel
	price.Channel = "openai-direct"
	price.UsageVersion = aicapability.ReferenceImagesTotalsV1
	price.SourceURL = "https://developers.openai.com/api/docs/"
	price.Rates = aicapability.OfficialModelRates{InputText: 10000, InputImage: 20000, OutputImage: 10000}
	price.Image.ReferenceEstimate = aicapability.ReferenceImageEstimate{InputTokens: 100, OutputTokens: 100, OperationScope: aicapability.ReferenceImageOperationScope, Size: "1024x1024", Quality: "auto", AllowAutoSize: true, MaximumPromptBytes: 65536, MaximumReferenceImages: 3, MaximumReferenceBytes: 16777216, MaximumReferencePixels: 64000000}
	var err error
	m.prices, err = d.FreezePlatformPrices(d.PlatformPricesInput{Prices: []d.PublishedModelPrice{price}}, "reference-image-1", nil, now)
	require.NoError(t, err)
	m.credential.EndpointDigest = governed.EndpointIdentityDigest("https://api.openai.com/v1")
	def := Definition{Manufacturer: "openai", Region: "global", ServiceType: "image-sync", Scope: d.Scope{OrganizationID: "org", Capability: d.ImageCapability}, Channel: "openai-direct", PriceID: price.ID, ReferenceVersion: ref.Version, ConnectionID: "fixture-connection", ConnectionProvider: "fixture-provider", Image: &ImageAdmission{DeadlineSeconds: 30, UnitVersion: aicapability.AIPointMicrounitV1, MaximumMicropoints: 300, ClientName: ref.ID, Provider: "openai", Model: aicapability.FlareImageModel, Endpoint: "https://api.openai.com/v1", CredentialVersion: m.credential.Reference}}
	c.Catalog, err = NewCatalog([]Definition{def})
	require.NoError(t, err)
	bindTestChannel(c.Catalog, &m.credential)
	q, err := c.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.True(t, q.Ready, q.Reason)
	require.EqualValues(t, 300, q.Image.Points)
	require.Empty(t, q.BoundEvidenceVersion)
	require.Equal(t, price.Image.ReferenceEstimate, q.Image.Billing.ReferenceEstimate)
	oldVersion := q.ProfileVersion
	price.Image.ReferenceEstimate.InputTokens++
	m.prices, err = d.FreezePlatformPrices(d.PlatformPricesInput{Prices: []d.PublishedModelPrice{price}}, "reference-image-2", nil, now)
	require.NoError(t, err)
	q, err = c.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.False(t, q.Ready, "a new estimate cannot bypass the original point ceiling")
	def.Image.MaximumMicropoints = 1000
	c.Catalog, err = NewCatalog([]Definition{def})
	require.NoError(t, err)
	q, err = c.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.True(t, q.Ready, q.Reason)
	require.NotEqual(t, oldVersion, q.ProfileVersion)
	m.credential.KeyPresent = false
	q, err = c.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.False(t, q.Ready)
}
