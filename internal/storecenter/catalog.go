package storecenter

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"gorm.io/gorm"
)

var ErrCatalogSummaryIncomplete = errors.New("store catalog summary unavailable for complete filtering")

type CatalogMetric struct {
	Value    *int64 `json:"value"`
	Unknown  int64  `json:"unknown"`
	Complete bool   `json:"complete"`
	Reason   string `json:"reason"`
}
type CatalogSignals struct {
	Connection      ConnectionStatus        `json:"connectionStatus"`
	ApplicationType OfficialApplicationType `json:"applicationType"`
	Products        CatalogMetric           `json:"products"`
	Orders          CatalogMetric           `json:"orders"`
	Inventory       CatalogMetric           `json:"inventory"`
	ProductSync     string                  `json:"productSync"`
	OrderSync       string                  `json:"orderSync"`
	SyncHealthy     bool                    `json:"syncHealthy"`
	ObservedAt      *time.Time              `json:"observedAt"`
	Reason          string                  `json:"reason"`
}
type CatalogSignalsReader interface {
	ReadCatalogSignals(context.Context, string, []string, map[string]*int64) (map[string]CatalogSignals, error)
}
type CatalogRequest struct {
	OrganizationID                                                string
	Page, PageSize                                                int
	Search, Platform, Region, ApplicationType, GroupID, Attention string
	Status                                                        RecordStatus
}
type CatalogItem struct {
	Store     Store
	Profile   StoreProfile
	Signals   CatalogSignals
	Attention string
}
type CatalogMetrics struct {
	Total       int64     `json:"total"`
	Normal      *int64    `json:"normal"`
	Attention   *int64    `json:"attention"`
	SyncHealthy *int64    `json:"syncHealthy"`
	Unknown     int64     `json:"unknown"`
	Complete    bool      `json:"complete"`
	ObservedAt  time.Time `json:"observedAt"`
}
type CatalogGroup struct {
	StoreGroup
	Count int64 `json:"count"`
}
type CatalogFacets struct {
	Regions          []string                  `json:"regions"`
	ApplicationTypes []OfficialApplicationType `json:"applicationTypes"`
	Groups           []CatalogGroup            `json:"groups"`
	CanManageGroups  bool                      `json:"canManageGroups"`
}
type CatalogResult struct {
	Items          []CatalogItem
	Total          int64
	Page, PageSize int
	Metrics        CatalogMetrics
	Facets         CatalogFacets
}
type Catalog struct {
	scoped  *MemberScopedStoreRepository
	signals CatalogSignalsReader
}

func NewCatalog(db *gorm.DB, access StoreMemberAuthorizer, signals CatalogSignalsReader) (*Catalog, error) {
	scoped, err := NewMemberScopedStoreRepository(db, access)
	if err != nil {
		return nil, err
	}
	return &Catalog{scoped: scoped, signals: signals}, nil
}
func (c *Catalog) Get(ctx context.Context, org, id string) (CatalogItem, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	store, err := c.scoped.Get(ctx, org, id)
	if err != nil {
		return CatalogItem{}, err
	}
	metadata, _ := NewMetadataRepository(c.scoped.db, c.scoped.access, nil)
	profile, err := metadata.Profile(ctx, org, id)
	if err != nil {
		return CatalogItem{}, err
	}
	signal := CatalogSignals{Connection: ConnectionStatusUnavailable, Reason: "summary_unavailable", Products: CatalogMetric{Reason: "summary_unavailable"}, Orders: CatalogMetric{Reason: "summary_unavailable"}, Inventory: CatalogMetric{Reason: "summary_unavailable"}}
	if c.signals != nil {
		if values, err := c.signals.ReadCatalogSignals(ctx, org, []string{id}, map[string]*int64{id: profile.LowStockThreshold}); err == nil {
			if value, ok := values[id]; ok {
				signal = value
			}
		}
	}
	if _, err = c.scoped.Get(ctx, org, id); err != nil {
		return CatalogItem{}, err
	}
	return CatalogItem{Store: *store, Profile: profile, Signals: signal, Attention: classifySignals(signal)}, nil
}

