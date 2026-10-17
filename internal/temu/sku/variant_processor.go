// Package sku 提供TEMU平台的SKU变体处理功能
package sku

import (
	"fmt"
	"task-processor/internal/integration/openai"
	"task-processor/internal/model"
	temucontext "task-processor/internal/temu/context"
	"task-processor/internal/temu/spec"

	"github.com/sirupsen/logrus"
)

// SkuVariantProcessor SKU变体处理器
type SkuVariantProcessor struct {
	logger           *logrus.Entry
	aiClient         openai.ChatCompleter
	specHandler      *SkuSpecHandler
	mappingProcessor *SkuMappingProcessor
	specResolver     *spec.SpecResolverService
}

// NewSkuVariantProcessor 创建新的SKU变体处理器
func NewSkuVariantProcessor(logger *logrus.Entry, aiClient openai.ChatCompleter, specHandler *SkuSpecHandler, specResolver *spec.SpecResolverService) *SkuVariantProcessor {
	return &SkuVariantProcessor{
		logger:           logger,
		aiClient:         aiClient,
		specHandler:      specHandler,
		mappingProcessor: NewSkuMappingProcessor(logger, specHandler),
		specResolver:     specResolver,
	}
}

// generateAISkuMapping 生成AI SKU映射
func (vp *SkuVariantProcessor) GenerateAISkuMapping(temuCtx *temucontext.TemuTaskContext, variants []*model.Product) (*temucontext.AISkuMappingResponse, error) {
	if vp.aiClient == nil {
		return nil, fmt.Errorf("AI客户端未初始化")
	}

	if err := vp.validateAIMappingVariantCount(variants); err != nil {
		return nil, err
	}

	// 根据token限制决定是否需要分批处理
	// Gemini 2.0 Flash输出限制约8000 tokens，每个SKU约400 tokens
	// 安全起见，每批最多处理20个变体
	const maxVariantsPerBatch = 20

	return vp.generateAIMappingByVariantCount(temuCtx, variants, maxVariantsPerBatch)
}

func (vp *SkuVariantProcessor) validateAIMappingVariantCount(variants []*model.Product) error {
	if len(variants) <= 100 {
		return nil
	}

	vp.logger.Errorf("❌ 变体数量超过限制: %d > 100，系统无法处理如此多的变体", len(variants))
	vp.logger.Error("❌ 此错误不应重试，请检查产品数据或联系技术支持")
	return fmt.Errorf("变体数量超过限制: %d > 100，系统无法处理", len(variants))
}

func (vp *SkuVariantProcessor) generateAIMappingByVariantCount(
	temuCtx *temucontext.TemuTaskContext,
	variants []*model.Product,
	maxVariantsPerBatch int,
) (*temucontext.AISkuMappingResponse, error) {
	if len(variants) > maxVariantsPerBatch {
		vp.logger.Infof("🔧 变体数量(%d)超过单批限制(%d)，将分批处理", len(variants), maxVariantsPerBatch)
		return vp.generateAISkuMappingInBatches(temuCtx, variants, maxVariantsPerBatch)
	}

	return vp.generateAISkuMappingSingle(temuCtx, variants)
}

func (vp *SkuVariantProcessor) validateAIMappingSpecs(aiMapping *temucontext.AISkuMappingResponse) error {
	var validationErr error
	aiMapping.ForEachSKUIndexed(func(i int, aiSku *temucontext.AIGeneratedSku) {
		if validationErr != nil {
			return
		}
		if len(aiSku.Spec) > 2 {
			vp.logger.Errorf("AI mapping[%d] has too many specs: %d", i, len(aiSku.Spec))
			vp.logger.Errorf("Specs: %+v", aiSku.Spec)
			validationErr = fmt.Errorf("AI mapping[%d] has too many specs: %d", i, len(aiSku.Spec))
			return
		}

		if err := vp.specHandler.ValidateSpecs(convertSpecInfos(aiSku.Spec)); err != nil {
			vp.logger.Errorf("AI mapping[%d] spec validation failed: %v", i, err)
			vp.logger.Error("AI generated specs do not match TEMU spec constraints")
			validationErr = fmt.Errorf("AI mapping[%d] spec validation failed: %w", i, err)
			return
		}

		vp.logger.Infof("SKU[%d] AI dimensions: weight=%s, length=%s, width=%s, height=%s",
			i, aiSku.Weight, aiSku.Length, aiSku.Width, aiSku.Height)
	})

	return validationErr
}

