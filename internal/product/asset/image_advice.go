package asset

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"task-processor/internal/product/collection"
)

const ImageAdviceSchema = "image-advice-v1"
const MaxImageAdviceBytes = 512 << 10
const MaxImageAdviceItems = 10000

var ErrImageAdvice = errors.New("bounded image advice does not match exact owner facts")

type ImageAdviceTarget struct {
	RecordID              string `json:"recordId" jsonschema:"minLength=1,maxLength=128"`
	Revision              string `json:"revision" jsonschema:"minLength=1,maxLength=19"`
	StoreID               string `json:"storeId" jsonschema:"minLength=1,maxLength=128"`
	Site                  string `json:"site" jsonschema:"minLength=1,maxLength=128"`
	CategoryID            int64  `json:"categoryId" jsonschema:"minimum=0"`
	RuleDigest            string `json:"ruleDigest" jsonschema:"minLength=1,maxLength=128"`
	SourceID              string `json:"sourceId" jsonschema:"minLength=1,maxLength=128"`
	OriginalPublicationID string `json:"originalPublicationId" jsonschema:"minLength=1,maxLength=128"`
	OriginalVersion       string `json:"originalVersion" jsonschema:"minLength=1,maxLength=19"`
	ApplyReceiptID        string `json:"applyReceiptId" jsonschema:"maxLength=128"`
}
type ImageAdviceBinding struct {
	ProductKey      string             `json:"productKey" jsonschema:"minLength=1,maxLength=128"`
	CatalogVersion  string             `json:"catalogVersion" jsonschema:"minLength=1,maxLength=19"`
	PublicationID   string             `json:"publicationId" jsonschema:"minLength=1,maxLength=128"`
	TargetPlatform  string             `json:"targetPlatform" jsonschema:"minLength=1,maxLength=128"`
	InventoryDigest string             `json:"inventoryDigest" jsonschema:"minLength=1,maxLength=128"`
	Target          *ImageAdviceTarget `json:"target,omitempty"`
}
type ImageAdviceEvidence struct {
	ToolID      string `json:"toolId" jsonschema:"minLength=1,maxLength=128"`
	ToolVersion string `json:"toolVersion" jsonschema:"minLength=1,maxLength=128"`
	CallID      string `json:"callId" jsonschema:"minLength=1,maxLength=128"`
	Digest      string `json:"digest" jsonschema:"minLength=1,maxLength=128"`
}
type ImageAdviceFact struct {
	AssetID          string                  `json:"assetId" jsonschema:"minLength=1,maxLength=128"`
	Role             Role                    `json:"role"`
	Approved         bool                    `json:"approved"`
	Width            int                     `json:"width" jsonschema:"minimum=0"`
	Height           int                     `json:"height" jsonschema:"minimum=0"`
	DimensionsStatus string                  `json:"dimensionsStatus" jsonschema:"enum=known,enum=unknown"`
	Placement        *ImageOfficialPlacement `json:"placement,omitempty"`
}
type ImageAdviceCheck struct {
	ID       string   `json:"id" jsonschema:"minLength=1,maxLength=128"`
	Code     string   `json:"code" jsonschema:"enum=dimensions_known,enum=dimensions_unknown,enum=dimensions_conflict,enum=placement_conflict,enum=required_image_missing,enum=approved_assets_unavailable,enum=rule_unavailable,enum=platform_not_evaluated,enum=owner_image_issue"`
	Status   string   `json:"status" jsonschema:"enum=known_satisfied,enum=known_conflict,enum=unknown,enum=not_applicable"`
	AssetIDs []string `json:"assetIds" jsonschema:"maxItems=10000"`
	RuleRef  string   `json:"ruleRef" jsonschema:"maxLength=128"`
	Reason   string   `json:"reason" jsonschema:"minLength=1,maxLength=1024"`
}
type ImageAdviceSuggestion struct {
	Kind     string   `json:"kind" jsonschema:"enum=verify_fact,enum=select_approved_asset,enum=check_placement,enum=open_existing_images"`
	CheckIDs []string `json:"checkIds" jsonschema:"minItems=1,maxItems=10000"`
	AssetIDs []string `json:"assetIds" jsonschema:"maxItems=10000"`
	Reason   string   `json:"reason" jsonschema:"minLength=1,maxLength=1024"`
}
type ImageAdvice struct {
	SchemaVersion string                  `json:"schemaVersion" jsonschema:"enum=image-advice-v1"`
	Binding       ImageAdviceBinding      `json:"binding"`
	Evidence      ImageAdviceEvidence     `json:"evidence"`
	Facts         []ImageAdviceFact       `json:"facts" jsonschema:"maxItems=10000"`
	Checks        []ImageAdviceCheck      `json:"checks" jsonschema:"maxItems=10000"`
	Suggestions   []ImageAdviceSuggestion `json:"suggestions" jsonschema:"maxItems=10000"`
	Unresolved    []string                `json:"unresolved" jsonschema:"maxItems=10000"`
}

