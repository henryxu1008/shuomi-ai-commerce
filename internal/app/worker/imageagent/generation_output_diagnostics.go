package imageagentworker

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"

	"task-processor/internal/imageagent"
	"task-processor/internal/integration/httpimage"
)

// Local metadata only: never attach the raw error, URL, prompt or credentials.
// Classification does not change the caller's error, retries or settlement.
func logOutputRecoveryFailure(logger *logrus.Logger, runID, stage string, err error) {
	if logger == nil {
		return
	}
	fields := logrus.Fields{
		"event": "image_output_recovery_diagnostic", "stage": stage,
		"run_id_hash": fmt.Sprintf("%x", sha256.Sum256([]byte(runID))),
	}
	var status *httpimage.HTTPStatusError
	var dns *net.DNSError
	var authority x509.UnknownAuthorityError
	var certificate x509.CertificateInvalidError
	var hostname x509.HostnameError
	var verification *tls.CertificateVerificationError
	var header tls.RecordHeaderError
	var network net.Error
	var operation *net.OpError
	reason := "transport_error"
	switch {
	case errors.Is(err, imageagent.ErrValidation):
		reason = "validation_error"
	case errors.Is(err, imageagent.ErrInvalidGeneratedOutput):
		reason = "invalid_output"
	case errors.Is(err, context.Canceled):
		reason = "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		reason = "deadline_exceeded"
	case errors.Is(err, httpimage.ErrBodyTooLarge):
		reason = "body_too_large"
	case errors.As(err, &status):
		reason = "http_error"
		if status.StatusCode >= 100 && status.StatusCode <= 599 {
			fields["http_status"] = status.StatusCode
		}
	case errors.As(err, &dns):
		reason = "dns_error"
	case errors.As(err, &authority), errors.As(err, &certificate), errors.As(err, &hostname), errors.As(err, &verification):
		reason = "tls_certificate_error"
	case errors.As(err, &header):
		reason = "tls_error"
	case errors.As(err, &network) && network.Timeout():
		reason = "network_timeout"
	case errors.As(err, &operation):
		reason = "network_error"
	}
	fields["reason"] = reason
	logger.WithFields(fields).Warn("Image output recovery diagnostic")
}
