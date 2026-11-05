package collection

import "testing"

func TestSHEINUSPublicationKind(t *testing.T) {
	const kind = "shein_us_data"
	scope := Scope{OrganizationID: "fixture-org", ActorID: "fixture-actor", MemberID: "fixture-member"}
	id := "00000000-0000-4000-8000-000000000301"
	if !ValidSourceKind(kind) {
		t.Fatal("Collection rejected shein_us_data")
	}
	batch, err := NewPublicationBatch(scope, id, kind, "SHEIN · US")
	if err != nil || batch.Kind != kind {
		t.Fatalf("SHEIN publication batch: %+v, %v", batch, err)
	}
}
