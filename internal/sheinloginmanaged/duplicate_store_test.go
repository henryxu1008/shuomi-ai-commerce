package sheinloginmanaged

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"task-processor/internal/listingadmin"
	"task-processor/internal/sheinlogin"
)

func TestFindDuplicateStoreScopesEveryPageAndPreservesEligibleDuplicates(t *testing.T) {
	first := make([]*listingadmin.StoreRespDTO, 200)
	for i := range first {
		first[i] = &listingadmin.StoreRespDTO{ID: int64(i + 100), TenantID: 7, Platform: "SHEIN", StoreID: "456"}
	}
	first[0] = &listingadmin.StoreRespDTO{ID: 5, TenantID: 7, Platform: "SHEIN", StoreID: "123"} // self
	first[1] = &listingadmin.StoreRespDTO{ID: 6, TenantID: 7, Platform: "SHEIN", StoreID: "123", Status: 1}
	first[2] = &listingadmin.StoreRespDTO{ID: 7, TenantID: 7, Platform: "TEMU", StoreID: "123"}
	want := &listingadmin.StoreRespDTO{ID: 99, TenantID: 7, Platform: " shein ", StoreID: "123"}
	calls := 0
	api := stubStoreAPI{pageFunc: func(req *listingadmin.StorePageReqDTO) (*listingadmin.PageResult[*listingadmin.StoreRespDTO], error) {
		calls++
		require.Equal(t, int64(7), req.TenantID, "each page must be tenant-qualified")
		require.Equal(t, "SHEIN", req.Platform)
		require.Equal(t, calls, req.PageNo)
		require.Equal(t, 200, req.PageSize)
		rows := first
		if calls == 2 {
			rows = []*listingadmin.StoreRespDTO{want}
		}
		return &listingadmin.PageResult[*listingadmin.StoreRespDTO]{List: rows, Total: 201, PageNo: calls, PageSize: 200}, nil
	}}
	got, err := findDuplicateStore(context.Background(), api, sheinlogin.Account{TenantID: 7, StoreID: 5}, "123")
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.Equal(t, 2, calls)
}

func TestFindDuplicateStoreRejectsInvalidInputBeforeQuery(t *testing.T) {
	for _, tc := range []struct {
		name    string
		account sheinlogin.Account
		actual  string
	}{
		{"zero tenant", sheinlogin.Account{StoreID: 5}, "123"},
		{"negative tenant", sheinlogin.Account{TenantID: -1, StoreID: 5}, "123"},
		{"zero row", sheinlogin.Account{TenantID: 7}, "123"},
		{"negative row", sheinlogin.Account{TenantID: 7, StoreID: -1}, "123"},
		{"empty external id", sheinlogin.Account{TenantID: 7, StoreID: 5}, ""},
		{"invalid external id", sheinlogin.Account{TenantID: 7, StoreID: 5}, "abc"},
		{"zero external id", sheinlogin.Account{TenantID: 7, StoreID: 5}, "0"},
		{"negative external id", sheinlogin.Account{TenantID: 7, StoreID: 5}, "-1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := stubStoreAPI{pageFunc: func(*listingadmin.StorePageReqDTO) (*listingadmin.PageResult[*listingadmin.StoreRespDTO], error) {
				t.Fatal("invalid scope/id must not query StoreAPI")
				return nil, nil
			}}
			got, err := findDuplicateStore(context.Background(), api, tc.account, tc.actual)
			require.Error(t, err)
			require.Nil(t, got)
		})
	}
}

