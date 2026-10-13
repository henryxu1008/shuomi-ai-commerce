package imageagent

import (
	"reflect"
	"strings"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
)

// ArtworkSchema is an explicit source-free input, never a product image fallback.
const ArtworkSchema = "pod-artwork-v1"

type ArtworkPlan struct {
	Schema, Theme, Requirements, PromptVersion, Prompt string
	Configuration                                      agent.ConfigurationSnapshotRef
	ModeSnapshotRef                                    agent.ModeSnapshotRef
	ConfigurationEpoch, ParametersDigest, InputDigest  string
	QuoteDigest                                        string
	MaxPoints                                          int64
	Quote                                              ImageGenerationQuote
}

func CloneArtworkPlan(plan *ArtworkPlan) *ArtworkPlan {
	if plan == nil {
		return nil
	}
	copy := *plan
	return &copy
}

func ArtworkQuoteDigest(quote ImageGenerationQuote) string {
	return generationHash(struct {
		Schema string
		Quote  ImageGenerationQuote
	}{ArtworkSchema, quote})
}

func ValidateArtworkPlan(plan Plan) error {
	a := plan.Artwork
	if a == nil || a.Schema != ArtworkSchema || plan.Set != nil || plan.Revision < 1 || plan.Revision > MaxJSONSafePlanRevision || plan.ParentRevision != 0 || len(plan.Slots) != 1 || len(plan.SourceAssetIDs) != 0 || len(plan.StyleReferenceIDs) != 0 || !canonicalImageValue(plan.CreatedBy) {
		return ErrValidation
	}
	if !agentconfig.ValidSetText(a.Theme, 512) || strings.TrimSpace(a.Theme) == "" || !agentconfig.ValidSetText(a.Requirements, 4096) || a.PromptVersion != ArtworkSchema || !agentconfig.ValidSetText(a.Prompt, 16<<10) || strings.TrimSpace(a.Prompt) == "" || !a.Quote.Valid() || a.MaxPoints != a.Quote.Points || a.QuoteDigest != ArtworkQuoteDigest(a.Quote) || !canonicalImageValue(a.ConfigurationEpoch) || !generationDigest(a.ParametersDigest) || !generationDigest(a.InputDigest) || a.Configuration.Kind != agentconfig.SnapshotKind || !agentconfig.UUID(a.Configuration.ID) || !generationDigest(a.Configuration.Digest) || !a.ModeSnapshotRef.Valid() {
		return ErrValidation
	}
	if validatePersistedIdempotencyKey("plan", plan.IdempotencyKey) != nil {
		return ErrValidation
	}
	slot := plan.Slots[0]
	if ValidateArtifactKeyIdentifier(slot.ID) != nil || slot.Role != SlotRoleMain || slot.Recipe != nil || len(slot.SourceAssetIDs) != 0 || len(slot.StyleReferenceIDs) != 0 || slot.Brief != a.Theme || validatePersistedIdempotencyKey("slot", slot.IdempotencyKey) != nil || slot.Status != "" && slot.Status != SlotStatusPending {
		return ErrValidation
	}
	return nil
}

func ValidateArtworkExecution(input SlotExecutionInput) error {
	if input.Artwork == nil || input.ImageSet != nil || input.TargetPlatform != "sds" || input.ImagePolicyContext != nil || input.Attempt < 1 || !canonicalImageValue(input.TenantID) || !canonicalImageValue(input.UserID) || ValidateArtifactKeyIdentifier(input.RunID) != nil || validatePersistedIdempotencyKey("attempt", input.IdempotencyKey) != nil || len(input.AssetCatalog.Assets) != 0 || !ProductContextRefIsZero(input.ProductContext) || !ProductContextRefIsZero(input.AssetCatalog.ProductContext) {
		return ErrValidation
	}
	catalog, err := NormalizeAssetCatalog(input.AssetCatalog)
	if err != nil || input.AssetCatalog.Manifest.Hash != catalog.Manifest.Hash {
		return ErrValidation
	}
	slot := input.Slot
	slot.Status = SlotStatusPending
	return ValidateArtworkPlan(Plan{Revision: input.PlanRevision, IdempotencyKey: input.IdempotencyKey, CreatedBy: input.UserID, Artwork: input.Artwork, Slots: []Slot{slot}})
}

