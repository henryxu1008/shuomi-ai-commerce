package temu

import (
	"bytes"
	"crypto/md5" // TEMU's specified request signature, not a password hash.
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"task-processor/internal/storecenter"
)

func (c *OfficialClient) encodeRequest(api, token string, fields map[string]json.RawMessage) ([]byte, error) {
	if !boundedParameter(api, 128) || !boundedParameter(token, 4096) {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	timestamp := strconv.FormatInt(c.now().Unix(), 10)
	if len(timestamp) != 10 {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	values := make(map[string]json.RawMessage, len(fields)+6)
	for key, value := range fields {
		switch key {
		case "type", "app_key", "access_token", "timestamp", "data_type", "sign", "version":
			return nil, storecenter.ErrOfficialConnectionUnavailable
		}
		if !boundedParameter(key, 128) || len(value) > maxOfficialBody || !json.Valid(value) {
			return nil, storecenter.ErrOfficialConnectionUnavailable
		}
		for _, r := range key {
			if r > 127 {
				return nil, storecenter.ErrOfficialConnectionUnavailable
			}
		}
		var compact bytes.Buffer
		if json.Compact(&compact, value) != nil {
			return nil, storecenter.ErrOfficialConnectionUnavailable
		}
		values[key] = compact.Bytes()
	}
	for key, value := range map[string]string{"type": api, "app_key": c.application.AppID, "access_token": token, "timestamp": timestamp, "data_type": "JSON"} {
		encoded, err := exactJSON(value)
		if err != nil {
			return nil, storecenter.ErrOfficialConnectionUnavailable
		}
		values[key] = encoded
	}
	// Check size before constructing the signed representation and again after
	// adding sign. Strings sign as their exact decoded value; other values sign
	// as the same compact JSON bytes included in the HTTP body.
	unsigned, err := exactJSON(values)
	if err != nil || len(unsigned) > maxOfficialBody {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	clear(unsigned)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hash := md5.New()
	_, _ = hash.Write([]byte(c.appSecret))
	for _, key := range keys {
		_, _ = hash.Write([]byte(key))
		value := values[key]
		if value[0] == '"' {
			var text string
			if json.Unmarshal(value, &text) != nil {
				return nil, storecenter.ErrOfficialConnectionUnavailable
			}
			_, _ = hash.Write([]byte(text))
		} else {
			_, _ = hash.Write(value)
		}
	}
	_, _ = hash.Write([]byte(c.appSecret))
	values["sign"], _ = exactJSON(strings.ToUpper(hex.EncodeToString(hash.Sum(nil))))
	body, err := exactJSON(values)
	if err != nil || len(body) > maxOfficialBody {
		clear(body)
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	return body, nil
}

func exactJSON(value any) ([]byte, error) {
	var body bytes.Buffer
	encoder := json.NewEncoder(&body)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(body.Bytes(), []byte("\n")), nil
}
