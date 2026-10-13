package agentconfig

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	"task-processor/internal/agent"
)

const (
	ArtworkAgentID         = "pod.design.agent"
	ArtworkAgentVersion    = "v1.0.0"
	ArtworkParameterSchema = "pod-artwork-config-v1"
	ArtworkSnapshotContext = "pod-artwork-v1"
)

// Artwork templates describe original creation, without product images or
// knowledge bindings. Model, route, price and output limits retain their owners.
type ArtworkTemplate struct {
	Schema      string `json:"schema"`
	Style       string `json:"style"`
	Constraints string `json:"constraints"`
}

func (t ArtworkTemplate) Valid() bool {
	bounded := func(s string, max int) bool {
		return len(s) <= max && utf8.ValidString(s) && strings.IndexFunc(s, func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\t' }) < 0
	}
	return t.Schema == ArtworkParameterSchema && bounded(t.Style, 512) && bounded(t.Constraints, 4096)
}

// Distinct public types prevent the product-image entry points from authorizing
// creation. The existing snapshot and admission rows remain the sole owners.
type ArtworkStartCommand ImageStartCommand

type ArtworkConfigurationSnapshot struct {
	ImageConfigurationSnapshot
	ArtworkParameters ArtworkTemplate `json:"-"`
}

type ArtworkRunAdmissionCommand ImageRunAdmissionCommand
type ArtworkRunAdmissionReceipt ImageRunAdmissionReceipt

type ArtworkConfigurationRepository interface {
	PrepareArtworkConfiguration(context.Context, ArtworkStartCommand) (ArtworkConfigurationSnapshot, error)
	LoadArtworkConfiguration(context.Context, agent.Scope, agent.ConfigurationSnapshotRef) (ArtworkConfigurationSnapshot, error)
	AdmitArtworkRun(context.Context, ArtworkRunAdmissionCommand, ImageRunLimits) (ArtworkRunAdmissionReceipt, error)
	ReadArtworkRunAdmission(context.Context, agent.Scope, agent.ConfigurationSnapshotRef) (ArtworkRunAdmissionReceipt, error)
	CheckArtworkExecutionMode(context.Context, agent.ExecutionIdentity, agent.ConfigurationSnapshotRef, agent.ModeSnapshotRef) (ModeSnapshot, error)
}
