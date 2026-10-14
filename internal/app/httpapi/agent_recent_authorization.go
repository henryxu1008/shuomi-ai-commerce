package httpapi

import (
	"errors"
	"task-processor/internal/agentconfig"
	"task-processor/internal/knowledge"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/operationscockpit"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/review"
)

// Only an explicit resource denial can hide a recent run. Dependency failures
// retain their error so the caller cannot report them as empty history.
func recentRunAccessError(err error) error {
	for _, denied := range []error{agentconfig.ErrForbidden, agentconfig.ErrNotFound, collection.ErrForbidden, collection.ErrNotFound, review.ErrForbidden, review.ErrNotFound, preparation.ErrForbidden, preparation.ErrNotFound, operationscockpit.ErrForbidden, operationscockpit.ErrNotFound, knowledge.ErrForbidden, knowledge.ErrNotFound} {
		if errors.Is(err, denied) {
			return agentconfig.ErrForbidden
		}
	}
	return err
}
