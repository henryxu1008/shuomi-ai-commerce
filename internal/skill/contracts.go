package skill

import (
	"context"
	"errors"
	"task-processor/internal/agent"
	"time"
)

var (
	ErrForbidden   = errors.New("SKILL_FORBIDDEN")
	ErrNotFound    = errors.New("SKILL_NOT_FOUND")
	ErrConflict    = errors.New("SKILL_IDEMPOTENCY_CONFLICT")
	ErrRevision    = errors.New("SKILL_REVISION_MISMATCH")
	ErrUnavailable = errors.New("SKILL_UNAVAILABLE")
	ErrNotEnabled  = errors.New("SKILL_NOT_ENABLED")
)

type Owner struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

const (
	UserOwner         = "USER"
	OrganizationOwner = "ORGANIZATION"
	PlatformOwner     = "PLATFORM"
	PlatformID        = "official"
)

type Principal struct {
	UserID, OrganizationID, MemberID                                             string
	ReadOrganization, UseOrganization, ConfigureOrganization, PlatformMaintainer bool
}
type Metadata struct {
	ID          string    `json:"id"`
	Owner       Owner     `json:"owner"`
	Creator     string    `json:"creator"`
	HeadVersion uint64    `json:"headVersion"`
	Revision    uint64    `json:"revision"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
type Version struct {
	Metadata    Metadata               `json:"skill"`
	Number      uint64                 `json:"version"`
	Package     Package                `json:"package"`
	State       string                 `json:"state"`
	Visibility  string                 `json:"visibility"`
	AllowCopy   bool                   `json:"allowCopy"`
	AllowExport bool                   `json:"allowExport"`
	Source      *agent.SkillVersionRef `json:"source,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
}

const (
	Draft    = "DRAFT"
	Ready    = "READY"
	Enabled  = "ENABLED"
	Disabled = "DISABLED"
	Revoked  = "REVOKED"
	Private  = "PRIVATE"
	Shared   = "SHARED"
)

type Action string

const (
	Read    Action = "read"
	Use     Action = "use"
	Edit    Action = "edit"
	Publish Action = "publish"
	Enable  Action = "enable"
	Copy    Action = "copy"
	Export  Action = "export"
)

func Authorized(p Principal, v Version, a Action) bool {
	if !agent.ValidID(p.UserID) || !v.Metadata.Owner.Valid() {
		return false
	}
	own, maintain, read, use := false, false, false, false
	switch o := v.Metadata.Owner; o.Kind {
	case UserOwner:
		own = p.UserID == o.ID
		maintain, read, use = own, own, own
	case OrganizationOwner:
		if p.OrganizationID != o.ID {
			return false
		}
		own = p.UserID == v.Metadata.Creator
		maintain = p.ConfigureOrganization
		read = maintain || own && p.UseOrganization || v.Visibility == Shared && v.State != Draft && v.State != Revoked && p.ReadOrganization
		use = p.UseOrganization && v.Visibility == Shared
	case PlatformOwner:
		maintain = p.PlatformMaintainer
		read = maintain || v.Visibility == Shared && v.State != Draft && v.State != Revoked
		use = v.Visibility == Shared
	}
	switch a {
	case Read:
		return read
	case Use:
		return use && v.State == Enabled
	case Edit:
		return maintain || v.Metadata.Owner.Kind == OrganizationOwner && own && p.UseOrganization && v.Visibility == Private
	case Publish, Enable:
		return maintain
	case Copy:
		return read && v.State != Revoked && v.AllowCopy && v.AllowExport
	case Export:
		return read && v.AllowExport
	default:
		return false
	}
}

func (o Owner) Valid() bool {
	return (o.Kind == UserOwner || o.Kind == OrganizationOwner) && agent.ValidID(o.ID) || o.Kind == PlatformOwner && o.ID == PlatformID
}

const MethodPolicyVersion = "skill-method-v1"

func (v Version) Ref() agent.SkillVersionRef {
	return agent.SkillVersionRef{Kind: "skill-version-v1", ID: v.Metadata.ID, OwnerKind: v.Metadata.Owner.Kind, OwnerID: v.Metadata.Owner.ID, Version: v.Number, Digest: v.Package.Digest, PolicyVersion: MethodPolicyVersion}
}

// Transition changes availability, never the immutable method package.
func Transition(p Principal, v Version, operation string) (string, string, error) {
	if !Authorized(p, v, Enable) {
		return "", "", ErrForbidden
	}
	state, visibility := v.State, v.Visibility
	switch operation {
	case "publish":
		if state != Draft {
			return "", "", ErrConflict
		}
		state = Ready
		if v.Metadata.Owner.Kind != UserOwner {
			visibility = Shared
		}
	case "enable":
		if state != Ready && state != Disabled {
			return "", "", ErrConflict
		}
		state = Enabled
	case "disable":
		if state != Enabled {
			return "", "", ErrConflict
		}
		state = Disabled
	case "revoke":
		if state == Revoked {
			return "", "", ErrConflict
		}
		state = Revoked
	default:
		return "", "", ErrInvalid
	}
	return state, visibility, nil
}

type ResolvePrincipal func(context.Context) (Principal, error)

type Command struct {
	Principal              Principal
	Owner                  Owner
	Key, Operation, ID     string
	Expected               uint64
	Version                uint64
	Package                Package
	AllowCopy, AllowExport bool
	Source                 *agent.SkillVersionRef
}
type Receipt struct {
	ID        string    `json:"skillId"`
	Version   uint64    `json:"version"`
	Revision  uint64    `json:"revision"`
	Operation string    `json:"operation"`
	At        time.Time `json:"at"`
}
type ShareRequest struct {
	ID             string                `json:"id"`
	Source         agent.SkillVersionRef `json:"source"`
	SubmittedBy    string                `json:"submittedBy"`
	OrganizationID string                `json:"organizationId"`
	Consent        bool                  `json:"consent"`
	State          string                `json:"state"`
	ResultID       string                `json:"resultSkillId,omitempty"`
	CreatedAt      time.Time             `json:"createdAt"`
	UpdatedAt      time.Time             `json:"updatedAt"`
}
type Repository interface {
	List(context.Context, Principal, Owner, string, int) ([]Version, string, error)
	Read(context.Context, Principal, Owner, string, uint64) (Version, error)
	Execute(context.Context, Command, ResolvePrincipal) (Receipt, error)
	Share(context.Context, Principal, string, string, uint64, bool, ResolvePrincipal) (ShareRequest, error)
	Shares(context.Context, Principal, bool) ([]ShareRequest, error)
	ReadShared(context.Context, Principal, string) (Version, error)
	DecideShare(context.Context, Principal, string, string, string, ResolvePrincipal) (ShareRequest, error)
}
