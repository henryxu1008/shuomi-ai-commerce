package goods

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"task-processor/internal/marketplace/temu/model"
)

// JSONB may reorder nested object keys. Restore the fixed official typed body
// before hashing a retained draft; unknown wire fields never disappear silently.
func (value *Draft) UnmarshalJSON(raw []byte) error {
	type plain Draft
	var decoded plain
	if len(raw) > 2<<20 {
		return errors.New("TEMU draft exceeds limit")
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	if len(decoded.SubmissionPayload) > 0 {
		var payload any
		switch decoded.Mode {
		case "local":
			payload = &model.LocalGoodsV2{}
		case "fully_managed", "semi_managed":
			payload = &model.SupplierGoods{}
		default:
			return errors.New("unknown TEMU draft mode")
		}
		reader := json.NewDecoder(bytes.NewReader(decoded.SubmissionPayload))
		reader.DisallowUnknownFields()
		if err := reader.Decode(payload); err != nil {
			return err
		}
		if err := reader.Decode(new(any)); err != io.EOF {
			return errors.New("invalid TEMU draft payload")
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		decoded.SubmissionPayload = body
	}
	*value = Draft(decoded)
	return nil
}
