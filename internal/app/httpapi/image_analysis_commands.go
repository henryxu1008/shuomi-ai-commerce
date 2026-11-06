package httpapi

import (
	"context"
	"errors"
	"strconv"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/authz"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
)

type imageAnalysisPrepareBody struct {
	Kind                 string `json:"kind"`
	ImageID              string `json:"imageId"`
	CatalogVersion       string `json:"catalogVersion"`
	PublicationID        string `json:"publicationId"`
	InventoryDigest      string `json:"inventoryDigest,omitempty"`
	ExpectedModeRevision string `json:"expectedModeRevision"`
}
type imageAnalysisConfirmBody struct {
	PreparationID       string `json:"preparationId"`
	ConfigurationDigest string `json:"configurationDigest"`
	QuoteDigest         string `json:"quoteDigest"`
	SelectionDigest     string `json:"selectionDigest"`
	ContentSHA256       string `json:"contentSha256"`
	ModelID             string `json:"modelId"`
	MaximumMicropoints  string `json:"maximumMicropoints"`
	Confirmed           bool   `json:"confirmed"`
}

func (c imageAnalysisConfirmBody) matches(s agentconfig.Snapshot) bool {
	return s.ImageAnalysis != nil && c.Confirmed && c.PreparationID == s.ID && c.ConfigurationDigest == s.Digest && c.QuoteDigest == s.ImageAnalysis.QuoteDigest &&
		c.SelectionDigest == s.ImageAnalysis.Image.SelectionDigest && c.ContentSHA256 == s.ImageAnalysis.Image.ContentSHA256 && c.ModelID == s.ExecutionModelProfile.ModelID && c.MaximumMicropoints == strconv.FormatInt(s.ImageAnalysis.MaximumMicropoints, 10)
}

type imageAnalysisChoice struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	URL  string `json:"url"`
}
type imageAnalysisSourceDTO struct {
	ContextKind     string                `json:"contextKind"`
	ContextID       string                `json:"contextId"`
	ProductKey      string                `json:"productKey"`
	CatalogVersion  string                `json:"catalogVersion"`
	PublicationID   string                `json:"publicationId"`
	Ready           bool                  `json:"ready"`
	Reason          string                `json:"reason"`
	InventoryStatus string                `json:"inventoryStatus"`
	InventoryDigest string                `json:"inventoryDigest"`
	Choices         []imageAnalysisChoice `json:"choices"`
}

func (a *imageAnalysisApplication) currentBinding(ctx context.Context, kind, id string) (agent.Binding, error) {
	i, err := a.parent.freshWorkbenchIdentity(ctx, authz.PermissionWorkbenchAgentUse)
	if err != nil {
		return agent.Binding{}, err
	}
	if kind == "own_product" {
		s, e := a.parent.ownSource(ctx, i, id, nil)
		return s.Binding, e
	}
	var original catalog.PublishedSnapshot
	if kind == "acquisition" {
		original, err = a.parent.acquisitionSnapshot(ctx, i, id)
	} else if kind == "collection" && a.supply != nil && a.supply.Sources != nil {
		selected, e := a.supply.Sources.Select(ctx, id)
		err = e
		if e == nil {
			_, _, original, err = selected.Read(ctx)
		}
	} else {
		return agent.Binding{}, agent.ErrUnavailable
	}
	if err != nil {
		return agent.Binding{}, err
	}
	current, err := a.parent.snapshots.GetCurrentSnapshot(ctx, original.Identity)
	if err != nil {
		return agent.Binding{}, err
	}
	if err = a.parent.checkAcquisitionLineage(ctx, i, original, current); err != nil {
		return agent.Binding{}, err
	}
	b := acquisitionBinding(id, "product", current)
	b.ContextKind = kind
	_, err = a.Authorize(ctx, b)
	return b, err
}

