package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSelectionHTTPServerUsesBoundedAgentWriteDeadline(t *testing.T) {
	const base = productSelectionBase + "/contexts/11111111-1111-4111-8111-111111111111"
	const key = "22222222-2222-4222-8222-222222222222"
	for _, tc := range []struct {
		method, path string
		long         bool
	}{
		{http.MethodPost, base + "/runs", true},
		{http.MethodPost, base + "/runs/" + key + "/resume", true},
		{http.MethodGet, base + "/runs/" + key, false},
		{http.MethodGet, base + "/runs", false},
		{http.MethodPost, productSelectionBase + "/prepare", false},
		{http.MethodPost, base + "/runs/not-a-uuid/resume", false},
		{http.MethodPost, base + "/runs/" + key + "/resume/extra", false},
		{http.MethodPost, productSelectionBase + "/contexts/not-a-uuid/runs", false},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			server := buildCurrentApplicationHTTPServer(nil, routeAuthDependencies{})
			writer := &materialsDeadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
			before := time.Now()
			server.Handler.ServeHTTP(writer, httptest.NewRequest(tc.method, tc.path, nil))
			if tc.long {
				require.WithinDuration(t, before.Add(125*time.Second), writer.deadline, time.Second)
			} else {
				require.True(t, writer.deadline.IsZero())
			}
		})
	}
}
