package agent

// ScheduledSourceRef is immutable provenance, never an authorization token.
// Only the Workbench transaction binds it to a server-created BusinessTask.
type ScheduledSourceRef struct{ PlanID, OccurrenceID string }

func (r ScheduledSourceRef) Absent() bool { return r == (ScheduledSourceRef{}) }
func (r ScheduledSourceRef) Valid() bool {
	return selectionUUID(r.PlanID) && selectionUUID(r.OccurrenceID)
}