func (vp *SkuVariantProcessor) resolveAIMappingSpecIDs(temuCtx *temucontext.TemuTaskContext, aiMapping *temucontext.AISkuMappingResponse) error {
	resolveRuntime, err := spec.BuildResolveSpecRuntimeInput(temuCtx)
	if err != nil {
		return fmt.Errorf("build spec resolve runtime: %w", err)
	}
	if err := vp.specResolver.ResolveTemporarySpecIDs(resolveRuntime, aiMapping); err != nil {
		vp.logger.Errorf("failed to resolve temporary spec IDs: %v", err)
		return fmt.Errorf("resolve temporary spec IDs: %w", err)
	}

	vp.logger.Info("Resolved all temporary spec IDs")
	return nil
}

func (vp *SkuVariantProcessor) prepareAIMappingForBuild(temuCtx *temucontext.TemuTaskContext, variants []*model.Product, aiMapping *temucontext.AISkuMappingResponse) error {
	if err := vp.ensureAIMappingMatchesVariants(aiMapping, variants); err != nil {
		return err
	}

	return vp.normalizeAIMappingForBuild(temuCtx, aiMapping)
}

func (vp *SkuVariantProcessor) ensureAIMappingMatchesVariants(aiMapping *temucontext.AISkuMappingResponse, variants []*model.Product) error {
	if aiMapping.SkuCount() == len(variants) {
		return nil
	}

	vp.logger.Warnf("⚠️ AI映射数量(%d)与变体数量(%d)不匹配", aiMapping.SkuCount(), len(variants))
	if err := vp.mappingProcessor.FixMappingCountMismatch(aiMapping, variants); err != nil {
		return fmt.Errorf("修复映射数量不匹配失败: %w", err)
	}

	return nil
}

func (vp *SkuVariantProcessor) normalizeAIMappingForBuild(temuCtx *temucontext.TemuTaskContext, aiMapping *temucontext.AISkuMappingResponse) error {
	if err := vp.validateAIMappingSpecs(aiMapping); err != nil {
		return err
	}
	if err := vp.resolveAIMappingSpecIDs(temuCtx, aiMapping); err != nil {
		return err
	}

	return nil
}

func (vp *SkuVariantProcessor) normalizeGeneratedAIMapping(aiMapping *temucontext.AISkuMappingResponse) error {
	if err := vp.unifyAIMappingSpecDimensions(aiMapping); err != nil {
		vp.logger.Errorf("single-batch spec dimension unify failed: %v", err)
		return fmt.Errorf("single-batch spec dimension unify: %w", err)
	}

	return nil
}

func (vp *SkuVariantProcessor) finalizeGeneratedAIMapping(aiMapping *temucontext.AISkuMappingResponse) (*temucontext.AISkuMappingResponse, error) {
	if err := vp.normalizeGeneratedAIMapping(aiMapping); err != nil {
		return nil, err
	}

	return aiMapping, nil
}

func (vp *SkuVariantProcessor) generateAISkuMappingSingle(temuCtx *temucontext.TemuTaskContext, variants []*model.Product) (*temucontext.AISkuMappingResponse, error) {
	response, err := vp.GenerateAISkuMappingSingleBatch(temuCtx, variants)
	if err != nil {
		return nil, err
	}

	return vp.finalizeGeneratedAIMapping(response)
}

func (vp *SkuVariantProcessor) unifyAIMappingSpecDimensions(aiMapping *temucontext.AISkuMappingResponse) error {
	unifier := spec.NewSpecDimensionUnifier()
	return unifier.UnifySpecDimensions(aiMapping)
}