func TestFindDuplicateStoreRejectsIncompleteOrWrongScopeResponses(t *testing.T) {
	valid := &listingadmin.StoreRespDTO{ID: 99, TenantID: 7, Platform: "SHEIN", StoreID: "123"}
	foreign := &listingadmin.StoreRespDTO{ID: 987654, TenantID: 876543, Name: "foreign-private-name", Platform: "SHEIN", StoreID: "123"}
	for _, tc := range []struct {
		name string
		page *listingadmin.PageResult[*listingadmin.StoreRespDTO]
		err  error
	}{
		{"nil page", nil, nil},
		{"query failure", nil, errors.New("foreign-private-name tenant=876543 row=987654")},
		{"wrong tenant after candidate", &listingadmin.PageResult[*listingadmin.StoreRespDTO]{List: []*listingadmin.StoreRespDTO{valid, foreign}, Total: 2, PageNo: 1, PageSize: 200}, nil},
		{"zero tenant", &listingadmin.PageResult[*listingadmin.StoreRespDTO]{List: []*listingadmin.StoreRespDTO{{ID: 99}}, Total: 1, PageNo: 1, PageSize: 200}, nil},
		{"nil row", &listingadmin.PageResult[*listingadmin.StoreRespDTO]{List: []*listingadmin.StoreRespDTO{nil}, Total: 1, PageNo: 1, PageSize: 200}, nil},
		{"missing rows", &listingadmin.PageResult[*listingadmin.StoreRespDTO]{Total: 1, PageNo: 1, PageSize: 200}, nil},
		{"short incomplete page", &listingadmin.PageResult[*listingadmin.StoreRespDTO]{List: []*listingadmin.StoreRespDTO{valid}, Total: 201, PageNo: 1, PageSize: 200}, nil},
		{"wrong page", &listingadmin.PageResult[*listingadmin.StoreRespDTO]{PageNo: 2, PageSize: 200}, nil},
		{"invalid page size", &listingadmin.PageResult[*listingadmin.StoreRespDTO]{PageNo: 1}, nil},
		{"negative total", &listingadmin.PageResult[*listingadmin.StoreRespDTO]{Total: -1, PageNo: 1, PageSize: 200}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// An invalid page must return an error rather than panic.
			require.NotPanics(t, func() {
				got, err := findDuplicateStore(context.Background(), stubStoreAPI{pageFunc: func(*listingadmin.StorePageReqDTO) (*listingadmin.PageResult[*listingadmin.StoreRespDTO], error) {
					return tc.page, tc.err
				}}, sheinlogin.Account{TenantID: 7, StoreID: 5}, "123")
				require.Error(t, err)
				require.Nil(t, got)
				if err != nil {
					for _, private := range []string{"foreign-private-name", "876543", "987654"} {
						require.NotContains(t, err.Error(), private)
					}
				}
			})
		})
	}
	got, err := findDuplicateStore(context.Background(), nil, sheinlogin.Account{TenantID: 7, StoreID: 5}, "123")
	require.Error(t, err, "missing API is incomplete lookup")
	require.Nil(t, got)
}

func TestFindDuplicateStoreGormKeepsSameExternalIDInOtherTenantIndependent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	// Only isolated fixture columns consumed by the real GORM read path.
	require.NoError(t, db.Exec(`CREATE TABLE listing_store (
		id INTEGER PRIMARY KEY, tenant_id INTEGER, store_id TEXT, platform TEXT,
		name TEXT, status INTEGER, deleted INTEGER DEFAULT 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO listing_store (id,tenant_id,store_id,platform,name,status) VALUES
		(5,7,'123','SHEIN','current',0), (99,8,'123','SHEIN','foreign-private-name',0),
		(6,7,'123','SHEIN','disabled',1)`).Error)
	lookup := NewDuplicateStoreLookupWithStoreAPI(listingadmin.NewGormStoreAPI(listingadmin.NewGormStoreRepository(db)))
	account := sheinlogin.Account{TenantID: 7, StoreID: 5}
	got, err := lookup(context.Background(), account, "123")
	require.NoError(t, err)
	require.Nil(t, got, "other tenant must not disable the current store")
	require.NoError(t, db.Exec(`INSERT INTO listing_store (id,tenant_id,store_id,platform,name,status) VALUES (7,7,'123','SHEIN','same tenant',0)`).Error)
	got, err = lookup(context.Background(), account, "123")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, int64(7), got.TenantID)
	require.Equal(t, int64(7), got.ID)
}
