package routeconfiguration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"sort"
	"task-processor/internal/agentconfig"
	"time"

	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

// SharedCatalog resolves admission metadata and published official price facts.
// It neither reads keys nor owns a second current selection or resource balance.
type SharedCatalog struct {
	Catalog     *Catalog
	Credentials d.CredentialReader
	Prices      *d.PlatformPricesService
	Now         func() time.Time
}

func (c *SharedCatalog) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}
func (c *SharedCatalog) ValidateDefaultChannel(ctx context.Context, org, channel string) error {
	if c == nil || c.Catalog == nil || ctx == nil || ctx.Err() != nil || !d.ValidFact(org, 128) {
		return d.ErrUnavailable
	}
	// The approved initial default remains visible before credentials are admitted.
	if channel == "openai-direct" {
		return nil
	}
	for scope, defs := range c.Catalog.definitions {
		if scope.OrganizationID == org {
			for _, def := range defs {
				if def.Channel == channel {
					return nil
				}
			}
		}
	}
	return d.ErrNeedsConfiguration
}
func (c *SharedCatalog) QualifyModel(ctx context.Context, org string, category d.ModelCategory, ref d.ModelBindingReference) (d.ModelQualification, error) {
	q := d.ModelQualification{OrganizationID: org, Category: category, Reference: ref, Reason: "BINDING_NOT_ADMITTED"}
	if c == nil || c.Catalog == nil || ctx == nil || ctx.Err() != nil {
		return q, d.ErrUnavailable
	}
	defs := []Definition{}
	scopes := c.Catalog.Scopes(org)
	for _, scope := range scopes {
		def, ok := c.Catalog.definitions[scope][ref.ID]
		if !ok || sharedDefinitionCategory(def) != category {
			continue
		}
		defs = append(defs, def)
	}
	if len(defs) == 0 {
		q.Channel, q.Model = "UNKNOWN", "UNKNOWN"
		if ref.Version != "approved-default-v1" || ref.ID == "openai-mini" && category != d.ModelCategoryText && category != d.ModelCategoryMultimodal || ref.ID == "openai-flare" && category != d.ModelCategoryImage {
			return q, nil
		}
		switch ref.ID {
		case "openai-mini":
			q.Channel, q.Model = "openai-direct", "gpt-5.4-mini-2026-03-17"
		case "openai-flare":
			q.Channel, q.Model = "openai-direct", "gpt-image-2.5-flare-2026-09-08"
		default:
			return q, nil
		}
		if category == d.ModelCategoryImage {
			q.Reason = "PROTOCOL_AND_COST_BOUND_MISSING"
		} else if category == d.ModelCategoryMultimodal {
			q.Reason = "VISION_CONSUMER_NOT_IMPLEMENTED"
		}
		return q, nil
	}
	first := defs[0]
	q.Channel = first.Channel
	if q.Channel == "" {
		q.Channel = "unadmitted"
	}
	if first.Image != nil {
		q.Model = first.Image.Model
	} else {
		q.Model = first.Text.ShapeProfile().ModelID
	}
	if ref.Version != first.ReferenceVersion || !d.ValidFact(first.ReferenceVersion, 128) {
		return q, nil
	}
	q.BoundEvidenceVersion = first.BoundEvidenceVersion
	if c.Prices == nil {
		q.Reason = "OFFICIAL_PRICE_MISSING"
		return q, nil
	}
	published, err := c.Prices.Current(ctx)
	if errors.Is(err, d.ErrNotFound) {
		q.Reason = "OFFICIAL_PRICE_MISSING"
		return q, nil
	}
	if err != nil {
		return q, err
	}
	applicable := false
	priceModel := q.Model
	if first.ManufacturerModel != "" {
		priceModel = first.ManufacturerModel
	}
	var referenceEstimate aicapability.ReferenceImageEstimate
	for _, entry := range published.Payload.PlatformPrices.Prices {
		if entry.ID != first.PriceID || entry.Model != priceModel || entry.Channel != q.Channel || c.now().Before(entry.EffectiveFrom) || !c.now().Before(entry.EffectiveUntil) {
			continue
		}
		applicable = d.ValidFact(first.Manufacturer, 64) && entry.Provider == first.Manufacturer && d.ValidFact(first.Region, 64) && d.ValidFact(first.ServiceType, 64) && entry.Region == first.Region && entry.ServiceType == first.ServiceType
		if category == d.ModelCategoryImage {
			applicable = applicable && entry.ServiceType == "image-sync" && entry.Image != nil && entry.Image.Width == 1024 && entry.Image.Height == 1024 && entry.Image.Quality == "auto"
			if applicable {
				referenceEstimate = entry.Image.ReferenceEstimate
			}
		} else {
			applicable = applicable && entry.ServiceType == "standard-sync" && entry.Image == nil
		}
	}
	if !applicable {
		q.Reason = "PRICE_APPLICABILITY_MISSING"
		return q, nil
	}
	price, err := published.Payload.PlatformPrices.Resolve(first.PriceID, priceModel, q.Channel, c.now())
	if err != nil {
		q.Reason = "PRICE_OR_FX_NOT_APPLICABLE"
		return q, nil
	}
	if priceModel != q.Model {
		price.WireModel = q.Model
	}
	q.Price = price
	if !price.IsReferenceEstimate() && !price.IsReferenceImageEstimate() && !d.ValidFact(first.BoundEvidenceVersion, 128) {
		q.Reason = "COST_BOUND_MISSING"
		return q, nil
	}
	q.TextProfiles = map[string]aicapability.ModelProfile{}
	var components []string
	for _, def := range defs {
		if def.ManufacturerModel != first.ManufacturerModel || def.ConnectionID != first.ConnectionID || def.ConnectionProvider != first.ConnectionProvider || def.Manufacturer != first.Manufacturer || def.Region != first.Region || def.ServiceType != first.ServiceType || def.Channel != q.Channel || def.PriceID != first.PriceID || def.ReferenceVersion != ref.Version || def.BoundEvidenceVersion != first.BoundEvidenceVersion {
			q.Reason = "CONSUMER_BINDING_MISMATCH"
			return q, nil
		}
		if !d.ValidFact(def.ConnectionID, 128) || !d.ValidFact(def.ConnectionProvider, 128) {
			q.Reason = "CREDENTIAL_NOT_ADMITTED"
			return q, nil
		}
		credentials := c.Credentials
		if credentials == nil {
			q.Reason = "CREDENTIAL_READER_MISSING"
			return q, nil
		}
		row, err := credentials.Credential(ctx, org, def.ConnectionID)
		if errors.Is(err, d.ErrNotFound) {
			q.Reason = "CREDENTIAL_NOT_ADMITTED"
			return q, nil
		}
		if err != nil {
			return q, err
		}
		binding, err := c.Catalog.Qualify(def.Scope, ref.ID, row)
		if err != nil {
			q.Reason = "CREDENTIAL_NOT_ADMITTED"
			return q, nil
		}
		if binding.Model != q.Model {
			q.Reason = "CONSUMER_BINDING_MISMATCH"
			return q, nil
		}
		if binding.Image != nil {
			// This native response has a validated complete image-count dimension.
			// URL-only GRSAI responses cannot establish an actual customer charge.
			reference := binding.Provider == "openai" && binding.Model == aicapability.FlareImageModel && binding.Protocol == aicapability.FlareImageProtocol && price.IsReferenceImageEstimate() && referenceEstimate.Validate() == nil
			if !reference && (binding.Provider != "bailian" || price.UsageVersion != "qwen-image-count-v1") {
				q.Reason = "COMPLETE_BILLING_USAGE_MISSING"
				return q, nil
			}
			image := *binding.Image
			image.Billing.OfficialTariff = price
			image.Billing.BoundEvidenceVersion = def.BoundEvidenceVersion
			if reference {
				image.Billing.ReferenceEstimate = referenceEstimate
			}
			maximum, err := image.Billing.ReservationMicropoints(image.Model)
			if err != nil || maximum > def.Image.MaximumMicropoints {
				q.Reason = "COST_BOUND_MISSING"
				return q, nil
			}
			image.Points, image.PriceVersion, image.ConfigurationVersion = maximum, price.PriceVersion, ref.Version
			q.Image = &image
		} else {
			p := *binding.Text
			// Each mapping has explicit fields and applicability. Reference
			// totals cannot qualify an unopened image or vision consumer.
			mappingReady := p.AdapterKind == string(governed.AdapterOpenAICompatible) && price.UsageVersion == "openai-text-v1"
			if price.IsReferenceEstimate() {
				mappingReady = governed.ReferenceMappingReady(p)
			}
			if !mappingReady || p.AdapterPolicyVersion == governed.DashScopeDeepSeekV41FlashNonThinkingPolicy && !price.IsReferenceEstimate() {
				q.Reason = "METERING_MAPPING_NOT_ADMITTED"
				return q, nil
			}
			p.OfficialTariff = price
			if p.Validate() != nil {
				q.Reason = "CONSUMER_NOT_QUALIFIED"
				return q, nil
			}
			if category == d.ModelCategoryMultimodal && (!validVisionConsumer(def.Scope.Capability, p) || p.MaximumPromptTokens <= aicapability.SingleImagePromptEstimate(p)+1024) {
				q.Reason = "VISION_PROTOCOL_NOT_ADMITTED"
				return q, nil
			}
			costLimit := def.MaximumCostMicros
			if def.Scope.Capability == string(aicapability.OperationOwnProductIntake) &&
				def.Channel == "openai-direct" && p.AdapterPolicyVersion == aicapability.SingleImageMiniAdapterPolicy {
				costLimit = min(costLimit, 300_000)
				if !miniIntakeRatesCoverTariff(p) {
					q.Reason = "COST_BOUND_MISSING"
					return q, nil
				}
				if def.MaximumMicropoints <= 0 {
					q.Reason = "POINT_BUDGET_MISSING"
					return q, nil
				}
				if def.MaximumMicropoints > 30_000_000 {
					q.Reason = "POINT_BUDGET_EXCEEDED"
					return q, nil
				}
			}
			maximum, err := p.MaximumCost()
			if err != nil || maximum > costLimit {
				q.Reason = "COST_BOUND_MISSING"
				return q, nil
			}
			advisory := def.Scope.Capability == string(aicapability.OperationProductResearchDecision) || def.Scope.Capability == string(aicapability.OperationProductSelectionDecision)
			if price.IsReferenceEstimate() && advisory && def.MaximumMicropoints == 0 {
				q.Reason = "POINT_BUDGET_MISSING"
				return q, nil
			}
			if def.MaximumMicropoints > 0 {
				points, err := price.ReserveTextMicropoints(p.MaximumPromptTokens, p.MaximumCompletionTokens)
				if err != nil || points > def.MaximumMicropoints {
					q.Reason = "POINT_BUDGET_EXCEEDED"
					return q, nil
				}
			}
			q.TextProfiles[def.Scope.Capability] = p
			components = append(components, governed.ComponentExecutionBinding(p))
			if q.Text == nil {
				q.Text = &p
			}
		}
	}
	if category == d.ModelCategoryText {
		for _, scope := range scopes {
			if scope.Capability != d.ImageCapability && !aicapability.VisionTextOperation(aicapability.Operation(scope.Capability)) {
				if _, ok := q.TextProfiles[scope.Capability]; !ok {
					q.Reason = "OPEN_CONSUMER_MISSING"
					return q, nil
				}
			}
		}
	}
	raw, _ := json.Marshal(struct {
		Definitions []Definition
		Price       aicapability.OfficialModelTariff
		Components  []string
	}{defs, price, components})
	sum := sha256.Sum256(raw)
	q.ProfileVersion = hex.EncodeToString(sum[:])
	q.Ready, q.Reason = true, ""
	return q, nil
}

