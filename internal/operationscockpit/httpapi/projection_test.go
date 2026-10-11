package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	c "task-processor/internal/operationscockpit"
)

type projectionFixture struct {
	c.Repository
	stores []c.StoreReference
	query  c.Query
	reads  int
}

func (r *projectionFixture) Snapshot(_ context.Context, _ c.Scope, q c.Query) (c.Snapshot, error) {
	r.query = q
	r.reads++
	s := c.Snapshot{Period: q.Period, CapturedAt: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), Stores: map[string]c.StoreAggregate{}}
	for _, id := range q.StoreIDs {
		var n int
		fmt.Sscanf(id, "123e4567-e89b-42d3-a456-%012d", &n)
		s.Stores[id] = c.StoreAggregate{StoreID: id, Period: q.Period, Complete: n >= 35, Totals: c.Totals{Procurement: 100, NetProfit: -100}}
	}
	s.Goal = &c.GoalVersion{ID: r.stores[0].ID, Revision: 1, Config: c.GoalConfig{StoreIDs: []string{r.stores[0].ID}, Frequency: "month", Period: c.Period{Start: "2026-10-01", End: "2026-10-31"}, Profit: 10000, NormalBPS: 9000, AttentionBPS: 7000}}
	s.Evaluation = &c.GoalEvaluation{State: "abnormal", Through: "2026-10-09"}
	return s, nil
}

func projectionRouter(t *testing.T) (*gin.Engine, *projectionFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := &projectionFixture{}
	for i := 0; i < 60; i++ {
		platform, region := "shein", "US"
		if i%2 == 1 {
			platform = "other"
		}
		if i >= 30 {
			region = "CA"
		}
		r.stores = append(r.stores, c.StoreReference{ID: fmt.Sprintf("123e4567-e89b-42d3-a456-%012d", i), Name: fmt.Sprintf("店铺-%02d", i), Platform: platform, Region: region})
	}
	h := &Handler{Repository: r, Bind: func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil }, Scope: func(context.Context) (c.Scope, error) { return c.Scope{}, nil }, Current: func(context.Context, c.Scope) (c.Access, error) {
		return c.Access{StoresRead: true, AlertsRead: true, AdviceRead: true, GoalsRead: true}, nil
	}, Directory: func(context.Context, c.Scope) ([]c.StoreReference, error) { return r.stores, nil }, Revalidate: func(context.Context, c.Scope, []string) error { return nil }}
	routes, err := BuildRoutes(h)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	for _, route := range routes {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	return router, r
}

func projectionGET(router *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", Base+path, nil))
	return w
}

const projectionPeriod = "startDate=2026-10-01&endDate=2026-10-09"

func TestRuleSummaryCountsFilteredResultsBeforePagination(t *testing.T) {
	for _, module := range []string{"alerts", "advice"} {
		t.Run(module, func(t *testing.T) {
			router, _ := projectionRouter(t)
			for _, tc := range []struct {
				query                                  string
				total, critical, attention, data, rows int
			}{
				{"&page=2", 61, 1, 25, 35, 11},
				{"&level=attention", 25, 0, 25, 0, 25},
				{"&kind=data_missing&search=" + url.QueryEscape("店铺-0"), 10, 0, 0, 10, 10},
				{"&search=" + url.QueryEscape("目标评判"), 1, 1, 0, 0, 1},
				{"&search=missing", 0, 0, 0, 0, 0},
			} {
				w := projectionGET(router, "/"+module+"?"+projectionPeriod+tc.query)
				var body struct {
					Rules   []c.Rule
					Total   int
					Summary map[string]int
				}
				if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil {
					t.Fatalf("%d %s", w.Code, w.Body.String())
				}
				if body.Total != tc.total || len(body.Rules) != tc.rows || body.Summary["total"] != tc.total || body.Summary["critical"] != tc.critical || body.Summary["attention"] != tc.attention || body.Summary["data"] != tc.data {
					t.Fatalf("wrong full summary: %+v", body)
				}
			}
		})
	}
}

func TestMatrixMetadataFiltersBoundScopeBeforeSnapshot(t *testing.T) {
	router, repo := projectionRouter(t)
	w := projectionGET(router, "/stores?"+projectionPeriod+"&platform=shein&region=US&search="+url.QueryEscape("店铺-00")+"&state=data_incomplete")
	var body struct {
		Total  int
		States map[string]int
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	if len(repo.query.StoreIDs) != 15 || body.Total != 1 || body.States["data_incomplete"] != 15 {
		t.Fatalf("wrong scope/count: ids=%v body=%+v", repo.query.StoreIDs, body)
	}
	w = projectionGET(router, "/stores?"+projectionPeriod+"&platform=missing")
	if w.Code != 200 || len(repo.query.StoreIDs) != 0 {
		t.Fatalf("empty filter expanded: %d %+v", w.Code, repo.query)
	}
	before := repo.reads
	w = projectionGET(router, "/stores?"+projectionPeriod+"&platform=missing&storeId=123e4567-e89b-42d3-a456-999999999999")
	if w.Code != 403 || repo.reads != before {
		t.Fatalf("filter hid unauthorized ID: %d", w.Code)
	}
}

func TestMatrixSummaryRetainsCompleteLossAcrossListFiltersAndPages(t *testing.T) {
	router, _ := projectionRouter(t)
	w := projectionGET(router, "/stores?"+projectionPeriod+"&state=loss&page=2&search="+url.QueryEscape("店铺-5"))
	var body struct {
		Rows    []matrixRow
		Total   int
		States  map[string]int
		Summary c.Totals
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	if len(body.Rows) != 0 || body.Total != 10 || body.States["loss"] != 25 || body.States["data_incomplete"] != 35 || body.Summary.NetProfit != -2500 {
		t.Fatalf("list filter rewrote scope statistics: total=%d states=%v summary=%+v", body.Total, body.States, body.Summary)
	}
}

func TestProjectionRejectsAmbiguousOrUnboundedFilters(t *testing.T) {
	router, repo := projectionRouter(t)
	for _, path := range []string{
		"/stores?" + projectionPeriod + "&platform=x&platform=y",
		"/stores?" + projectionPeriod + "&platform=" + url.QueryEscape(strings.Repeat("中", 65)),
		"/stores?" + projectionPeriod + "&region=" + url.QueryEscape(strings.Repeat("中", 101)),
		"/alerts?" + projectionPeriod + "&search=" + url.QueryEscape(strings.Repeat("中", 101)),
		"/advice?" + projectionPeriod + "&level=unknown",
		"/alerts?" + projectionPeriod + "&kind=unknown",
		"/stores?" + projectionPeriod + "&state=normal",
		"/stores?" + projectionPeriod + "&sort=unknown",
		"/goals?" + projectionPeriod + "&platform=shein",
	} {
		w := projectionGET(router, path)
		if w.Code != 400 || repo.reads != 0 {
			t.Fatalf("invalid query consumed: %s %d", path, w.Code)
		}
	}
	w := projectionGET(router, "/alerts?"+projectionPeriod+"&search="+url.QueryEscape(strings.Repeat("中", 100)))
	if w.Code != 200 {
		t.Fatalf("valid unicode bound rejected: %d", w.Code)
	}
}
