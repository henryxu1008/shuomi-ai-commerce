package sourcing

import "context"

// OwnHistoryReadStore exposes only immutable Source evidence. Its caller derives
// the subject from an authorized Collection item, never from browser identity.
type OwnHistoryReadStore interface {
	PublicationReadStore
	ListOwnVersions(context.Context, PublicationScope, string, uint64, OwnHistoryQuery) ([]OwnHistoryReference, error)
}
type OwnHistoryQuery struct {
	Before, Version uint64
	Limit           int
}
type OwnHistoryReference struct {
	PublicationID string
	Version       uint64
}
