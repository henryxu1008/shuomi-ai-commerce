package observations

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const SupplyTimeContract = "shein-supply-business-times/v1"
const SupplyTimeSource = "/open-api/order/purchase-order-infos"

// Fixed response fields only. Query windows and local observation timestamps
// are deliberately absent: neither is an original business time.
var supplyTimeFields = [...]string{"addTime", "allocateTime", "updateTime", "requestDeliveryTime", "requestReceiptTime", "requestTakeParcelTime", "reserveTime", "receiptTime", "checkTime", "storageTime", "returnTime", "requestCompleteTime", "deliveryTime"}

type SupplyTimeValue struct {
	State   string  `json:"state"`
	RawType string  `json:"rawType"`
	Raw     *string `json:"raw"`
}
type SupplyBusinessTimes struct {
	Contract string                     `json:"contract"`
	Source   string                     `json:"source"`
	Timezone string                     `json:"timezone"`
	Values   map[string]SupplyTimeValue `json:"values"`
}

// ParseSupplyBusinessTimes must run on the original JSON tokens, before a
// string decoder could lose missing/null, numeric lexical form or invalid UTF8.
func ParseSupplyBusinessTimes(fields map[string]json.RawMessage) (SupplyBusinessTimes, error) {
	out := SupplyBusinessTimes{Contract: SupplyTimeContract, Source: SupplyTimeSource, Timezone: "unconfirmed", Values: map[string]SupplyTimeValue{}}
	for _, field := range supplyTimeFields {
		value, err := parseSupplyTime(fields[field])
		if err != nil {
			return SupplyBusinessTimes{}, err
		}
		out.Values[field] = value
	}
	return out, nil
}

func supplyStringState(raw string) string {
	if raw == "" {
		return "empty"
	}
	// Only classify syntax/calendar. Never retain or convert the parsed instant.
	if len(raw) == 19 {
		if _, err := time.Parse("2006-01-02 15:04:05", raw); err == nil {
			return "local_datetime"
		}
	}
	if _, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return "offset_datetime"
	}
	return "unrecognized"
}
func parseSupplyTime(token json.RawMessage) (SupplyTimeValue, error) {
	if len(token) == 0 {
		return SupplyTimeValue{State: "missing", RawType: "missing"}, nil
	}
	if !utf8.Valid(token) {
		return SupplyTimeValue{}, ErrInvalid
	}
	var value any
	decoder := json.NewDecoder(strings.NewReader(string(token)))
	decoder.UseNumber()
	if decoder.Decode(&value) != nil {
		return SupplyTimeValue{}, ErrInvalid
	}
	out := SupplyTimeValue{}
	switch v := value.(type) {
	case nil:
		out.State = "null"
		out.RawType = "null"
	case string:
		// encoding/json replaces invalid surrogate escapes. Reject them rather
		// than silently recording a different source string.
		if !validSupplyJSONString(token) || !validSupplyTimeText(v) {
			return SupplyTimeValue{}, ErrInvalid
		}
		out.State = supplyStringState(v)
		out.RawType = "string"
		out.Raw = &v
	case json.Number:
		raw := v.String()
		if !validSupplyTimeText(raw) {
			return SupplyTimeValue{}, ErrInvalid
		}
		out.State = "unexpected_type"
		out.RawType = "number"
		out.Raw = &raw
	case bool:
		raw := "false"
		if v {
			raw = "true"
		}
		out.State = "unexpected_type"
		out.RawType = "boolean"
		out.Raw = &raw
	default:
		return SupplyTimeValue{}, ErrInvalid
	}
	return out, nil
}
func validSupplyTimeText(v string) bool {
	return len(v) <= 256 && utf8.ValidString(v) && !strings.ContainsRune(v, 0)
}

func (b SupplyBusinessTimes) Valid() bool {
	if b.Contract != SupplyTimeContract || b.Source != SupplyTimeSource || b.Timezone != "unconfirmed" || len(b.Values) != len(supplyTimeFields) {
		return false
	}
	for _, field := range supplyTimeFields {
		v, ok := b.Values[field]
		if !ok {
			return false
		}
		switch v.RawType {
		case "missing", "null":
			if v.Raw != nil || v.State != v.RawType {
				return false
			}
		case "string":
			if v.Raw == nil || !validSupplyTimeText(*v.Raw) || v.State != supplyStringState(*v.Raw) {
				return false
			}
		case "number", "boolean":
			if v.Raw == nil || !validSupplyTimeText(*v.Raw) {
				return false
			}
			p, e := parseSupplyTime(json.RawMessage(*v.Raw))
			if e != nil || !reflect.DeepEqual(p, v) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// JSON permits escaped UTF16 pairs. Unpaired surrogates are not Unicode source
// text and the standard decoder repairs them, so reject before that repair.
func validSupplyJSONString(raw []byte) bool {
	for i := 1; i < len(raw)-1; i++ {
		if raw[i] != '\\' {
			continue
		}
		i++
		if raw[i] != 'u' {
			continue
		}
		u, e := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
		if e != nil {
			return false
		}
		i += 4
		if u >= 0xdc00 && u <= 0xdfff {
			return false
		}
		if u >= 0xd800 && u <= 0xdbff {
			if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
				return false
			}
			low, e := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
			if e != nil || low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 6
		}
	}
	return true
}
