package openai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"gorm.io/gorm"
)

// CredentialVersion excludes secret material and preserves the existing route
// identity. Rotation changes UpdatedAt; metadata readers never select the key.
func CredentialVersion(row AIClientCredential) string {
	identity := fmt.Sprintf("v1|%d|%d|%s|%s|%s|%s|%d", row.ID, row.UpdatedAt.UTC().UnixNano(), row.ClientName, row.BaseURL, row.Model, row.APIStyle, row.TimeoutSecond)
	sum := sha256.Sum256([]byte(identity))
	return "org-credential:v1:" + hex.EncodeToString(sum[:])
}
func CredentialReference(row AIClientCredential) string {
	return fmt.Sprintf("db:%d:%d:%s", row.ID, row.UpdatedAt.UnixNano(), row.ClientName)
}

// InstallCredentialMetadataProjection is used only by explicit owner schema
// installation. Serving code only reads this non-secret projection.
func InstallCredentialMetadataProjection(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Exec(`DO $$ BEGIN
 IF to_regclass('public.ai_client_credential_bindings') IS NULL THEN
 CREATE VIEW public.ai_client_credential_bindings AS
 SELECT id,tenant_id,user_id,client_name,base_url,model,api_style,timeout_second,enabled,updated_at,
 length(btrim(api_key))>0 AS key_present FROM public.ai_client_credentials;
 END IF; END $$`).Error
}
