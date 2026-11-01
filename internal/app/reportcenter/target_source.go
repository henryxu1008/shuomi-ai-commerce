package reportcenterapp

import (
	"context"
	"encoding/hex"
	"strconv"
	"time"

	"task-processor/internal/authidentity"
	target "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
	rc "task-processor/internal/reportcenter"
	"task-processor/internal/storecenter"
)

func (s *Sources) readTarget(ctx context.Context, scope rc.Scope, id string) (rc.Snapshot, error) {
	if s.TargetRecords == nil {
		return rc.Snapshot{}, rc.ErrUnavailable
	}
	r, err := s.TargetRecords.ReadRecord(ctx, id)
	if err != nil {
		return rc.Snapshot{}, sourceError(err)
	}
	identity, _ := authidentity.AuthenticatedIdentityFromContext(ctx)
	nativeScope := collection.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: identity.EffectiveMemberID}
	// The native repository verifies its body hash and scoped immutable row.
	// Check the returned identity/variant rather than running another validator.
	if r.ID != id || !rc.UUID(id) || !rc.UUID(r.Source.ID) || !rc.UUID(r.Input.StoreID) ||
		r.TargetID != target.TargetIdentity(nativeScope, r.Source.ID, r.Input.StoreID) || r.Revision <= 0 || r.Revision != r.Input.ExpectedRevision+1 ||
		r.Input.SourceID != r.Source.ID || r.EffectiveVersion == 0 || r.EffectiveVersion != r.Input.EffectiveVersion ||
		r.Merchant.OrganizationID != scope.OrganizationID || r.Merchant.StoreID != r.Input.StoreID || r.CreatedAt.IsZero() ||
		(r.Merchant.Platform != "" && r.Merchant.Platform != storecenter.PlatformShein) ||
		r.Input.Temu != nil || r.Input.TikTok != nil || r.Input.Walmart != nil || !target.ValidPlatformRecord(r) {
		return rc.Snapshot{}, rc.ErrUnavailable
	}
	for _, hash := range []string{r.ProductHash, r.InventoryHash, r.RulesHash} {
		b, e := hex.DecodeString(hash)
		if e != nil || len(b) != 32 || hex.EncodeToString(b) != hash {
			return rc.Snapshot{}, rc.ErrUnavailable
		}
	}
	bodyDigest := collection.Digest(r)
	version := "sha256:" + collection.Digest(struct {
		Projection string `json:"projection"`
		BodyDigest string `json:"bodyDigest"`
	}{"shein-target-report-v1", bodyDigest})
	result := rc.Snapshot{SourceInfo: rc.SourceInfo{
		Ref:   rc.SourceRef{Kind: "SHEIN_TARGET_RECORD", ID: id, Version: version},
		Title: "SHEIN 已保存资料 · " + r.Source.Source.ProductKey, ProductKey: r.Source.Source.ProductKey,
		StoreID: r.Input.StoreID, SourceAt: &r.CreatedAt,
	}, Content: rc.Document{SchemaVersion: 1}}
	result.Content.Sections = []rc.Section{{Title: "保存时资料身份", Fields: []rc.Field{
		{Label: "资料记录 ID", Value: r.ID}, {Label: "目标 ID", Value: r.TargetID}, {Label: "资料修订", Value: strconv.FormatInt(r.Revision, 10)},
		{Label: "原来源 ID", Value: r.Source.ID}, {Label: "原来源类型", Value: r.Source.Source.Kind},
		{Label: "原商品版本", Value: strconv.FormatUint(r.Source.Source.Version, 10)}, {Label: "有效商品版本", Value: strconv.FormatUint(r.EffectiveVersion, 10)},
		{Label: "店铺 ID", Value: r.Input.StoreID}, {Label: "站点", Value: r.Merchant.Site}, {Label: "资料保存时间", Value: r.CreatedAt.UTC().Format(time.RFC3339Nano)},
		{Label: "原资料 SHA-256", Value: bodyDigest}, {Label: "商品 SHA-256", Value: r.ProductHash}, {Label: "资产清单 SHA-256", Value: r.InventoryHash}, {Label: "规则 SHA-256", Value: r.RulesHash},
	}}}
	product := r.Input.Draft.Product
	skus := 0
	for _, skc := range product.SKCs {
		skus += len(skc.SKUs)
	}
	fields := []rc.Field{{Label: "SKU 数量（摘要）", Value: strconv.Itoa(skus)}, {Label: "图片槽位数量（摘要）", Value: strconv.Itoa(len(r.Input.Draft.Images))}}
	for _, name := range product.Names {
		fields = append(fields, rc.Field{Label: "保存时标题 · " + name.Language, Value: name.Name})
	}
	for _, description := range product.Descriptions {
		fields = append(fields, rc.Field{Label: "保存时描述 · " + description.Language, Value: description.Name})
	}
	result.Content.Sections = append(result.Content.Sections, rc.Section{Title: "保存时资料摘要（非完整 SKU/媒体导出）", Fields: fields})
	fields = []rc.Field{{Label: "保存时 ReadyForUpload", Value: strconv.FormatBool(r.Result.ReadyForUpload)}}
	for _, issue := range r.Result.Issues {
		fields = append(fields, rc.Field{Label: issue.Field + " · " + issue.Code, Value: issue.Message})
	}
	result.Content.Sections = append(result.Content.Sections,
		rc.Section{Title: "保存时构建结果（不代表当前上传许可或离线诊断）", Fields: fields},
		rc.Section{Title: "报告范围", Fields: []rc.Field{
			{Label: "来源", Value: "本人已保存的准确历史资料版本，不代表当前 head。"},
			{Label: "未重新评估", Value: "未重新执行规则或资产校验；旧离线诊断未覆盖，在线条件 UNKNOWN。"},
			{Label: "未执行", Value: "平台授权、审核、上架、库存和实际效果 NOT_RUN；报告不是发布许可或恢复输入。"},
		}})
	// Reuse all existing byte/field/text limits. Never truncate required issues.
	if _, _, err := result.Bytes(); err != nil {
		return rc.Snapshot{}, err
	}
	return result, nil
}
