package openai

import (
	"context"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/aicapability"
	productimage "task-processor/internal/product/image"
	"testing"
)

func TestFlareSourceQuoteUsesReferenceMoneyWithoutClaimingVendorBound(t *testing.T) {
	cfg := validSourceImageAdapterConfig(&productImageGeneratorStub{})
	cfg.ImageCostMicrosPerOutput, cfg.CostUpperBoundKnown = 0, false
	cfg.Provider = "openai"
	cfg.ImageModel = aicapability.FlareImageModel
	cfg.PricingVersion = "reference-v1"
	cfg.ReferenceEstimate = SourceImageReferenceQuote{PriceVersion: cfg.PricingVersion, ProfileDigest: strings.Repeat("a", 64), MaximumCostMicros: 3}
	a, err := NewSourceImageAdapter(cfg)
	require.NoError(t, err)
	q, err := a.QuoteUsage(context.Background(), productimage.UsageQuoteRequest{Operation: productimage.SourceWhiteBackgroundOperation, InputFingerprint: "synthetic-input", MaximumOutputs: 1})
	require.NoError(t, err)
	require.EqualValues(t, 3, q.MaximumCostMicros)
	require.False(t, q.CostUpperBoundKnown)
	require.NoError(t, func() error { _, err := productimage.NormalizeUsageAuthorization(q, q.Operation); return err }())
}
