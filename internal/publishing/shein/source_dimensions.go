package shein

type SourceVariantDimension struct {
	Name          string   `json:"name,omitempty"`
	Values        []string `json:"values,omitempty"`
	DistinctCount int      `json:"distinct_count,omitempty"`
	SampleValue   string   `json:"sample_value,omitempty"`
}
