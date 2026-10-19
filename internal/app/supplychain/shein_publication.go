package supplychainapp

import (
	"context"
	"errors"
	storeapp "task-processor/internal/app/storecenter"
	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	"task-processor/internal/marketplace/shein/goods"
	"task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"time"
)

type SheinPublicationMerchant interface {
	Binding() storecenter.ProductMerchantBinding
	QueryDocumentState(context.Context, string, string) (model.DocumentAudit, error)
	QueryShelfProduct(context.Context, string) (model.ShelfProduct, error)
	QuerySKUInventory(context.Context, []string, string) (model.InventorySnapshot, error)
	Warehouses(context.Context) ([]model.Warehouse, error)
	ShelfOn(context.Context, string) (model.ShelfResult, error)
}
type SheinPublicationStore interface {
	PublicationMerchant(context.Context, collection.Scope, string, string, *storecenter.ProductMerchantBinding) (SheinPublicationMerchant, error)
}
type OfficialSheinPublicationStore struct {
	Access *storeapp.OfficialProductAccess
}

func (s OfficialSheinPublicationStore) PublicationMerchant(ctx context.Context, scope collection.Scope, storeID, purpose string, expected *storecenter.ProductMerchantBinding) (SheinPublicationMerchant, error) {
	if s.Access == nil {
		return nil, record.ErrUnavailable
	}
	return s.Access.Authorize(ctx, storecenter.ProductExecutionSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: purpose}, storeID, expected)
}

type SheinShelfAttemptReader interface {
	ReadShelfAttempt(context.Context, submission.ExecutionScope, submission.ExecutionTarget) (submission.ExecutionAttempt, error)
}
type SheinPublicationDependencies struct {
	Sources                *preparation.SourceSelector
	Records                record.TargetRepository
	Authorization          preparation.Authorizer
	ExecutionAuthorization collection.ExecutionAuthorizer
	Receipts               submission.OfficialReceiptRepository
	Intents                submission.ShelfIntentRepository
	Kernel                 UploadKernel
	Attempts               SheinShelfAttemptReader
	Stores                 SheinPublicationStore
}
type SheinPublicationService struct {
	d   SheinPublicationDependencies
	now func() time.Time
}

func NewSheinPublicationService(d SheinPublicationDependencies) (*SheinPublicationService, error) {
	if d.Sources == nil || d.Records == nil || d.Authorization == nil || d.ExecutionAuthorization == nil || d.Receipts == nil || d.Intents == nil || d.Kernel == nil || d.Attempts == nil || d.Stores == nil {
		return nil, record.ErrUnavailable
	}
	return &SheinPublicationService{d: d, now: time.Now}, nil
}

type SheinShelfInput struct {
	Key         string `json:"key,omitempty"`
	SourceID    string `json:"sourceId"`
	StoreID     string `json:"storeId"`
	ReceiptID   string `json:"receiptId"`
	SKC         string `json:"skc"`
	Token       string `json:"token,omitempty"`
	StockOffset int    `json:"stockOffset,omitempty"`
}
type SheinShelfAttemptView struct {
	ID     string                     `json:"id"`
	Key    string                     `json:"key"`
	Status submission.ExecutionStatus `json:"status"`
}
type SheinSKCStatus struct {
	goods.PublicationSKCObservation
	Attempt *SheinShelfAttemptView `json:"attempt,omitempty"`
}
type SheinPublicationStatus struct {
	SourceID        string                `json:"sourceId"`
	StoreID         string                `json:"storeId"`
	ReceiptID       string                `json:"receiptId"`
	SPU             string                `json:"spu"`
	Version         string                `json:"version"`
	Mode            model.ApplicationMode `json:"mode"`
	ObservedAt      time.Time             `json:"observedAt"`
	AuditAt         *time.Time            `json:"auditAt,omitempty"`
	LiveAt          *time.Time            `json:"liveAt,omitempty"`
	StockAt         *time.Time            `json:"stockAt,omitempty"`
	AuditError      string                `json:"auditError,omitempty"`
	LiveError       string                `json:"liveError,omitempty"`
	StockError      string                `json:"stockError,omitempty"`
	StockType       string                `json:"stockType"`
	StockOffset     int                   `json:"stockOffset"`
	StockTotal      int                   `json:"stockTotal"`
	NextStockOffset *int                  `json:"nextStockOffset,omitempty"`
	StockComplete   bool                  `json:"stockComplete"`
	SKCs            []SheinSKCStatus      `json:"skcs"`
}
type loadedSheinPublication struct {
	scope   collection.Scope
	source  preparation.AuthorizedSource
	record  record.TargetRecord
	receipt submission.OfficialReceipt
	binding storecenter.ProductMerchantBinding
}

