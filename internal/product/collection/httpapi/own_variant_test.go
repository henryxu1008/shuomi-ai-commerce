package httpapi

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOwnVariantNumbersMustBeDeclared(t *testing.T) {
	for _, fields := range []string{`"stock":0`, `"price":0`, `"price":null,"stock":0`, `"price":0,"stock":null`, `"price":0,"stock":0`} {
		t.Run(fields, func(t *testing.T) {
			req := httptest.NewRequest("POST", BasePath+"/commands", strings.NewReader(fmt.Sprintf(`{"action":"create_product","product":{"title":"商品","images":[],"variants":[{"sourceId":"red","title":"红","sku":"RED","attributes":{},"currency":"CNY",%s}]}}`, fields)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", uuid.NewString())
			_, _, err := readMutation(req)
			if fields == `"price":0,"stock":0` {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