func (a *imageAnalysisApplication) describe(ctx context.Context, kind, id string) (imageAnalysisSourceDTO, error) {
	var out imageAnalysisSourceDTO
	b, err := a.currentBinding(ctx, kind, id)
	if err != nil {
		return out, err
	}
	i, err := a.parent.freshWorkbenchIdentity(ctx, authz.PermissionWorkbenchAgentUse)
	if err != nil {
		return out, err
	}
	v, _ := strconv.ParseUint(b.CatalogVersion, 10, 63)
	s, err := a.parent.snapshots.GetSnapshot(ctx, catalog.SnapshotIdentity{TenantID: i.TenantID, ProductKey: b.ProductKey}, v)
	if err != nil {
		return out, err
	}
	out = imageAnalysisSourceDTO{ContextKind: kind, ContextID: id, ProductKey: b.ProductKey, CatalogVersion: b.CatalogVersion, PublicationID: b.PublicationID, InventoryStatus: "UNKNOWN", Choices: []imageAnalysisChoice{}}
	for _, img := range catalog.IdentifyImages(s.Snapshot.Images) {
		out.Choices = append(out.Choices, imageAnalysisChoice{"source", img.ID, img.URL})
	}
	inventory, err := a.assets.GetApprovedInventory(ctx, asset.InventoryScope{TenantID: i.TenantID, ProductKey: b.ProductKey, SourceSnapshotVersion: v, TargetPlatform: "product"})
	if errors.Is(err, asset.ErrApprovedAssetsNotReady) {
		out.InventoryStatus = "NOT_READY"
	} else if err != nil {
		return out, err
	} else {
		out.InventoryStatus = "READY"
		out.InventoryDigest = collection.Digest(inventory)
		for _, img := range inventory.Assets {
			out.Choices = append(out.Choices, imageAnalysisChoice{"approved", img.ID, img.URL})
		}
	}
	out.Ready = true
	if !authz.AllowedOrganization(ctx, a.parent.authorizer, i.UserID, i.TenantID, i.Roles, authz.PermissionLocalAgentWrite) {
		out.Ready = false
		out.Reason = "RUN_PERMISSION_REQUIRED"
	}
	if _, err = a.profile(ctx, i.TenantID); err != nil {
		out.Ready = false
		out.Reason = "MODEL_NOT_READY"
	}
	activation, e := a.parent.configuration.ReadAgent(ctx, agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}, agentconfig.ImageAgentID)
	if e != nil || activation.Activation != "ENABLED" {
		out.Ready = false
		out.Reason = "AGENT_NOT_ENABLED"
	}
	return out, nil
}

func (a *imageAnalysisApplication) preparation(ctx context.Context, key string) (agentconfig.Snapshot, bool, error) {
	i, err := a.parent.freshWorkbenchIdentity(ctx, authz.PermissionWorkbenchAgentUse)
	if err != nil {
		return agentconfig.Snapshot{}, false, err
	}
	s, found, err := a.parent.configuration.FindImageAnalysisPreparation(ctx, agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}, key)
	if err != nil || !found {
		return s, found, err
	}
	if s.MemberID != i.EffectiveMemberID {
		return agentconfig.Snapshot{}, false, agentconfig.ErrForbidden
	}
	_, err = a.Authorize(ctx, s.Request.Binding)
	return s, true, err
}

func (a *imageAnalysisApplication) prepare(ctx context.Context, kind, id, key string, in imageAnalysisPrepareBody) (agentconfig.Snapshot, error) {
	existing, found, err := a.preparation(ctx, key)
	if err != nil {
		return existing, err
	}
	if found {
		b := existing.Request.Binding
		v := existing.ImageAnalysis
		if b.ContextKind != kind || b.ContextID != id || b.CatalogVersion != in.CatalogVersion || b.PublicationID != in.PublicationID || v == nil || v.Kind != in.Kind || v.ImageID != in.ImageID || v.InventoryDigest != in.InventoryDigest || existing.ExpectedModeRevision != in.ExpectedModeRevision {
			return agentconfig.Snapshot{}, agentconfig.ErrConflict
		}
		return existing, nil
	}
	b, err := a.currentBinding(ctx, kind, id)
	if err != nil {
		return existing, err
	}
	if b.CatalogVersion != in.CatalogVersion || b.PublicationID != in.PublicationID {
		return existing, agentconfig.ErrChanged
	}
	i, err := a.parent.freshIdentity(ctx)
	if err != nil {
		return existing, err
	}
	p, err := a.profile(ctx, i.TenantID)
	if err != nil {
		return existing, err
	}
	c := agentconfig.StartCommand{Scope: agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}, MemberID: i.EffectiveMemberID, ExpectedModeRevision: in.ExpectedModeRevision, AgentID: agentconfig.ImageAgentID, AgentVersion: agentconfig.ImageAgentVersion, ExecutionModelProfile: p, Request: agent.Request{Key: key, Binding: b, PolicyVersion: agentconfig.ImageAnalysisPolicyVersion, PromptVersion: agentconfig.ImageAnalysisPromptVersion, Limits: a.limits}}
	c, err = a.sources.FreezeImageAnalysis(ctx, c, in.Kind, in.ImageID, in.InventoryDigest, "")
	if err != nil {
		return existing, err
	}
	return a.parent.configuration.Prepare(ctx, c)
}

type imageAnalysisResultDTO struct {
	ContextKind         string                    `json:"contextKind"`
	ContextID           string                    `json:"contextId"`
	RequestKey          string                    `json:"requestKey"`
	CatalogVersion      string                    `json:"catalogVersion"`
	PublicationID       string                    `json:"publicationId"`
	UnitVersion         string                    `json:"unitVersion"`
	Status              string                    `json:"status"`
	StopReason          string                    `json:"stopReason"`
	RunID               string                    `json:"runId"`
	Kind                string                    `json:"kind"`
	ImageID             string                    `json:"imageId"`
	ModelID             string                    `json:"modelId"`
	PreparationID       string                    `json:"preparationId"`
	ConfigurationDigest string                    `json:"configurationDigest"`
	QuoteDigest         string                    `json:"quoteDigest"`
	SelectionDigest     string                    `json:"selectionDigest"`
	ContentSHA256       string                    `json:"contentSha256"`
	MaximumMicropoints  string                    `json:"maximumMicropoints"`
	ActualMicropoints   string                    `json:"actualMicropoints"`
	Width               int                       `json:"width"`
	Height              int                       `json:"height"`
	Opinion             *asset.ImageVisualOpinion `json:"opinion,omitempty"`
}

