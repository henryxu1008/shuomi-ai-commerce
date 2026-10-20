package shared

import "testing"

func TestCrawlerTaskRetainsTrustedTenantIDInWorkerPayload(t *testing.T) {
	task := NewCrawlerTask("https://detail.1688.com/offer/3001.html")
	task.TenantID = 101

	if task.TenantID != 101 {
		t.Fatalf("TenantID = %d, want 101", task.TenantID)
	}
}
