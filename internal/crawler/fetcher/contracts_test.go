package fetcher

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"task-processor/internal/core/config"
	"task-processor/internal/marketplace/sourceproduct"
	"task-processor/internal/platform/queue/rabbitmq"
)

func TestContractsOwnFetcherTypeAndProductFetcherInterface(t *testing.T) {
	source, err := os.ReadFile("contracts.go")
	if err != nil {
		t.Fatalf("ReadFile(contracts.go) error = %v", err)
	}
	content := string(source)
	if !strings.Contains(content, "type FetcherType string") {
		t.Fatal("internal/crawler/fetcher should own FetcherType instead of aliasing app/crawler/fetcher")
	}
	if !strings.Contains(content, "type ProductFetcher interface") {
		t.Fatal("internal/crawler/fetcher should own the ProductFetcher contract interface")
	}
	if strings.Contains(content, "type FetcherType = appfetcher.FetcherType") {
		t.Fatal("FetcherType should not be an alias to internal/app/crawler/fetcher")
	}
	if strings.Contains(content, "type ProductFetcher = appfetcher.ProductFetcher") {
		t.Fatal("ProductFetcher should not be an alias to internal/app/crawler/fetcher")
	}
	if strings.Contains(content, "type RemoteAPIProductFetcher = appfetcher.RemoteAPIProductFetcher") {
		t.Fatal("RemoteAPIProductFetcher concrete app implementation should not be re-exported from the neutral fetcher package")
	}
	if strings.Contains(content, "type DistributedProductFetcher = appfetcher.DistributedProductFetcher") {
		t.Fatal("DistributedProductFetcher concrete app implementation should not be re-exported from the neutral fetcher package")
	}
	if strings.Contains(content, "appFactory") {
		t.Fatal("neutral fetcher factory should not store an app/crawler/fetcher factory")
	}
	if strings.Contains(content, ".appFactory.CreateFetcher") {
		t.Fatal("neutral fetcher factory should own selection logic instead of delegating to app factory CreateFetcher")
	}
}

func TestProductFetcherExposesComposableCapabilities(t *testing.T) {
	var _ ProductReader = (ProductFetcher)(nil)
	var _ ProductCache = (ProductFetcher)(nil)
	var _ ProductFetcherStats = (ProductFetcher)(nil)
}

func TestFetcherFactoryRejectsRetiredDistributedMode(t *testing.T) {
	for _, broker := range []*rabbitmq.Client{nil, {}} {
		productFetcher, err := NewFetcherFactory().CreateFetcher(
			FetcherType("distributed"),
			nil,
			&config.AmazonConfig{Enabled: true},
			nil,
			broker,
		)
		if err == nil {
			t.Fatal("CreateFetcher() error = nil, want unsupported retired mode")
		}
		if !strings.Contains(err.Error(), "unsupported fetcher type: distributed") {
			t.Fatalf("CreateFetcher() error = %v, want unsupported retired mode", err)
		}
		if productFetcher != nil {
			t.Fatal("retired mode must not construct a fetcher")
		}
	}
}

func TestFetcherFactoryPassesLoggerToLocalFetcher(t *testing.T) {
	var output bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&output)
	factory := &FetcherFactory{logger: logrus.NewEntry(logger)}

	fetcher, err := factory.CreateFetcher(LocalFetcher, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateFetcher() error = %v", err)
	}
	if err := fetcher.CacheProduct(&sourceproduct.FetchRequest{}, nil); err != nil {
		t.Fatalf("CacheProduct() error = %v", err)
	}
	if !strings.Contains(output.String(), "product is nil, skipping cache") {
		t.Fatalf("local fetcher log output = %q, want product warning", output.String())
	}
}

func TestFetcherFactoryBuildsRemoteFetcherWithBoundDependencies(t *testing.T) {
	productFetcher, err := NewFetcherFactory().CreateFetcher(RemoteAPIFetcher, nil, &config.AmazonConfig{
		RemoteAPI: config.RemoteAPIConfig{Enabled: true, BaseURL: "http://amazon-crawler-api:8080", Timeout: 30},
	}, nil, nil)
	if err != nil {
		t.Fatalf("CreateFetcher() error = %v", err)
	}
	if got := productFetcher.GetStats()["type"]; got != "remote-api" {
		t.Fatalf("CreateFetcher() fetcher type = %v, want remote-api", got)
	}
}
