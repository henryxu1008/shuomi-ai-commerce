package content

import (
	"context"

	"task-processor/internal/shein/authorizedbrand"
)

// CleanupContext supplies only the values needed for content cleanup.
type CleanupContext struct {
	Context         context.Context
	AuthorizedBrand *authorizedbrand.Resolved
	SourceBrand     string
}