// Mini's CNY quote must cover the same uncached input and output that the
// resource owner reserves from the current USD tariff and its frozen FX.
func miniIntakeRatesCoverTariff(p aicapability.ModelProfile) bool {
	tariff := p.OfficialTariff
	if p.Currency != "CNY" || tariff.Currency != "USD" ||
		tariff.UsageVersion != "openai-text-v1" || tariff.Rates.InputText <= 0 || tariff.Rates.OutputText <= 0 {
		return false
	}
	covers := func(cny, usd int64) bool {
		// Comparing scaled integers enforces upward rounding without overflow.
		configured := new(big.Int).Mul(big.NewInt(cny), big.NewInt(1_000_000))
		required := new(big.Int).Mul(big.NewInt(usd), big.NewInt(tariff.CNYMicrosPerCurrencyUnit))
		return configured.Cmp(required) >= 0
	}
	return covers(p.InputMicrosPerMillion, tariff.Rates.InputText) &&
		covers(p.OutputMicrosPerMillion, tariff.Rates.OutputText)
}

func validVisionConsumer(capability string, p aicapability.ModelProfile) bool {
	if capability == string(aicapability.OperationOwnProductIntake) {
		return agentconfig.OwnIntakeProfile(p)
	}
	return capability == string(aicapability.OperationProductImageAnalysis) && aicapability.SingleImageProfile(p) && p.PromptVersion == agentconfig.ImageAnalysisPromptVersion && p.OutputSchemaVersion == agentconfig.ImageAnalysisResultVersion
}

