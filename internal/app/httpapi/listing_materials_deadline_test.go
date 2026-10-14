package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMaterialsModelContextRetainsTheLocalSaveReserve(t *testing.T) {
	parentDeadline := time.Now().Add(45 * time.Second)
	parent, stop := context.WithDeadline(context.Background(), parentDeadline)
	defer stop()
	model, cancel := materialsModelContext(parent)
	deadline, ok := model.Deadline()
	require.True(t, ok)
	require.Equal(t, parentDeadline.Add(-30*time.Second), deadline)
	cancel()
	require.NoError(t, parent.Err(), "automatic save keeps its enclosing context")
	short, stopShort := context.WithTimeout(context.Background(), time.Second)
	defer stopShort()
	model, cancel = materialsModelContext(short)
	defer cancel()
	require.ErrorIs(t, model.Err(), context.DeadlineExceeded, "too little route time cannot dispatch model work")
}

type materialsDeadlineRecorder struct {
	*httptest.ResponseRecorder
	deadline time.Time
}

func (w *materialsDeadlineRecorder) SetWriteDeadline(deadline time.Time) error {
	w.deadline = deadline
	return nil
}

func TestMaterialsHTTPServerUsesBoundedAgentWriteDeadline(t *testing.T) {
	const base = "/api/v1/workbench/supply-preparations/records/11111111-1111-4111-8111-111111111111/materials-agent"
	const key = "22222222-2222-4222-8222-222222222222"
	for _, tc := range []struct {
		method, path string
		long         bool
	}{
		{http.MethodPost, base + "/runs", true},
		{http.MethodPost, base + "/runs/" + key + "/resume", true},
		{http.MethodPost, base + "/runs/" + key + "/adopt", true},
		{http.MethodGet, base + "/runs/" + key + "/adoption", false},
		{http.MethodGet, base + "/runs", false},
		{http.MethodPost, base + "/prepare", false},
		{http.MethodPost, base + "/runs/not-a-uuid/adopt", false},
		{http.MethodPost, base + "/runs/" + key + "/adopt/extra", false},
		{http.MethodPost, "/api/v1/workbench/supply-preparations/records/not-a-uuid/materials-agent/runs", false},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			server := buildCurrentApplicationHTTPServer(nil, routeAuthDependencies{})
			require.Equal(t, 32*time.Second, server.WriteTimeout)
			writer := &materialsDeadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
			before := time.Now()
			server.Handler.ServeHTTP(writer, httptest.NewRequest(tc.method, tc.path, nil))
			if tc.long {
				require.WithinDuration(t, before.Add(125*time.Second), writer.deadline, time.Second)
			} else {
				require.True(t, writer.deadline.IsZero(), "ordinary and malformed routes retain the server deadline")
			}
		})
	}
}
