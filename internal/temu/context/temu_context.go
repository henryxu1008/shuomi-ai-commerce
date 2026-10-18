package context

import (
	"context"

	"task-processor/internal/app/ports"
	"task-processor/internal/listingadmin"
	"task-processor/internal/model"
	"task-processor/internal/pipeline"
	"task-processor/internal/state"
	"task-processor/internal/temu/api"
	temutemplate "task-processor/internal/temu/api/template"
)

type RuntimeState struct {
	MemoryManager *state.MemoryManager
	CrawlSource   ports.CrawlSource
	APIClient     api.APIClientInterface
	QueryAPI      api.QueryAPIInterface
}

type ProductState struct {
	AmazonProduct *model.Product
	Variants      []*model.Product
	TemuProduct   *api.Product
	StoreInfo     *listingadmin.StoreRespDTO
}

type TemplateState struct {
	TemplateInfo            *temutemplate.TemplateInfo
	UserInputParentSpecList []temutemplate.UserInputParentSpec
	InputMaxSpecNum         int
	SingleSpecValueNum      int
}

type PublishState struct {
	PriceQueryResponse *api.PriceQueryResponse
}

type AssetState struct {
	PaddedImages     map[string][]byte
	PaddedImageSizes map[string][2]int
}

type VariantState struct {
	AsinSkuMap         map[string]string
	VariantAsins       []string
	CleanedTitle       string
	ProductDescription string
}

type RuleState struct {
	ProfitRule *listingadmin.ProfitRuleRespDTO
	FilterRule *listingadmin.FilterRuleRespDTO
}

type TemuTaskContext struct {
	*pipeline.DefaultTaskContext
	RuntimeState
	ProductState
	TemplateState
	PublishState
	AssetState
	VariantState
	RuleState
	AISkuMapping *AISkuMappingResponse
}

func NewTemuTaskContext(ctx context.Context, task *model.Task) *TemuTaskContext {
	return &TemuTaskContext{
		DefaultTaskContext: pipeline.NewTaskContext(ctx, task),
		AssetState: AssetState{
			PaddedImages:     make(map[string][]byte),
			PaddedImageSizes: make(map[string][2]int),
		},
		VariantState: VariantState{
			AsinSkuMap: make(map[string]string),
		},
	}
}

func (tc *TemuTaskContext) AttachRuntime(memoryManager *state.MemoryManager, crawlSource ports.CrawlSource) {
	tc.MemoryManager = memoryManager
	tc.CrawlSource = crawlSource
}

func (tc *TemuTaskContext) SetAPIClients(apiClient api.APIClientInterface, queryAPI api.QueryAPIInterface) {
	tc.APIClient = apiClient
	tc.QueryAPI = queryAPI
}

func (tc *TemuTaskContext) GetAmazonProduct() *model.Product {
	return tc.AmazonProduct
}

func (tc *TemuTaskContext) SetAmazonProduct(product *model.Product) {
	tc.AmazonProduct = product
}

func (tc *TemuTaskContext) GetVariants() []*model.Product {
	return tc.Variants
}

func (tc *TemuTaskContext) SetVariants(variants []*model.Product) {
	tc.Variants = variants
}

func (tc *TemuTaskContext) AddVariant(variant *model.Product) {
	tc.Variants = append(tc.Variants, variant)
}

func (tc *TemuTaskContext) SetAsinSkuMap(mapping map[string]string) {
	tc.AsinSkuMap = mapping
}

func (tc *TemuTaskContext) SetAISkuMapping(mapping *AISkuMappingResponse) {
	tc.AISkuMapping = mapping
}

func (tc *TemuTaskContext) SetTemuProduct(product *api.Product) {
	tc.TemuProduct = product
}

var _ pipeline.AmazonContext = (*TemuTaskContext)(nil)
