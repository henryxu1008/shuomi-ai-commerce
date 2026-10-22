package client

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSetCookieValueDoesNotLogCookieValues(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "create", true: "update"}[existing], func(t *testing.T) {
			var output bytes.Buffer
			log := logrus.New()
			log.SetOutput(&output)
			c := NewAPIClient(1, DefaultConfig(), "", nil, func(component string) *logrus.Entry {
				return log.WithFields(logrus.Fields{"component": component, "request_id": "synthetic-request"})
			})
			if c.logger.Logger != log || c.logger.Data["component"] != "TEMUAPIClient" || c.logger.Data["request_id"] != "synthetic-request" || c.logger.Data["storeID"] != int64(1) {
				t.Fatal("API client lost its injected logger or existing context")
			}
			if existing {
				c.SetCookies([]*http.Cookie{{Name: "fixture_cookie", Value: "old_fixture_value"}})
			}
			const secret = "synthetic_cookie_value_893"
			c.SetCookieValue("fixture_cookie", secret)
			if c.GetCookieValue("fixture_cookie") != secret || c.GetCookieCount() != 1 {
				t.Fatal("Cookie write/select behavior changed")
			}
			if strings.Contains(output.String(), secret) {
				t.Fatalf("Cookie value reached diagnostic output: %s", output.String())
			}
			if !strings.Contains(output.String(), "fixture_cookie") {
				t.Fatal("Cookie name/action diagnostics were lost")
			}
		})
	}
}
