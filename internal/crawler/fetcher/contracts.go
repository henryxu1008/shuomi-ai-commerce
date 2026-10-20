package fetcher

import (
	"context"
	"fmt"
	"reflect"

	"task-processor/internal/app/ports"
	"task-processor/internal/core/config"
	coreLogger "task-processor/internal/core/logger"
	sourceamazon "task-processor/internal/integration/crawler/amazon"
	"task-processor/internal/marketplace/sourceproduct"
	"task-processor/internal/model"
	"task-processor/internal/platform/queue/rabbitmq"

	"github.com/sirupsen/logrus"
)

type FetcherType string

const (
	LocalFetcher     FetcherType = "local"
	RemoteAPIFetcher FetcherType = "remote-api"
)

type ProductReader interface {
	FetchProduct(ctx context.Context, req *sourceproduct.FetchRequest) (*model.Product, error)
	FetchVariants(ctx context.Context, req *sourceproduct.FetchRequest, variantASINs []string) ([]*model.Product, error)
}

type ProductCache interface {
	CacheProduct(req *sourceproduct.FetchRequest, product *model.Product) error
	CacheVariants(req *sourceproduct.FetchRequest, variants []*model.Product) error
}

type ProductFetcherStats interface {
	GetStats() map[string]any
}

type ProductFetcher interface {
	ProductReader
	ProductCache
	ProductFetcherStats
}

type amazonSourceFetcherAdapter struct {
	crawlSource ports.CrawlSource
	delegate    sourceamazon.AmazonSourceFetcher
}

func newAmazonSourceFetcher(crawlSource ports.CrawlSource, zipcodes map[string]string) sourceproduct.SourceFetcher {
	zipcodesSnapshot := make(map[string]string, len(zipcodes))
	for region, zipcode := range zipcodes {
		zipcodesSnapshot[region] = zipcode
	}
	return &amazonSourceFetcherAdapter{
		crawlSource: crawlSource,
		delegate: sourceamazon.AmazonSourceFetcher{
			Planner: sourceamazon.AmazonCrawlRequestPlanner{
				DomainResolver: sourceamazon.AmazonDefaultDomainResolver{},
				ZipcodePolicy:  sourceamazon.AmazonDefaultZipcodePolicy{},
				Zipcodes:       zipcodesSnapshot,
			},
			Source: crawlSource,
		},
	}
}

func (a *amazonSourceFetcherAdapter) Configured() bool {
	return a != nil && !isNilCrawlSource(a.crawlSource)
}

func (a *amazonSourceFetcherAdapter) Fetch(ctx context.Context, req sourceproduct.SourceFetchRequest) (*model.Product, error) {
	if !a.Configured() {
		return nil, fmt.Errorf("amazon crawler source is not configured")
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	product, err := a.delegate.Fetch(ctx, sourceamazon.AmazonCrawlRequestInput{
		Region:    req.Region,
		ProductID: req.ProductID,
		Zipcode:   req.Zipcode,
	})
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	return product, err
}

func isNilCrawlSource(source ports.CrawlSource) bool {
	if source == nil {
		return true
	}
	value := reflect.ValueOf(source)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

type FetcherFactory struct {
	logger *logrus.Entry
}

func NewFetcherFactory() *FetcherFactory {
	return &FetcherFactory{
		logger: coreLogger.GetGlobalLogger("FetcherFactory"),
	}
}

func (f *FetcherFactory) CreateFetcher(
	fetcherType FetcherType,
	rawJsonDataClient sourceproduct.RawJsonDataClient,
	amazonConfig *config.AmazonConfig,
	crawlSource ports.CrawlSource,
	rabbitmqClient *rabbitmq.Client,
) (ProductFetcher, error) {
	if f == nil || f.logger == nil {
		f = NewFetcherFactory()
	}
	switch fetcherType {
	case LocalFetcher:
		f.logger.Info("creating local product fetcher")
		options := sourceproduct.ProductFetcherOptions{}
		var zipcodes map[string]string
		if amazonConfig != nil {
			options.Enabled = amazonConfig.Enabled
			options.DataFreshnessDays = amazonConfig.DataFreshnessDays
			zipcodes = amazonConfig.Zipcodes
		}
		return sourceproduct.NewProductFetcherWithLogger(
			rawJsonDataClient,
			options,
			newAmazonSourceFetcher(crawlSource, zipcodes),
			f.logger,
		), nil
	case RemoteAPIFetcher:
		f.logger.Info("creating remote api product fetcher")
		return NewRemoteAPIProductFetcher(rawJsonDataClient, amazonConfig)
	default:
		return nil, fmt.Errorf("unsupported fetcher type: %s", fetcherType)
	}
}

func NewRemoteAPIProductFetcher(
	rawJsonDataClient sourceproduct.RawJsonDataClient,
	amazonConfig *config.AmazonConfig,
) (ProductFetcher, error) {
	return newRemoteAPIProductFetcher(rawJsonDataClient, amazonConfig)
}
