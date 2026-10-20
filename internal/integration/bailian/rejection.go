package bailian

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	strictjson "sigs.k8s.io/json"
)

// QuotaRejection is minted only after a complete, bounded HTTP response has
// passed the refusal contract. Diagnostic logs do not implement this contract.
type QuotaRejection struct{ RequestID, ResponseDigest string }

func (*QuotaRejection) Error() string {
	return "Bailian image request rejected: AllocationQuota.FreeTierOnly"
}
func (*QuotaRejection) NoExternalEffect() bool { return true }

func ParseQuotaRejection(status int, data []byte, headerID string) (*QuotaRejection, bool) {
	if status != http.StatusForbidden || len(data) == 0 || len(data) > 64<<10 {
		return nil, false
	}
	var body struct {
		RequestID string `json:"request_id"`
		Code      string `json:"code"`
		Message   string `json:"message"`
	}
	strict, err := strictjson.UnmarshalStrict(data, &body, strictjson.DisallowDuplicateFields, strictjson.DisallowUnknownFields)
	if err != nil || len(strict) != 0 || body.Code != "AllocationQuota.FreeTierOnly" || !diagnosticRequestID.MatchString(body.RequestID) || headerID != "" && headerID != body.RequestID {
		return nil, false
	}
	sum := sha256.Sum256(data)
	return &QuotaRejection{RequestID: body.RequestID, ResponseDigest: hex.EncodeToString(sum[:])}, true
}
