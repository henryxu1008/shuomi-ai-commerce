package einomodel

import (
	"context"
	"strings"

	"task-processor/internal/aicapability"
	credential "task-processor/internal/aicapability/channelcredential"
	"task-processor/internal/integration/googleinteractions"
)

type RouteKey struct {
	OrganizationID string
	Operation      aicapability.Operation
}

type RouteReadiness string

const (
	RouteAvailable          RouteReadiness = "AVAILABLE"
	RouteNeedsConfiguration RouteReadiness = "NEEDS_CONFIGURATION"
	RouteUnavailable        RouteReadiness = "UNAVAILABLE"
)

// RoutePolicy is trusted deployment configuration for one organization and
// one capability. The admitted version is bound after restricted provisioning.
type RoutePolicy struct {
	Profile                        aicapability.ModelProfile
	AdmittedCredentialVersion      string
	AdmittedEndpointIdentityDigest string
}

// ShapeProfile validates a deployment profile before its first credential
// write. An empty admitted version keeps execution unavailable until the
// operator binds the returned row version in deployment configuration.
func (p RoutePolicy) ShapeProfile() aicapability.ModelProfile {
	profile := p.Profile
	profile.CredentialVersion = p.AdmittedCredentialVersion
	if profile.CredentialVersion == "" {
		profile.CredentialVersion = "unadmitted"
	}
	profile.EndpointIdentityDigest = p.AdmittedEndpointIdentityDigest
	return profile
}

type OrganizationRouteResolver struct {
	credentials credential.SecretReader
	policies    map[RouteKey]RoutePolicy
	current     RoutePolicyReader
}

type RoutePolicyReader interface {
	ReadPolicy(context.Context, RouteKey) (RoutePolicy, error)
}

// Managed mode has a single durable policy source. An unavailable reader must
// never borrow a deployment policy, member credential, or process default.
func NewManagedOrganizationRouteResolver(credentials credential.SecretReader, current RoutePolicyReader) (*OrganizationRouteResolver, error) {
	if current == nil {
		return nil, ErrInvalid
	}
	return &OrganizationRouteResolver{credentials: credentials, current: current}, nil
}

func EndpointIdentityDigest(endpoint string) string { return endpointDigest(endpoint) }

func NewOrganizationRouteResolver(credentials credential.SecretReader, policies map[RouteKey]RoutePolicy) (*OrganizationRouteResolver, error) {
	// A separately configured registered consumer may leave this registry
	// empty. Resolve still rejects every unconfigured exact operation.
	if credentials == nil || len(policies) > 128 {
		return nil, ErrInvalid
	}
	frozen := make(map[RouteKey]RoutePolicy, len(policies))
	for key, policy := range policies {
		if key.OrganizationID == "" || len(key.OrganizationID) > 128 ||
			!aicapability.SupportsObservedUsageFailure(key.Operation) ||
			policy.AdmittedEndpointIdentityDigest == "" {
			return nil, ErrInvalid
		}
		if !ValidRouteProfile(policy.ShapeProfile(), key.Operation) {
			return nil, ErrInvalid
		}
		frozen[key] = policy
	}
	return &OrganizationRouteResolver{credentials: credentials, policies: frozen}, nil
}

func (r *OrganizationRouteResolver) Resolve(ctx context.Context, input aicapability.TextInputIdentity) (QualifiedRoute, error) {
	route, _, err := r.resolve(ctx, input)
	return route, err
}

// Readiness reads the same scoped policy and credential as Resolve. It is a
// current configuration hint, never execution authorization or provider health.
func (r *OrganizationRouteResolver) Readiness(ctx context.Context, input aicapability.TextInputIdentity) RouteReadiness {
	_, status, _ := r.resolve(ctx, input)
	return status
}

