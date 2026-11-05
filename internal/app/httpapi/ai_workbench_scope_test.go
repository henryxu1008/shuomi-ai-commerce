package httpapi

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"task-processor/internal/aiworkbench"
)

func workbenchCreateFixtureBody(t *testing.T, server *httptest.Server, actor, org string, selection aiworkbench.ScopeSelection, favorite bool) string {
	t.Helper()
	q := url.Values{"kind": {selection.Kind}}
	if selection.ProjectID != "" {
		q.Set("projectId", selection.ProjectID)
	}
	if selection.StoreID != "" {
		q.Set("storeId", selection.StoreID)
	}
	status, raw, err := acquisitionHTTPRequest(server, "GET", workbenchChatBase+"/scopes?"+q.Encode(), actor, org, "", "")
	require.NoError(t, err)
	require.Equal(t, 200, status, string(raw))
	var page struct {
		Options []struct{ Confirmation string }
	}
	require.NoError(t, json.Unmarshal(raw, &page))
	require.Len(t, page.Options, 1)
	body, err := json.Marshal(map[string]any{"scopeSelection": selection, "scopeConfirmation": page.Options[0].Confirmation, "favorite": favorite})
	require.NoError(t, err)
	return string(body)
}