func ArtworkGenerationInputDigest(input SlotExecutionInput) (string, error) {
	if err := ValidateArtworkExecution(input); err != nil {
		return "", err
	}
	return generationHash(struct{ Schema, Fingerprint string }{ArtworkSchema, SlotExecutionFingerprint(input)}), nil
}

// ArtworkProvenance binds a real creative attempt, without inventing an original.
// Only the execution owner constructs it from the persisted GenerationFact.
type ArtworkProvenance struct {
	Protocol, TenantID, OwnerUserID, RunID, SlotID string
	PlanRevision                                   int64
	Attempt                                        int
	InputDigest, IntentID, Fingerprint             string
}

func (p ArtworkProvenance) Validate() error {
	if p.Protocol != ArtworkSchema || !canonicalImageValue(p.TenantID) || !canonicalImageValue(p.OwnerUserID) || ValidateArtifactKeyIdentifier(p.RunID) != nil || ValidateArtifactKeyIdentifier(p.SlotID) != nil || p.PlanRevision < 1 || p.PlanRevision > MaxJSONSafePlanRevision || p.Attempt < 1 || !generationDigest(p.InputDigest) || !generationDigest(p.IntentID) || !generationDigest(p.Fingerprint) {
		return ErrValidation
	}
	return nil
}

func (p ArtworkProvenance) ValidateForInput(input SlotExecutionInput) error {
	digest, err := ArtworkGenerationInputDigest(input)
	if err != nil || p.Validate() != nil || p.TenantID != input.TenantID || p.OwnerUserID != input.UserID || p.RunID != input.RunID || p.PlanRevision != input.PlanRevision || p.SlotID != input.Slot.ID || p.Attempt != input.Attempt || p.InputDigest != digest {
		return ErrRevisionConflict
	}
	identity := SlotExternalEffectIdentity{RunScope: RunScope{TenantID: input.TenantID, OwnerUserID: input.UserID, RunID: input.RunID}, PlanRevision: input.PlanRevision, SlotID: input.Slot.ID, Attempt: input.Attempt}
	if p.IntentID != generationHash(identity) {
		return ErrRevisionConflict
	}
	return nil
}

func NewArtworkProvenance(input SlotExecutionInput, fact GenerationFact) (ArtworkProvenance, error) {
	p := ArtworkProvenance{Protocol: ArtworkSchema, TenantID: input.TenantID, OwnerUserID: input.UserID, RunID: input.RunID, SlotID: input.Slot.ID, PlanRevision: input.PlanRevision, Attempt: input.Attempt, InputDigest: fact.Intent.InputDigest, IntentID: fact.IntentID, Fingerprint: fact.Fingerprint}
	if fact.Validate() != nil || fact.Intent.InputProtocol != ArtworkSchema || fact.Intent.CatalogHash != input.AssetCatalog.Manifest.Hash || p.ValidateForInput(input) != nil {
		return ArtworkProvenance{}, ErrRevisionConflict
	}
	quote := input.Artwork.Quote
	i := fact.Intent
	if i.Provider != quote.Provider || i.Model != quote.Model || i.Protocol != quote.Protocol || i.Resolution != quote.Resolution || i.Quality != quote.Quality || i.Points != quote.Points || i.PriceVersion != quote.PriceVersion || i.RouteReference != quote.RouteReference || i.CredentialReference != quote.CredentialReference || i.ConfigurationVersion != quote.ConfigurationVersion {
		return ArtworkProvenance{}, ErrRevisionConflict
	}
	return p, nil
}

func cloneArtworkProvenance(provenance *ArtworkProvenance) *ArtworkProvenance {
	if provenance == nil {
		return nil
	}
	copy := *provenance
	return &copy
}

func validateArtifactSource(source string, artwork *ArtworkProvenance, operations []string) error {
	if artwork != nil {
		if source != "" || artwork.Validate() != nil || !reflect.DeepEqual(operations, []string{"generate_artwork"}) {
			return ErrValidation
		}
		return nil
	}
	for _, operation := range operations {
		if operation == "generate_artwork" {
			return ErrValidation
		}
	}
	return ValidateProvenanceAssetID(source)
}