func classifySignals(s CatalogSignals) string {
	if s.Connection == ConnectionStatusDisconnected || s.Connection == ConnectionStatusExpired || s.ProductSync == "failed" || s.ProductSync == "partial" || s.ProductSync == "suspended" || s.OrderSync == "failed" || s.OrderSync == "partial" || s.OrderSync == "suspended" || s.Inventory.Value != nil && *s.Inventory.Value > 0 {
		return "attention"
	}
	if s.Connection == ConnectionStatusConnected && s.SyncHealthy && s.Products.Complete && s.Orders.Complete && (s.Inventory.Reason == "threshold_not_set" || s.Inventory.Complete) {
		return "normal"
	}
	return "unknown"
}
func (c *Catalog) List(ctx context.Context, q CatalogRequest) (CatalogResult, error) {
	result := CatalogResult{Items: []CatalogItem{}, Page: q.Page, PageSize: q.PageSize, Facets: CatalogFacets{Regions: []string{}, ApplicationTypes: []OfficialApplicationType{}, Groups: []CatalogGroup{}}}
	if _, _, err := normalizeListStoresRequest(ListStoresRequest{OrganizationID: q.OrganizationID, Page: q.Page, PageSize: q.PageSize, Platform: q.Platform, Status: q.Status}); err != nil {
		return result, err
	}
	q.Search = strings.TrimSpace(q.Search)
	if !utf8.ValidString(q.Search) || len(q.Search) > 200 || strings.ContainsFunc(q.Search, unicode.IsControl) || len(q.Region) > 128 || strings.ContainsFunc(q.Region, unicode.IsControl) || q.ApplicationType != "" && !OfficialApplicationType(q.ApplicationType).Valid() || q.Attention != "" && q.Attention != "normal" && q.Attention != "attention" && q.Attention != "unknown" {
		return result, ErrVersionConflict
	}
	if q.GroupID != "" && q.GroupID != "ungrouped" {
		if _, err := canonicalUUID(q.GroupID); err != nil {
			return result, ErrVersionConflict
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	access, err := c.scoped.authorize(ctx, q.OrganizationID, false)
	if err != nil {
		return result, err
	}
	result.Facets.CanManageGroups = access.Administrator && access.CanWrite
	records := []workbenchStoreRecord{}
	profiles := []StoreProfile{}
	groups := []StoreGroup{}
	err = c.scoped.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		base := tx.Where("organization_id=? AND deleted_at IS NULL", q.OrganizationID)
		if !access.Administrator {
			base = base.Where("EXISTS(SELECT 1 FROM workbench_store_member_grants g WHERE g.organization_id=workbench_stores.organization_id AND g.store_id=workbench_stores.id AND g.member_id=? AND g.active=?)", access.MemberID, true)
		}
		if err := base.Order("updated_at DESC,id ASC").Find(&records).Error; err != nil {
			return err
		}
		ids := []string{}
		for _, record := range records {
			ids = append(ids, record.ID)
		}
		if len(ids) > 0 {
			if err := tx.Where("organization_id=? AND store_id IN ?", q.OrganizationID, ids).Find(&profiles).Error; err != nil {
				return err
			}
		}
		return tx.Where("organization_id=? AND deleted_at IS NULL", q.OrganizationID).Order("name ASC,id ASC").Find(&groups).Error
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return result, dependencyError(err)
	}
	byID := map[string]StoreProfile{}
	groupCount := map[string]int64{}
	regions := map[string]bool{}
	for _, profile := range profiles {
		byID[profile.StoreID] = profile
		if profile.GroupID != nil {
			groupCount[*profile.GroupID]++
		}
	}
	folded := cases.Fold()
	search := folded.String(q.Search)
	items := []CatalogItem{}
	thresholds := map[string]*int64{}
	for _, record := range records {
		regions[record.Region] = true
		profile, exists := byID[record.ID]
		if !exists {
			profile = StoreProfile{OrganizationID: q.OrganizationID, StoreID: record.ID}
		}
		if q.Platform != "" && record.Platform != q.Platform || q.Region != "" && record.Region != q.Region || q.Status != "" && record.RecordStatus != string(q.Status) {
			continue
		}
		if search != "" && !strings.Contains(folded.String(record.Name), search) && !strings.Contains(folded.String(record.ID), search) && !strings.Contains(folded.String(record.ExternalStoreID), search) {
			continue
		}
		if q.GroupID == "ungrouped" && profile.GroupID != nil || q.GroupID != "" && q.GroupID != "ungrouped" && (profile.GroupID == nil || *profile.GroupID != q.GroupID) {
			continue
		}
		store, err := rehydrateRecord(record)
		if err != nil {
			return result, dependencyError(err)
		}
		items = append(items, CatalogItem{Store: *store, Profile: profile, Attention: "unknown"})
		thresholds[record.ID] = profile.LowStockThreshold
	}
	for region := range regions {
		result.Facets.Regions = append(result.Facets.Regions, region)
	}
	sort.Strings(result.Facets.Regions)
	for _, group := range groups {
		result.Facets.Groups = append(result.Facets.Groups, CatalogGroup{StoreGroup: group, Count: groupCount[group.ID]})
	}
	signals := map[string]CatalogSignals{}
	summaryAvailable := len(items) == 0
	if c.signals != nil && len(items) > 0 && len(items) <= 500 {
		ids := []string{}
		for _, item := range items {
			ids = append(ids, item.Store.ID())
		}
		if resolved, e := c.signals.ReadCatalogSignals(ctx, q.OrganizationID, ids, thresholds); e == nil {
			signals = resolved
			summaryAvailable = true
		}
	}
	typeSet := map[OfficialApplicationType]bool{}
	filtered := []CatalogItem{}
	normal, attention, healthy, unknown := int64(0), int64(0), int64(0), int64(0)
	for _, item := range items {
		signal, exists := signals[item.Store.ID()]
		if !exists {
			signal = CatalogSignals{Connection: ConnectionStatusUnavailable, Reason: "summary_unavailable", Products: CatalogMetric{Reason: "summary_unavailable"}, Orders: CatalogMetric{Reason: "summary_unavailable"}, Inventory: CatalogMetric{Reason: "summary_unavailable"}}
		}
		item.Signals = signal
		item.Attention = classifySignals(signal)
		if signal.ApplicationType.Valid() {
			typeSet[signal.ApplicationType] = true
		}
		if q.ApplicationType != "" && string(signal.ApplicationType) != q.ApplicationType {
			if !summaryAvailable {
				return result, ErrCatalogSummaryIncomplete
			}
			continue
		}
		if q.Attention != "" && q.Attention != "unknown" && item.Attention == "unknown" {
			return result, ErrCatalogSummaryIncomplete
		}
		if q.Attention != "" && item.Attention != q.Attention {
			continue
		}
		filtered = append(filtered, item)
		switch item.Attention {
		case "normal":
			normal++
		case "attention":
			attention++
		default:
			unknown++
		}
		if signal.SyncHealthy {
			healthy++
		}
	}
	if q.ApplicationType != "" && !summaryAvailable {
		return result, ErrCatalogSummaryIncomplete
	}
	for kind := range typeSet {
		result.Facets.ApplicationTypes = append(result.Facets.ApplicationTypes, kind)
	}
	sort.Slice(result.Facets.ApplicationTypes, func(i, j int) bool { return result.Facets.ApplicationTypes[i] < result.Facets.ApplicationTypes[j] })
	result.Total = int64(len(filtered))
	result.Metrics = CatalogMetrics{Total: result.Total, Unknown: unknown, Complete: unknown == 0, ObservedAt: time.Now().UTC()}
	if unknown == 0 {
		result.Metrics.Normal = &normal
		result.Metrics.Attention = &attention
		result.Metrics.SyncHealthy = &healthy
	}
	start := (q.Page - 1) * q.PageSize
	if start >= 0 && start < len(filtered) {
		result.Items = filtered[start:min(start+q.PageSize, len(filtered))]
	}
	// Revalidate the live member capability after reading every count and facet.
	again, err := c.scoped.authorize(ctx, q.OrganizationID, false)
	if err != nil || again != access {
		return CatalogResult{}, ErrNotFound
	}
	current := []workbenchStoreRecord{}
	base := c.scoped.db.WithContext(ctx).Select("id").Where("organization_id=? AND deleted_at IS NULL", q.OrganizationID)
	if !access.Administrator {
		base = base.Where("EXISTS(SELECT 1 FROM workbench_store_member_grants g WHERE g.organization_id=workbench_stores.organization_id AND g.store_id=workbench_stores.id AND g.member_id=? AND g.active=?)", access.MemberID, true)
	}
	if err = base.Find(&current).Error; err != nil {
		return CatalogResult{}, dependencyError(err)
	}
	visible := map[string]bool{}
	for _, record := range current {
		visible[record.ID] = true
	}
	for _, record := range records {
		if !visible[record.ID] {
			return CatalogResult{}, ErrNotFound
		}
	}
	for _, item := range result.Items {
		if _, err := c.scoped.Get(ctx, q.OrganizationID, item.Store.ID()); err != nil {
			return CatalogResult{}, err
		}
	}
	return result, nil
}
