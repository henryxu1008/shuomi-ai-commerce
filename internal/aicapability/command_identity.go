package aicapability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// ScopedCommandID derives a UUIDv8 from live command identity. The namespace
// is the versioned command kind. Callers must not accept this ID from clients.
func ScopedCommandID(prefix, kind, org, actor, key string) string {
	data, _ := json.Marshal([4]string{kind, org, actor, key})
	id := sha256.Sum256(data)
	id[6] = id[6]&0x0f | 0x80
	id[8] = id[8]&0x3f | 0x80
	raw := hex.EncodeToString(id[:16])
	return prefix + raw[:8] + "-" + raw[8:12] + "-" + raw[12:16] + "-" + raw[16:20] + "-" + raw[20:]
}

func IsScopedCommandID(prefix, id string) bool {
	raw, ok := strings.CutPrefix(id, prefix)
	if !ok || len(raw) != 36 || raw[8] != '-' || raw[13] != '-' || raw[18] != '-' || raw[23] != '-' || raw[14] != '8' {
		return false
	}
	value, err := hex.DecodeString(strings.ReplaceAll(raw, "-", ""))
	return err == nil && len(value) == 16 && value[8]&0xc0 == 0x80 && raw == strings.ToLower(raw)
}
