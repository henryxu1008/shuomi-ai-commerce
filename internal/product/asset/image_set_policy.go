package asset

import (
	"context"
	"strconv"
	"task-processor/internal/agent"
)

const ImageSetPolicyAdoptionVersion = "product.image.policy-adopt-v1"

// Persisted explanation only. The current application resolves the exact
// run/config/mode and current source grants before any new inventory write.
type ImageSetPolicyProvenance struct {
	Identity                 agent.ExecutionIdentity        `json:"identity"`
	Mode                     agent.ExecutionMode            `json:"mode"`
	ModeRevision             string                         `json:"modeRevision"`
	ModeSnapshotRef          agent.ModeSnapshotRef          `json:"modeSnapshotRef"`
	ConfigurationSnapshotRef agent.ConfigurationSnapshotRef `json:"configurationSnapshotRef"`
	RunID                    string                         `json:"runId"`
	PlanRevision             int64                          `json:"planRevision"`
	ResultDigest             string                         `json:"resultDigest"`
	PolicyVersion            string                         `json:"policyVersion"`
	ActionVersion            string                         `json:"actionVersion"`
}

func (p *ImageSetPolicyProvenance) Valid() bool {
	if p == nil {
		return true
	}
	revision, err := strconv.ParseUint(p.ModeRevision, 10, 63)
	return p.Identity.Valid() && p.Mode == agent.ModeFullAccess && p.ModeSnapshotRef.Valid() && !p.ConfigurationSnapshotRef.Absent() && p.ConfigurationSnapshotRef.ValidOrAbsent() && validIdentityPart(p.RunID) && p.PlanRevision > 0 && imageSetDigest(p.ResultDigest) && err == nil && revision > 0 && strconv.FormatUint(revision, 10) == p.ModeRevision && p.PolicyVersion == agent.ExecutionModePolicyVersion && p.ActionVersion == ImageSetPolicyAdoptionVersion
}

type ImageSetMutationAdmission func(context.Context, SourceSelection, ImageSetCommand) error

func (s *ImageSetService) WithExecutionPolicy(admit ImageSetMutationAdmission) *ImageSetService {
	s.executionPolicy = admit
	return s
}
