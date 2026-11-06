package materials

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	sigjson "sigs.k8s.io/json"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
)

const (
	MaterialsEditPurpose  = "materials-edit"
	CategoryRulesPurpose  = "category-rules"
	CandidateRulesSchema  = "shein-candidate-rules-v1"
	RuleExplanationSchema = "listing-rule-explanation-v1"
)

type RuleCoverage struct {
	Section string `json:"section" jsonschema:"enum=category,enum=fill,enum=attributes,enum=images,enum=conditional"`
	Status  string `json:"status" jsonschema:"enum=available,enum=unknown,enum=not_evaluated,enum=not_applicable"`
	Reason  string `json:"reason"`
}
type CategoryRulePath struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ProductTypeID string `json:"productTypeId"`
}
type RuleEvidence struct {
	Tool                  commercetool.ToolRef `json:"tool"`
	CallID                string               `json:"callId"`
	RuleDigest            string               `json:"ruleDigest"`
	ReadAt                time.Time            `json:"readAt"`
	AlgorithmVersion      string               `json:"algorithmVersion"`
	OfficialVersionStatus string               `json:"officialVersionStatus" jsonschema:"enum=unknown"`
}
type CandidateRulesDisclosure struct {
	OrganizationID        string                           `json:"-"`
	SchemaVersion         string                           `json:"schemaVersion" jsonschema:"enum=shein-candidate-rules-v1"`
	Binding               agent.Binding                    `json:"binding"`
	SourceID              string                           `json:"sourceId"`
	OriginalPublicationID string                           `json:"originalPublicationId"`
	OriginalVersion       string                           `json:"originalVersion"`
	ApplyReceiptID        string                           `json:"applyReceiptId"`
	EvidenceID            string                           `json:"evidenceId"`
	Evidence              RuleEvidence                     `json:"evidence"`
	CategoryPath          []CategoryRulePath               `json:"categoryPath"`
	Fill                  model.FillStandards              `json:"fill"`
	Attributes            model.AttributeTemplate          `json:"attributes"`
	Images                *goods.OfficialImageRequirements `json:"images,omitempty"`
	Coverage              []RuleCoverage                   `json:"coverage"`
	Unknowns              []string                         `json:"unknowns"`
}
type CandidateRulesReader interface {
	ReadCandidateRules(context.Context, agent.Binding) (CandidateRulesDisclosure, error)
}

type Reader interface {
	InspectionReader
	CandidateRulesReader
}

func RulesToolRef() commercetool.ToolRef { return agentconfig.MaterialsRulesToolRef() }

func CandidateRulesBinding(base record.TargetRecord, product catalog.PublishedSnapshot, category string) (agent.Binding, error) {
	b := agent.Binding{ContextKind: agent.CandidateRulesContextKind, ContextID: base.ID, ProductKey: product.Identity.ProductKey, CatalogVersion: strconv.FormatUint(product.Version, 10), PublicationID: product.PublicationID, TargetPlatform: "shein"}
	b.CandidateRules = agent.CandidateRulesRef{BaseRevision: strconv.FormatInt(base.Revision, 10), BaseDigest: collection.Digest(base), StoreID: base.Input.StoreID, Site: base.Merchant.Site, MerchantDigest: collection.Digest(base.Merchant), CandidateCategoryID: category}
	b.CandidateRules.SelectorDigest = collection.Digest(b)
	if !b.Valid() || !CandidateRulesMatchesBase(b, base) || product.Identity.TenantID != base.Merchant.OrganizationID || product.Version != base.EffectiveVersion || collection.Digest(product.Snapshot) != base.ProductHash || base.Source.Source.ProductKey != product.Identity.ProductKey {
		return agent.Binding{}, ErrInspectionChanged
	}
	return b, nil
}

func CandidateRulesMatchesBase(b agent.Binding, base record.TargetRecord) bool {
	if !b.Valid() || b.ContextKind != agent.CandidateRulesContextKind || b.ContextID != base.ID {
		return false
	}
	r := b.CandidateRules
	if r.BaseRevision != strconv.FormatInt(base.Revision, 10) || r.BaseDigest != collection.Digest(base) || r.StoreID != base.Input.StoreID || r.Site != "shein-us" || r.MerchantDigest != collection.Digest(base.Merchant) {
		return false
	}
	b.CandidateRules.SelectorDigest = ""
	return collection.Digest(b) == r.SelectorDigest
}