func (s *SheinPublicationService) load(ctx context.Context, sourceID, storeID, receiptID string, write bool) (loadedSheinPublication, error) {
	var out loadedSheinPublication
	if s == nil || ctx == nil || !collection.ValidID(sourceID) || !collection.ValidID(storeID) || !collection.ValidID(receiptID) {
		return out, record.ErrInvalid
	}
	scope, e := s.d.Authorization.Authorize(ctx, preparation.PermissionRead)
	if e != nil {
		return out, e
	}
	if write {
		if e = s.d.ExecutionAuthorization.AuthorizeExecution(ctx, scope, preparation.PermissionSubmit); e != nil {
			return out, e
		}
	}
	selected, e := s.d.Sources.Select(ctx, sourceID)
	if e != nil {
		return out, e
	}
	owner, item, _, e := selected.Read(ctx)
	if e != nil || owner != scope {
		return out, record.ErrForbidden
	}
	original, e := s.d.Receipts.ReadOfficial(ctx, scope, receiptID)
	if e != nil || submission.ValidateOfficialReceipt(original) != nil || original.Kind != "publish" || original.Target.Platform != "shein" || original.Target.StoreID != storeID || original.ProductKey != item.Source.ProductKey {
		return out, record.ErrForbidden
	}
	saved, e := s.d.Records.ReadTargetRecord(ctx, scope, original.RecordID)
	if e != nil || saved.Source != item || saved.Merchant != original.Binding || !goods.CorrelatesPublishedMembership(saved.Result.Product, original.Product) {
		return out, record.ErrConflict
	}
	purpose := storecenter.ProductPurposeStatus
	if write {
		purpose = storecenter.ProductPurposeShelfOn
	}
	m, e := s.d.Stores.PublicationMerchant(ctx, scope, storeID, purpose, nil)
	if e != nil {
		return out, record.ErrForbidden
	}
	binding := m.Binding()
	if !submission.CompatibleShelfBinding(original.Binding, binding) {
		return out, record.ErrConflict
	}
	return loadedSheinPublication{scope, selected, saved, original, binding}, nil
}
func (s *SheinPublicationService) merchant(ctx context.Context, p loadedSheinPublication, purpose string) (SheinPublicationMerchant, error) {
	if _, _, _, e := p.source.Read(ctx); e != nil {
		return nil, e
	}
	return s.d.Stores.PublicationMerchant(ctx, p.scope, p.binding.StoreID, purpose, &p.binding)
}
func (s *SheinPublicationService) Status(ctx context.Context, sourceID, storeID, receiptID string, offset int) (SheinPublicationStatus, error) {
	if ctx == nil {
		return SheinPublicationStatus{}, record.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	p, e := s.load(ctx, sourceID, storeID, receiptID, false)
	if e != nil {
		return SheinPublicationStatus{}, e
	}
	return s.observe(ctx, p, true, offset)
}
func (s *SheinPublicationService) observe(ctx context.Context, p loadedSheinPublication, inventory bool, offset int) (SheinPublicationStatus, error) {
	v := SheinPublicationStatus{SourceID: p.record.Source.ID, StoreID: p.binding.StoreID, ReceiptID: p.receipt.ID, SPU: p.receipt.Product.SPUName, Version: p.receipt.Product.Version, Mode: model.ApplicationMode(p.binding.ApplicationType), ObservedAt: s.now().UTC(), StockType: "VI", StockOffset: offset, SKCs: []SheinSKCStatus{}}
	if v.Mode == model.ModeFullyManaged {
		v.StockType = "PI"
	}
	ids := []string{}
	for _, skc := range p.receipt.Product.SKCs {
		for _, sku := range skc.SKUs {
			ids = append(ids, sku.SKUCode)
		}
	}
	v.StockTotal = len(ids)
	if offset < 0 || offset%100 != 0 || offset >= len(ids) && offset != 0 {
		return v, record.ErrInvalid
	}
	var audit model.DocumentAudit
	var live model.ShelfProduct
	var stock model.InventorySnapshot
	var warehouses []model.Warehouse
	end := min(offset+100, len(ids))
	page := map[string]bool{}
	if inventory {
		for _, id := range ids[offset:end] {
			page[id] = true
		}
	}
	if !authidentity.IsBoundedIdentifier(v.Version) {
		v.AuditError = "原上传回执缺少审核版本，不能借用最新版本"
	} else {
		m, e := s.merchant(ctx, p, storecenter.ProductPurposeStatus)
		if e != nil {
			return v, record.ErrForbidden
		}
		audit, e = m.QueryDocumentState(ctx, v.SPU, v.Version)
		at := s.now().UTC()
		v.AuditAt = &at
		if e != nil {
			audit = model.DocumentAudit{}
			v.AuditError = "原版本审核查询失败或返回不匹配"
		}
	}
	m, e := s.merchant(ctx, p, storecenter.ProductPurposeStatus)
	if e != nil {
		return v, record.ErrForbidden
	}
	live, e = m.QueryShelfProduct(ctx, v.SPU)
	at := s.now().UTC()
	v.LiveAt = &at
	if e != nil {
		live = model.ShelfProduct{}
		v.LiveError = "当前商品状态查询失败，可能尚未审核通过"
	}
	if inventory && len(ids) > 0 {
		m, e = s.merchant(ctx, p, storecenter.ProductPurposeStatus)
		if e != nil {
			return v, record.ErrForbidden
		}
		warehouses, e = m.Warehouses(ctx)
		if e != nil {
			warehouses = nil
		}
		m, e = s.merchant(ctx, p, storecenter.ProductPurposeStatus)
		if e != nil {
			return v, record.ErrForbidden
		}
		stock, e = m.QuerySKUInventory(ctx, ids[offset:end], v.StockType)
		at := s.now().UTC()
		v.StockAt = &at
		if e != nil || stock.Type != v.StockType {
			stock = model.InventorySnapshot{}
			v.StockError = "库存查询失败或类型不匹配"
		} else {
			v.StockComplete = offset == 0 && end == len(ids)
		}
		if end < len(ids) {
			v.NextStockOffset = &end
			v.StockError = "本次仅查询部分SKU，可继续查询下一组"
		}
	}
	for _, row := range goods.ObservePublication(p.receipt.Product, p.record.Result.Product, v.Mode, audit, live, stock, warehouses, s.now(), page) {
		for _, sku := range row.SKUs {
			if !sku.InventoryScopeKnown {
				v.StockComplete = false
				if inventory && v.StockError == "" {
					v.StockError = "部分SKU的美国可用库存缺失或仓库归属不明"
				}
			}
		}
		skc := SheinSKCStatus{PublicationSKCObservation: row}
		target, _ := submission.ShelfSubjectID(row.SKC, "shein-us")
		attempt, e := s.d.Attempts.ReadShelfAttempt(ctx, submission.ExecutionScope{OrganizationID: p.scope.OrganizationID}, submission.ExecutionTarget{Platform: "shein", StoreID: p.binding.StoreID, SubjectID: target})
		if e == nil {
			if attempt.Status == submission.ExecutionClaimed && !s.now().Before(attempt.LeaseExpiresAt) {
				attempt, e = s.d.Kernel.Expire(ctx, submission.ExecutionScope{OrganizationID: p.scope.OrganizationID}, attempt.AttemptID)
				if e != nil {
					return v, record.ErrUnavailable
				}
			}
			skc.Attempt = &SheinShelfAttemptView{attempt.AttemptID, attempt.IntentKey, attempt.Status}
			if !submission.ShelfCooldownReady(attempt, s.now()) {
				skc.CanShelf = false
				skc.BlockingReason = "已有上架请求或操作间隔尚未满足；仅可查询原结果"
			}
		} else if !errors.Is(e, submission.ErrExecutionNotFound) {
			return v, record.ErrUnavailable
		}
		v.SKCs = append(v.SKCs, skc)
	}
	if _, _, _, e = p.source.Read(ctx); e != nil {
		return v, e
	}
	if _, e = s.merchant(ctx, p, storecenter.ProductPurposeStatus); e != nil {
		return v, record.ErrForbidden
	}
	return v, nil
}
