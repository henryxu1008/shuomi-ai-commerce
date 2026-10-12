package agentconfig

import (
	"context"
	"encoding/hex"
	"math"
	"strconv"
	"time"

	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
)

type ModeSubjectKind string

const (
	RegisteredAgentSubject ModeSubjectKind = "registered_agent"
	PrivateDeliverySubject ModeSubjectKind = "private_delivery"
)

type ModeSubject struct {
	Kind ModeSubjectKind `json:"kind"`
	ID   string          `json:"id"`
}

func (s ModeSubject) Valid() bool {
	return s.Kind == RegisteredAgentSubject && agent.ValidID(s.ID) || s.Kind == PrivateDeliverySubject && UUID(s.ID)
}

// ResolvedModeSubject is constructed only by the current registration/delivery
// owner. A mode record or an HTTP ID does not prove a delivery or activation.
type ResolvedModeSubject struct {
	Subject    ModeSubject
	Definition commercetool.AgentRef
}

func (s ResolvedModeSubject) Valid() bool {
	if !s.Subject.Valid() {
		return false
	}
	if s.Subject.Kind == RegisteredAgentSubject {
		return s.Definition.Validate() == nil && s.Subject.ID == s.Definition.ID
	}
	// Delivery versions belong to the private-delivery owner (currently 2.0.0),
	// not to public Commerce Tool registration's v-prefixed version contract.
	return agent.ValidID(s.Definition.ID) && agent.ValidID(s.Definition.Version)
}

func ParseModeRevision(value string) (uint64, error) {
	n, err := strconv.ParseUint(value, 10, 63)
	if err != nil || strconv.FormatUint(n, 10) != value {
		return 0, ErrInvalid
	}
	return n, nil
}

type ExecutionModeRecord struct {
	Subject   ModeSubject         `json:"subject"`
	Mode      agent.ExecutionMode `json:"mode"`
	Revision  string              `json:"revision"`
	UpdatedBy string              `json:"updatedBy,omitempty"`
	UpdatedAt time.Time           `json:"updatedAt,omitzero"`
}

type ModeCommand struct {
	Identity         agent.ExecutionIdentity
	Subject          ResolvedModeSubject
	Key              string
	ExpectedRevision uint64
	Mode             agent.ExecutionMode
}

func (c ModeCommand) Validate() error {
	if !c.Identity.Valid() || !c.Subject.Valid() || !UUID(c.Key) || !c.Mode.Valid() || c.ExpectedRevision > math.MaxInt64 {
		return ErrInvalid
	}
	return nil
}

type ModeReceipt struct {
	UpdatedBy      string              `json:"updatedBy,omitempty"`
	UpdatedAt      time.Time           `json:"updatedAt,omitzero"`
	CommandID      string              `json:"commandId"`
	RequestKey     string              `json:"requestKey"`
	Operation      string              `json:"operation"`
	Subject        ModeSubject         `json:"subject"`
	Mode           agent.ExecutionMode `json:"mode"`
	BeforeRevision string              `json:"beforeRevision"`
	Revision       string              `json:"revision"`
	Noop           bool                `json:"noop"`
	CommittedAt    time.Time           `json:"committedAt"`
}

// IntentDigest binds all exact consumer input, including its content/versions,
// without copying Product, Asset or private-delivery facts into configuration.
type ModeBinding struct {
	ContextKind, ContextID, RequestKey, IntentDigest string
}

func (b ModeBinding) Valid() bool {
	d, err := hex.DecodeString(b.IntentDigest)
	return agent.ValidID(b.ContextKind) && agent.ValidID(b.ContextID) && UUID(b.RequestKey) && err == nil && len(d) == 32 && hex.EncodeToString(d) == b.IntentDigest
}

type ModeStartCommand struct {
	Identity         agent.ExecutionIdentity
	Subject          ResolvedModeSubject
	Binding          ModeBinding
	ExpectedRevision uint64
}

func (c ModeStartCommand) Validate() error {
	if !c.Identity.Valid() || !c.Subject.Valid() || !c.Binding.Valid() || c.ExpectedRevision > math.MaxInt64 {
		return ErrInvalid
	}
	return nil
}

type ModeSnapshot struct {
	ID            string
	Digest        string `json:"-"`
	Identity      agent.ExecutionIdentity
	Subject       ResolvedModeSubject
	Binding       ModeBinding
	Mode          agent.ExecutionMode
	Revision      string
	PolicyVersion string
	CreatedAt     time.Time
}

func (s ModeSnapshot) Ref() agent.ModeSnapshotRef {
	return agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: s.ID, Digest: s.Digest}
}

type ModeRepository interface {
	ReadExecutionMode(context.Context, agent.ExecutionIdentity, ResolvedModeSubject) (ExecutionModeRecord, error)
	ExecuteMode(context.Context, ModeCommand, ...func(context.Context) error) (ModeReceipt, error)
	PrepareMode(context.Context, ModeStartCommand) (ModeSnapshot, error)
	LoadModeSnapshot(context.Context, agent.ExecutionIdentity, agent.ModeSnapshotRef) (ModeSnapshot, error)
	CheckModeSnapshot(context.Context, agent.ExecutionIdentity, agent.ModeSnapshotRef) (ModeSnapshot, error)
}
