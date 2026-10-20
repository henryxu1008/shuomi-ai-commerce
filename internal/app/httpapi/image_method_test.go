package httpapi

import (
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImagePreparationAcceptsOnlyBoundedSkillSelectionFromBrowser(t *testing.T) {
	base := `{"skill":{"skillId":"2bb819bd-af13-4e68-95e0-1cc86130fb1f","version":1,"digest":"` + strings.Repeat("a", 64) + `"}}`
	var input fullImagePrepareBody
	req := httptest.NewRequest("POST", "/", strings.NewReader(base))
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, readFullImageJSON(req, &input))
	bad := strings.Replace(base, `"version":1`, `"version":1,"ownerKind":"PLATFORM"`, 1)
	req = httptest.NewRequest("POST", "/", strings.NewReader(bad))
	req.Header.Set("Content-Type", "application/json")
	require.Error(t, readFullImageJSON(req, &input))
}
