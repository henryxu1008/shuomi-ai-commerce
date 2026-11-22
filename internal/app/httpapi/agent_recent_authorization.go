package httpapi

import (
	"errors"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/knowledge"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/operationscockpit"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/review"
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
