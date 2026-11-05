package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"task-processor/internal/agent"
	"task-processor/internal/aiworkbench"
	pc "task-processor/internal/aiworkbench/projectcenter"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	ps "task-processor/internal/integration/persistence/aiworkbench/projectcenter"
	"task-processor/internal/storecenter"
)

type workbenchScopeSources struct {
	reader      aiworkbench.ScopeReader
	projectPage func(context.Context, aiworkbench.Scope, string, bool) ([]pc.Project, string, error)
	storePage   func(context.Context, aiworkbench.Scope, int) (storecenter.StorePage, error)
}

func (a *aiWorkbenchApplication) scopeIdentity(ctx context.Context, owner aiworkbench.Scope, permission string) (context.Context, error) {
	i, err := a.freshWorkbenchIdentity(ctx, permission)
	original, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if err != nil || !ok || i.TenantID != owner.OrganizationID || i.UserID != owner.ActorID ||
		i.EffectiveMemberID == "" || original.EffectiveMemberID != i.EffectiveMemberID {
		return nil, aiworkbench.ErrNotFound
	}
	return authidentity.WithAuthenticatedIdentity(ctx, i), nil
}

func (a *aiWorkbenchApplication) attachStoreScopes(stores *storecenter.MemberScopedStoreRepository) {
	a.scopes.reader.Store = func(ctx context.Context, owner aiworkbench.Scope, id string) (aiworkbench.StoreScopeFacts, error) {
		ctx, err := a.scopeIdentity(ctx, owner, authz.PermissionWorkbenchStoreRead)
		if err != nil {
			return aiworkbench.StoreScopeFacts{}, err
		}
		s, err := stores.Get(ctx, owner.OrganizationID, id)
		if errors.Is(err, storecenter.ErrNotFound) {
			return aiworkbench.StoreScopeFacts{}, aiworkbench.ErrNotFound
		}
		if err != nil || s == nil {
			return aiworkbench.StoreScopeFacts{}, aiworkbench.ErrUnavailable
		}
		return aiworkbench.StoreScopeFacts{ID: s.ID(), Name: s.Name(), Platform: string(s.Platform()), Region: s.Region(), Version: uint64(s.Version())}, nil
	}
	a.scopes.storePage = func(ctx context.Context, owner aiworkbench.Scope, page int) (storecenter.StorePage, error) {
		ctx, err := a.scopeIdentity(ctx, owner, authz.PermissionWorkbenchStoreRead)
		if err != nil {
			return storecenter.StorePage{}, err
		}
		return stores.List(ctx, owner.OrganizationID, storecenter.StoreListQuery{Page: page, PageSize: 20})
	}
}

func (a *aiWorkbenchApplication) attachProjectScopes(projects *ps.Store) {
	a.scopes.reader.Project = func(ctx context.Context, owner aiworkbench.Scope, id string) (aiworkbench.ProjectScopeFacts, error) {
		ctx, err := a.scopeIdentity(ctx, owner, pc.PermissionRead)
		if err != nil {
			return aiworkbench.ProjectScopeFacts{}, err
		}
		p, err := projects.GetMetadata(ctx, pc.Scope{OrganizationID: owner.OrganizationID, ActorID: owner.ActorID}, id)
		if errors.Is(err, pc.ErrNotFound) {
			return aiworkbench.ProjectScopeFacts{}, aiworkbench.ErrNotFound
		}
		if err != nil {
			return aiworkbench.ProjectScopeFacts{}, aiworkbench.ErrUnavailable
		}
		return aiworkbench.ProjectScopeFacts{ID: p.ID, Title: p.Title, Goal: p.Goal, Revision: p.Revision, Archived: p.Archived, StoreID: p.StoreID}, nil
	}
	a.scopes.projectPage = func(ctx context.Context, owner aiworkbench.Scope, after string, includeArchived bool) ([]pc.Project, string, error) {
		ctx, err := a.scopeIdentity(ctx, owner, pc.PermissionRead)
		if err != nil {
			return nil, "", err
		}
		return projects.ListWorkScopeOptions(ctx, pc.Scope{OrganizationID: owner.OrganizationID, ActorID: owner.ActorID}, after, includeArchived)
	}
}

