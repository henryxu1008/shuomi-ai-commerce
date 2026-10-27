// Package aicapability defines provider-neutral AI capability contracts.
package aicapability

type Capability string

const (
	CapabilityProductImageScene       Capability = "productimage.scene_generation"
	CapabilityProductEnrichText       Capability = "productenrich.text_understanding"
	CapabilityAIWorkbenchChatPlanning Capability = "aiworkbench.chat_planning"
)

type Operation string

const (
	OperationProductAgentDecision     Operation = "product_agent_decision"
	OperationProductContentDecision   Operation = "product.content.decision"
	OperationListingMaterialsDecision Operation = "listing.materials.decision"
	OperationProductSelectionDecision Operation = "product.selection.decision"
	OperationShopDiagnosisDecision    Operation = "shop.operations-diagnosis.decision"
	OperationAIWorkbenchChatPlan      Operation = "aiworkbench_chat_plan"
	OperationProductImageReview       Operation = "productimage_review"
)