// ImageDimensionFact is an already verified observation supplied by the
// authorized consumer. It carries no URL, provider or execution instruction.
type ImageDimensionFact struct {
	AssetID       string
	Width, Height int
}

// AnalyzeImageFacts consumes authorized, already read owner facts. It never
// reads bytes, prepares a plan, approves material or infers image content.
func AnalyzeImageFacts(binding ImageAdviceBinding, inventory ApprovedAssetInventory, ready bool, observations ...ImageDimensionFact) (ImageAdvice, error) {
	a := ImageAdvice{SchemaVersion: ImageAdviceSchema, Binding: binding, Facts: []ImageAdviceFact{}, Checks: []ImageAdviceCheck{}, Suggestions: []ImageAdviceSuggestion{}, Unresolved: []string{}}
	version, err := strconv.ParseUint(binding.CatalogVersion, 10, 64)
	if err != nil || version == 0 || strconv.FormatUint(version, 10) != binding.CatalogVersion || ValidateInventoryScope(inventory.Scope) != nil ||
		inventory.Scope.ProductKey != binding.ProductKey || inventory.Scope.SourceSnapshotVersion != version || inventory.Scope.TargetPlatform != binding.TargetPlatform ||
		binding.InventoryDigest != collection.Digest(inventory) || !validAdviceID(binding.PublicationID) || len(inventory.Assets) > MaxImageAdviceItems {
		return ImageAdvice{}, ErrImageAdvice
	}
	if len(observations) > MaxImageAdviceItems {
		return ImageAdvice{}, ErrImageAdvice
	}
	dimensions := map[string]ImageDimensionFact{}
	for _, observed := range observations {
		if !validAdviceID(observed.AssetID) || observed.Width <= 0 || observed.Height <= 0 {
			return ImageAdvice{}, ErrImageAdvice
		}
		if _, duplicate := dimensions[observed.AssetID]; duplicate {
			return ImageAdvice{}, ErrImageAdvice
		}
		dimensions[observed.AssetID] = observed
	}
	if ready && len(inventory.Assets) > 0 {
		if ValidateApprovalCommit(ApprovalCommit{TenantID: inventory.Scope.TenantID, ProductKey: binding.ProductKey, TargetPlatform: binding.TargetPlatform, SourceSnapshotVersion: version, ActionID: "image-advice-exact-read", Assets: inventory.Assets}) != nil {
			return ImageAdvice{}, ErrImageAdvice
		}
	}
	if !ready || len(inventory.Assets) == 0 {
		status, reason := "known_conflict", "当前作用版本没有已批准素材，请在原图片入口核对或补充。"
		if !ready {
			status, reason = "unknown", "批准素材尚未准备好，不能据此判断必需图片缺失。"
		}
		a.AddCheck("approved_assets_unavailable", status, nil, "", reason, "open_existing_images")
		if !ready {
			a.Unresolved = append(a.Unresolved, reason)
		}
	} else {
		for _, image := range inventory.Assets {
			fact := ImageAdviceFact{AssetID: image.ID, Role: image.Role, Approved: true, Width: image.Width, Height: image.Height, DimensionsStatus: "known", Placement: image.OfficialPlacement}
			if observed, ok := dimensions[image.ID]; ok {
				fact.Width, fact.Height = observed.Width, observed.Height
			}
			code, status, reason, kind := "dimensions_known", "known_satisfied", "仅确认已有真实尺寸记录；不代表图片内容或平台审核通过。", "select_approved_asset"
			if fact.Width <= 0 || fact.Height <= 0 {
				fact.DimensionsStatus = "unknown"
				code, status, reason, kind = "dimensions_unknown", "unknown", "真实尺寸缺少依据，请在原图片入口核对。", "verify_fact"
				a.Unresolved = append(a.Unresolved, reason)
			}
			a.Facts = append(a.Facts, fact)
			a.AddCheck(code, status, []string{image.ID}, "", reason, kind)
		}
	}
	if binding.Target == nil {
		a.AddCheck("platform_not_evaluated", "not_applicable", nil, "", "没有绑定原生Target和具体平台规则，本结果未评价官方发布条件。", "")
	}
	return a, nil
}

