package einomodel

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http/httptrace"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

type textDiagnosticKey struct{}

// Diagnostics are local log metadata, never invocation, usage or recovery facts.
// In particular, a transport handoff does not prove the provider received it.
type textDiagnostics struct {
	logger    *logrus.Entry
	started   time.Time
	caller    context.Context
	mu        sync.Mutex
	fields    logrus.Fields
	status    int
	validJSON bool
}

func newTextDiagnostics(ctx context.Context, invocationID string, logger *logrus.Entry) *textDiagnostics {
	d := &textDiagnostics{logger: logger, started: time.Now(), caller: ctx,
		fields: logrus.Fields{
			"event": "governed_text_diagnostic", "handoffs": 0,
			"tls_handshake_success_count": 0, "tls_handshake_failure_count": 0,
			"wrote_request_success_count": 0, "wrote_request_failure_count": 0,
		}}
	if invocationID != "" {
		d.fields["invocation_id_hash"] = diagnosticHash(invocationID)
	}
	return d
}

func diagnosticHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func diagnosticContextState(ctx context.Context) string {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "deadline_exceeded"
	}
	if ctx.Err() != nil {
		return "canceled"
	}
	return "active"
}

func diagnosticErrorCategory(err error) string {
	var dns *net.DNSError
	var authority x509.UnknownAuthorityError
	var certificate x509.CertificateInvalidError
	var network net.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.As(err, &dns):
		return "dns_error"
	case errors.As(err, &authority), errors.As(err, &certificate):
		return "tls_certificate_error"
	case errors.As(err, &network) && network.Timeout():
		return "network_timeout"
	case errors.Is(err, io.EOF):
		return "eof"
	case errors.Is(err, io.ErrUnexpectedEOF):
		return "unexpected_eof"
	case errors.Is(err, syscall.ECONNRESET):
		return "connection_reset"
	case errors.Is(err, syscall.ECONNREFUSED):
		return "connection_refused"
	case errors.Is(err, syscall.EPIPE):
		return "broken_pipe"
	case errors.Is(err, net.ErrClosed), errors.Is(err, io.ErrClosedPipe):
		return "connection_closed"
	default:
		return "transport_error"
	}
}

func (d *textDiagnostics) set(key string, value any) {
	d.mu.Lock()
	d.fields[key] = value
	d.mu.Unlock()
}

// These are callback observations, not provider execution or transport handoffs.
// Keep counters and the last failure together; later success cannot erase it.
func (d *textDiagnostics) traceResult(phase, prefix string, err error) {
	category := ""
	if err != nil {
		category = diagnosticErrorCategory(err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.fields["network_phase"] = phase
	countKey := prefix + "_success_count"
	if err != nil {
		countKey = prefix + "_failure_count"
		d.fields[prefix+"_last_failure_category"] = category
	}
	d.fields[countKey] = d.fields[countKey].(int) + 1
}

// Both sources are opaque, bounded hashes. Even a well-formed provider ID can
// echo a credential or private input; no raw ID is safe merely due to its shape.
func (d *textDiagnostics) response(status int, requestID string, raw []byte, adapter AdapterKind) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if status >= 100 && status <= 599 {
		d.status = status
		d.fields["http_status"] = status
	}
	if len(requestID) > 0 && len(requestID) <= 128 {
		d.fields["http_request_id_hash"] = diagnosticHash(requestID)
	}
	d.validJSON = json.Valid(raw)
	if adapter == AdapterOpenAICompatible && d.validJSON {
		var metadata struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &metadata) == nil && len(metadata.ID) > 0 && len(metadata.ID) <= 128 {
			d.fields["completion_id_hash"] = diagnosticHash(metadata.ID)
		}
	}
}

func (d *textDiagnostics) sdkCategory(err error) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.status >= 300 {
		return "http_error"
	}
	var syntax *json.SyntaxError
	var valueType *json.UnmarshalTypeError
	if d.status >= 200 && (!d.validJSON || errors.As(err, &syntax) || errors.As(err, &valueType)) {
		return "response_json_invalid"
	}
	return "sdk_error"
}

func (d *textDiagnostics) trace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart:          func(httptrace.DNSStartInfo) { d.set("network_phase", "dns") },
		ConnectStart:      func(string, string) { d.set("network_phase", "connect") },
		TLSHandshakeStart: func() { d.set("network_phase", "tls") },
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			d.traceResult("tls_done", "tls_handshake", err)
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			d.traceResult("write_attempt_done", "wrote_request", info.Err)
		},
		GotFirstResponseByte: func() { d.set("network_phase", "first_response_byte") },
	}
}

func (d *textDiagnostics) emit(ctx context.Context, stage, category string) {
	if d == nil || d.logger == nil {
		return
	}
	d.mu.Lock()
	fields := make(logrus.Fields, len(d.fields)+5)
	for key, value := range d.fields {
		fields[key] = value
	}
	d.mu.Unlock()
	fields["stage"], fields["category"] = stage, category
	fields["context_state"], fields["caller_context_state"] = diagnosticContextState(ctx), diagnosticContextState(d.caller)
	fields["elapsed_ms"] = min(max(time.Since(d.started).Milliseconds(), 0), int64(120000))
	entry := d.logger.WithFields(fields)
	if category == "recorded" || category == "response_received" {
		entry.Info("governed text diagnostic")
	} else {
		entry.Warn("governed text diagnostic")
	}
}
