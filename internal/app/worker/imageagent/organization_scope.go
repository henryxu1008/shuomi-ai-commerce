package imageagentworker

import (
	"context"
	"task-processor/internal/authidentity"
	zitadel "task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	"task-processor/internal/imageagent"
	openai "task-processor/internal/integration/openai"
	"task-processor/internal/workbenchcontext"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// OrganizationExecutionAuthorizer is explicitly assembled; no default worker
// installs it or obtains service credentials on behalf of the process.
type OrganizationExecutionAuthorizer struct {
	Client             *zitadel.AuthorizationClient
	ServiceToken       func(context.Context) (string, error)
	ProjectID          string
	Authorizer         *authz.ListingKitAuthorizer
	OrganizationStatus workbenchcontext.OrganizationBusinessStatusChecker
}

func (a OrganizationExecutionAuthorizer) SetRolePolicyReader(reader authz.RolePolicyReader) {
	if a.Authorizer != nil {
		a.Authorizer.SetRolePolicyReader(reader)
	}
}

func (a OrganizationExecutionAuthorizer) AuthorizeExecution(ctx context.Context, identity imageagent.ExecutionIdentity) error {
	if a.Client == nil || a.ServiceToken == nil || a.Authorizer == nil || a.ProjectID == "" {
		return imageagent.ErrIdentityRequired
	}
	if err := imageagent.ValidateOrganizationExecution(identity, identity.RunID); err != nil {
		return err
	}
	token, err := a.ServiceToken(ctx)
	if err != nil {
		return imageagent.ErrIdentityRequired
	}
	grant, err := a.Client.ReadExactServiceProjectAuthorization(ctx, token, identity.UserID, a.ProjectID, identity.TenantID)
	if err != nil {
		return imageagent.ErrIdentityRequired
	}
	if a.OrganizationStatus != nil {
		suspended, err := a.OrganizationStatus.IsOrganizationSuspended(ctx, identity.TenantID)
		if err != nil || suspended {
			return imageagent.ErrIdentityRequired
		}
	}
	if grant.Found && grant.State == "STATE_ACTIVE" && grant.AuthorizationID == identity.MemberID && authz.AllowedOrganization(ctx, a.Authorizer, identity.UserID, identity.TenantID, grant.Roles, authz.PermissionImageAgentWrite) {
		return nil
	}
	return imageagent.ErrIdentityRequired
}

// BuildOrganizationImageCapabilities reuses the current production ports and
// adapters with a mandatory organization credential resolver.
func BuildOrganizationImageCapabilities(manager *openai.Manager, db *gorm.DB, logger *logrus.Logger) (ImageCapabilities, error) {
	if manager == nil || db == nil || logger == nil {
		return ImageCapabilities{}, imageagent.ErrIdentityRequired
	}
	manager.SetConfigResolver(organizationCredentialAdmission{resolver: openai.NewOrganizationCredentialResolver(db), logger: safeReviewTransportLogger{logger: logger}})
	provider, err := newRoutedOpenAIProductImageProvider(manager)
	if err != nil {
		return ImageCapabilities{}, err
	}
	resolver, err := loadEmbeddedImagePolicyResolver()
	if err != nil {
		return ImageCapabilities{}, err
	}
	return buildImageCapabilities(providerDependencies{Subject: provider, WhiteBackground: provider, Scene: provider, Review: provider}, resolver)
}

var _ imageagent.ExecutionAuthorizer = OrganizationExecutionAuthorizer{}

// App owns the authenticated-to-provider identity boundary. The Integration
// resolver consumes only its existing shared identity contract and never
// imports application authentication policy.
type organizationCredentialAdmission struct {
	resolver openai.ClientConfigResolver
	logger   openai.Logger
}

func (r organizationCredentialAdmission) ResolveClientConfig(ctx context.Context, name string, fallback *openai.ClientConfig) (*openai.ResolvedClientConfig, error) {
	identity := openai.IdentityFromContext(ctx)
	verified, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || verified.EffectiveOrganizationID == "" || verified.EffectiveOrganizationID != identity.TenantID || verified.TenantID != identity.TenantID || verified.UserID != identity.UserID || r.resolver == nil {
		return nil, openai.ErrClientConfigurationUnavailable
	}
	resolved, err := r.resolver.ResolveClientConfig(ctx, name, fallback)
	if err == nil && resolved != nil && resolved.Config != nil && r.logger != nil {
		resolved.Config.Logger = r.logger
	}
	return resolved, err
}

type safeReviewTransportLogger struct{ logger *logrus.Logger }

func (l safeReviewTransportLogger) Debug(string, map[string]any) {}
func (l safeReviewTransportLogger) Info(string, map[string]any)  {}
func (l safeReviewTransportLogger) Warn(string, map[string]any) {
	l.logger.WithField("event", "organization_provider_transport_warning").Warn("organization provider transport warning")
}
func (l safeReviewTransportLogger) Error(string, map[string]any) {
	l.logger.WithField("event", "organization_provider_transport_error").Error("organization provider transport error")
}
