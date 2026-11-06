package httpapi

import (
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
	"time"
)

func TestImageAnalysisConfirmUsesOriginalBoundedAgentWriteDeadline(t *testing.T) {
	id, key := "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"
	for _, owner := range []string{productAcquisitionBase + "/", "/api/v1/workbench/supply-preparations/sources/", "/api/v1/workbench/collections/items/"} {
		for _, tc := range []struct {
			method, path string
			extended     bool
		}{
			{"POST", owner + id + "/images/analysis/by-key/" + key + "/confirm", true},
			{"GET", owner + id + "/images/analysis/by-key/" + key + "/confirm", false},
			{"POST", owner + id + "/images/analysis/prepare", false},
			{"GET", owner + id + "/images/analysis/by-key/" + key, false},
			{"POST", owner + "invalid/images/analysis/by-key/" + key + "/confirm", false},
			{"POST", owner + id + "/images/analysis/by-key/invalid/confirm", false},
			{"POST", owner + id + "/images/analysis/by-key/" + key + "/confirm/extra", false},
		} {
			t.Run(tc.method+tc.path, func(t *testing.T) {
				server := buildCurrentApplicationHTTPServer(nil, routeAuthDependencies{})
				require.Equal(t, 32*time.Second, server.WriteTimeout)
				writer := &materialsDeadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
				before := time.Now()
				server.Handler.ServeHTTP(writer, httptest.NewRequest(tc.method, tc.path, nil))
				if tc.extended {
					require.WithinDuration(t, before.Add(125*time.Second), writer.deadline, time.Second)
				} else {
					require.True(t, writer.deadline.IsZero())
				}
			})
		}
	}
}
