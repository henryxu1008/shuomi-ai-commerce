package httpapi

import (
	"context"
	"errors"
	"task-processor/internal/authidentity"
	zitadelruntime "task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	"task-processor/internal/knowledge"
	knowledgehttp "task-processor/internal/knowledge/httpapi"
	"task-processor/internal/workbenchcontext"
)

// WithKnowledge borrows a service prepared from the explicit Knowledge owner
// pool; its processor and database lifecycle belong to current-application.
func WithKnowledge(service *knowledge.Service) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.knowledgeServices++; o.knowledge = service }
}
func authorizeKnowledgeOriginalDelete(deps routeAuthDependencies) func(context.Context, knowledge.Scope) error {
	return func(ctx context.Context, scope knowledge.Scope) error {
		initial, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
		token := zitadelruntime.BearerTokenFromContext(ctx)
		if !ok || token == "" || initial.UserID != scope.ActorID || initial.EffectiveOrganizationID != scope.OrganizationID {
			return knowledge.ErrForbidden
		}
		if deps.workbenchVerifier == nil || deps.organizationResolver == nil || deps.authorizer == nil {
			return knowledge.ErrUnavailable
		}
		verified, err := deps.workbenchVerifier.Verify(ctx, token)
		if err != nil {
			if zitadelruntime.IsVerificationInvalid(err) {
				return knowledge.ErrForbidden
			}
			return knowledge.ErrUnavailable
		}
		if verified.UserID != initial.UserID {
			return knowledge.ErrForbidden
		}
		resolved, err := deps.organizationResolver.Resolve(ctx, httproute.OrganizationAccessPolicyLiveWrite, workbenchcontext.ResolveInput{
			Identity: verified, BearerToken: token, RequestedOrganizationID: scope.OrganizationID,
		})
		if err != nil {
			if workbenchIdentityRejected(err) {
				return knowledge.ErrForbidden
			}
			return knowledge.ErrUnavailable
		}
		if resolved.UserID != scope.ActorID || resolved.EffectiveOrganizationID != scope.OrganizationID {
			return knowledge.ErrForbidden
		}
		allowed, err := deps.authorizer.AuthorizeScoped(ctx, resolved.UserID, scope.OrganizationID, resolved.Roles, authz.PermissionWorkbenchKnowledgeManage)
		if err != nil {
			return knowledge.ErrUnavailable
		}
		if !allowed {
			return knowledge.ErrForbidden
		}
		return nil
	}
}

func validateKnowledgeDescriptor(d httproute.Descriptor) error {
	for _, expected := range knowledgehttp.Routes(&knowledgehttp.Handler{}) {
		if d.Method == expected.Method && d.Path == expected.Path && d.Module == expected.Module && d.AuthPolicy == expected.AuthPolicy && d.OrganizationAccessPolicy == expected.OrganizationAccessPolicy && d.OrganizationTargetResolver == nil && d.Permission == expected.Permission && d.RequestTimeout == expected.RequestTimeout && d.RejectUnreadRequestBody == expected.RejectUnreadRequestBody && d.Handler != nil {
			return nil
		}
	}
	return errors.New("knowledge route loses fresh content admission or bounded request contract")
}
