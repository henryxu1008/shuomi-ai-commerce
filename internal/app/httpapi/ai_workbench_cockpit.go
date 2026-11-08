package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"task-processor/internal/aiworkbench"
	"task-processor/internal/app/workbenchmetrics"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/product/review"
)

func (a *aiWorkbenchApplication) taskCockpitMetrics(c *gin.Context, ctx context.Context, owner aiworkbench.Scope) {
	now := time.Now()
	filter, err := workbenchmetrics.Parse(c.Request.URL.RawQuery, now)
	if err != nil {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	result, err := a.readTaskCockpitMetrics(ctx, owner, filter, now)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	workbenchReply(c, http.StatusOK, result)
}

func (a *aiWorkbenchApplication) readTaskCockpitMetrics(ctx context.Context, owner aiworkbench.Scope, filter workbenchmetrics.Filter, now time.Time) (gin.H, error) {
	original, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || original.EffectiveMemberID == "" || a.scopes.reader.Store == nil {
		return nil, aiworkbench.ErrUnavailable
	}
	stores := make(map[string]aiworkbench.StoreScopeFacts, len(filter.Stores))
	checkStores := func() error {
		for _, id := range filter.Stores {
			store, err := a.scopes.reader.Store(ctx, owner, id)
			if err != nil || store.ID != id {
				return aiworkbench.ErrUnavailable
			}
			stores[id] = store
		}
		return nil
	}
	if err := checkStores(); err != nil {
		return nil, err
	}
	tasks, err := a.store.ListNewStoreTasks(ctx, owner, original.EffectiveMemberID, filter.Stores, filter.Start, filter.End, workbenchmetrics.MaxCandidates+1)
	if err != nil {
		return nil, err
	}
	result := gin.H{"capturedAt": now.UTC(), "status": "limited"}
	projects := make(map[string]aiworkbench.ProjectScopeFacts)
	checkScope := a.scopes.reader
	checkScope.Store = func(_ context.Context, _ aiworkbench.Scope, id string) (aiworkbench.StoreScopeFacts, error) {
		return stores[id], nil
	}
	if a.scopes.reader.Project != nil {
		checkScope.Project = func(ctx context.Context, scope aiworkbench.Scope, id string) (aiworkbench.ProjectScopeFacts, error) {
			if saved, ok := projects[id]; ok {
				return saved, nil
			}
			facts, err := a.scopes.reader.Project(ctx, scope, id)
			if err == nil {
				projects[id] = facts
			}
			return facts, err
		}
	}
	if len(tasks) <= workbenchmetrics.MaxCandidates {
		counts := map[aiworkbench.TaskState]int{aiworkbench.TaskRunning: 0, aiworkbench.TaskWaitingConfirmation: 0, aiworkbench.TaskCompleted: 0, aiworkbench.TaskError: 0, aiworkbench.TaskPaused: 0, "UNKNOWN": 0}
		reader := a.taskProjectionReader()
		reader.Now = func() time.Time { return now }
		for _, task := range tasks {
			frozen, err := a.store.ReadFrozenScope(ctx, owner, task.WorkbenchContextRef)
			if err != nil || frozen.Owner != owner || frozen.ConversationID != task.ConversationID || stores[frozen.Scope.StoreID].ID == "" {
				return nil, aiworkbench.ErrUnavailable
			}
			if frozen.MemberID != original.EffectiveMemberID {
				continue
			}
			if err := checkScope.Check(ctx, frozen, original.EffectiveMemberID, false); err != nil {
				// The scope reader can collapse a dependency error into a not-found.
				// Do not convert an ambiguous grant failure into a smaller count.
				return nil, aiworkbench.ErrUnavailable
			}
			state := aiworkbench.TaskState("UNKNOWN")
			if a.agent != nil && a.agent.workbenchKind(task.Kind) {
				if projection, err := reader.Read(ctx, owner, task); err == nil {
					if _, known := counts[projection.State]; known {
						state = projection.State
					}
				}
			}
			counts[state]++
		}
		result["status"], result["counts"] = "complete", counts
	}
	fresh, err := a.freshWorkbenchIdentity(ctx, authz.PermissionWorkbenchTaskRead)
	if err != nil {
		return nil, err
	}
	if fresh.UserID != owner.ActorID || fresh.TenantID != owner.OrganizationID || fresh.EffectiveMemberID != original.EffectiveMemberID {
		return nil, review.ErrForbidden
	}
	if err := checkStores(); err != nil {
		return nil, err
	}
	for id := range projects {
		if project, err := a.scopes.reader.Project(ctx, owner, id); err != nil || project.ID != id {
			return nil, aiworkbench.ErrUnavailable
		}
	}
	return result, nil
}
