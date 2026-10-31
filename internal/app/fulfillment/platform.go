package fulfillmentapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	f "task-processor/internal/fulfillment"
	"task-processor/internal/storecenter"
	sf "task-processor/internal/storecenter/fulfillment"
	"time"
)

type Starter interface {
	Ensure(context.Context, string, string, string) error
}
type ExecutionMerchant interface {
	Merchant
	Options(context.Context, sf.Source, sf.Selection) (sf.ShippingOptions, error)
	Build(context.Context, sf.Source, sf.Delivery, sf.Selection) (sf.Plan, error)
	Send(context.Context, sf.Plan) (sf.MutationResult, error)
}
type Prepared struct {
	Purpose   string          `json:"purpose"`
	Demand    string          `json:"demand"`
	Delivery  string          `json:"delivery,omitempty"`
	Version   string          `json:"version"`
	Data      json.RawMessage `json:"data"`
	Selection sf.Selection    `json:"selection"`
}
type Loaded struct {
	f.Receipt
	Actor   string   `json:"actor"`
	Member  string   `json:"member"`
	Binding string   `json:"binding"`
	Input   Prepared `json:"input"`
	Source  struct {
		Store        string `json:"store"`
		Kind         string `json:"source_kind"`
		ID           string `json:"source_id"`
		Organization string `json:"organization"`
	} `json:"source"`
}

func preparedVersion(source sf.Source, delivery sf.Delivery, selection sf.Selection, plan sf.Plan) string {
	raw, _ := json.Marshal([]any{source, delivery, selection, plan})
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}
func native(ctx context.Context, erp f.ERP, base f.Command, command string, payload any) (json.RawMessage, error) {
	base.Command = command
	base.OperationID = uuid.NewString()
	base.Payload, _ = json.Marshal(payload)
	return erp.Execute(ctx, base)
}
func load(ctx context.Context, erp f.ERP, base f.Command, id string) (Loaded, error) {
	var loaded Loaded
	raw, err := native(ctx, erp, base, "load_operation", map[string]any{"operation": id})
	if err != nil {
		return loaded, err
	}
	if json.Unmarshal(raw, &loaded) != nil || loaded.Operation != id || loaded.Source.Organization != base.Organization || loaded.Actor == "" || loaded.Member == "" || loaded.Binding == "" || loaded.Target == "" || loaded.Target != loaded.Input.Demand {
		return loaded, f.ErrUnknown
	}
	return loaded, nil
}
func checkedTransition(raw json.RawMessage, original Loaded, stage string) error {
	var result f.Receipt
	if decode(raw, &result) != nil || result.Operation != original.Operation || result.Stage != stage || result.Revision != original.Revision || result.Target != original.Target {
		return f.ErrUnknown
	}
	return nil
}
func (s *Service) freeze(ctx context.Context, erp f.ERP, base f.Command, merchant ExecutionMerchant, source sf.Source, selection sf.Selection) (Prepared, error) {
	if err := eligibleSource(source, time.Now()); err != nil {
		return Prepared{}, err
	}
	var delivery sf.Delivery
	if selection.Purpose == "consumer_tracking" || selection.Purpose == "supply_shipping" {
		raw, err := native(ctx, erp, base, "delivery_details", map[string]any{"demand": selection.Demand, "delivery": selection.Delivery})
		if err != nil {
			return Prepared{}, err
		}
		if json.Unmarshal(raw, &delivery) != nil {
			return Prepared{}, f.ErrUnknown
		}
	}
	if selection.Purpose == "accept_address" || selection.Purpose == "read_address" {
		address, err := s.Open(ctx, storecenter.FulfillmentSubject{OrganizationID: base.Organization, ActorID: base.Actor, MemberID: base.Member, Purpose: "fulfillment.address"}, base.Store, base.SourceKind)
		if err != nil {
			return Prepared{}, err
		}
		if err = address.Check(ctx); err != nil {
			return Prepared{}, err
		}
	}
	plan, err := merchant.Build(ctx, source, delivery, selection)
	if err != nil {
		return Prepared{}, err
	}
	return Prepared{Purpose: selection.Purpose, Demand: selection.Demand, Delivery: selection.Delivery, Version: preparedVersion(source, delivery, selection, plan), Data: plan.Body, Selection: selection}, nil
}
func (s *Service) platformCommand(ctx context.Context, scope f.Scope, in Input, base f.Command, m Merchant, source sf.Source) (json.RawMessage, error) {
	merchant, ok := m.(ExecutionMerchant)
	if !ok {
		return nil, f.ErrUnavailable
	}
	erp := s.ERPs[scope.OrganizationID]
	if in.Action == "options" {
		var selection sf.Selection
		if decode(in.Payload, &selection) != nil {
			return nil, f.ErrInvalid
		}
		options, err := merchant.Options(ctx, source, selection)
		if err != nil {
			return nil, err
		}
		return json.Marshal(options)
	}
	if in.Action == "prepare" {
		var selection sf.Selection
		if decode(in.Payload, &selection) != nil {
			return nil, f.ErrInvalid
		}
		prepared, err := s.freeze(ctx, erp, base, merchant, source, selection)
		if err != nil {
			return nil, err
		}
		base.Payload, _ = json.Marshal(prepared)
		if err = merchant.Check(ctx); err != nil {
			return nil, err
		}
		return erp.Execute(ctx, base)
	}
	var transition struct {
		Operation string `json:"operation"`
		Version   string `json:"version"`
	}
	if decode(in.Payload, &transition) != nil || uuid.Validate(transition.Operation) != nil {
		return nil, f.ErrInvalid
	}
	original, err := load(ctx, erp, base, transition.Operation)
	if err != nil {
		return nil, err
	}
	if original.Actor != scope.ActorID || original.Member != scope.MemberID || original.Binding != base.Binding || original.Source.Store != in.Store || original.Source.Kind != in.Kind || original.Source.ID != in.SourceID || transition.Version != original.Revision {
		return nil, f.ErrConflict
	}
	if original.Stage == "PREPARED" {
		if in.Action != "confirm" {
			return nil, f.ErrConflict
		}
		prepared, err := s.freeze(ctx, erp, base, merchant, source, original.Input.Selection)
		if err != nil {
			return nil, err
		}
		if prepared.Version != original.Input.Version || !bytes.Equal(prepared.Data, original.Input.Data) {
			return nil, f.ErrConflict
		}
		if in.Action != "confirm" {
			return nil, f.ErrConflict
		}
		raw, err := erp.Execute(ctx, base)
		if err != nil {
			return nil, err
		}
		if err = checkedTransition(raw, original, "CONFIRMED"); err != nil {
			return nil, err
		}
		original.Stage = "CONFIRMED"
	}
	if original.Stage == "CONFIRMED" {
		if s.Starter == nil || s.Aliases[scope.OrganizationID] == "" {
			return nil, f.ErrUnavailable
		}
		if err = s.Starter.Ensure(ctx, s.Aliases[scope.OrganizationID], scope.OrganizationID, original.Operation); err != nil {
			return nil, err
		}
	}
	return json.Marshal(original.Receipt)
}