func sharedDefinitionCategory(def Definition) d.ModelCategory {
	if def.Image != nil {
		return d.ModelCategoryImage
	}
	if aicapability.VisionTextOperation(aicapability.Operation(def.Scope.Capability)) {
		return d.ModelCategoryMultimodal
	}
	return d.ModelCategoryText
}
func (c *SharedCatalog) Options(ctx context.Context, org string) ([]d.ModelOption, error) {
	refs := map[d.ModelCategory]map[d.ModelBindingReference]bool{}
	for _, cat := range []d.ModelCategory{d.ModelCategoryText, d.ModelCategoryImage, d.ModelCategoryMultimodal} {
		refs[cat] = map[d.ModelBindingReference]bool{}
	}
	refs[d.ModelCategoryText][d.ModelBindingReference{ID: "openai-mini", Version: "approved-default-v1"}] = true
	refs[d.ModelCategoryMultimodal][d.ModelBindingReference{ID: "openai-mini", Version: "approved-default-v1"}] = true
	refs[d.ModelCategoryImage][d.ModelBindingReference{ID: "openai-flare", Version: "approved-default-v1"}] = true
	for scope, defs := range c.Catalog.definitions {
		if scope.OrganizationID != org {
			continue
		}
		for id, def := range defs {
			if !d.ValidFact(def.ReferenceVersion, 128) {
				continue
			}
			cat := sharedDefinitionCategory(def)
			for old := range refs[cat] {
				if old.ID == id {
					delete(refs[cat], old)
				}
			}
			refs[cat][d.ModelBindingReference{ID: id, Version: def.ReferenceVersion}] = true
		}
	}
	out := []d.ModelOption{}
	for category, values := range refs {
		for ref := range values {
			q, err := c.QualifyModel(ctx, org, category, ref)
			if err != nil {
				return nil, err
			}
			option := d.ModelOption{Category: category, Reference: ref, Channel: q.Channel, Model: q.Model, Ready: q.Ready, Reason: q.Reason}
			if q.Price.Validate() == nil {
				p := q.Price
				option.Price = &p
			}
			out = append(out, option)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Category != b.Category {
			return a.Category < b.Category
		}
		if a.Reference.ID != b.Reference.ID {
			return a.Reference.ID < b.Reference.ID
		}
		return a.Reference.Version < b.Reference.Version
	})
	return out, nil
}

