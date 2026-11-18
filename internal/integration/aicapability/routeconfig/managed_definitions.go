package routeconfiguration

import (
	"context"
	"errors"
	"net/url"

	"task-processor/internal/aicapability"
	credential "task-processor/internal/aicapability/channelcredential"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

type ManagedDefinitions struct {
	Catalog       *Catalog
	Bindings      d.ManagedBindingReader
	Registrations credential.RegistrationReader
	Metadata      credential.MetadataReader
}

func (r *ManagedDefinitions) Supports(org, template string) bool {
	_, err := r.templates(org, template, credential.OfficialOpenAIBaseURL, "", "")
	return err == nil
}

func (r *ManagedDefinitions) templates(org, template, endpoint, connection, binding string) ([]Definition, error) {
	policy, usage, channel := governed.OpenAIMiniTextAdapterPolicy, "openai-text-v1", credential.OfficialOpenAIChannel
	priceID := "openai-mini"
	switch template {
	case credential.OfficialMiniTemplate:
		if endpoint != credential.OfficialOpenAIBaseURL {
			return nil, d.ErrInvalid
		}
	case credential.CompatibleMiniTemplate:
		policy, usage, channel = governed.RegisteredCompatibleMiniPolicy, aicapability.ReferenceTextTotalsV1, credential.CompatibleOpenAIChannel
		priceID = "openai-compatible-mini"
		if credential.CompatibleBaseURL(endpoint) != endpoint {
			return nil, d.ErrInvalid
		}
	default:
		return nil, d.ErrInvalid
	}
	if r == nil || r.Catalog == nil {
		return nil, d.ErrUnavailable
	}
	definitions := []Definition{}
	chat := false
	for _, scope := range r.Catalog.Scopes(org) {
		if scope.Capability == d.ImageCapability || aicapability.VisionTextOperation(aicapability.Operation(scope.Capability)) {
			continue
		}
		var anchor Definition
		for _, id := range r.Catalog.Candidates(scope) {
			candidate := r.Catalog.definitions[scope][id]
			// Use an already bounded Mini anchor for the core operation. The
			// exact model, usage and verified ceiling are never inferred.
			if candidate.Text == nil {
				continue
			}
			adapterPolicy := candidate.Text.ShapeProfile().AdapterPolicyVersion
			if adapterPolicy == governed.OpenAIMiniTextAdapterPolicy ||
				template == credential.OfficialMiniTemplate && adapterPolicy == governed.TitlePilotOpenAIAdapterPolicy {
				anchor = candidate
				break
			}
		}
		if anchor.Text == nil {
			return nil, d.ErrNeedsConfiguration
		}
		chat = chat || scope.Capability == string(aicapability.OperationAIWorkbenchChatPlan)
		id := binding
		if id == "" {
			id = "managed-mini-template"
		}
		profile := anchor.Text.ShapeProfile()
		profile.ClientName, profile.ConnectionID = id, connection
		if profile.AdapterPolicyVersion != governed.TitlePilotOpenAIAdapterPolicy {
			profile.AdapterPolicyVersion = policy
		}
		profile.UsageMappingVersion = usage
		profile.OfficialTariff = aicapability.OfficialModelTariff{}
		p := governed.RoutePolicy{Profile: profile, AdmittedEndpointIdentityDigest: governed.EndpointIdentityDigest(endpoint)}
		if !governed.ValidRouteShape(p.ShapeProfile(), aicapability.Operation(scope.Capability)) {
			return nil, d.ErrNeedsConfiguration
		}
		selectedPrice := priceID
		if template == credential.OfficialMiniTemplate && anchor.Channel == channel && anchor.PriceID != "" {
			selectedPrice = anchor.PriceID
		}
		definitions = append(definitions, Definition{Scope: scope, ConnectionID: connection, ConnectionProvider: "openai",
			ConnectionEndpoint: endpoint, Manufacturer: "openai", Region: "global", ServiceType: "standard-sync",
			Channel: channel, PriceID: selectedPrice, BoundEvidenceVersion: anchor.BoundEvidenceVersion,
			ReferenceVersion: d.ManagedTextBindingVersion, MaximumCostMicros: anchor.MaximumCostMicros,
			MaximumMicropoints: anchor.MaximumMicropoints, Text: &p})
	}
	if !chat {
		return nil, d.ErrNeedsConfiguration
	}
	return definitions, nil
}

type modelDefinitions struct {
	owner       string
	publication string
	values      []Definition
	label       string
	reason      string
}

func (r *ManagedDefinitions) read(ctx context.Context, org string, ref d.ModelBindingReference) (modelDefinitions, error) {
	var out modelDefinitions
	if r == nil || r.Bindings == nil || r.Registrations == nil || r.Metadata == nil {
		return out, d.ErrUnavailable
	}
	if !d.IsManagedBindingID(ref.ID) || ref.Version != d.ManagedTextBindingVersion {
		return out, d.ErrNotFound
	}
	binding, err := r.Bindings.ManagedBinding(ctx, org, ref.ID)
	if err != nil {
		return out, err
	}
	if binding.OrganizationID != org || binding.BindingID != ref.ID {
		return out, d.ErrNotFound
	}
	registration, err := r.Registrations.Registration(ctx, org, binding.ConnectionID)
	if err != nil || registration.OrganizationID != org || registration.ConnectionID != binding.ConnectionID || registration.Template != binding.Template {
		return out, d.ErrUnavailable
	}
	metadata, err := r.Metadata.Metadata(ctx, org, binding.ConnectionID)
	if err != nil || metadata.OrganizationID != org || metadata.ConnectionID != binding.ConnectionID {
		return out, d.ErrUnavailable
	}
	out.label = registration.Name
	if origin, err := url.Parse(metadata.BaseURL); err == nil && origin.Host != "" {
		out.label += " · " + origin.Host
	}
	if metadata.Account != "default" {
		out.label += " · " + metadata.Account
	}
	if !binding.Enabled {
		out.reason = "MODEL_BINDING_DISABLED"
	} else if !metadata.Enabled {
		out.reason = "CHANNEL_DISABLED"
	}
	out.values, err = r.templates(org, registration.Template, metadata.BaseURL, binding.ConnectionID, ref.ID)
	if errors.Is(err, d.ErrNeedsConfiguration) {
		if out.reason == "" {
			out.reason = "OPEN_CONSUMER_MISSING"
		}
		out.values = []Definition{{Channel: metadata.Channel}}
		return out, nil
	}
	return out, err
}
