package prompt

// Prompt key 常量，与 prompts/ 目录下 YAML 文件的展平路径对应。
const (

	// ── productenrich/llm_scorer.yaml ──────────────────────────────────────
	KProductEnrichLlmScorerTextScoring  = "productenrich.llm_scorer.text_scoring"
	KProductEnrichLlmScorerImageScoring = "productenrich.llm_scorer.image_scoring"

	// ── productenrich/understanding.yaml ───────────────────────────────────
	KProductEnrichUnderstandingAnalyzeImage   = "productenrich.understanding.analyze_image"
	KProductEnrichUnderstandingExtractText    = "productenrich.understanding.extract_text"
	KProductEnrichUnderstandingFuseMultimodal = "productenrich.understanding.fuse_multimodal"

	// ── productenrich/generation.yaml ──────────────────────────────────────
	KProductEnrichGenerationProductJSON       = "productenrich.generation.product_json"
	KProductEnrichGenerationSpecs             = "productenrich.generation.specs"
	KProductEnrichGenerationVariants          = "productenrich.generation.variants"
	KProductEnrichGenerationExtractDimensions = "productenrich.generation.extract_dimensions"
	KProductEnrichGenerationExtractWeight     = "productenrich.generation.extract_weight"

	// ── productimage/generation.yaml ───────────────────────────────────────
	KProductImageSubjectExtract                        = "productimage.subject.extract"
	KProductImageWhiteBackgroundDefault                = "productimage.white_background.default"
	KProductImageSceneDefault                          = "productimage.scene.default"
	KProductImageSceneShoes                            = "productimage.scene.shoes"
	KProductImageSceneJewelry                          = "productimage.scene.jewelry"
	KProductImageSceneBags                             = "productimage.scene.bags"
	KProductImageReviewDefault                         = "productimage.review.default"
	KProductImageStudioGenerationPodDesign             = "productimage.studio_generation.pod_design"
	KProductImageStudioGenerationPodDesignHotReference = "productimage.studio_generation.pod_design_hot_reference"
	KProductImageStudioGenerationAmazonProductImage    = "productimage.studio_generation.amazon_product_image"
)
