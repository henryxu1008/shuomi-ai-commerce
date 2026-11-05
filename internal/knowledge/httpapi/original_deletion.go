package httpapi

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"task-processor/internal/knowledge"
)

// ConfigureOriginalDeletionAuthority uses the application's live grant resolver
// again at the deletion dispatch boundary.
func (h *Handler) ConfigureOriginalDeletionAuthority(authorize func(context.Context, knowledge.Scope) error) {
	h.authorizeOriginalDelete = authorize
}

func (h *Handler) OriginalFile(c *gin.Context) {
	s, ok := scope(c)
	if !ok || !noQuery(c) {
		return
	}
	view, err := h.service.OriginalFile(c.Request.Context(), s, c.Param("source_id"))
	if err != nil {
		failure(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *Handler) originalMutation(c *gin.Context, resume bool) {
	s, ok := scope(c)
	if !ok || !noQuery(c) {
		return
	}
	if !emptyBody(c) || h.authorizeOriginalDelete == nil {
		failure(c, knowledge.ErrInvalid)
		return
	}
	authorize := func(ctx context.Context) error { return h.authorizeOriginalDelete(ctx, s) }
	id := c.Param("original_id")
	var view knowledge.OriginalFile
	var err error
	if resume {
		keys := c.Request.Header.Values("Idempotency-Key")
		if len(keys) != 1 || keys[0] != id || c.GetHeader("Content-Encoding") != "" {
			failure(c, knowledge.ErrInvalid)
			return
		}
		view, err = h.service.ContinueOriginalDeletion(c.Request.Context(), s, c.Param("source_id"), id, authorize)
	} else {
		cmd, ok := command(c, s, "original_delete")
		if !ok {
			return
		}
		view, err = h.service.DeleteOriginal(c.Request.Context(), s, cmd.SourceID, id, cmd.Key, cmd.Version, authorize)
	}
	if err != nil {
		failure(c, err)
		return
	}
	status := http.StatusAccepted
	if view.State == knowledge.OriginalReleased {
		status = http.StatusOK
	}
	c.JSON(status, view)
}

func (h *Handler) DeleteOriginalFile(c *gin.Context)       { h.originalMutation(c, false) }
func (h *Handler) ContinueOriginalDeletion(c *gin.Context) { h.originalMutation(c, true) }