func CandidateBaseBinding(b agent.Binding) agent.Binding {
	b.ContextKind, b.CandidateRules = "listing-target-record", agent.CandidateRulesRef{}
	return b
}

func candidateCategoryPath(nodes []model.Category, id int64) []CategoryRulePath {
	for _, n := range nodes {
		item := CategoryRulePath{ID: strconv.FormatInt(n.ID, 10), Name: n.Name, ProductTypeID: strconv.FormatInt(n.ProductTypeID, 10)}
		if n.ID == id && n.Leaf != nil && *n.Leaf && len(n.Children) == 0 {
			return []CategoryRulePath{item}
		}
		if path := candidateCategoryPath(n.Children, id); len(path) > 0 {
			return append([]CategoryRulePath{item}, path...)
		}
	}
	return nil
}

func NewCandidateRulesDisclosure(base record.TargetRecord, product catalog.PublishedSnapshot, category string, rules goods.OfficialRuleSnapshot, readAt time.Time) (CandidateRulesDisclosure, error) {
	b, err := CandidateRulesBinding(base, product, category)
	if err != nil {
		return CandidateRulesDisclosure{}, err
	}
	id, _ := strconv.ParseInt(category, 10, 64)
	path := candidateCategoryPath(rules.Categories, id)
	if len(path) == 0 || len(rules.Linked) != 0 {
		return CandidateRulesDisclosure{}, ErrInspectionChanged
	}
	value := CandidateRulesDisclosure{SchemaVersion: CandidateRulesSchema, Binding: b, SourceID: base.Source.ID, OriginalPublicationID: base.Source.Source.PublicationID, OriginalVersion: strconv.FormatUint(base.Source.Source.Version, 10), ApplyReceiptID: base.ApplyReceiptID, CategoryPath: path, Fill: rules.Fill, Attributes: rules.Attributes, Unknowns: []string{}, Coverage: []RuleCoverage{}}
	value.Evidence = RuleEvidence{Tool: RulesToolRef(), ReadAt: readAt.UTC(), AlgorithmVersion: CandidateRulesSchema, OfficialVersionStatus: "unknown"}
	value.Coverage = append(value.Coverage, RuleCoverage{"category", "available", "实际店铺目录中的候选叶子类目。"}, RuleCoverage{"fill", "available", "官方实际返回的基础填写模板；未返回的标志保持未知。"}, RuleCoverage{"attributes", "available", "类目基础属性模板，不包含具体取值的关联分支。"})
	if rules.Fill.Fields == nil {
		value.Coverage[1].Status = "unknown"
	}
	for _, f := range rules.Fill.Fields {
		if f.Required == nil || f.Show == nil {
			value.Coverage[1].Status = "unknown"
		}
	}
	if rules.Attributes.MainAttributeStatus == nil {
		value.Coverage[2].Status = "unknown"
	}
	for _, a := range rules.Attributes.Attributes {
		if a.Show == nil || a.Mode == nil || a.Status == nil {
			value.Coverage[2].Status = "unknown"
		}
	}
	// The existing goods owner derives requirements from the frozen SKU shape.
	// No old attribute selection, image content or approval is evaluated here.
	images, e := goods.ResolveOfficialImageRequirements(base.Input.Draft.Product, rules.Fill, nil)
	if e == nil {
		value.Images = &images
		value.Coverage = append(value.Coverage, RuleCoverage{"images", "available", "按当前记录的 SKU 结构计算；未检查真实图片。"})
	} else {
		value.Coverage = append(value.Coverage, RuleCoverage{"images", "unknown", "图片规范或 SKU 结构证据不足，不能推断无需图片。"})
	}
	value.Coverage = append(value.Coverage, RuleCoverage{"conditional", "not_evaluated", "尚未提供该候选类目下的具体属性取值，关联条件未评价；不提供保存或发布许可。"})
	for _, c := range value.Coverage {
		if c.Status != "available" {
			value.Unknowns = append(value.Unknowns, c.Reason)
		}
	}
	value.Unknowns = append(value.Unknowns, "官方规则版本未提供证据；模型解释不能代替原规则或正式保存校验。")
	value.Evidence.RuleDigest = collection.Digest(struct {
		Path       []CategoryRulePath
		Fill       model.FillStandards
		Attributes model.AttributeTemplate
		Images     *goods.OfficialImageRequirements
		Coverage   []RuleCoverage
	}{value.CategoryPath, value.Fill, value.Attributes, value.Images, value.Coverage})
	value.EvidenceID = rulesDisclosureDigest(value)
	raw, e := json.Marshal(value)
	if e != nil || len(raw) > MaxInspectionBytes {
		return CandidateRulesDisclosure{}, ErrInspectionTooLarge
	}
	var copied CandidateRulesDisclosure
	if json.Unmarshal(raw, &copied) != nil {
		return CandidateRulesDisclosure{}, ErrInspectionChanged
	}
	copied.OrganizationID = product.Identity.TenantID
	return copied, nil
}

