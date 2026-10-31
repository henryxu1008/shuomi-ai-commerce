// Package fulfillment defines bounded commands. Quantities remain native ERPNext facts.
package fulfillment

import (
	"context"
	"encoding/json"
	"errors"
)

var (
	ErrInvalid     = errors.New("invalid fulfillment command")
	ErrForbidden   = errors.New("fulfillment permission changed")
	ErrConflict    = errors.New("fulfillment source or operation changed")
	ErrUnknown     = errors.New("fulfillment result unknown; read the original operation")
	ErrUnavailable = errors.New("fulfillment is not configured or unavailable")
)

type Scope struct{ OrganizationID, ActorID, MemberID string }
type Command struct {
	Store        string          `json:"store"`
	SourceKind   string          `json:"source_kind"`
	SourceID     string          `json:"source_id"`
	OperationID  string          `json:"operation_id"`
	Organization string          `json:"organization"`
	Actor        string          `json:"actor"`
	Member       string          `json:"member"`
	Binding      string          `json:"binding"`
	Command      string          `json:"command"`
	Payload      json.RawMessage `json:"payload"`
}
type Receipt struct {
	Operation string          `json:"operation"`
	Stage     string          `json:"stage"`
	Target    string          `json:"target"`
	Revision  string          `json:"revision"`
	Result    json.RawMessage `json:"result"`
}
type ERP interface {
	Execute(context.Context, Command) (json.RawMessage, error)
}
type Authority interface {
	Check(context.Context, Scope, string, string, string) error
}

func Permission(command string) string {
	switch command {
	case "list", "details", "delivery_details", "catalog", "read", "options", "configuration":
		return "fulfillment.read"
	case "bind_demand", "create_purchase", "record_external_order":
		return "fulfillment.purchase"
	case "receive", "reserve", "pack", "dispatch":
		return "fulfillment.stock"
	case "read_private":
		return "fulfillment.address"
	case "prepare", "confirm", "ensure", "cancel", "claim", "mark_dispatching", "record_result", "load_operation":
		return "fulfillment.execute"
	}
	return ""
}
