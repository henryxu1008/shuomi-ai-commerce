package openai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"task-processor/internal/shared/aiidentity"

	"gorm.io/gorm"
)

type AIClientCredential struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	TenantID      string    `json:"tenant_id" gorm:"type:varchar(64);not null;uniqueIndex:idx_ai_client_credential_scope,priority:1"`
	UserID        string    `json:"user_id,omitempty" gorm:"type:varchar(128);not null;default:'';uniqueIndex:idx_ai_client_credential_scope,priority:2"`
	ClientName    string    `json:"client_name" gorm:"type:varchar(64);not null;uniqueIndex:idx_ai_client_credential_scope,priority:3"`
	APIKey        string    `json:"-" gorm:"type:text;not null"`
	BaseURL       string    `json:"base_url" gorm:"type:text;not null"`
	Model         string    `json:"model" gorm:"type:varchar(128);not null"`
	APIStyle      string    `json:"api_style,omitempty" gorm:"type:varchar(64);not null;default:''"`
	TimeoutSecond int       `json:"timeout_second" gorm:"not null;default:0"`
	Enabled       bool      `json:"enabled" gorm:"not null;default:true"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (AIClientCredential) TableName() string {
	return "ai_client_credentials"
}

type GormCredentialResolver struct {
	db *gorm.DB
}

func NewOrganizationOnlyCredentialResolver(db *gorm.DB) *GormCredentialResolver {
	return &GormCredentialResolver{db: db}
}

func NewGormCredentialResolver(db *gorm.DB) *GormCredentialResolver {
	return &GormCredentialResolver{db: db}
}

func (r *GormCredentialResolver) SaveCredential(ctx context.Context, credential AIClientCredential) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("ai credential resolver database is nil")
	}
	credential.TenantID = strings.TrimSpace(credential.TenantID)
	credential.UserID = strings.TrimSpace(credential.UserID)
	credential.ClientName = normalizeClientName(credential.ClientName)
	credential.APIKey = strings.TrimSpace(credential.APIKey)
	credential.BaseURL = strings.TrimSpace(credential.BaseURL)
	credential.Model = strings.TrimSpace(credential.Model)
	credential.APIStyle = strings.TrimSpace(credential.APIStyle)
	if credential.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if credential.ClientName == "" {
		return fmt.Errorf("client_name is required")
	}
	if credential.APIKey == "" {
		return fmt.Errorf("api_key is required")
	}
	if credential.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}
	if credential.Model == "" {
		return fmt.Errorf("model is required")
	}
	updates := map[string]any{
		"api_key":        credential.APIKey,
		"base_url":       credential.BaseURL,
		"model":          credential.Model,
		"api_style":      credential.APIStyle,
		"timeout_second": credential.TimeoutSecond,
		"enabled":        credential.Enabled,
	}
	return r.db.WithContext(ctx).Where(
		"tenant_id = ? AND user_id = ? AND client_name = ?",
		credential.TenantID,
		credential.UserID,
		credential.ClientName,
	).Assign(updates).FirstOrCreate(&credential).Error
}

func (r *GormCredentialResolver) GetCredential(ctx context.Context, tenantID, userID, clientName string) (*AIClientCredential, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("ai credential resolver database is nil")
	}
	tenantID = strings.TrimSpace(tenantID)
	userID = strings.TrimSpace(userID)
	clientName = normalizeClientName(clientName)
	if tenantID == "" {
		return nil, nil
	}
	var credential AIClientCredential
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND client_name = ?", tenantID, userID, clientName).
		First(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

func (r *GormCredentialResolver) ResolveClientConfig(ctx context.Context, clientName string) (*ResolvedClientConfig, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("ai credential resolver database is nil")
	}
	identity := aiidentity.FromContext(ctx)
	tenantID := strings.TrimSpace(identity.TenantID)
	if tenantID == "" {
		return nil, ErrClientConfigurationUnavailable
	}
	credential, err := r.findCredential(ctx, tenantID, "", normalizeClientName(clientName))
	if err != nil {
		return nil, err
	}
	if credential == nil {
		return nil, ErrClientConfigurationUnavailable
	}
	return credential.toResolvedClientConfig(), nil
}

func (r *GormCredentialResolver) findCredential(ctx context.Context, tenantID, userID, clientName string) (*AIClientCredential, error) {
	var credential AIClientCredential
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND client_name = ? AND enabled = ?", tenantID, userID, clientName, true).
		First(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

func (c AIClientCredential) toResolvedClientConfig() *ResolvedClientConfig {
	cfg := &ClientConfig{APIKey: c.APIKey, BaseURL: c.BaseURL, Model: c.Model, APIStyle: c.APIStyle}
	if c.TimeoutSecond > 0 {
		cfg.Timeout = time.Duration(c.TimeoutSecond) * time.Second
	}
	return &ResolvedClientConfig{CacheKey: CredentialReference(c), Config: cfg}
}

func normalizeClientName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "default"
	}
	return name
}

// NewOrganizationCredentialResolver retains the current generation/quote entry;
// ResolveClientConfig always selects the organization row, never a member/global fallback.
func NewOrganizationCredentialResolver(db *gorm.DB) *GormCredentialResolver {
	return &GormCredentialResolver{db: db}
}
