package proposal

import (
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/sourcing"
)

func validRequest() Request {
	return Request{
		Snapshot: catalog.ProductSnapshot{Title: "Bottle"},
		Source: sourcing.SourceEnvelope{
			Identity: sourcing.SourceIdentity{
				SourceType:     sourcing.SourceTypeCrawler,
				SourcePlatform: "amazon",
				SourceID:       "B001",
			},
			RawReference: sourcing.RawSourceReference{ReferenceID: "raw-1"},
		},
		Policy: PolicySnapshot{
			Version:             "v1",
			AllowedFields:       []string{"description"},
			RequiredFields:      []string{"description"},
			MinimumQualityScore: 80,
		},
	}
}
