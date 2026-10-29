package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"task-processor/internal/authidentity"
)

func TestBusinessRoleHTTPFixesDirectionAndRejectsRoleSets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, direction := range []string{"grant", "revoke"} {
		for _, invalid := range []bool{false, true} {
			commands := &commandStub{}
			factories := 0
			h := NewCommandHandler(nil, func(*http.Request) (CommandService, error) { factories++; return commands, nil })
			router := gin.New()
			handler := h.GrantBusinessRole
			if direction == "revoke" {
				handler = h.RevokeBusinessRole
			}
			router.POST("/api/v1/account/members/:member_id/business-roles/"+direction, handler)
			body := `{"role":"selected","expectedVersion":"` + strings.Repeat("a", 64) + `"}`
			if invalid {
				body = `{"roleKeys":["platform_admin"],"nativeSequence":"7"}`
			}
			r := httptest.NewRequest("POST", "/api/v1/account/members/grant/business-roles/"+direction, strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Idempotency-Key", uuid.NewString())
			r.Header.Set("X-Requested-Organization-ID", "org")
			r = r.WithContext(authidentity.WithAuthenticatedIdentity(r.Context(), authidentity.AuthenticatedIdentity{UserID: "actor", EffectiveOrganizationID: "org", TokenExpiresAt: time.Now().Add(time.Hour)}))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if invalid {
				if w.Code != 400 || factories != 0 {
					t.Fatal("untrusted plan reached command")
				}
			} else if w.Code != 200 || factories != 1 || !strings.Contains(w.Body.String(), `"kind":"business_role_`+direction+`"`) {
				t.Fatalf("direction=%s status=%d body=%s", direction, w.Code, w.Body.String())
			}
		}
	}
}