func (a *aiWorkbenchApplication) scopeOptions(c *gin.Context, ctx context.Context, owner aiworkbench.Scope, forTasks bool) {
	query := c.Request.URL.Query()
	for k, values := range query {
		if len(values) != 1 || k != "kind" && k != "after" && k != "projectId" && k != "storeId" {
			writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
			return
		}
	}
	kind, after := c.Query("kind"), c.Query("after")
	selection := aiworkbench.ScopeSelection{Kind: kind, ProjectID: c.Query("projectId"), StoreID: c.Query("storeId")}
	type option struct {
		aiworkbench.ScopeView
		Confirmation string `json:"confirmation"`
	}
	options := []option{}
	read := a.scopes.reader.Preview
	if forTasks {
		read = a.scopes.reader.FilterOption
	}
	add := func(sel aiworkbench.ScopeSelection) error {
		view, confirmation, err := read(ctx, owner, sel)
		if err == nil {
			options = append(options, option{view, confirmation})
		}
		return err
	}
	next := ""
	incomplete := false
	var err error
	switch {
	case selection.Valid() && after == "":
		err = add(selection)
	case kind == "project" && selection.ProjectID == "" && selection.StoreID == "":
		if a.scopes.projectPage == nil {
			err = aiworkbench.ErrUnavailable
			break
		}
		var rows []pc.Project
		rows, next, err = a.scopes.projectPage(ctx, owner, after, forTasks)
		for _, row := range rows {
			incomplete = add(aiworkbench.ScopeSelection{Kind: kind, ProjectID: row.ID}) != nil || incomplete
		}
	case kind == "store" && selection.ProjectID == "" && selection.StoreID == "":
		if a.scopes.storePage == nil {
			err = aiworkbench.ErrUnavailable
			break
		}
		page := 1
		if after != "" {
			page, err = strconv.Atoi(after)
			if err != nil || page < 2 || page > 10000 || strconv.Itoa(page) != after {
				err = aiworkbench.ErrInvalid
				break
			}
		}
		var rows storecenter.StorePage
		rows, err = a.scopes.storePage(ctx, owner, page)
		for _, row := range rows.Stores {
			incomplete = add(aiworkbench.ScopeSelection{Kind: kind, StoreID: row.ID()}) != nil || incomplete
		}
		if int64(page)*20 < rows.Total {
			next = strconv.Itoa(page + 1)
		}
	default:
		err = aiworkbench.ErrInvalid
	}
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	workbenchReply(c, http.StatusOK, gin.H{"options": options, "next": next, "incomplete": incomplete})
}

func (a *aiWorkbenchApplication) listScopedTasks(c *gin.Context, ctx context.Context, owner aiworkbench.Scope) {
	filter := aiworkbench.ScopeSelection{Kind: c.Query("scopeKind"), ProjectID: c.Query("projectId"), StoreID: c.Query("storeId")}
	limit, err := workbenchSize(c)
	if err != nil || !filter.ValidTaskFilter() {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	if filter.Kind == "project" {
		if a.scopes.reader.Project == nil {
			writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
			return
		}
		if _, err := a.scopes.reader.Project(ctx, owner, filter.ProjectID); err != nil {
			writeAIWorkbenchError(c, err)
			return
		}
	}
	if filter.Kind == "store" {
		if a.scopes.reader.Store == nil {
			writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
			return
		}
		if _, err := a.scopes.reader.Store(ctx, owner, filter.StoreID); err != nil {
			writeAIWorkbenchError(c, err)
			return
		}
	}
	page, err := a.store.ListScopedTasks(ctx, owner, filter, c.Query("after"), limit, func(s *aiworkbench.BusinessScope, task aiworkbench.BusinessTask) bool {
		if s == nil {
			return true
		}
		f, err := a.authorizedFrozenScope(ctx, owner, task.WorkbenchContextRef, false)
		return err == nil && f.Scope == *s && f.ConversationID == task.ConversationID
	})
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	views := make([]workbenchTaskView, 0, len(page.Tasks))
	for _, task := range page.Tasks {
		view, err := a.taskView(ctx, owner, task, false)
		if err != nil || view.ScopeUnavailable {
			// A late revocation may have invalidated a visible cursor anchor.
			page.Incomplete, page.Next = true, ""
			continue
		}
		views = append(views, view)
	}
	workbenchReply(c, http.StatusOK, gin.H{"tasks": views, "next": page.Next, "incomplete": page.Incomplete})
}

func (a *aiWorkbenchApplication) authorizedConversation(ctx context.Context, owner aiworkbench.Scope, conversation aiworkbench.Conversation) (aiworkbench.Conversation, bool) {
	if conversation.BusinessScope == nil {
		return conversation, true
	}
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || a.store.ConversationMember(ctx, owner, conversation.ID, i.EffectiveMemberID) != nil {
		conversation.Title, conversation.BusinessScope = "", nil
		return conversation, false
	}
	if _, err := a.scopes.reader.View(ctx, owner, *conversation.BusinessScope); err != nil {
		conversation.Title, conversation.BusinessScope = "", nil
		return conversation, false
	}
	return conversation, true
}

func (a *aiWorkbenchApplication) authorizedFrozenScope(ctx context.Context, owner aiworkbench.Scope, ref agent.WorkbenchContextRef, freshInput bool) (aiworkbench.FrozenScope, error) {
	f, err := a.store.ReadFrozenScope(ctx, owner, ref)
	if err != nil {
		return aiworkbench.FrozenScope{}, err
	}
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || i.TenantID != owner.OrganizationID || i.UserID != owner.ActorID {
		return aiworkbench.FrozenScope{}, aiworkbench.ErrNotFound
	}
	if err = a.scopes.reader.Check(ctx, f, i.EffectiveMemberID, freshInput); err != nil {
		return aiworkbench.FrozenScope{}, err
	}
	return f, nil
}