// AddCheck only builds a view result; kind selects explanatory guidance,
// never a task/action payload. Seal validates the final bounded result.
func (a *ImageAdvice) AddCheck(code, status string, ids []string, rule, reason, kind string) {
	if ids == nil {
		ids = []string{}
	}
	id := collection.Digest(struct {
		Binding                    ImageAdviceBinding
		Code, Status, Rule, Reason string
		Assets                     []string
	}{a.Binding, code, status, rule, reason, ids})
	a.Checks = append(a.Checks, ImageAdviceCheck{ID: id, Code: code, Status: status, AssetIDs: ids, RuleRef: rule, Reason: reason})
	if kind != "" {
		a.Suggestions = append(a.Suggestions, ImageAdviceSuggestion{Kind: kind, CheckIDs: []string{id}, AssetIDs: ids, Reason: reason})
	}
}

func (a *ImageAdvice) Seal(tool, version, call string) error {
	a.Evidence = ImageAdviceEvidence{ToolID: tool, ToolVersion: version, CallID: call}
	a.Evidence.Digest = a.semanticDigest()
	return a.Verify()
}
func (a ImageAdvice) semanticDigest() string {
	a.Evidence.Digest = ""
	a.Evidence.CallID = ""
	return collection.Digest(a)
}
func (a ImageAdvice) Verify() error {
	if a.SchemaVersion != ImageAdviceSchema || !validAdviceID(a.Binding.ProductKey) || !validAdviceID(a.Binding.PublicationID) ||
		!validAdviceID(a.Binding.TargetPlatform) || !validAdviceID(a.Binding.InventoryDigest) ||
		!validAdviceID(a.Evidence.ToolID) || !validAdviceID(a.Evidence.ToolVersion) || len(a.Evidence.CallID) > 128 ||
		a.Evidence.Digest != a.semanticDigest() || len(a.Facts) > MaxImageAdviceItems || len(a.Checks) > MaxImageAdviceItems || len(a.Suggestions) > MaxImageAdviceItems || len(a.Unresolved) > MaxImageAdviceItems ||
		a.Facts == nil || a.Checks == nil || a.Suggestions == nil || a.Unresolved == nil {
		return ErrImageAdvice
	}
	if v, e := strconv.ParseUint(a.Binding.CatalogVersion, 10, 64); e != nil || v == 0 || strconv.FormatUint(v, 10) != a.Binding.CatalogVersion {
		return ErrImageAdvice
	}
	if t := a.Binding.Target; t != nil {
		// An incomplete saved draft may have no category yet. Preserve that
		// exact zero fact; the existing rule owner reports what is unavailable.
		if !validAdviceID(t.RecordID) || !validAdviceID(t.StoreID) || !validAdviceID(t.Site) || !validAdviceID(t.RuleDigest) || !validAdviceID(t.SourceID) || !validAdviceID(t.OriginalPublicationID) || t.CategoryID < 0 || len(t.ApplyReceiptID) > 128 {
			return ErrImageAdvice
		}
		for _, s := range []string{t.Revision, t.OriginalVersion} {
			if v, e := strconv.ParseUint(s, 10, 64); e != nil || v == 0 || strconv.FormatUint(v, 10) != s {
				return ErrImageAdvice
			}
		}
	}
	facts := map[string]bool{}
	for _, f := range a.Facts {
		if !validAdviceID(f.AssetID) || facts[f.AssetID] || !f.Approved || f.Width < 0 || f.Height < 0 || !oneOf(f.DimensionsStatus, "known", "unknown") || f.DimensionsStatus == "known" && (f.Width == 0 || f.Height == 0) {
			return ErrImageAdvice
		}
		facts[f.AssetID] = true
	}
	checks := map[string]bool{}
	for _, c := range a.Checks {
		if !validAdviceID(c.ID) || checks[c.ID] || !oneOf(c.Code, "dimensions_known", "dimensions_unknown", "dimensions_conflict", "placement_conflict", "required_image_missing", "approved_assets_unavailable", "rule_unavailable", "platform_not_evaluated", "owner_image_issue") || !oneOf(c.Status, "known_satisfied", "known_conflict", "unknown", "not_applicable") || !adviceText(c.Reason) || len(c.RuleRef) > 128 || c.AssetIDs == nil || len(c.AssetIDs) > MaxImageAdviceItems {
			return ErrImageAdvice
		}
		for _, id := range c.AssetIDs {
			if !facts[id] {
				return ErrImageAdvice
			}
		}
		checks[c.ID] = true
	}
	for _, s := range a.Suggestions {
		if !oneOf(s.Kind, "verify_fact", "select_approved_asset", "check_placement", "open_existing_images") || !adviceText(s.Reason) || len(s.CheckIDs) == 0 || len(s.CheckIDs) > MaxImageAdviceItems || s.AssetIDs == nil || len(s.AssetIDs) > MaxImageAdviceItems {
			return ErrImageAdvice
		}
		for _, id := range s.CheckIDs {
			if !checks[id] {
				return ErrImageAdvice
			}
		}
		for _, id := range s.AssetIDs {
			if !facts[id] {
				return ErrImageAdvice
			}
		}
	}
	for _, s := range a.Unresolved {
		if !adviceText(s) {
			return ErrImageAdvice
		}
	}
	raw, e := json.Marshal(a)
	if e != nil || len(raw) > MaxImageAdviceBytes {
		return ErrImageAdvice
	}
	return nil
}
func DecodeImageAdvice(raw []byte) (ImageAdvice, error) {
	var a ImageAdvice
	if len(raw) == 0 || len(raw) > MaxImageAdviceBytes {
		return a, ErrImageAdvice
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&a) != nil {
		return ImageAdvice{}, ErrImageAdvice
	}
	var extra any
	if d.Decode(&extra) != io.EOF || a.Verify() != nil {
		return ImageAdvice{}, ErrImageAdvice
	}
	return a, nil
}
func validAdviceID(s string) bool {
	return s != "" && s == strings.TrimSpace(s) && utf8.ValidString(s) && utf8.RuneCountInString(s) <= 128
}
func adviceText(s string) bool { return strings.TrimSpace(s) != "" && len([]rune(s)) <= 1024 }
func oneOf(s string, allowed ...string) bool {
	for _, v := range allowed {
		if s == v {
			return true
		}
	}
	return false
}
