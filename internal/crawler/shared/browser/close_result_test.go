package browser

import (
	"errors"
	"github.com/mxschmitt/playwright-go"
	"testing"
)

type uncertainContext struct {
	playwright.BrowserContext
	failure error
	calls   int
}

func (c *uncertainContext) Close(...playwright.BrowserContextCloseOptions) error {
	c.calls++
	return c.failure
}
func TestManagerCloseRetainsUnknownClosureResult(t *testing.T) {
	failure := errors.New("synthetic close acknowledgement lost")
	native := &uncertainContext{failure: failure}
	m := &Manager{context: native}
	if e := m.CloseWithResult(); !errors.Is(e, failure) {
		t.Fatal("lost closure proof", e)
	}
	if e := m.CloseWithResult(); !errors.Is(e, failure) {
		t.Fatal("second Close invented successful closure", e)
	}
	if native.calls != 1 || m.GetContext() != nil {
		t.Fatal("native owner lifecycle not closed exactly once")
	}
}
