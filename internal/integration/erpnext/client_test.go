package erpnext

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	f "task-processor/internal/fulfillment"
	"testing"
)

func TestNativeMethodAndLostResponse(t *testing.T) {
	received := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received++
		if r.URL.Path != "/api/method/commerce_fulfillment.api.execute" || r.Method != "POST" || r.Header.Get("Authorization") != "token test-key:test-secret" {
			t.Error("lost fixed credential/method boundary")
		}
		if received == 1 {
			fmt.Fprint(w, `{"message":{"operation":"original","stage":"SUCCEEDED"}}`)
			return
		}
		fmt.Fprint(w, `{"message":`)
	}))
	defer server.Close()
	client, err := New(Config{Origin: server.URL, Organization: "org", APIKey: "test-key", APISecret: "test-secret"}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	request := f.Command{OperationID: "fa7daf54-70c0-4a89-ab74-9b815a8e2cdf", Organization: "org", Actor: "actor", Member: "member", Binding: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Command: "read", Payload: json.RawMessage(`{"operation":"original"}`)}
	result, err := client.Execute(context.Background(), request)
	if err != nil || !json.Valid(result) {
		t.Fatalf("%s %v", result, err)
	}
	_, err = client.Execute(context.Background(), request)
	if !errors.Is(err, f.ErrUnknown) || received != 2 {
		t.Fatalf("lost response retried or not UNKNOWN: %v %d", err, received)
	}
	request.Organization = "other"
	if _, err = client.Execute(context.Background(), request); !errors.Is(err, f.ErrForbidden) || received != 2 {
		t.Fatal("cross-site request sent")
	}
}

func TestOriginAndRedirect(t *testing.T) {
	for _, origin := range []string{"http://localhost:8000", "https://erp.invalid/base", "https://user:password@erp.invalid", "https://erp.invalid?site=other"} {
		if _, err := New(Config{Origin: origin, Organization: "org", APIKey: "k", APISecret: "s"}, nil); err == nil {
			t.Fatal("unsafe origin", origin)
		}
	}
	reached := false
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))
	defer target.Close()
	redirect := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	client, _ := New(Config{Origin: redirect.URL, Organization: "org", APIKey: "k", APISecret: "s"}, redirect.Client())
	_, err := client.Execute(context.Background(), f.Command{Organization: "org", Command: "read", Payload: json.RawMessage(`{}`)})
	if !errors.Is(err, f.ErrUnknown) || reached {
		t.Fatal("redirect followed or misclassified", err)
	}
}
