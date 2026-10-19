package bootstrap

import (
	"context"
	"net"
	"strconv"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"

	"task-processor/internal/core/config"
	"task-processor/internal/listingadmin"
)

type stubStoreRepository struct {
	items []listingadmin.Store
}

func (s *stubStoreRepository) ListStores(_ context.Context, query listingadmin.StoreQuery) (*listingadmin.StorePage, error) {
	items := make([]listingadmin.Store, 0, len(s.items))
	for _, item := range s.items {
		if query.TenantID > 0 && item.TenantID != query.TenantID {
			continue
		}
		if query.Platform != "" && item.Platform != query.Platform {
			continue
		}
		items = append(items, item)
	}
	return &listingadmin.StorePage{Items: items, Total: int64(len(items)), Page: 1, PageSize: len(items)}, nil
}

func (s *stubStoreRepository) GetStore(_ context.Context, tenantID, id int64) (*listingadmin.Store, error) {
	for _, item := range s.items {
		if item.TenantID == tenantID && item.ID == id {
			store := item
			return &store, nil
		}
	}
	return nil, listingadmin.ErrStoreNotFound
}

func (s *stubStoreRepository) UpdateStoreID(context.Context, int64, string) (*listingadmin.Store, error) {
	panic("unexpected UpdateStoreID")
}

func (s *stubStoreRepository) UpdateStoreStatus(context.Context, int64, int64, int16, string) (*listingadmin.Store, error) {
	panic("unexpected UpdateStoreStatus")
}

func TestBuildServiceReturnsNilWithoutLocalStoreRepository(t *testing.T) {
	t.Parallel()

	result, err := BuildService(BuildInput{
		Config: &config.Config{
			Platforms: config.PlatformsConfig{
				Shein: config.PlatformConfig{
					CookieRedis: config.RedisConfig{Host: "127.0.0.1"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildService() error = %v", err)
	}
	if result != nil {
		t.Fatalf("BuildService() result = %#v, want nil", result)
	}
}

func TestBuildServiceReturnsServiceAndClose(t *testing.T) {
	t.Parallel()

	redisServer := miniredis.RunT(t)
	host, portText, err := net.SplitHostPort(redisServer.Addr())
	if err != nil {
		t.Fatalf("SplitHostPort() error = %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("Atoi() error = %v", err)
	}

	result, err := BuildService(BuildInput{
		Config: &config.Config{
			Browser: config.BrowserConfig{ViewportWidth: 1280, ViewportHeight: 720},
			Platforms: config.PlatformsConfig{
				Shein: config.PlatformConfig{
					CookieRedis: config.RedisConfig{Host: host, Port: port},
				},
			},
		},
		AccountRepository: &stubStoreRepository{
			items: []listingadmin.Store{{
				ID:       12,
				TenantID: 7,
				Platform: "SHEIN",
				Username: "demo-user",
				Password: "secret",
				LoginURL: "sellerhub.shein.com",
			}},
		},
	})
	if err != nil {
		t.Fatalf("BuildService() error = %v", err)
	}
	if result == nil || result.Service == nil {
		t.Fatalf("BuildService() returned incomplete result: %#v", result)
	}
	if result.Close == nil {
		t.Fatal("BuildService() close func is nil")
	}
	if err := result.Close(); err != nil {
		t.Fatalf("result.Close() error = %v", err)
	}
}
