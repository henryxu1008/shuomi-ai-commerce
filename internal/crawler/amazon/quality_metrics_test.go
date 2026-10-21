package amazon

import "testing"

// Retained quality assertions extracted from the retired Service statistics test.
func TestQualityMetricsRetainsProcessorStatistics(t *testing.T) {
	metrics := newQualityMetrics()
	metrics.RecordValidationRetryAttempt()
	metrics.RecordValidationRetryRecovered()
	metrics.RecordTargetContextSkip("us")
	metrics.RecordTargetContextFallback("us")
	metrics.RecordTargetContextCheckError("us")
	stats := metrics.Snapshot()
	targetContextSkipByRegion := stats["target_context_skip_by_region"].(map[string]int64)
	targetContextFallbackByRegion := stats["target_context_fallback_by_region"].(map[string]int64)
	if stats["quality_validation_retry_attempt_total"].(int64) != 1 {
		t.Fatalf("expected quality_validation_retry_attempt_total=1, got %#v", stats["quality_validation_retry_attempt_total"])
	}
	if stats["quality_validation_retry_recovered_total"].(int64) != 1 {
		t.Fatalf("expected quality_validation_retry_recovered_total=1, got %#v", stats["quality_validation_retry_recovered_total"])
	}
	if stats["target_context_skip_total"].(int64) != 1 {
		t.Fatalf("expected target_context_skip_total=1, got %#v", stats["target_context_skip_total"])
	}
	if stats["target_context_fallback_total"].(int64) != 1 {
		t.Fatalf("expected target_context_fallback_total=1, got %#v", stats["target_context_fallback_total"])
	}
	if stats["target_context_check_error_total"].(int64) != 1 {
		t.Fatalf("expected target_context_check_error_total=1, got %#v", stats["target_context_check_error_total"])
	}
	if targetContextSkipByRegion["us"] != 1 {
		t.Fatalf("expected us target_context_skip count=1, got %#v", targetContextSkipByRegion)
	}
	if targetContextFallbackByRegion["us"] != 1 {
		t.Fatalf("expected us target_context_fallback count=1, got %#v", targetContextFallbackByRegion)
	}
	if stats["quality_validation_retry_recovery_rate"].(float64) != 1 {
		t.Fatalf("expected quality_validation_retry_recovery_rate=1, got %#v", stats["quality_validation_retry_recovery_rate"])
	}
}
