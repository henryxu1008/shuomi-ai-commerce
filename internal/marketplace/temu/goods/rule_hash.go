package goods

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Observation time is audit metadata. Re-reading unchanged rules must not
// invalidate the approved content; actual definitions and provenance still do.
func RuleHash(value RuleSnapshot) string {
	value.ObservedAt = time.Time{}
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}
