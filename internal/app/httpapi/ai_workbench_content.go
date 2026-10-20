package httpapi

import (
	"context"
	"github.com/gin-gonic/gin"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/authidentity"
)

// Read-only selection preview delegates identity and current effective version
// to the original subject owner. The browser cannot choose a product key.
func (a *aiWorkbenchApplication) readTaskSubject(c *gin.Context, ctx context.Context) {
	if a.agent == nil || !a.agent.workbenchKind(aiworkbench.ContentTaskKind) {
		writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
		return
	}
	kind, id := c.Param("context_kind"), c.Param("context_id")
	if !acquisitionHTTPUUID(id) || (kind != "own_product" && kind != "acquisition") || c.Request.URL.RawQuery != "" {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	i, err := a.agent.freshIdentity(ctx)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	var b agent.Binding
	if kind == "own_product" {
		source, e := a.agent.ownSource(ctx, i, id, nil)
		b, err = source.Binding, e
	} else {
		b, err = a.agent.currentAcquisitionBinding(ctx, i, id, "product")
	}
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	if _, err = a.agent.freshIdentity(ctx); err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	workbenchReply(c, 200, gin.H{"subject": aiworkbench.Subject{Kind: b.ContextKind, ID: b.ContextID, CatalogVersion: b.CatalogVersion, PublicationID: b.PublicationID}, "productKey": b.ProductKey})
}

func (a *productAgentApplication) workbenchKind(kind string) bool {
	if a == nil {
		return false
	}
	if kind == aiworkbench.ContentTaskKind {
		return a.definition.ID == agentconfig.ContentAgentID && !a.config.ContentTitleOnly
	}
	return kind == aiworkbench.TitleTaskKind && a.definition.ID == "product.title.agent"
}

// New intent always selects the current effective head. Reading a receipt uses
// the original owner's historical authorization instead of this selector.
func (a *productAgentApplication) workbenchSelectionBinding(ctx context.Context, i authidentity.AuthenticatedIdentity, w aiworkbench.WorkScope) (agent.Binding, error) {
	if w.TaskSelection == nil {
		if !a.workbenchKind(aiworkbench.TitleTaskKind) {
			return agent.Binding{}, aiworkbench.ErrUnavailable
		}
		return a.bindingForIdentity(ctx, i, w.OperationID, w.TargetPlatform)
	}
	s := w.TaskSelection
	if w.OperationID != "" || w.TargetPlatform != "" || !s.Valid() || !a.workbenchKind(s.Capability) {
		return agent.Binding{}, aiworkbench.ErrInvalid
	}
	var b agent.Binding
	var err error
	switch s.Subject.Kind {
	case "own_product":
		source, e := a.ownSource(ctx, i, s.Subject.ID, nil)
		b, err = source.Binding, e
	case "acquisition":
		b, err = a.currentAcquisitionBinding(ctx, i, s.Subject.ID, "product")
	default:
		return b, aiworkbench.ErrInvalid
	}
	if err != nil {
		return b, err
	}
	if b.ContextKind != s.Subject.Kind || b.ContextID != s.Subject.ID || b.CatalogVersion != s.Subject.CatalogVersion || b.PublicationID != s.Subject.PublicationID {
		return agent.Binding{}, aiworkbench.ErrProposalStale
	}
	return b, nil
}
func (a *productAgentApplication) workbenchProposalBinding(ctx context.Context, i authidentity.AuthenticatedIdentity, p aiworkbench.ExecutionProposal) (agent.Binding, error) {
	w := aiworkbench.WorkScope{OperationID: p.OperationID, TargetPlatform: p.TargetPlatform}
	if p.Kind == aiworkbench.ContentTaskKind {
		w = aiworkbench.WorkScope{TaskSelection: p.Selection()}
	}
	return a.workbenchSelectionBinding(ctx, i, w)
}
func (a *productAgentApplication) workbenchCurrentBinding(ctx context.Context, i authidentity.AuthenticatedIdentity, b agent.Binding) (agent.Binding, error) {
	if a.definition.ID != agentconfig.ContentAgentID {
		return a.bindingForIdentity(ctx, i, b.ContextID, b.TargetPlatform)
	}
	var current agent.Binding
	var err error
	if b.ContextKind == "own_product" {
		source, e := a.ownSource(ctx, i, b.ContextID, nil)
		current, err = source.Binding, e
	} else if b.ContextKind == "acquisition" {
		current, err = a.currentAcquisitionBinding(ctx, i, b.ContextID, b.TargetPlatform)
	} else {
		return current, agent.ErrInvalid
	}
	return current, err
}
func (a *productAgentApplication) workbenchAuthorizeBinding(ctx context.Context, i authidentity.AuthenticatedIdentity, b agent.Binding) error {
	if b.ContextKind == "own_product" {
		return a.authorizeOwnBinding(ctx, i, b)
	}
	if b.ContextKind == "acquisition" {
		return a.authorizeAcquisitionBinding(ctx, i, b)
	}
	return agent.ErrInvalid
}
