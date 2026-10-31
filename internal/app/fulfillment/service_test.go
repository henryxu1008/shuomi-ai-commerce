package fulfillmentapp

import (
	"context"
	"encoding/json"
	f "task-processor/internal/fulfillment"
	sf "task-processor/internal/marketplace/shein/fulfillment"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type fakeMerchant struct {
	denied    bool
	source    sf.Source
	sourceErr error
}

func (m *fakeMerchant) Check(context.Context) error {
	if m.denied {
		return f.ErrForbidden
	}
	return nil
}
func (m *fakeMerchant) BindingDigest() string {
	return "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}
func (m *fakeMerchant) Source(context.Context, string) (sf.Source, error) {
	return m.source, m.sourceErr
}

type fakeERP struct{ calls []f.Command }

func TestActualReceiptAndSavedFactsDoNotDependOnPlatformAvailability(t *testing.T) {
	m := &fakeMerchant{sourceErr: f.ErrUnavailable}
	erp := &fakeERP{}
	service := &Service{ERPs: map[string]f.ERP{"org": erp}, Open: func(context.Context, storecenter.FulfillmentSubject, string, string) (Merchant, error) { return m, nil }}
	input := Input{Store: "store", Kind: "platform_supply", SourceID: "original", OperationID: "a182d2e0-6f91-4e71-8b7f-5df25b75fb11", Action: "receive", Payload: json.RawMessage(`{"purchase":"native-po","lines":[{"purchase_line":"line","qty":1}]}`)}
	scope := f.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	if _, err := service.Execute(context.Background(), scope, input); err != nil {
		t.Fatal(err)
	}
	input.Action = "reserve"
	if _, err := service.Execute(context.Background(), scope, input); err != f.ErrUnavailable || len(erp.calls) != 1 {
		t.Fatal("new allocation proceeded without current platform state", err)
	}
	m.denied = true
	input.Action = "receive"
	if _, err := service.Execute(context.Background(), scope, input); err != f.ErrForbidden || len(erp.calls) != 1 {
		t.Fatal("receipt ignored live authority", err)
	}
}

func (e *fakeERP) Execute(_ context.Context, c f.Command) (json.RawMessage, error) {
	e.calls = append(e.calls, c)
	return json.RawMessage(`{"stage":"SUCCEEDED","target":"native-order"}`), nil
}

func TestDemandTakesOriginalPlatformQuantitiesAndCurrentAuthority(t *testing.T) {
	now := time.Now()
	m := &fakeMerchant{source: sf.Source{Kind: "platform_supply", ID: "original", Status: 2, Scene: 1, Type: 2, Deadline: now.Add(time.Hour), Lines: []sf.Line{{ID: "sku", SKU: "sku", Quantity: 5}}}}
	erp := &fakeERP{}
	service := &Service{ERPs: map[string]f.ERP{"org": erp}, SupplyMovement: map[string]string{"org": "owned_transfer"}, Open: func(context.Context, storecenter.FulfillmentSubject, string, string) (Merchant, error) { return m, nil }}
	input := Input{Store: "store", Kind: "platform_supply", SourceID: "original", OperationID: "a182d2e0-6f91-4e71-8b7f-5df25b75fb11", Action: "bind_demand", Payload: json.RawMessage(`{"delivery_date":"2026-11-05","lines":[{"source_line":"sku","item_code":"native-item","uom":"Nos","warehouse":"Stores"}]}`)}
	scope := f.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	if _, err := service.Execute(context.Background(), scope, input); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Lines []struct {
			Qty int `json:"qty"`
		} `json:"lines"`
	}
	_ = json.Unmarshal(erp.calls[0].Payload, &payload)
	if erp.calls[0].Organization != "org" || erp.calls[0].SourceID != "original" || payload.Lines[0].Qty != 5 {
		t.Fatal("browser quantity replaced original source")
	}
	m.denied = true
	if _, err := service.Execute(context.Background(), scope, input); err != f.ErrForbidden || len(erp.calls) != 1 {
		t.Fatal("revoked operation reached ERP", err)
	}
}
