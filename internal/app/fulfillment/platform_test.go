package fulfillmentapp

import (
	"context"
	"encoding/json"
	f "task-processor/internal/fulfillment"
	"task-processor/internal/storecenter"
	sf "task-processor/internal/storecenter/fulfillment"
	"testing"
	"time"
)

type executionMerchant struct {
	fakeMerchant
	sends int
}

func (m *executionMerchant) Options(context.Context, sf.Source, sf.Selection) (sf.ShippingOptions, error) {
	return sf.ShippingOptions{}, nil
}
func (m *executionMerchant) Build(_ context.Context, s sf.Source, _ sf.Delivery, selection sf.Selection) (sf.Plan, error) {
	return sf.Plan{Purpose: selection.Purpose, Body: json.RawMessage(`{"orderNo":"original","handleType":2}`)}, nil
}
func (m *executionMerchant) Send(ctx context.Context, p sf.Plan) (sf.MutationResult, error) {
	if err := m.Check(ctx); err != nil {
		return sf.MutationResult{}, err
	}
	m.sends++
	return sf.MutationResult{Result: map[string]any{}}, nil
}

type executionERP struct {
	original      Loaded
	commands      []string
	loseMarker    bool
	invalidMarker bool
	onMarker      func()
}

func (e *executionERP) Execute(_ context.Context, c f.Command) (json.RawMessage, error) {
	e.commands = append(e.commands, c.Command)
	if c.Command == "load_operation" {
		return json.Marshal(e.original)
	}
	if c.Command == "mark_dispatching" {
		if e.onMarker != nil {
			e.onMarker()
		}
		if e.invalidMarker {
			return json.RawMessage(`{}`), nil
		}
		if e.loseMarker {
			return nil, f.ErrUnknown
		}
	}
	if c.Command == "record_result" {
		var p struct {
			Stage string `json:"stage"`
		}
		_ = json.Unmarshal(c.Payload, &p)
		e.original.Stage = p.Stage
	}
	stage := map[string]string{"claim": "CLAIMED", "mark_dispatching": "DISPATCHING", "confirm": "CONFIRMED"}[c.Command]
	if stage == "" {
		stage = e.original.Stage
	}
	return json.Marshal(f.Receipt{Operation: e.original.Operation, Target: e.original.Target, Revision: e.original.Revision, Stage: stage, Result: json.RawMessage(`{}`)})
}
func executionFixture() (*Service, *executionERP, *executionMerchant) {
	source := sf.Source{Kind: "consumer_order", ID: "original", Market: "shein-us", Type: 1, StockMode: 3, Status: 1, PrintStatus: 1, SelectedLogistics: 2, Logistics: []int{2}, Deadline: time.Now().Add(time.Hour), Lines: []sf.Line{{ID: "123", SKU: "sku", Quantity: 1, GoodsStatus: 1, ExchangeTag: 1}}}
	merchant := &executionMerchant{fakeMerchant: fakeMerchant{source: source}}
	selection := sf.Selection{Purpose: "accept_address", Demand: "native-demand"}
	plan, _ := merchant.Build(context.Background(), source, sf.Delivery{}, selection)
	original := Loaded{Receipt: f.Receipt{Operation: "a182d2e0-6f91-4e71-8b7f-5df25b75fb11", Stage: "CONFIRMED", Target: "native-demand"}, Actor: "actor", Member: "member", Binding: merchant.BindingDigest(), Input: Prepared{Purpose: selection.Purpose, Demand: selection.Demand, Selection: selection, Data: plan.Body, Version: preparedVersion(source, sf.Delivery{}, selection, plan)}}
	original.Revision = original.Input.Version
	original.Source.Store, original.Source.Kind, original.Source.ID, original.Source.Organization = "store", "consumer_order", "original", "org"
	erp := &executionERP{original: original}
	service := &Service{ERPs: map[string]f.ERP{"org": erp}, Aliases: map[string]string{"org": "site"}, Open: func(context.Context, storecenter.FulfillmentSubject, string, string) (Merchant, error) {
		return merchant, nil
	}}
	return service, erp, merchant
}
func TestLostDispatchMarkerNeverSendsProviderRequest(t *testing.T) {
	service, erp, merchant := executionFixture()
	erp.loseMarker = true
	if err := service.Run(context.Background(), "site", "org", erp.original.Operation); err != f.ErrUnknown {
		t.Fatal(err)
	}
	if merchant.sends != 0 {
		t.Fatal("provider sent after marker response loss")
	}
}
func TestPermissionDriftAfterMarkerPreservesUnknownAndNoDisclosure(t *testing.T) {
	service, erp, merchant := executionFixture()
	erp.onMarker = func() { merchant.denied = true }
	if err := service.Run(context.Background(), "site", "org", erp.original.Operation); err != nil {
		t.Fatal(err)
	}
	if merchant.sends != 0 || erp.original.Stage != "UNKNOWN" {
		t.Fatal("revocation disclosed private address", merchant.sends, erp.original.Stage)
	}
}
func TestOriginalCompletedEffectIsNeverResent(t *testing.T) {
	service, erp, merchant := executionFixture()
	for i := 0; i < 2; i++ {
		if err := service.Run(context.Background(), "site", "org", erp.original.Operation); err != nil {
			t.Fatal(err)
		}
	}
	if merchant.sends != 1 || erp.original.Stage != "SUCCEEDED" {
		t.Fatal(merchant.sends, erp.original.Stage)
	}
}

func TestMalformedDispatchMarkerNeverAuthorizesProviderSend(t *testing.T) {
	service, erp, merchant := executionFixture()
	erp.invalidMarker = true
	if err := service.Run(context.Background(), "site", "org", erp.original.Operation); err != f.ErrUnknown || merchant.sends != 0 {
		t.Fatal("invalid marker authorized send", err, merchant.sends)
	}
}
func TestOriginalDeadlineExpiringAfterMarkerStopsProviderSend(t *testing.T) {
	service, erp, merchant := executionFixture()
	merchant.source.Deadline = time.Now().Add(200 * time.Millisecond)
	plan, _ := merchant.Build(context.Background(), merchant.source, sf.Delivery{}, erp.original.Input.Selection)
	erp.original.Input.Version = preparedVersion(merchant.source, sf.Delivery{}, erp.original.Input.Selection, plan)
	erp.original.Revision = erp.original.Input.Version
	marked := false
	erp.onMarker = func() { marked = true; time.Sleep(250 * time.Millisecond) }
	err := service.Run(context.Background(), "site", "org", erp.original.Operation)
	if !marked || err != f.ErrConflict || merchant.sends != 0 {
		t.Fatal("expired original deadline authorized a provider send", marked, err, merchant.sends)
	}
}
