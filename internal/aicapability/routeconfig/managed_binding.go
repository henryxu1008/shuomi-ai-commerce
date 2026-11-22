package routeconfig

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
)

const ManagedTextBindingVersion = "v1"

type ManagedBinding struct {
	OrganizationID string `json:"organizationId"`
	BindingID      string `json:"bindingId"`
	ConnectionID   string `json:"connectionId"`
	Template       string `json:"template"`
	Enabled        bool   `json:"enabled"`
}

func (b ManagedBinding) Reference() ModelBindingReference {
	return ModelBindingReference{ID: b.BindingID, Version: ManagedTextBindingVersion}
}

type ManagedBindingInput struct {
	Action           string `json:"action"`
	BindingID        string `json:"bindingId,omitempty"`
	ExpectedRevision int64  `json:"expectedRevision,string"`
}

type ManagedBindingCommand struct {
	OrganizationID string
	ConnectionID   string
	ActorID        string
	RequestKey     string
	Input          ManagedBindingInput
}

func (c ManagedBindingCommand) BindingID() string {
	if c.Input.Action == "disable" {
		return c.Input.BindingID
	}
	return aicapability.ScopedCommandID("um-", "attach-text-model-v1", c.OrganizationID, c.ActorID, c.RequestKey)
}

func IsManagedBindingID(id string) bool {
	return aicapability.IsScopedCommandID("um-", id)
}

type ManagedBindingReader interface {
	ManagedBinding(context.Context, string, string) (ManagedBinding, error)
	ManagedBindings(context.Context, string) ([]ManagedBinding, error)
}

type ManagedBindingWriter interface {
	WriteManagedBinding(context.Context, ManagedBindingCommand, func(string) error) (ManagedBinding, error)
	ManagedBindingReceipt(context.Context, string, string, string) (ManagedBinding, error)
}
