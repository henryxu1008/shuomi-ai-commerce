package dataacquisition

import (
	"encoding/json"
	"fmt"
)

const (
	PriceFrozen              = "FROZEN"
	PriceUnavailableTerminal = "UNAVAILABLE_TERMINAL"
)

// RequireFrozenPrice never interprets an absent snapshot as a free price.
func (j Job) RequireFrozenPrice() error {
	if j.PriceStatus != PriceFrozen || j.Price.Validate() != nil {
		return fmt.Errorf("%w: task %s price_snapshot unavailable; read only", ErrUnavailable, j.ID)
	}
	return nil
}

// The internal numeric fields remain useful for priced execution. The current
// wire contract makes unavailable amounts explicitly null, never numeric zero.
func (j Job) MarshalJSON() ([]byte, error) {
	type jobJSON Job
	if j.PriceStatus != PriceUnavailableTerminal {
		return json.Marshal(jobJSON(j))
	}
	return json.Marshal(struct {
		*jobJSON
		Price        any `json:"price"`
		ConfirmedFen any `json:"confirmedFen"`
		PendingFen   any `json:"pendingFen"`
	}{jobJSON: (*jobJSON)(&j)})
}