func rulesDisclosureDigest(v CandidateRulesDisclosure) string {
	v.EvidenceID = ""
	v.Evidence.CallID = ""
	v.Evidence.ReadAt = time.Time{}
	return collection.Digest(v)
}

func ParseCandidateRulesDisclosure(raw json.RawMessage) (CandidateRulesDisclosure, error) {
	var v CandidateRulesDisclosure
	if len(raw) == 0 || len(raw) > MaxInspectionBytes {
		return v, ErrInspectionTooLarge
	}
	strict, err := sigjson.UnmarshalStrict(raw, &v)
	if err != nil || len(strict) != 0 || v.SchemaVersion != CandidateRulesSchema || !v.Binding.Valid() || v.Binding.ContextKind != agent.CandidateRulesContextKind || v.Evidence.Tool != RulesToolRef() || v.Evidence.AlgorithmVersion != CandidateRulesSchema || v.Evidence.OfficialVersionStatus != "unknown" || v.Evidence.ReadAt.IsZero() || v.EvidenceID != rulesDisclosureDigest(v) || len(v.Coverage) != 5 {
		return CandidateRulesDisclosure{}, ErrInspectionChanged
	}
	return v, nil
}

type RuleExplanationRef struct {
	EvidenceID string `json:"evidenceId"`
	CallID     string `json:"callId"`
	RuleDigest string `json:"ruleDigest"`
}
type RuleExplanation struct {
	SchemaVersion string               `json:"schemaVersion"`
	Summary       string               `json:"summary"`
	References    []RuleExplanationRef `json:"references"`
	Unknowns      []string             `json:"unknowns"`
}
type DomainResult struct {
	Purpose     string           `json:"purpose"`
	Proposal    *Proposal        `json:"proposal,omitempty"`
	Explanation *RuleExplanation `json:"explanation,omitempty"`
}

func ParseDomainResult(raw json.RawMessage) (DomainResult, error) {
	var p DomainResult
	if len(raw) == 0 || len(raw) > agent.MaxModelOutputBytes {
		return p, ErrInvalidProposal
	}
	strict, err := sigjson.UnmarshalStrict(raw, &p)
	if err != nil || len(strict) != 0 {
		return DomainResult{}, ErrInvalidProposal
	}
	switch p.Purpose {
	case MaterialsEditPurpose:
		if p.Proposal == nil || p.Explanation != nil || validateProposal(*p.Proposal) != nil {
			return DomainResult{}, ErrInvalidProposal
		}
	case CategoryRulesPurpose:
		e := p.Explanation
		if p.Proposal != nil || e == nil || e.SchemaVersion != RuleExplanationSchema || strings.TrimSpace(e.Summary) == "" || len(e.Summary) > 8192 || len(e.References) == 0 || len(e.References) > 16 || len(e.Unknowns) > 64 {
			return DomainResult{}, ErrInvalidProposal
		}
		for _, r := range e.References {
			if !agent.ValidID(r.CallID) || len(r.EvidenceID) != 64 || len(r.RuleDigest) != 64 {
				return DomainResult{}, ErrInvalidProposal
			}
		}
		for _, s := range e.Unknowns {
			if strings.TrimSpace(s) == "" || len(s) > 1024 {
				return DomainResult{}, ErrInvalidProposal
			}
		}
	default:
		return DomainResult{}, ErrInvalidProposal
	}
	return p, nil
}

func ParseRequestProposal(request agent.Request, p agent.DomainProposal) (Proposal, error) {
	kind := agentconfig.MaterialsStoredProposalKind(request.PolicyVersion, request.PromptVersion)
	if kind == "" || p.Kind != kind || request.Binding.ContextKind != "listing-target-record" {
		return Proposal{}, ErrInvalidProposal
	}
	if kind == "listing-materials-v1" {
		return ParseProposal(p.Payload)
	}
	result, err := ParseDomainResult(p.Payload)
	if err != nil || result.Purpose != MaterialsEditPurpose {
		return Proposal{}, ErrInvalidProposal
	}
	return *result.Proposal, nil
}
