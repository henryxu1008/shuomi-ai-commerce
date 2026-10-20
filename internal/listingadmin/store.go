package listingadmin

import (
	"context"
	"errors"
	"time"
)

var ErrStoreNotFound = errors.New("store not found")

type Store struct {
	ID          int64      `json:"id"`
	TenantID    int64      `json:"tenantId"`
	OwnerUserID string     `json:"ownerUserId,omitempty"`
	CreatedBy   string     `json:"createdBy,omitempty"`
	UpdatedBy   string     `json:"updatedBy,omitempty"`
	StoreID     string     `json:"-"`
	Name        string     `json:"name"`
	Username    string     `json:"username"`
	Password    string     `json:"password,omitempty"`
	LoginURL    string     `json:"loginUrl,omitempty"`
	ShopType    string     `json:"shopType"`
	Region      string     `json:"region"`
	Platform    string     `json:"platform"`
	Proxy       string     `json:"proxy,omitempty"`
	Remark      string     `json:"remark,omitempty"`
	Status      int16      `json:"status"`
	ValidFrom   *time.Time `json:"validFrom,omitempty"`
	ValidUntil  *time.Time `json:"validUntil,omitempty"`
	Expired     bool       `json:"expired"`
	CreateTime  *time.Time `json:"createTime,omitempty"`
	UpdateTime  *time.Time `json:"updateTime,omitempty"`
}

type StoreQuery struct {
	TenantID    int64
	OwnerUserID string
	// ReadAccess applies the same owner-or-shared scope used by GetStore.
	// It is intended for read-only catalogs that must include tenant-shared stores.
	ReadAccess bool
	Page       int
	PageSize   int
	Platform   string
}

type StorePage struct {
	Items    []Store `json:"items"`
	Total    int64   `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

type StoreRepository interface {
	ListStores(ctx context.Context, query StoreQuery) (*StorePage, error)
	GetStore(ctx context.Context, tenantID, id int64) (*Store, error)
	UpdateStoreID(ctx context.Context, id int64, storeID string) (*Store, error)
	UpdateStoreStatus(ctx context.Context, tenantID, id int64, status int16, remark string) (*Store, error)
}

type listingStore struct {
	ID          int64      `gorm:"column:id;primaryKey;autoIncrement"`
	TenantID    int64      `gorm:"column:tenant_id;not null;index"`
	OwnerUserID string     `gorm:"column:owner_user_id;type:varchar(128);index"`
	StoreID     string     `gorm:"column:store_id"`
	Name        string     `gorm:"column:name;not null"`
	Username    string     `gorm:"column:username;not null"`
	Password    string     `gorm:"column:password;not null"`
	LoginURL    string     `gorm:"column:login_url"`
	ShopType    string     `gorm:"column:shop_type;not null"`
	Region      string     `gorm:"column:region"`
	Platform    string     `gorm:"column:platform;not null"`
	Proxy       string     `gorm:"column:proxy"`
	Remark      string     `gorm:"column:remark"`
	Status      int16      `gorm:"column:status"`
	ValidFrom   *time.Time `gorm:"column:valid_from"`
	ValidUntil  *time.Time `gorm:"column:valid_until"`
	Expired     bool       `gorm:"column:expired"`
	Creator     string     `gorm:"column:creator"`
	CreatedBy   string     `gorm:"column:created_by;type:varchar(128)"`
	CreateTime  *time.Time `gorm:"column:create_time;autoCreateTime"`
	Updater     string     `gorm:"column:updater"`
	UpdatedBy   string     `gorm:"column:updated_by;type:varchar(128)"`
	UpdateTime  *time.Time `gorm:"column:update_time;autoUpdateTime"`
	Deleted     int16      `gorm:"column:deleted;not null;default:0;index"`
}

func (listingStore) TableName() string {
	return "listing_store"
}
