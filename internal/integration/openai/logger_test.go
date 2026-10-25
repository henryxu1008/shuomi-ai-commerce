package openai

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

type recordingLogger struct {
	messages []string
	fields   []map[string]any
}

func (l *recordingLogger) record(message string, fields map[string]any) {
	l.messages = append(l.messages, message)
	l.fields = append(l.fields, fields)
}
func (l *recordingLogger) Debug(message string, fields map[string]any) { l.record(message, fields) }
func (l *recordingLogger) Info(message string, fields map[string]any)  { l.record(message, fields) }
func (l *recordingLogger) Warn(message string, fields map[string]any)  { l.record(message, fields) }
func (l *recordingLogger) Error(message string, fields map[string]any) { l.record(message, fields) }

func TestAdaptLogrusIsNilSafeAndForwardsStructuredFields(t *testing.T) {
	AdaptLogrus(nil).Info("nil-safe", map[string]any{"key": "value"})
	base := logrus.New()
	var output bytes.Buffer
	base.SetOutput(&output)
	base.SetFormatter(&logrus.JSONFormatter{})
	AdaptLogrus(logrus.NewEntry(base)).Info("provider-log", map[string]any{"provider": "openai"})
	got := output.String()
	if !strings.Contains(got, `"msg":"provider-log"`) || !strings.Contains(got, `"provider":"openai"`) {
		t.Fatalf("structured log output = %s", got)
	}
}

func TestNewManagerPropagatesLoggerToStaticClients(t *testing.T) {
	recorder := &recordingLogger{}
	manager, err := NewManager(&ManagerConfig{
		Clients: map[string]*ClientConfig{"default": testClientConfig("key", "model", "https://example.test/v1")},
		Logger:  recorder,
	})
	if err != nil {
		t.Fatal(err)
	}
	if manager.logger != recorder || manager.clients["default"].logger != recorder || manager.clients["default"].pool.logger != recorder {
		t.Fatal("manager logger was not propagated to the static client graph")
	}
	if len(recorder.messages) == 0 || len(recorder.fields) == 0 || recorder.fields[0]["client"] != "default" {
		t.Fatalf("constructor logs = %#v / %#v", recorder.messages, recorder.fields)
	}
}

func TestManagerPropagatesLoggerToResolverCreatedDynamicClient(t *testing.T) {
	recorder := &recordingLogger{}
	manager, err := NewManager(&ManagerConfig{
		Clients: map[string]*ClientConfig{"image": testClientConfig("static", "model", "https://static.example.test/v1")},
		ConfigResolver: fakeClientConfigResolver{tenantConfigs: map[string]*ClientConfig{
			"tenant-a": testClientConfig("dynamic", "dynamic-model", "https://dynamic.example.test/v1"),
		}},
		Logger: recorder,
	})
	if err != nil {
		t.Fatal(err)
	}
	client, err := manager.resolveClient(WithTenantID(context.Background(), "tenant-a"), "image")
	if err != nil {
		t.Fatal(err)
	}
	if client.logger != recorder || client.pool.logger != recorder || client.config.Logger != recorder {
		t.Fatal("manager logger was not propagated to a resolver-created client")
	}
}
