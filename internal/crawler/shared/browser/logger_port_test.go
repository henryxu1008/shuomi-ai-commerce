package browser

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestChromeDownloaderUsesInjectedRawLoggerWithoutDownloading(t *testing.T) {
	var output bytes.Buffer
	raw := logrus.New()
	raw.SetOutput(&output)
	cd := NewChromeDownloaderWithTimeout("144.0.0.0", "unused-fixture-path", time.Second, func(component string) *logrus.Entry {
		return raw.WithField("component", component)
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	defer server.Close()
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := cd.httpClient.Do(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if !strings.Contains(output.String(), "HTTP请求成功") {
		t.Fatal("HTTP client did not use the injected raw logger")
	}
}

func TestMissingDiagnosticDependencyIsInstanceLocal(t *testing.T) {
	for _, logs := range []func(string) *logrus.Entry{nil, func(string) *logrus.Entry { return nil }} {
		entry := componentLogger(logs, "fixture")
		if entry.Logger.Out != io.Discard || entry.Data["component"] != "fixture" {
			t.Fatal("missing diagnostic dependency must not look up a global manager or output")
		}
	}
}
