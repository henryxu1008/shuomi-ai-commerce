package bailian

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"regexp"

	strictjson "sigs.k8s.io/json"
)

// Diagnostic contains only bounded operational metadata. It is not evidence of
// no provider effect and must never change the existing UNKNOWN/retry decision.
type Diagnostic struct {
	Stage, Reason, ProviderCode, RequestID string
	HTTPStatus                             int
}

var diagnosticRequestID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (c *Client) unknown(stage string, status int, data []byte, headerID, reason string) error {
	if c.cfg.Diagnose == nil {
		return errUnknown
	}
	d := Diagnostic{Stage: stage, HTTPStatus: status, Reason: reason}
	if diagnosticRequestID.MatchString(headerID) && headerID != c.cfg.APIKey {
		d.RequestID = headerID
	}
	var body struct {
		Code      string `json:"code"`
		RequestID string `json:"request_id"`
	}
	if strict, err := strictjson.UnmarshalStrict(data, &body, strictjson.DisallowDuplicateFields); err == nil && len(strict) == 0 {
		if diagnosticRequestID.MatchString(body.RequestID) && body.RequestID != c.cfg.APIKey {
			d.RequestID = body.RequestID
		}
		switch body.Code {
		case "":
		case "InvalidParameter", "InvalidApiKey", "AccessDenied", "Arrearage", "DataInspectionFailed", "InternalError", "Throttling", "ModelNotFound", "AllocationQuota.FreeTierOnly", "BadRequest.EmptyModel":
			if body.Code != c.cfg.APIKey {
				d.ProviderCode = body.Code
			}
		default:
			d.ProviderCode = "unrecognized"
		}
	}
	c.cfg.Diagnose(d)
	return errUnknown
}

func transportReason(err error) string {
	var certificate x509.UnknownAuthorityError
	var network net.Error
	switch {
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "context_deadline"
	case errors.As(err, &certificate):
		return "tls_certificate"
	case errors.As(err, &network) && network.Timeout():
		return "network_timeout"
	default:
		return "network_other"
	}
}
