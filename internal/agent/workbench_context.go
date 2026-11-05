package agent

import "regexp"

const WorkbenchContextContract = "workbench-scope-v1"

var workbenchContextDigest = regexp.MustCompile("^[a-f0-9]{64}$")

// WorkbenchContextRef identifies an immutable Workbench command. The Agent
// can carry it, but only the Workbench owner can resolve its input.
type WorkbenchContextRef struct {
	Contract       string
	ConversationID string
	CommandKey     string
	Digest         string
}

func (r WorkbenchContextRef) Absent() bool { return r == (WorkbenchContextRef{}) }
func (r WorkbenchContextRef) Valid() bool {
	return r.Contract == WorkbenchContextContract && ValidID(r.ConversationID) &&
		ValidID(r.CommandKey) && workbenchContextDigest.MatchString(r.Digest)
}
func (r WorkbenchContextRef) ValidOrAbsent() bool { return r.Absent() || r.Valid() }
