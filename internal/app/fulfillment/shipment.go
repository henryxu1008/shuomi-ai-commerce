package fulfillmentapp

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	f "task-processor/internal/fulfillment"
	sf "task-processor/internal/marketplace/shein/fulfillment"
)

func (s *Service) shipment(ctx context.Context, scope f.Scope, in Input, base f.Command, merchant Merchant) (json.RawMessage, error) {
	var input struct {
		Operation string `json:"operation"`
	}
	if decode(in.Payload, &input) != nil || uuid.Validate(input.Operation) != nil || in.Kind != "platform_supply" {
		return nil, f.ErrInvalid
	}
	original, err := load(ctx, s.ERPs[scope.OrganizationID], base, input.Operation)
	if err != nil {
		return nil, err
	}
	if original.Binding != base.Binding || original.Source.ID != in.SourceID || original.Source.Store != in.Store || original.Source.Kind != in.Kind || original.Stage != "SUCCEEDED" || original.Input.Purpose != "supply_shipping" {
		return nil, f.ErrForbidden
	}
	var result struct {
		Shipping string `json:"shipping_id"`
	}
	if json.Unmarshal(original.Result, &result) != nil || result.Shipping == "" {
		return nil, f.ErrUnknown
	}
	reader, ok := merchant.(interface {
		Shipment(context.Context, string) (sf.SupplyShipment, error)
	})
	if !ok {
		return nil, f.ErrUnavailable
	}
	if err = merchant.Check(ctx); err != nil {
		return nil, err
	}
	projection, err := reader.Shipment(ctx, result.Shipping)
	if err != nil {
		return nil, err
	}
	return json.Marshal(projection)
}
