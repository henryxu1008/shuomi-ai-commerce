package routeconfiguration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"task-processor/internal/agentconfig"
	"time"

	"task-processor/internal/aicapability"
	credential "task-processor/internal/aicapability/channelcredential"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

// SharedCatalog resolves admission metadata and published official price facts.
// It neither reads keys nor owns a second current selection or resource balance.
type SharedCatalog struct {
	Platform    *PlatformChannels
	Managed     *ManagedDefinitions
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
	if channel == credential.CompatibleOpenAIChannel && c.Managed != nil {
		rows, err := c.Managed.Registrations.Registrations(ctx, org)
		if err != nil {
			return d.ErrUnavailable
		}
		for _, registration := range rows {
			if registration.OrganizationID == org && registration.Template == credential.CompatibleMiniTemplate {
				return nil
			}
		}
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
	if d.PlatformConnection(ref.ID) != "" {
		if category != d.ModelCategoryText {
			return q, d.ErrInvalid
		}
		if c.Platform == nil {
			return c.qualifyDefinitions(ctx, org, category, ref, modelDefinitions{owner: credential.PlatformOwner, reason: "PLATFORM_CHANNEL_UNAVAILABLE"})
		}
		resolved, err := c.Platform.admitted(ctx, org, ref)
		if err != nil {
			return q, err
		}
		return c.qualifyDefinitions(ctx, org, category, ref, resolved)
	}
	defs := []Definition{}
	label := ""
	scopes := c.Catalog.Scopes(org)
	if d.IsManagedBindingID(ref.ID) {
		if category != d.ModelCategoryText {
			return q, d.ErrUnavailable
		}
		resolved, err := c.Managed.read(ctx, org, ref)
		if errors.Is(err, d.ErrNotFound) {
			return q, d.ErrUnavailable
		}
		if err != nil {
			return q, err
		}
		defs, label = resolved.values, resolved.label
		q.ChannelName, q.Source = label, "enterprise"
		if resolved.reason != "" {
			q.Channel, q.Model, q.Reason = defs[0].Channel, OpenAIQueryModel, resolved.reason
			return q, nil
		}
	} else {
		for _, scope := range scopes {
			def, ok := c.Catalog.definitions[scope][ref.ID]
			if !ok || sharedDefinitionCategory(def) != category {
				continue
			}
			defs = append(defs, def)
		}
	}
	return c.qualifyDefinitions(ctx, org, category, ref, modelDefinitions{values: defs, label: label})
}

func (c *SharedCatalog) qualifyDefinitions(ctx context.Context, org string, category d.ModelCategory, ref d.ModelBindingReference, resolved modelDefinitions) (d.ModelQualification, error) {
	q := d.ModelQualification{OrganizationID: org, Category: category, Reference: ref, ChannelName: resolved.label, Reason: "BINDING_NOT_ADMITTED", CredentialOwner: resolved.owner}
	if resolved.owner == credential.PlatformOwner {
		q.Source = "platform"
	} else if d.IsManagedBindingID(ref.ID) {
		q.Source = "enterprise"
	}
	if resolved.reason != "" {
		q.Channel, q.Model, q.Reason = "UNKNOWN", "UNKNOWN", resolved.reason
		if resolved.reason != "PLATFORM_ACCESS_WITHDRAWN" && resolved.reason != "PLATFORM_BINDING_REPLACED" && resolved.reason != "PLATFORM_CHANNEL_UNAVAILABLE" {
			q.Model = OpenAIQueryModel
		}
		return q, nil
	}
	defs, scopes := resolved.values, c.Catalog.Scopes(org)
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
	now := c.now()
	var price aicapability.OfficialModelTariff
	var referenceEstimate aicapability.ReferenceImageEstimate
	if trial := first.PrivateTrialPrice; trial != nil {
		if len(defs) != 1 || now.Before(trial.NotBefore) || !now.Before(trial.NotAfter) {
			q.Reason = "PRICE_APPLICABILITY_MISSING"
			return q, nil
		}
		price = trial.Tariff
	} else {
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
		for _, entry := range published.Payload.PlatformPrices.Prices {
			if entry.PlatformConnectionID != "" && (resolved.owner != credential.PlatformOwner || entry.PlatformConnectionID != first.ConnectionID) {
				continue
			}
			if entry.ID != first.PriceID || entry.Model != priceModel || entry.Channel != q.Channel || now.Before(entry.EffectiveFrom) || !now.Before(entry.EffectiveUntil) {
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
		price, err = published.Payload.PlatformPrices.Resolve(first.PriceID, priceModel, q.Channel, now)
		if err != nil {
			q.Reason = "PRICE_OR_FX_NOT_APPLICABLE"
			return q, nil
		}
		if priceModel != q.Model {
			price.WireModel = q.Model
		}
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
		row, err := credentials.Credential(ctx, def.credentialOrganization(), def.ConnectionID)
		if errors.Is(err, d.ErrNotFound) {
			q.Reason = "CREDENTIAL_NOT_ADMITTED"
			return q, nil
		}
		if err != nil {
			return q, err
		}
		binding, err := qualifyDefinition(def, ref.ID, row)
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
			if p.AdapterPolicyVersion == governed.OpenAIMiniTextAdapterPolicy && !p.CNYRatesCoverUSDTariff() {
				q.Reason = "COST_BOUND_MISSING"
				return q, nil
			}
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
		Owner       string `json:"Owner,omitempty"`
		Publication string `json:"Publication,omitempty"`
	}{defs, price, components, resolved.owner, resolved.publication})
	sum := sha256.Sum256(raw)
	q.ProfileVersion = hex.EncodeToString(sum[:])
	q.Ready, q.Reason = true, ""
	return q, nil
}

// Mini's CNY quote must cover the same uncached input and output that the
// resource owner reserves from the current USD tariff and its frozen FX.
func miniIntakeRatesCoverTariff(p aicapability.ModelProfile) bool {
	return p.CNYRatesCoverUSDTariff()
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
	return c.options(ctx, org, "")
}
func (c *SharedCatalog) TextOptions(ctx context.Context, org string) ([]d.ModelOption, error) {
	return c.options(ctx, org, d.ModelCategoryText)
}
func (c *SharedCatalog) options(ctx context.Context, org string, only d.ModelCategory) ([]d.ModelOption, error) {
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
	if c.Managed != nil {
		bindings, err := c.Managed.Bindings.ManagedBindings(ctx, org)
		if err != nil {
			return nil, err
		}
		for _, binding := range bindings {
			refs[d.ModelCategoryText][binding.Reference()] = true
		}
	}
	if c.Platform != nil {
		published, err := c.Platform.references(ctx, org)
		if err != nil {
			return nil, err
		}
		for _, ref := range published {
			refs[d.ModelCategoryText][ref] = true
		}
	}
	out := []d.ModelOption{}
	for category, values := range refs {
		if only != "" && only != category {
			continue
		}
		for ref := range values {
			q, err := c.QualifyModel(ctx, org, category, ref)
			if err != nil {
				return nil, err
			}
			if q.Reason == "PLATFORM_ACCESS_WITHDRAWN" {
				continue
			}
			option := d.ModelOption{Source: q.Source, Category: category, Reference: ref, Channel: q.Channel, Model: q.Model, Ready: q.Ready, Reason: q.Reason, ChannelName: q.ChannelName}
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
	return governed.RoutePolicy{CredentialOwner: q.CredentialOwner, Profile: p, AdmittedCredentialVersion: p.CredentialVersion, AdmittedEndpointIdentityDigest: p.EndpointIdentityDigest}, nil
}
