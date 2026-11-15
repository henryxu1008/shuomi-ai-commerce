// Package skillsandbox is the fixed OpenSandbox adapter. It owns no business
// authorization, durable run, SSE parser, Docker client or recovery worker.
package skillsandbox

import (
	"errors"
	"io"
	"net/http"
	"strings"

	osb "github.com/alibaba/OpenSandbox/sdks/sandbox/go"
	strict "sigs.k8s.io/json"
)

var errBudget = errors.New("sandbox payload budget exceeded")
var errOrigin = errors.New("sandbox endpoint rejected")
var errProtocol = errors.New("sandbox protocol rejected")

type limitedBody struct {
	io.ReadCloser
	left int64
}

func (b *limitedBody) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if b.left == 0 {
		var probe [1]byte
		n, err := b.ReadCloser.Read(probe[:])
		if n > 0 {
			_ = b.Close()
			return 0, errBudget
		}
		return 0, err
	}
	if int64(len(p)) > b.left {
		p = p[:b.left]
	}
	n, err := b.ReadCloser.Read(p)
	b.left -= int64(n)
	return n, err
}

type boundedTransport struct {
	base   http.RoundTripper
	origin string
}

func (t boundedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme+"://"+r.URL.Host != t.origin || r.URL.User != nil {
		return nil, errOrigin
	}
	response, err := t.base.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	encoding := response.Header.Get("Content-Encoding")
	if encoding != "" && encoding != "identity" {
		_ = response.Body.Close()
		return nil, errProtocol
	}
	size := int64(16 << 10)
	if strings.HasSuffix(r.URL.Path, "/command") {
		size = 64 << 10
	}
	if strings.HasSuffix(r.URL.Path, "/files/download") {
		size = 8 << 10
	}
	response.Body = &limitedBody{ReadCloser: response.Body, left: size}
	return response, nil
}

// observe receives events already framed by the official SDK. Every event,
// including unknown types, counts; no high-level SDK terminal inference is used.
type observation struct {
	id                                   string
	events, init, complete, failed, logs int
	code, errorCode                      *int
}

func (o *observation) observe(event osb.StreamEvent) error {
	o.events++
	if o.events > 256 {
		return errBudget
	}
	var value struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		ExitCode *int   `json:"exit_code"`
		Error    *struct {
			ExitCode *int `json:"exit_code"`
		} `json:"error"`
	}
	violations, err := strict.UnmarshalStrict([]byte(event.Data), &value, strict.DisallowDuplicateFields)
	if err != nil || len(violations) != 0 {
		return errProtocol
	}
	switch value.Type {
	case "init":
		o.init++
		if value.Text == "" || len(value.Text) > 128 {
			return errProtocol
		}
		o.id = value.Text
	case "stdout", "stderr":
		o.logs += len(value.Text)
		if o.logs > 8<<10 {
			return errBudget
		}
	case "execution_complete":
		o.complete++
		o.code = value.ExitCode
	case "error":
		o.failed++
		if value.Error != nil {
			o.errorCode = value.Error.ExitCode
		}
	}
	return nil
}
func (o observation) outcome(err error) string {
	if err != nil || o.init != 1 || o.complete > 1 || o.failed > 1 || o.complete > 0 && o.failed > 0 {
		return "outcome_unknown"
	}
	if o.failed == 1 && o.errorCode != nil && *o.errorCode != 0 {
		return "failed"
	}
	if o.complete == 1 && o.code != nil && *o.code == 0 && o.failed == 0 {
		return "succeeded"
	}
	return "outcome_unknown"
}
