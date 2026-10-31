package grsai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"task-processor/internal/ai"
	"testing"
	"time"
)

func TestSourceEditDisablesGRSAISubmitReplay(t *testing.T) {
	for _, mode := range []string{"lost_response", "partial_response", "307", "308", "running", "http_error"} {
		t.Run(mode, func(t *testing.T) {
			var submits, queries, redirects, observations atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					queries.Add(1)
					http.Error(w, "no query", 500)
					return
				}
				submits.Add(1)
				switch mode {
				case "lost_response":
					h, ok := w.(http.Hijacker)
					if !ok {
						t.Error("hijack unavailable")
						return
					}
					c, _, e := h.Hijack()
					if e == nil {
						_ = c.Close()
					}
				case "partial_response":
					w.Header().Set("Content-Length", "100")
					_, _ = w.Write([]byte("{"))
				case "307":
					http.Redirect(w, r, "/redirected", 307)
				case "308":
					http.Redirect(w, r, "/redirected", 308)
				case "running":
					_, _ = w.Write([]byte(`{"id":"original-job","status":"running"}`))
				default:
					http.Error(w, "unknown", 500)
				}
			}))
			defer server.Close()
			hc := server.Client()
			hc.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
			client := NewClient(Config{Model: "gpt-image-2.5", SubmitURL: server.URL, HTTPClient: hc})
			observe := func(context.Context, GenerationObservation) error { observations.Add(1); return nil }
			for _, retries := range []*int{func() *int { x := 0; return &x }(), nil} {
				submits.Store(0)
				_, err := client.EditImageOnce(context.Background(), &ai.ImageEditRequest{Model: "gpt-image-2.5", Quality: "auto", Prompt: "white background", Image: []byte("source"), ImageContentType: "image/png", MaxRetries: retries}, observe)
				if err == nil || submits.Load() != 1 || queries.Load() != 0 || redirects.Load() != 0 || observations.Load() != 0 {
					t.Fatalf("ambiguous outcome replayed: err=%v post=%d query=%d redirect=%d observer=%d", err, submits.Load(), queries.Load(), redirects.Load(), observations.Load())
				}
			}
		})
	}
}
func TestReferenceMaterializationBudgetCapsConcurrentReferences(t *testing.T) {
	budget := newReferenceMaterializationBudget(maxMaterializedReferenceBytes)
	releaseFirst, err := budget.acquire(context.Background(), maxMaterializedReferenceBytes)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	secondAcquired := make(chan func(), 1)
	go func() {
		release, acquireErr := budget.acquire(context.Background(), maxMaterializedReferenceBytes)
		if acquireErr == nil {
			secondAcquired <- release
		}
	}()
	select {
	case <-secondAcquired:
		t.Fatal("second reference acquired before the first lease was released")
	case <-time.After(25 * time.Millisecond):
	}

	releaseFirst()
	select {
	case releaseSecond := <-secondAcquired:
		releaseSecond()
	case <-time.After(time.Second):
		t.Fatal("second reference did not acquire after the first lease was released")
	}
}
func TestBuildSubmitURLUsesGenerateEndpoint(t *testing.T) {
	tests := []struct {
		name  string
		base  string
		model string
		want  string
	}{
		{
			name:  "nano model on v1 base",
			base:  "https://grsaiapi.com/v1",
			model: "nano-banana-fast",
			want:  "https://grsaiapi.com/v1/api/generate",
		},
		{
			name:  "gpt model on host only",
			base:  "https://grsai.dakka.com.cn",
			model: "gpt-image-2",
			want:  "https://grsai.dakka.com.cn/v1/api/generate",
		},
		{
			name:  "legacy draw path",
			base:  "https://grsai.dakka.com.cn/v1/draw/nano-banana",
			model: "nano-banana-fast",
			want:  "https://grsai.dakka.com.cn/v1/api/generate",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildSubmitURL(tt.base, tt.model)
			if err != nil {
				t.Fatalf("buildSubmitURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("buildSubmitURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
