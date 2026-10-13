package materials

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/collection"
)

type MaterialPreview struct {
	Input         record.TargetInput  `json:"input"`
	Result        goods.OfficialDraft `json:"result"`
	RequiresHuman bool                `json:"requiresHuman"`
}

// Preview uses the existing official builder. Rule membership is necessary
// but is not evidence that a model-selected value describes the real product.
func Preview(i Inspection, p Proposal) (MaterialPreview, error) {
	if err := i.verify(); err != nil {
		return MaterialPreview{}, err
	}
	for _, id := range p.EvidenceIDs {
		if id != i.EvidenceID {
			return MaterialPreview{}, ErrInvalidProposal
		}
	}
	input, err := ApplyProposal(i.Base, p)
	if err != nil || !referencesDisclosed(p, i) {
		return MaterialPreview{}, ErrInvalidProposal
	}
	changedCategory := input.Draft.Product.CategoryID != i.Base.Input.Draft.Product.CategoryID
	result := goods.BuildOfficial(input.Draft, i.Rules, i.Inventory, i.Base.ImageObservations, i.Base.ProofObservations)
	if changedCategory {
		// These attribute/fill rules belong to the original category. The
		// actual Target owner will load the selected category on local save.
		result.ReadyForUpload = false
		result.SubmissionPayload = nil
		result.Issues = append(result.Issues, goods.OfficialIssue{Field: "product.category_id", Code: "rule_unavailable", Message: "新类目须人工核对；保存后使用该类目的实际规则完善资料"})
	}
	uncertain := changedCategory || p.Attributes != nil || len(p.SKCs) != 0
	if p.BrandCode != nil {
		for _, brand := range i.Rules.Brands {
			if brand.Code == *p.BrandCode {
				uncertain = uncertain || strings.TrimSpace(i.Product.Snapshot.Brand) == "" || !strings.EqualFold(strings.TrimSpace(i.Product.Snapshot.Brand), strings.TrimSpace(brand.Name))
			}
		}
	}
	return MaterialPreview{Input: input, Result: result, RequiresHuman: uncertain || len(result.Issues) != 0 || !result.ReadyForUpload}, nil
}

func referencesDisclosed(p Proposal, i Inspection) bool {
	if p.CategoryID != nil && !disclosedLeaf(i.Rules.Categories, *p.CategoryID) {
		return false
	}
	if p.BrandCode != nil {
		found := false
		for _, b := range i.Rules.Brands {
			found = found || b.Code == *p.BrandCode
		}
		if !found {
			return false
		}
	}
	attributes := func(values []model.AttributeValue) bool {
		for _, v := range values {
			if v.CustomValue != "" || v.Language != "" {
				return false
			}
			found := false
			for _, a := range i.Rules.Attributes.Attributes {
				if a.ID != v.AttributeID || a.Show == nil || *a.Show != 1 {
					continue
				}
				if v.AttributeValueID == nil {
					found = a.Mode != nil && *a.Mode == 0 && v.ExtraValue != ""
				} else {
					for _, option := range a.Options {
						found = found || option.ID == *v.AttributeValueID && option.Show != nil && *option.Show == 1
					}
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	if p.Attributes != nil && !attributes(*p.Attributes) {
		return false
	}
	for _, skc := range p.SKCs {
		if skc.SaleAttribute != nil && !attributes([]model.AttributeValue{*skc.SaleAttribute}) {
			return false
		}
		for _, sku := range skc.SKUs {
			if sku.Attributes != nil && !attributes(*sku.Attributes) || sku.SaleAttributes != nil && !attributes(*sku.SaleAttributes) {
				return false
			}
		}
	}
	if p.Images != nil {
		for _, slot := range *p.Images {
			found := false
			for _, a := range i.Inventory.Assets {
				found = found || a.ID == slot.AssetID
			}
			if !found {
				return false
			}
		}
	}
	return true
}

func disclosedLeaf(nodes []model.Category, id int64) bool {
	for _, n := range nodes {
		if n.ID == id {
			return n.Leaf != nil && *n.Leaf && len(n.Children) == 0
		}
		if disclosedLeaf(n.Children, id) {
			return true
		}
	}
	return false
}

type Validator struct{ Reader InspectionReader }

func (v Validator) ValidateDomain(ctx context.Context, request agent.Request, candidate agent.DomainProposal, history []agent.Observation) (agent.Validation, error) {
	invalid := agent.Validation{PolicyVersion: PolicyVersion, Unresolved: []string{"PLATFORM_MATERIAL_EVIDENCE_INVALID"}}
	if ctx == nil || v.Reader == nil || request.PolicyVersion != PolicyVersion || request.PromptVersion != PromptVersion || candidate.Kind != ProposalKind {
		return invalid, nil
	}
	p, err := ParseProposal(candidate.Payload)
	if err != nil {
		return invalid, nil
	}
	current, err := v.Reader.ReadMaterialsInspection(ctx, request.Binding)
	if err != nil {
		return agent.Validation{}, err
	}
	expected, err := current.Disclose()
	if err != nil || current.Binding != request.Binding {
		return agent.Validation{}, ErrInspectionChanged
	}
	disclosed := false
	for _, observation := range history {
		if observation.Tool != InspectToolRef() || observation.AuditStatus != commercetool.AuditStatusRecorded || !agent.ValidID(observation.CallID) || observation.InvocationID != "" || len(observation.Output) > MaxInspectionBytes {
			continue
		}
		var i Disclosure
		if json.Unmarshal(observation.Output, &i) == nil && i.Binding == request.Binding {
			for _, id := range p.EvidenceIDs {
				if id == i.EvidenceID {
					if collection.Digest(i) != collection.Digest(expected) {
						return agent.Validation{}, ErrInspectionChanged
					}
					disclosed = true
				}
			}
		}
	}
	if !disclosed {
		return invalid, nil
	}
	preview, err := Preview(current, p)
	if err != nil {
		return invalid, nil
	}
	result := agent.Validation{Valid: true, PolicyVersion: PolicyVersion}
	for _, issue := range preview.Result.Issues {
		result.Unresolved = append(result.Unresolved, fmt.Sprintf("%s:%s:%s", issue.Field, issue.Code, issue.Message))
	}
	if preview.RequiresHuman {
		result.Unresolved = append(result.Unresolved, "PLATFORM_MATERIAL_CONFIRMATION_REQUIRED")
	}
	return result, nil
}

var _ agent.DomainValidator = Validator{}