// History is strictly original snapshot/Run/native fact readback. It never
// downloads an image, requotes today's model/price, claims or resumes a Run.
func (a *imageAnalysisApplication) result(ctx context.Context, s agentconfig.Snapshot) (imageAnalysisResultDTO, error) {
	v := s.ImageAnalysis
	if v == nil {
		return imageAnalysisResultDTO{}, agent.ErrInvalid
	}
	b := s.Request.Binding
	out := imageAnalysisResultDTO{ContextKind: b.ContextKind, ContextID: b.ContextID, RequestKey: s.Request.Key, CatalogVersion: b.CatalogVersion, PublicationID: b.PublicationID, UnitVersion: aicapability.AIPointMicrounitV1, Status: "prepared", Kind: v.Kind, ImageID: v.ImageID, ModelID: s.ExecutionModelProfile.ModelID, PreparationID: s.ID, ConfigurationDigest: s.Digest, QuoteDigest: v.QuoteDigest, SelectionDigest: v.Image.SelectionDigest, ContentSHA256: v.Image.ContentSHA256, MaximumMicropoints: strconv.FormatInt(v.MaximumMicropoints, 10), Width: v.Image.Width, Height: v.Image.Height}
	r, found, err := a.parent.store.Lookup(ctx, s.Scope, b, s.Request.Key)
	if err != nil || !found {
		return out, err
	}
	want := s.Request
	want.ConfigurationSnapshotRef = agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: s.ID, Digest: s.Digest}
	if r.State.Request != want {
		return out, agent.ErrConflict
	}
	out.RunID = r.State.RunID
	out.StopReason = string(r.State.StopReason)
	out.Status = "unknown"
	if r.State.PendingInvocationID != "" || r.State.Phase == agent.Running {
		return out, nil
	}
	invocation := ""
	var observed *agent.ObservedUsage
	for _, h := range r.State.History {
		if h.InvocationID != "" {
			invocation = h.InvocationID
			observed = h.ObservedUsage
		}
	}
	if invocation == "" {
		out.Status = "not_sent"
		out.ActualMicropoints = "0"
		return out, nil
	}
	fact, err := a.parent.config.Ledger.ReadModelInvocation(ctx, s.Scope.OrganizationID, invocation)
	if err != nil {
		return out, err
	}
	if fact.AgentRunID != r.State.RunID || fact.UserID != s.Scope.ActorID || fact.MemberID != s.MemberID || fact.Operation != aicapability.OperationProductImageAnalysis {
		return out, agent.ErrConflict
	}
	if fact.Outcome == aicapability.InvocationDispatched || !fact.UsageKnown {
		return out, nil
	}
	if fact.BillableUsage != nil {
		actual, e := fact.OfficialTariff.SettlementMicropoints(*fact.BillableUsage)
		if e != nil {
			return out, e
		}
		out.ActualMicropoints = strconv.FormatInt(actual, 10)
	} else if r.State.StopReason == agent.StopDependency && fact.Outcome == aicapability.InvocationFailed && observed != nil && observed.Known && observed.Tokens == 0 && observed.CostMicros == 0 && fact.TotalTokens == 0 && fact.EstimatedCostKnown && fact.EstimatedCostMicros == 0 {
		out.Status = "not_sent"
		out.ActualMicropoints = "0"
		return out, nil
	} else {
		return out, nil
	}
	out.Status = "failed"
	if fact.Outcome == aicapability.InvocationSucceeded && r.State.Phase == agent.HumanReviewRequired && r.State.StopReason == "" && r.State.Validation != nil && r.State.Validation.Valid && r.State.Validation.PolicyVersion == agentconfig.ImageAnalysisPolicyVersion && r.State.DomainProposal != nil && r.State.DomainProposal.Kind == asset.ImageVisualAnalysisSchema {
		result, e := asset.DecodeImageVisualAnalysisResult(r.State.DomainProposal.Payload)
		if e != nil || result.InvocationID != invocation || result.ModelOutputSHA256 != fact.OutputHash || result.ConfigurationDigest != s.Digest || result.QuoteDigest != v.QuoteDigest || result.SelectionDigest != v.Image.SelectionDigest || result.ContentSHA256 != v.Image.ContentSHA256 || result.ModelID != fact.ModelID {
			return out, agent.ErrConflict
		}
		out.Status = "completed"
		out.Opinion = &result.Opinion
	}
	return out, nil
}
