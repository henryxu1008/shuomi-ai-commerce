package httpapi

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"strings"
	"task-processor/internal/product/collection"
	"testing"
)

func TestRevisionSourceVersionMustBeCanonicalString(t *testing.T) {
	for _, value := range []string{`"1"`, `"9223372036854775807"`, `"01"`, `"+1"`, `"0"`, `null`, `1`, `"9223372036854775808"`} {
		req := httptest.NewRequest("POST", BasePath+"/commands", strings.NewReader(fmt.Sprintf(`{"action":"revise_product","itemId":"%s","expectedRevision":1,"expectedSourceVersion":%s,"expectedPublicationId":"%s","product":{"title":"新版","images":[]}}`, uuid.NewString(), value, uuid.NewString())))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.NewString())
		_, _, err := readMutation(req)
		if value == `"1"` || value == `"9223372036854775807"` {
			require.NoError(t, err, value)
		} else {
			require.ErrorIs(t, err, collection.ErrInvalid, value)
		}
	}
}
