package agent

import (
	"encoding/hex"
	"github.com/google/uuid"
)

// SkillVersionRef is frozen method identity, never an authorization credential.
// The runtime carries it without importing the Skill owner or reading files.
type SkillVersionRef struct {
	Kind          string `json:"kind"`
	ID            string `json:"id"`
	OwnerKind     string `json:"ownerKind"`
	OwnerID       string `json:"ownerId"`
	Version       uint64 `json:"version"`
	Digest        string `json:"digest"`
	PolicyVersion string `json:"policyVersion"`
}

func (r SkillVersionRef) Absent() bool        { return r == SkillVersionRef{} }
func (r SkillVersionRef) ValidOrAbsent() bool { return r.Absent() || r.Valid() }
func (r SkillVersionRef) Valid() bool {
	id, e := uuid.Parse(r.ID)
	digest, de := hex.DecodeString(r.Digest)
	return r.Kind == "skill-version-v1" && e == nil && id != uuid.Nil && id.String() == r.ID && r.Version > 0 && r.Version < 1<<63 && ((r.OwnerKind == "USER" || r.OwnerKind == "ORGANIZATION") && ValidID(r.OwnerID) || r.OwnerKind == "PLATFORM" && r.OwnerID == "official") && de == nil && len(digest) == 32 && hex.EncodeToString(digest) == r.Digest && r.PolicyVersion == "skill-method-v1"
}