// Run sends at most once. The activity must have MaximumAttempts=1. ERP is the durable owner.
func (s *Service) Run(ctx context.Context, alias, org, id string) error {
	erp := s.ERPs[org]
	if erp == nil || s.Aliases[org] != alias || uuid.Validate(id) != nil {
		return f.ErrUnavailable
	}
	original, err := load(ctx, erp, f.Command{Organization: org}, id)
	if err != nil {
		return err
	}
	if original.Stage != "CONFIRMED" {
		return nil
	}
	scope := f.Scope{OrganizationID: org, ActorID: original.Actor, MemberID: original.Member}
	opened, err := s.Open(ctx, storecenter.FulfillmentSubject{OrganizationID: org, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: "fulfillment.execute"}, original.Source.Store, original.Source.Kind)
	if err != nil {
		return err
	}
	merchant, ok := opened.(ExecutionMerchant)
	if !ok || merchant.BindingDigest() != original.Binding {
		return f.ErrForbidden
	}
	source, err := merchant.Source(ctx, original.Source.ID)
	if err != nil {
		return err
	}
	base := f.Command{Organization: org, Actor: scope.ActorID, Member: scope.MemberID, Binding: original.Binding, Store: original.Source.Store, SourceKind: original.Source.Kind, SourceID: original.Source.ID}
	prepared, err := s.freeze(ctx, erp, base, merchant, source, original.Input.Selection)
	if err != nil {
		return err
	}
	if prepared.Version != original.Input.Version || !bytes.Equal(prepared.Data, original.Input.Data) {
		return f.ErrConflict
	}
	transition := map[string]any{"operation": id, "version": original.Revision}
	raw, err := native(ctx, erp, base, "claim", transition)
	if err != nil {
		return err
	}
	if err = checkedTransition(raw, original, "CLAIMED"); err != nil {
		return err
	}
	if err = merchant.Check(ctx); err != nil {
		return err
	}
	// A lost marker response never authorizes provider send. Read original key only.
	raw, err = native(ctx, erp, base, "mark_dispatching", transition)
	if err != nil {
		return err
	}
	if err = checkedTransition(raw, original, "DISPATCHING"); err != nil {
		return err
	}
	var delivery sf.Delivery
	if prepared.Delivery != "" {
		raw, e := native(ctx, erp, base, "delivery_details", map[string]any{"demand": prepared.Demand, "delivery": prepared.Delivery})
		if e != nil {
			return e
		}
		if json.Unmarshal(raw, &delivery) != nil {
			return f.ErrUnknown
		}
	}
	fresh, err := merchant.Source(ctx, source.ID)
	if err != nil {
		return err
	}
	plan, err := merchant.Build(ctx, fresh, delivery, prepared.Selection)
	if err != nil || preparedVersion(fresh, delivery, prepared.Selection, plan) != prepared.Version {
		return f.ErrConflict
	}
	result, sendErr := merchant.Send(ctx, plan)
	stage := "SUCCEEDED"
	if sendErr != nil {
		stage = "UNKNOWN"
		result = sf.MutationResult{Result: map[string]any{"reason": "provider_outcome_unproven"}}
	}
	payload := map[string]any{"operation": id, "stage": stage, "result": result.Result}
	if result.Private != nil {
		payload["private"] = result.Private
	}
	raw, err = native(ctx, erp, base, "record_result", payload)
	if err != nil {
		return err
	}
	return checkedTransition(raw, original, stage)
}
