package fulfillmentapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	sigjson "sigs.k8s.io/json"
	"task-processor/internal/authidentity"
	f "task-processor/internal/fulfillment"
	sf "task-processor/internal/marketplace/shein/fulfillment"
	"task-processor/internal/storecenter"
	"time"
)

type Merchant interface {
	Check(context.Context) error
	BindingDigest() string
	Source(context.Context, string) (sf.Source, error)
}
type Input struct {
	Store       string          `json:"store"`
	Kind        string          `json:"kind"`
	SourceID    string          `json:"source_id"`
	OperationID string          `json:"operation_id"`
	Action      string          `json:"action"`
	Payload     json.RawMessage `json:"payload"`
}
type Service struct {
	ERPs           map[string]f.ERP
	SupplyMovement map[string]string
	Aliases        map[string]string
	Starter        Starter
	Open           func(context.Context, storecenter.FulfillmentSubject, string, string) (Merchant, error)
}
type Configuration struct {
	ERPs           map[string]f.ERP
	Aliases        map[string]string
	SupplyMovement map[string]string
}

func decode(raw []byte, target any) error {
	issues, err := sigjson.UnmarshalStrict(raw, target)
	if err != nil || len(issues) > 0 {
		return f.ErrInvalid
	}
	return nil
}
func version(source sf.Source) string {
	raw, _ := json.Marshal(source)
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}
func (s *Service) Execute(ctx context.Context, scope f.Scope, in Input) (json.RawMessage, error) {
	if s == nil || s.Open == nil || s.ERPs[scope.OrganizationID] == nil {
		return nil, f.ErrUnavailable
	}
	if !authidentity.IsBoundedIdentifier(scope.OrganizationID) || !authidentity.IsBoundedIdentifier(scope.ActorID) || !authidentity.IsBoundedIdentifier(scope.MemberID) || !authidentity.IsBoundedIdentifier(in.Store) || !authidentity.IsBoundedIdentifier(in.SourceID) || in.Kind != "consumer_order" && in.Kind != "platform_supply" || len(in.Payload) > 256*1024 {
		return nil, f.ErrInvalid
	}
	permission := f.Permission(in.Action)
	if in.Action == "source" {
		permission = "fulfillment.read"
	}
	// Worker-only stages never accept an HTTP command.
	switch in.Action {
	case "source", "options", "shipment", "catalog", "bind_demand", "details", "find_demand", "create_purchase", "record_external_order", "close_purchase", "receive", "reserve", "pack", "dispatch", "prepare", "confirm", "ensure", "cancel", "cancel_original", "read", "read_private":
	default:
		return nil, f.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 55*time.Second)
	defer cancel()
	merchant, err := s.Open(ctx, storecenter.FulfillmentSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: permission}, in.Store, in.Kind)
	if err != nil {
		return nil, err
	}
	if err = merchant.Check(ctx); err != nil {
		return nil, err
	}
	if uuid.Validate(in.OperationID) != nil {
		return nil, f.ErrInvalid
	}
	command := f.Command{OperationID: in.OperationID, Organization: scope.OrganizationID, Actor: scope.ActorID, Member: scope.MemberID, Binding: merchant.BindingDigest(), Store: in.Store, SourceKind: in.Kind, SourceID: in.SourceID, Command: in.Action, Payload: in.Payload}
	switch in.Action {
	case "shipment":
		return s.shipment(ctx, scope, in, command, merchant)
	case "catalog", "details", "find_demand", "read", "read_private", "record_external_order", "close_purchase", "receive", "cancel", "cancel_original":
		if err = merchant.Check(ctx); err != nil {
			return nil, err
		}
		return s.ERPs[scope.OrganizationID].Execute(ctx, command)
	case "ensure":
		return s.platformCommand(ctx, scope, in, command, merchant, sf.Source{})
	}
	source, err := merchant.Source(ctx, in.SourceID)
	if err != nil {
		return nil, err
	}
	if source.ID != in.SourceID || source.Kind != in.Kind {
		return nil, f.ErrConflict
	}
	if in.Action == "source" {
		return json.Marshal(struct {
			Source   sf.Source `json:"source"`
			Version  string    `json:"version"`
			Eligible bool      `json:"eligible"`
		}{source, version(source), source.Eligible(time.Now()) == nil})
	}
	if in.Action == "create_purchase" || in.Action == "reserve" || in.Action == "pack" || in.Action == "prepare" || in.Action == "confirm" {
		command.CurrentLines = map[string]int64{}
		for _, line := range source.FulfillableLines() {
			command.CurrentLines[line.ID] = line.Quantity - line.Delivered
		}
	}
	if uuid.Validate(in.OperationID) != nil {
		return nil, f.ErrInvalid
	}
	if in.Action != "catalog" && in.Action != "details" && in.Action != "read" && in.Action != "read_private" && in.Action != "cancel" && in.Action != "ensure" && in.Action != "find_demand" {
		// Recording the actual handover follows this exact original platform
		// shipment, which may already have advanced to the platform's sent state.
		originalHandover := in.Action == "dispatch" && source.Kind == "platform_supply" && (source.Status == 4 || source.Status == 5 || source.Status == 6 || source.Status == 8)
		if err = source.Eligible(time.Now()); err != nil && !originalHandover {
			return nil, err
		}
	}
	if in.Action == "options" || in.Action == "prepare" || in.Action == "confirm" || in.Action == "ensure" {
		return s.platformCommand(ctx, scope, in, command, merchant, source)
	}
	if in.Action == "bind_demand" {
		var mapping struct {
			DeliveryDate string `json:"delivery_date"`
			Lines        []struct {
				SourceLine string `json:"source_line"`
				ItemCode   string `json:"item_code"`
				UOM        string `json:"uom"`
				Warehouse  string `json:"warehouse"`
			} `json:"lines"`
		}
		if decode(in.Payload, &mapping) != nil || len(mapping.Lines) != len(source.FulfillableLines()) {
			return nil, f.ErrInvalid
		}
		original := map[string]sf.Line{}
		for _, line := range source.FulfillableLines() {
			original[line.ID] = line
		}
		rows := []map[string]any{}
		for _, row := range mapping.Lines {
			line, ok := original[row.SourceLine]
			if !ok || line.Quantity-line.Delivered < 1 {
				return nil, f.ErrConflict
			}
			delete(original, row.SourceLine)
			rows = append(rows, map[string]any{"source_line": row.SourceLine, "item_code": row.ItemCode, "uom": row.UOM, "warehouse": row.Warehouse, "qty": line.Quantity - line.Delivered})
		}
		movement := "commercial_sale"
		if in.Kind == "platform_supply" {
			movement = s.SupplyMovement[scope.OrganizationID]
			if movement != "owned_transfer" && movement != "commercial_sale" {
				return nil, f.ErrUnavailable
			}
		}
		command.Payload, _ = json.Marshal(map[string]any{"source_kind": in.Kind, "store": in.Store, "source_id": in.SourceID, "movement_kind": movement, "delivery_date": mapping.DeliveryDate, "lines": rows})
	}
	if err = merchant.Check(ctx); err != nil {
		return nil, err
	}
	return s.ERPs[scope.OrganizationID].Execute(ctx, command)
}
