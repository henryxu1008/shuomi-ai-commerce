package fulfillmentmodule

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	app "task-processor/internal/app/fulfillment"
	runtime "task-processor/internal/app/runtime/fulfillment"
	"task-processor/internal/authidentity"
	f "task-processor/internal/fulfillment"
	sf "task-processor/internal/marketplace/shein/fulfillment"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type runningWorker struct{}

func (runningWorker) Start() error { return nil }
func (runningWorker) Stop()        {}

type merchant struct{}

func (merchant) Check(context.Context) error { return nil }
func (merchant) BindingDigest() string       { return strings.Repeat("a", 64) }
func (merchant) Source(context.Context, string) (sf.Source, error) {
	return sf.Source{}, f.ErrUnavailable
}

type receiver struct{ calls []f.Command }

func (r *receiver) Execute(_ context.Context, c f.Command) (json.RawMessage, error) {
	r.calls = append(r.calls, c)
	return json.Marshal(f.Receipt{Operation: c.OperationID, Stage: "SUCCEEDED", Target: "native", Result: json.RawMessage(`{}`)})
}
func TestCommandsBindOnlyCurrentIdentityAndRejectInjectedOrWorkerFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	erp := &receiver{}
	lifecycle := &runtime.Lifecycle{Worker: runningWorker{}}
	if err := lifecycle.Start(); err != nil {
		t.Fatal(err)
	}
	m := &Module{Service: &app.Service{ERPs: map[string]f.ERP{"org": erp}, Open: func(_ context.Context, s storecenter.FulfillmentSubject, store, kind string) (app.Merchant, error) {
		if s.OrganizationID != "org" || s.ActorID != "actor" || s.MemberID != "member" || s.Purpose != "fulfillment.stock" {
			t.Fatal("wrong current authority", s)
		}
		return merchant{}, nil
	}}, dependencies: Dependencies{Lifecycle: lifecycle, Bind: func(ctx context.Context, _ string) (context.Context, error) {
		return authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)}), nil
	}}}
	router := gin.New()
	router.POST(BasePath+"/commands", func(c *gin.Context) { m.handle(c, "POST") })
	valid := `{"store":"store","kind":"platform_supply","source_id":"original","operation_id":"a182d2e0-6f91-4e71-8b7f-5df25b75fb11","action":"receive","payload":{"purchase":"native-po","lines":[{"purchase_line":"row","qty":2}]}}`
	for _, body := range []string{strings.Replace(valid, `"store":"store"`, `"organization":"foreign","store":"store"`, 1), strings.Replace(valid, `"action":"receive"`, `"action":"claim"`, 1), strings.Replace(valid, `"store":"store"`, `"store":"other","store":"store"`, 1)} {
		writer := httptest.NewRecorder()
		request := httptest.NewRequest("POST", BasePath+"/commands", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(writer, request)
		if writer.Code != 400 || len(erp.calls) != 0 {
			t.Fatal(writer.Code, writer.Body.String(), erp.calls)
		}
	}
	writer := httptest.NewRecorder()
	request := httptest.NewRequest("POST", BasePath+"/commands", strings.NewReader(valid))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(writer, request)
	if writer.Code != 200 || len(erp.calls) != 1 || erp.calls[0].Actor != "actor" || erp.calls[0].Organization != "org" || writer.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal(writer.Code, writer.Body.String(), erp.calls)
	}
}
