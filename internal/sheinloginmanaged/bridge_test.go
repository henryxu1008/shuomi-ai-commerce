package sheinloginmanaged

import (
	"context"
	"testing"

	"task-processor/internal/listingadmin"
	"task-processor/internal/sheinlogin"
)

func TestNewStoreSyncClientFactoryReturnsFactory(t *testing.T) {
	if NewStoreSyncClientFactoryWithStoreAPI(nil) == nil {
		t.Fatal("expected store sync factory")
	}
}

func TestFindDuplicateStoreIgnoresDisabledHistoricalStore(t *testing.T) {
	page := &listingadmin.PageResult[*listingadmin.StoreRespDTO]{
		List: []*listingadmin.StoreRespDTO{{
			ID:       560,
			TenantID: 246,
			Platform: "SHEIN",
			StoreID:  "9843414915",
			Status:   1,
		}},
		Total:    1,
		PageNo:   1,
		PageSize: 200,
	}

	duplicate, err := findDuplicateStore(
		context.Background(),
		stubStoreAPI{page: page},
		sheinlogin.Account{StoreID: 867, TenantID: 246},
		"9843414915",
	)
	if err != nil {
		t.Fatalf("find duplicate store: %v", err)
	}
	if duplicate != nil {
		t.Fatalf("duplicate = %+v, want disabled historical store to be ignored", duplicate)
	}
}

type stubStoreAPI struct {
	page     *listingadmin.PageResult[*listingadmin.StoreRespDTO]
	pageFunc func(*listingadmin.StorePageReqDTO) (*listingadmin.PageResult[*listingadmin.StoreRespDTO], error)
}

func (s stubStoreAPI) GetStore(int64) (*listingadmin.StoreRespDTO, error) { return nil, nil }

func (s stubStoreAPI) PageStores(req *listingadmin.StorePageReqDTO) (*listingadmin.PageResult[*listingadmin.StoreRespDTO], error) {
	if s.pageFunc != nil {
		return s.pageFunc(req)
	}
	if req == nil || req.Platform != "SHEIN" {
		return nil, nil
	}
	return s.page, nil
}

func (s stubStoreAPI) GetStoreCookie(int64) (string, error) { return "", nil }

func (s stubStoreAPI) UpdateStoreId(*listingadmin.StoreIdUpdateReqDTO) (bool, error) {
	return false, nil
}

func (s stubStoreAPI) UpdateStoreStatus(*listingadmin.StoreStatusUpdateReqDTO) (bool, error) {
	return false, nil
}

func (s stubStoreAPI) DeleteStoreCookie(int64) (bool, error) { return false, nil }

func (s stubStoreAPI) SetStorePauseStatus(int64, bool, string) (bool, error) {
	return false, nil
}

func (s stubStoreAPI) GetStorePauseStatus(int64) (bool, error) { return false, nil }

func (s stubStoreAPI) GetStorePauseStatusDetail(int64) (*listingadmin.StorePauseStatusRespDTO, error) {
	return nil, nil
}
