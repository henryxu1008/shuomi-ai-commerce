package submission

import "encoding/json"

func limitedWalmartResult(record WalmartResultRecord) WalmartResultRecord {
	out := record
	out.Limited, out.Observation.Partial = true, true
	out.ObservedPages, out.RepairableSKUs = nil, nil
	if record.ReadProgress != nil {
		offset := record.ReadProgress.NextOffset
		out.ReadOffset = &offset
	}
	return out
}

func minimalWalmartResult(record WalmartResultRecord) WalmartResultRecord {
	out := limitedWalmartResult(record)
	out.ReadProgress = nil
	out.Observation.Items, out.Observation.ResponseHashes = nil, nil
	return out
}

// BoundWalmartResults limits only the display projection. Every live receipt
// stays queryable; canonical observations and resumable cursors are unchanged.
func BoundWalmartResults(records []WalmartResultRecord, budget int) ([]WalmartResultRecord, error) {
	if len(records) == 0 {
		return records, nil
	}
	out := make([]WalmartResultRecord, len(records))
	minimum := make([]int, len(records))
	for i, record := range records {
		out[i] = minimalWalmartResult(record)
		raw, err := json.Marshal(out[i])
		if err != nil {
			return nil, ErrExecutionUnavailable
		}
		minimum[i] = len(raw)
	}
	raw, err := json.Marshal(out)
	if err != nil || len(raw) > budget {
		return nil, ErrExecutionUnavailable
	}
	used := len(raw)
	for i, record := range records {
		cap := budget - used + minimum[i]
		full, err := json.Marshal(record)
		if err != nil {
			return nil, ErrExecutionUnavailable
		}
		if len(full) <= cap {
			out[i], used = record, used+len(full)-minimum[i]
			continue
		}
		current := limitedWalmartResult(record)
		body, err := json.Marshal(current)
		if err != nil {
			return nil, ErrExecutionUnavailable
		}
		if len(body) <= cap {
			out[i], used = current, used+len(body)-minimum[i]
			continue
		}
		current = out[i]
		low, high := 0, len(record.Observation.Items)
		for low < high {
			mid := (low + high + 1) / 2
			current.Observation.Items = record.Observation.Items[:mid]
			body, err := json.Marshal(current)
			if err != nil {
				return nil, ErrExecutionUnavailable
			}
			if len(body) <= cap {
				low = mid
			} else {
				high = mid - 1
			}
		}
		current.Observation.Items = record.Observation.Items[:low]
		body, _ = json.Marshal(current)
		out[i], used = current, used+len(body)-minimum[i]
	}
	return out, nil
}
