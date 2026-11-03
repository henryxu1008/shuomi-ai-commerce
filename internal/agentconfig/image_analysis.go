package agentconfig

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"task-processor/internal/agent"
	"task-processor/internal/aicapability"
)

const (
	ImageAnalysisAgentID       = "product.image.analysis"
	ImageAnalysisVersion       = "v1.0.0"
	ImageAnalysisSchema        = "image-analysis-config-v1"
	ImageAnalysisPolicyVersion = "product-image-analysis-policy-v1"
	ImageAnalysisPromptVersion = "product-image-analysis-v1"
	ImageAnalysisResultVersion = "image-analysis-result-v1"
)

// ImageAnalysisConfiguration is an immutable preparation intent in the original
// configuration snapshot, not a Product/Asset fact or a generation permission.
// MaximumMicropoints is the confirmed ceiling, never a second charge/reserve.
type ImageAnalysisConfiguration struct {
	Schema, ExecutionID, ExecutionVersion string
	Kind, ImageID, InventoryDigest        string
	ApplyReceiptID                        string
	Image                                 aicapability.SingleImageInput
	MaximumMicropoints                    int64 `json:"maximumMicropoints,string"`
	QuoteDigest                           string
}

func ImageAnalysisBinding(b agent.Binding) bool {
	return b.Valid() && b.TargetPlatform == "product" &&
		(b.ContextKind == "acquisition" || b.ContextKind == "own-product" || b.ContextKind == "collection")
}

func imageAnalysisHash(v any) string {
	raw, _ := json.Marshal(v)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}

func FreezeImageAnalysis(c StartCommand, kind, imageID, inventoryDigest, applyReceiptID string, image aicapability.SingleImageInput) (ImageAnalysisConfiguration, error) {
	invalid := func() (ImageAnalysisConfiguration, error) { return ImageAnalysisConfiguration{}, ErrInvalid }
	if !agent.ValidID(c.Scope.OrganizationID) || !agent.ValidID(c.Scope.ActorID) || !agent.ValidID(c.MemberID) || !UUID(c.Request.Key) ||
		c.AgentID != ImageAgentID || c.AgentVersion != ImageAgentVersion || c.Template != nil || c.KnowledgeBaseID != "" ||
		!ImageAnalysisBinding(c.Request.Binding) || c.Request.PolicyVersion != ImageAnalysisPolicyVersion || c.Request.PromptVersion != ImageAnalysisPromptVersion ||
		!c.Request.ContextSnapshotRef.Absent() || !c.Request.ConfigurationSnapshotRef.Absent() || !c.Request.ModeSnapshotRef.Absent() && !c.Request.ModeSnapshotRef.Valid() || !agent.ValidGoalSummary(c.Request.GoalSummary) || c.Request.ManualReviewOnly ||
		!c.Request.ContentSelection.Absent() || !c.Request.SkillVersionRef.Absent() || !c.Request.ValidContentSelection() ||
		!aicapability.SingleImageProfile(c.ExecutionModelProfile) || c.ExecutionModelProfile.PromptVersion != ImageAnalysisPromptVersion || c.ExecutionModelProfile.OutputSchemaVersion != ImageAnalysisResultVersion ||
		!c.Request.Limits.Valid() || c.Request.Limits.ModelCalls != 1 || c.Request.Limits.Currency != c.ExecutionModelProfile.Currency ||
		c.Request.Limits.Tokens < c.ExecutionModelProfile.MaximumPromptTokens+c.ExecutionModelProfile.MaximumCompletionTokens ||
		!agent.ValidID(imageID) || applyReceiptID != "" && !agent.ValidID(applyReceiptID) {
		return invalid()
	}
	if _, err := ParseModeRevision(c.ExpectedModeRevision); err != nil {
		return invalid()
	}
	steps := 3 // canonical pre-read, model, fixed validator
	if kind == "source" {
		n, err := strconv.Atoi(strings.TrimPrefix(imageID, "catalog-image-"))
		if !strings.HasPrefix(imageID, "catalog-image-") || err != nil || n < 1 || "catalog-image-"+strconv.Itoa(n) != imageID || inventoryDigest != "" {
			return invalid()
		}
	} else if kind == "approved" {
		if !ImageDigest(inventoryDigest) {
			return invalid()
		}
		steps++
	} else {
		return invalid()
	}
	maximumCost, err := c.ExecutionModelProfile.MaximumCost()
	if err != nil || maximumCost > c.Request.Limits.CostMicros || c.Request.Limits.Steps < steps ||
		c.ExecutionModelProfile.MaximumPromptTokens <= aicapability.SingleImagePromptTokenBound+1024 {
		return invalid()
	}
	image.SelectionDigest = imageAnalysisHash(struct {
		Binding                                agent.Binding
		Kind, ImageID, Inventory, ApplyReceipt string
	}{c.Request.Binding, kind, imageID, inventoryDigest, applyReceiptID})
	if !image.Valid() {
		return invalid()
	}
	// Before a Run exists, freeze the already-admitted profile ceiling. The real
	// Run owner generates its ID later; its exact invocation quote must fit this
	// ceiling and the same frozen profile. No placeholder Run or provider fact.
	maximum, err := c.ExecutionModelProfile.OfficialTariff.ReserveTextMicropoints(c.ExecutionModelProfile.MaximumPromptTokens, c.ExecutionModelProfile.MaximumCompletionTokens)
	if err != nil {
		return invalid()
	}
	a := ImageAnalysisConfiguration{Schema: ImageAnalysisSchema, ExecutionID: ImageAnalysisAgentID, ExecutionVersion: ImageAnalysisVersion, Kind: kind, ImageID: imageID, InventoryDigest: inventoryDigest, ApplyReceiptID: applyReceiptID, Image: image, MaximumMicropoints: maximum}
	profileDigest, _ := c.ExecutionModelProfile.Digest()
	a.QuoteDigest = imageAnalysisHash(struct {
		Scope                     agent.Scope
		Member, Key, ModeRevision string
		Binding                   agent.Binding
		Profile                   string
		Analysis                  ImageAnalysisConfiguration
	}{c.Scope, c.MemberID, c.Request.Key, c.ExpectedModeRevision, c.Request.Binding, profileDigest, a})
	return a, nil
}

func (a ImageAnalysisConfiguration) Valid(c StartCommand) bool {
	expected, err := FreezeImageAnalysis(c, a.Kind, a.ImageID, a.InventoryDigest, a.ApplyReceiptID, a.Image)
	return err == nil && a == expected
}
