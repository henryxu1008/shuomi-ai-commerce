package amazon

import (
	"context"
	"errors"
	"testing"
	"time"

	"task-processor/internal/model"

	"github.com/mxschmitt/playwright-go"
)

func TestInstanceProcessorRetriesOnProductQualityFailure(t *testing.T) {
	ip := NewInstanceProcessor(NewURLHelper(), NewProductChecker(nil), nil, NewProductResultValidator())
	ip.SetQualityControlOptions(true, 2)

	attempts := 0
	reloads := 0
	ip.extractProduct = func(ctx context.Context, page playwright.Page, url, zipcode string) (*model.Product, error) {
		attempts++
		if attempts == 1 {
			return nil, &ProductQualityError{Reasons: []string{"title is empty"}}
		}
		return &model.Product{
			Asin:         "B001234567",
			Title:        "Demo Product",
			ImageURL:     "https://example.com/1.jpg",
			FinalPrice:   19.99,
			IsAvailable:  true,
			Availability: "In Stock",
		}, nil
	}
	ip.prepareRetry = func(ctx context.Context, page playwright.Page, waitTimeout time.Duration) error {
		reloads++
		return nil
	}

	product, err := ip.extractWithQualityRetry(context.Background(), nil, "https://www.amazon.com/dp/B001234567", "10001", 5*time.Second)
	if err != nil {
		t.Fatalf("expected retry to recover, got error: %v", err)
	}
	if product == nil || product.Asin != "B001234567" {
		t.Fatalf("expected valid product after retry, got %+v", product)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 extraction attempts, got %d", attempts)
	}
	if reloads != 1 {
		t.Fatalf("expected 1 page retry preparation, got %d", reloads)
	}
}

func TestInstanceProcessorStopsAfterQualityRetryLimit(t *testing.T) {
	ip := NewInstanceProcessor(NewURLHelper(), NewProductChecker(nil), nil, NewProductResultValidator())
	ip.SetQualityControlOptions(true, 2)

	attempts := 0
	ip.extractProduct = func(ctx context.Context, page playwright.Page, url, zipcode string) (*model.Product, error) {
		attempts++
		return nil, &ProductQualityError{Reasons: []string{"primary image is missing"}}
	}
	ip.prepareRetry = func(ctx context.Context, page playwright.Page, waitTimeout time.Duration) error {
		return nil
	}

	product, err := ip.extractWithQualityRetry(context.Background(), nil, "https://www.amazon.com/dp/B001234567", "10001", 5*time.Second)
	if err == nil {
		t.Fatal("expected quality retry to fail after max attempts")
	}
	if product != nil {
		t.Fatalf("expected nil product on failure, got %+v", product)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 extraction attempts, got %d", attempts)
	}
}

func TestInstanceProcessorResolveDefaultZipcodeForTargetContext(t *testing.T) {
	tests := []struct {
		name               string
		targetURL          string
		matchTargetContext func(page playwright.Page, targetURL string) (bool, error)
		want               string
	}{
		{
			name:      "us-matched-context-skips-fallback",
			targetURL: "https://www.amazon.com/dp/B001234567",
			matchTargetContext: func(page playwright.Page, targetURL string) (bool, error) {
				return true, nil
			},
			want: "",
		},
		{
			name:      "us-mismatched-context-uses-default-zipcode",
			targetURL: "https://www.amazon.com/dp/B001234567",
			matchTargetContext: func(page playwright.Page, targetURL string) (bool, error) {
				return false, nil
			},
			want: "94107",
		},
		{
			name:      "us-context-check-error-uses-default-zipcode",
			targetURL: "https://www.amazon.com/dp/B001234567",
			matchTargetContext: func(page playwright.Page, targetURL string) (bool, error) {
				return false, errors.New("page read failed")
			},
			want: "94107",
		},
		{
			name:      "jp-target-does-not-fallback",
			targetURL: "https://www.amazon.co.jp/dp/B0FLFBXSD2",
			matchTargetContext: func(page playwright.Page, targetURL string) (bool, error) {
				t.Fatalf("matchTargetContext should not be called for non-US targets")
				return false, nil
			},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ip := NewInstanceProcessor(NewURLHelper(), NewProductChecker(nil), nil, NewProductResultValidator())
			ip.matchTargetContext = tc.matchTargetContext

			got := ip.resolveDefaultZipcodeForTargetContext(nil, tc.targetURL)
			if got != tc.want {
				t.Fatalf("resolveDefaultZipcodeForTargetContext(%q)=%q want %q", tc.targetURL, got, tc.want)
			}
		})
	}
}