func (r *OrganizationRouteResolver) resolve(ctx context.Context, input aicapability.TextInputIdentity) (QualifiedRoute, RouteReadiness, error) {
	if r == nil || r.credentials == nil || ctx == nil || ctx.Err() != nil {
		return QualifiedRoute{}, RouteUnavailable, ErrNotDispatched
	}
	policy, ok := r.policies[RouteKey{OrganizationID: input.OrganizationID, Operation: input.Operation}]
	if r.current != nil {
		var err error
		policy, err = r.current.ReadPolicy(ctx, RouteKey{OrganizationID: input.OrganizationID, Operation: input.Operation})
		if err != nil {
			return QualifiedRoute{}, RouteUnavailable, ErrNotDispatched
		}
		ok = true
	}
	if !ok {
		return QualifiedRoute{}, RouteUnavailable, ErrNotDispatched
	}
	if !credential.ValidID(policy.Profile.ConnectionID) || !strings.HasPrefix(policy.AdmittedCredentialVersion, "channel-credential-v1:") {
		return QualifiedRoute{}, RouteNeedsConfiguration, ErrNotDispatched
	}
	policy.Profile = policy.ShapeProfile()
	// Model identity comes only from the qualified definition. One exact current
	// connection observation supplies revision and secret; no model lives there.
	row, err := r.credentials.Resolve(ctx, input.OrganizationID, policy.Profile.ConnectionID, policy.Profile.CredentialVersion)
	if err != nil {
		if err == credential.ErrChanged || err == credential.ErrNotFound {
			return QualifiedRoute{}, RouteNeedsConfiguration, ErrNotDispatched
		}
		return QualifiedRoute{}, RouteUnavailable, ErrNotDispatched
	}
	m := row.Metadata
	if !m.Enabled || !m.KeyPresent || m.OrganizationID != input.OrganizationID || m.ConnectionID != policy.Profile.ConnectionID || strings.TrimSpace(row.APIKey) == "" || m.Reference != policy.AdmittedCredentialVersion || endpointDigest(m.BaseURL) != policy.AdmittedEndpointIdentityDigest {
		return QualifiedRoute{}, RouteNeedsConfiguration, ErrNotDispatched
	}
	switch AdapterKind(policy.Profile.AdapterKind) {
	case AdapterOpenAICompatible:
	case AdapterQwenCompatible:
	case AdapterClaudeNative:
	case AdapterGoogleGenerateContent:
		if !ValidAdapterEndpoint(AdapterGoogleGenerateContent, m.BaseURL) {
			return QualifiedRoute{}, RouteNeedsConfiguration, ErrNotDispatched
		}
	case AdapterGoogleInteractions:
		if !googleinteractions.ValidEndpoint(m.BaseURL) {
			return QualifiedRoute{}, RouteNeedsConfiguration, ErrNotDispatched
		}
	default:
		return QualifiedRoute{}, RouteUnavailable, ErrNotDispatched
	}
	route := QualifiedRoute{Profile: policy.Profile, Endpoint: m.BaseURL, APIKey: row.APIKey}
	if !sameRoute(route, route) {
		return QualifiedRoute{}, RouteUnavailable, ErrNotDispatched
	}
	return route, RouteAvailable, nil
}

// ValidRouteProfile is static protocol qualification, never user permission.
// The selected DashScope model requires its fixed non-thinking policy.
// Google is qualified only for the current title operation and fixed versions.
func ValidRouteShape(p aicapability.ModelProfile, operation aicapability.Operation) bool {
	if p.ValidateShape() != nil || !validFixedAdapterPolicy(AdapterKind(p.AdapterKind), p.ProviderID, p.ModelID, p.AdapterPolicyVersion) {
		return false
	}
	if (AdapterKind(p.AdapterKind) == AdapterGoogleGenerateContent || AdapterKind(p.AdapterKind) == AdapterQwenCompatible ||
		p.AdapterPolicyVersion == aicapability.EinoClaudeTextPolicy) && !ReferenceMappingReady(p) {
		return false
	}
	if p.AdapterPolicyVersion == TitlePilotOpenAIAdapterPolicy {
		return operation == aicapability.OperationProductAgentDecision && p.ImagePromptTokenEstimate == 0 &&
			p.MaximumPromptTokens == TitlePilotPromptTokens && p.MaximumCompletionTokens == TitlePilotCompletionTokens &&
			p.EndpointIdentityDigest == EndpointIdentityDigest(TitlePilotOpenAIEndpoint)
	}
	if p.AdapterPolicyVersion == OpenAIMiniTextAdapterPolicy {
		return (operation == aicapability.OperationAIWorkbenchChatPlan || operation == aicapability.OperationAIWorkbenchStoreRoute || operation == aicapability.OperationProductContentDecision) &&
			p.EndpointIdentityDigest == EndpointIdentityDigest(TitlePilotOpenAIEndpoint) && p.UsageMappingVersion == "openai-text-v1" &&
			p.ImagePromptTokenEstimate == 0 && p.MaximumCompletionTokens <= 4096
	}
	if aicapability.VisionTextOperation(operation) {
		return aicapability.SingleImageProfileShape(p)
	}
	if aicapability.SingleImagePurpose(p.AdapterPolicyVersion) || p.ImagePromptTokenEstimate != 0 {
		return false
	}
	if AdapterKind(p.AdapterKind) != AdapterGoogleInteractions {
		return true
	}
	return operation == aicapability.OperationProductAgentDecision && p.ProviderID == "google" && p.ModelID == googleinteractions.Model &&
		p.AdapterPolicyVersion == googleinteractions.AdapterPolicyVersion && p.UsageMappingVersion == googleinteractions.UsageMappingVersion &&
		p.MaximumInputBytes <= 128<<10 && p.MaximumOutputBytes <= 256<<10
}

var _ interface {
	Resolve(context.Context, aicapability.TextInputIdentity) (QualifiedRoute, error)
} = (*OrganizationRouteResolver)(nil)

func ValidRouteProfile(p aicapability.ModelProfile, operation aicapability.Operation) bool {
	if p.AdapterPolicyVersion == OpenAIMiniTextAdapterPolicy && (p.OfficialTariff.Channel != "openai-direct" || !p.CNYRatesCoverUSDTariff()) {
		return false
	}
	return p.Validate() == nil && ValidRouteShape(p, operation) &&
		((p.AdapterKind != string(AdapterGoogleGenerateContent) && p.AdapterKind != string(AdapterQwenCompatible) &&
			p.AdapterPolicyVersion != aicapability.EinoClaudeTextPolicy && p.AdapterPolicyVersion != aicapability.EinoOpenAIImagePolicy) ||
			p.OfficialTariff.IsReferenceEstimate())
}
