package collection

import (
	"context"
	"encoding/json"
	"time"

	"task-processor/internal/product/sourcing"
)

func (s *Service) WithSourceQuotes(reader SourceQuoteReader) *Service { s.quotes = reader; return s }

func (q SourceQuotes) MarshalJSON() ([]byte, error) {
	if q.State == "available" {
		return json.Marshal(struct {
			State         string        `json:"state"`
			PublicationID string        `json:"publicationId"`
			Version       uint64        `json:"version,string"`
			Prices        []SourcePrice `json:"prices"`
		}{q.State, q.PublicationID, q.Version, q.Prices})
	}
	return json.Marshal(struct {
		State string `json:"state"`
	}{q.State})
}

func (s *Service) readSourceQuotes(ctx context.Context, scope Scope, source Source) SourceQuotes {
	unavailable := SourceQuotes{State: "unavailable"}
	if source.Kind != "acquisition" && source.Kind != "amazon_data" {
		return SourceQuotes{State: "not_applicable"}
	}
	if s.quotes == nil {
		return unavailable
	}
	budget := 2 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline) - 500*time.Millisecond
		if remaining <= 0 {
			return unavailable
		}
		if remaining < budget {
			budget = remaining
		}
	}
	child, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	result, err := s.quotes.ReadSourceQuotes(child, scope, source)
	if err != nil || child.Err() != nil {
		return unavailable
	}
	if result.State == "not_applicable" && result.PublicationID == "" && result.Version == 0 && len(result.Prices) == 0 {
		return result
	}
	if result.State != "available" || result.PublicationID != source.PublicationID || result.Version != source.Version || result.Prices == nil {
		return unavailable
	}
	raw, err := json.Marshal(result)
	if err != nil || len(raw) > sourcing.MaxSourceQuoteBytes {
		return unavailable
	}
	return result
}