// SharedTextReader preserves each opened Agent's existing prompt/schema and
// smaller budget. All operations obtain their chosen model from one current.
type SharedTextReader struct{ Service d.SharedCategoryReader }

func (r SharedTextReader) ReadPolicy(ctx context.Context, key governed.RouteKey) (governed.RoutePolicy, error) {
	if r.Service == nil {
		return governed.RoutePolicy{}, d.ErrUnavailable
	}
	category := d.ModelCategoryText
	if aicapability.VisionTextOperation(key.Operation) {
		category = d.ModelCategoryMultimodal
	}
	q, digest, err := r.Service.ReadCategory(ctx, key.OrganizationID, category)
	if errors.Is(err, d.ErrNotFound) || errors.Is(err, d.ErrNeedsConfiguration) {
		return governed.RoutePolicy{}, nil
	}
	if err != nil {
		return governed.RoutePolicy{}, err
	}
	p, ok := q.TextProfiles[string(key.Operation)]
	if !ok {
		return governed.RoutePolicy{}, d.ErrNeedsConfiguration
	}
	p.RoutingPolicyVersion = digest
	return governed.RoutePolicy{Profile: p, AdmittedCredentialVersion: p.CredentialVersion, AdmittedEndpointIdentityDigest: p.EndpointIdentityDigest}, nil
}
