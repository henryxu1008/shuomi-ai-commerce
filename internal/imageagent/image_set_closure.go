package imageagent

import "github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"

const providerUsageUnit = agentconfig.ProviderImageUsageUnit

func (c ImageSlotClosure) Valid(attempt int) bool {
	if c.Kind == "not_dispatched" {
		return attempt == 0 && c == (ImageSlotClosure{UnitVersion: "ai-micropoint-v1", Kind: "not_dispatched"})
	}
	if attempt <= 0 || !agentconfig.ImageDigest(c.IntentID) || !agentconfig.ImageDigest(c.Fingerprint) {
		return false
	}
	if c.UnitVersion == providerUsageUnit {
		return (c.Kind == "provider_usage" || c.Kind == "provider_failed" || c.Kind == "provider_no_generation") &&
			c.Points == 0 && c.SettlementProofDigest == "" && agentconfig.ImageDigest(c.ProviderUsageProofDigest)
	}
	return c.UnitVersion == "ai-micropoint-v1" && c.ProviderUsageProofDigest == "" && agentconfig.ImageDigest(c.SettlementProofDigest) &&
		(c.Kind == "settled" && c.Points >= 0 || c.Kind == "no_generation" && c.Points == 0)
}

func (c ImageSlotClosure) MatchesQuote(quote ImageGenerationQuote) bool {
	if c.Kind == "not_dispatched" {
		return true
	}
	return (c.UnitVersion == providerUsageUnit) == quote.UsesProviderUsage() && c.Points <= quote.Points
}

func (c ImageSlotClosure) CandidateProof() *ImageGenerationProof {
	if c.Kind != "settled" && c.Kind != "provider_usage" {
		return nil
	}
	return &ImageGenerationProof{UnitVersion: c.UnitVersion, IntentID: c.IntentID, Fingerprint: c.Fingerprint,
		SettlementProofDigest: c.SettlementProofDigest, Points: c.Points, ProviderUsageProofDigest: c.ProviderUsageProofDigest}
}

func (q ImageGenerationQuote) MatchesGenerationIntent(intent GenerationIntent) bool {
	return q.Valid() && q.BillingMode == intent.BillingMode && q.ConnectionID == intent.ConnectionID && q.Billing == intent.Billing && q.Provider == intent.Provider && q.Model == intent.Model && q.Protocol == intent.Protocol && q.Resolution == intent.Resolution && q.Quality == intent.Quality && q.PriceVersion == intent.PriceVersion && q.Points == intent.Points && q.RouteReference == intent.RouteReference && q.CredentialReference == intent.CredentialReference && q.ConfigurationVersion == intent.ConfigurationVersion
}

// This is a view of original generation/settlement facts, never a new resource
// decision. Callers must resolve the original fact through its effect owner.
func ImageGenerationClosure(fact GenerationFact) (*ImageSlotClosure, error) {
	if fact.Validate() != nil || (fact.Intent.InputProtocol != ImageSetSchema && fact.Intent.InputProtocol != ArtworkSchema) {
		return nil, ErrRevisionConflict
	}
	if fact.Intent.UsesProviderUsage() {
		kind := ""
		switch fact.State {
		case GenerationSucceeded:
			kind = "provider_usage"
		case GenerationSupplierFailed:
			kind = "provider_failed"
		case GenerationNoEffect:
			kind = "provider_no_generation"
		default:
			return nil, ErrCommandBlocked
		}
		return &ImageSlotClosure{UnitVersion: providerUsageUnit, Kind: kind, IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, ProviderUsageProofDigest: fact.TerminalProofDigest()}, nil
	}
	if fact.Settlement == (GenerationSettlementReceipt{}) || fact.State != GenerationSucceeded && fact.State != GenerationNoEffect {
		return nil, ErrCommandBlocked
	}
	closure := &ImageSlotClosure{UnitVersion: fact.Settlement.UnitVersion, IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, SettlementProofDigest: fact.Settlement.ProofDigest}
	if fact.State == GenerationSucceeded {
		closure.Kind, closure.Points = "settled", fact.Settlement.Points
	} else {
		closure.Kind = "no_generation"
	}
	return closure, nil
}
