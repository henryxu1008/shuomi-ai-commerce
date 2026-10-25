package browser

import (
	"errors"
	"testing"

	"task-processor/internal/core/config"
)

type rotationMockInstanceManager struct {
	recreated []*BrowserInstance
}

func (m *rotationMockInstanceManager) CreateInstance(id int) (*BrowserInstance, error) {
	return &BrowserInstance{ID: id}, nil
}

func (m *rotationMockInstanceManager) RecreateInstanceSync(old *BrowserInstance) *BrowserInstance {
	return &BrowserInstance{ID: old.ID}
}

func (m *rotationMockInstanceManager) RecreateInstanceAsync(old *BrowserInstance) {
	m.recreated = append(m.recreated, old)
}

func TestReleaseRotatesInstanceAfterMaxUses(t *testing.T) {
	mockManager := &rotationMockInstanceManager{}
	bp := &BrowserPool{
		config:          &config.Config{},
		poolConfig:      &BrowserPoolConfig{MaxInstanceUses: 2},
		available:       make(chan *BrowserInstance, 1),
		instanceManager: mockManager,
	}

	instance := &BrowserInstance{ID: 1, UsageCount: 1, InUse: true}
	bp.Release(instance)

	if len(mockManager.recreated) != 1 {
		t.Fatalf("期望触发 1 次异步重建，实际: %d", len(mockManager.recreated))
	}
	if len(bp.available) != 0 {
		t.Fatalf("达到轮换阈值后实例不应归还池，实际可用: %d", len(bp.available))
	}
	if instance.UsageCount != 2 {
		t.Fatalf("期望 UsageCount 增加到 2，实际: %d", instance.UsageCount)
	}
}

func TestReleaseKeepsProxyUntilInstanceRebuildThreshold(t *testing.T) {
	mockManager := &rotationMockInstanceManager{}
	bp := &BrowserPool{
		config: &config.Config{Amazon: config.AmazonConfig{
			ProxyPool: config.AmazonProxyPoolConfig{Enabled: true},
		}},
		poolConfig:      &BrowserPoolConfig{MaxInstanceUses: 3},
		available:       make(chan *BrowserInstance, 1),
		instanceManager: mockManager,
	}

	instance := &BrowserInstance{ID: 1, UsageCount: 1, InUse: true, CurrentProxy: "http://127.0.0.1:31001"}
	bp.Release(instance)

	if len(mockManager.recreated) != 0 {
		t.Fatalf("expected no rebuild before max uses, got %d", len(mockManager.recreated))
	}
	if len(bp.available) != 1 {
		t.Fatalf("instance should return to pool before rebuild threshold, available=%d", len(bp.available))
	}
	if instance.CurrentProxy != "http://127.0.0.1:31001" {
		t.Fatalf("proxy changed before rebuild: %q", instance.CurrentProxy)
	}
}

func TestReleaseWithErrorRotatesInstanceAfterMaxUsesWhenErrorIsNotSerious(t *testing.T) {
	mockManager := &rotationMockInstanceManager{}
	bp := &BrowserPool{
		config:          &config.Config{},
		poolConfig:      &BrowserPoolConfig{MaxInstanceUses: 2},
		available:       make(chan *BrowserInstance, 1),
		errorDetector:   NewErrorDetector(),
		riskPolicy:      newRiskPolicy(&config.Config{}, NewErrorDetector()),
		instanceManager: mockManager,
	}

	instance := &BrowserInstance{ID: 2, UsageCount: 1, InUse: true}
	bp.ReleaseWithError(instance, errors.New("temporary validation error"))

	if len(mockManager.recreated) != 1 {
		t.Fatalf("期望错误释放后仍触发 1 次异步重建，实际: %d", len(mockManager.recreated))
	}
	if len(bp.available) != 0 {
		t.Fatalf("达到轮换阈值后实例不应归还池，实际可用: %d", len(bp.available))
	}
	if instance.UsageCount != 2 {
		t.Fatalf("期望 UsageCount 增加到 2，实际: %d", instance.UsageCount)
	}
}
