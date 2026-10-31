package shein

import (
	f "task-processor/internal/fulfillment"
	"testing"
)

func TestTrackingAcknowledgementIsNotShipmentOrBlanketSuccess(t *testing.T) {
	for _, raw := range []string{`{"code":0}`, `{"code":0,"info":null}`, `{"code":0,"info":[{"goodsId":123,"errorMsg":"denied"}]}`, `{"code":0,"info":[]`} {
		if _, err := trackingReceipt([]byte(raw), []string{"123"}); err != f.ErrUnknown {
			t.Fatalf("unproven tracking success: %s %v", raw, err)
		}
	}
	result, err := trackingReceipt([]byte(`{"code":0,"info":[]}`), []string{"123"})
	if err != nil || len(result) != 1 || result[0] != "123" {
		t.Fatal(result, err)
	}
}
