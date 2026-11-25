package toapis

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFlareUsesOneReferenceAndReadsOriginalTask(t *testing.T) {
	const source = "https://cbu01.alicdn.com/img/ibank/O1CN_fixture.jpg"
	const task = "tsk_img_fixture"
	var post, get int
	client := NewClient("fixture-key", &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "api.toapis.cn" || r.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Fatalf("request escaped fixed API origin: %s", r.URL)
		}
		body := `{"id":"tsk_img_fixture","model":"gpt-image-2.6-flare","status":"pending"}`
		if r.Method == http.MethodPost {
			post++
			var request map[string]any
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			refs, ok := request["reference_images"].([]any)
			if !ok || len(refs) != 1 || refs[0] != source || request["n"] != float64(1) ||
				request["model"] != "gpt-image-2.6-flare" || request["quality"] != "high" ||
				request["size"] != "1:1" || request["resolution"] != "1K" || request["output_format"] != "png" ||
				r.URL.Path != "/v1/images/generations" || r.GetBody != nil {
				t.Fatalf("unexpected submission: %#v", request)
			}
		} else {
			get++
			if r.Method != http.MethodGet || r.URL.Path != "/v1/images/generations/"+task {
				t.Fatalf("read did not use saved task: %s %s", r.Method, r.URL)
			}
			body = `{"id":"tsk_img_fixture","status":"completed","billing":{"status":"pending"},"result":{"type":"image","data":[{"url":"https://example.com/result.png"}]}}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})})
	id, err := client.Start(context.Background(), "preserve the original coat", source)
	if err != nil || id != task || post != 1 || get != 0 {
		t.Fatalf("want one submission and no lookup before task persistence: %s %v", id, err)
	}
	result, err := client.Lookup(context.Background(), id)
	if err != nil || result.TaskID != id || result.Status != "succeeded" || result.URL != "https://example.com/result.png" || len(result.Digest) != 64 || post != 1 || get != 1 {
		t.Fatalf("unexpected result: %+v %v", result, err)
	}
}

func TestFlareDoesNotFollowCredentialRedirect(t *testing.T) {
	var calls int
	client := NewClient("fixture-key", &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 307, Header: http.Header{"Location": {"https://example.com/other"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})})
	id, err := client.Start(context.Background(), "product", "https://cbu01.alicdn.com/img/ibank/coat.jpg")
	if id != "" || err == nil || calls != 1 {
		t.Fatalf("redirect was not treated as an uncertain single submission: %q %v, %d", id, err, calls)
	}
}
