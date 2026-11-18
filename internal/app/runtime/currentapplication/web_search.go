package currentapplication

import (
	"context"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"errors"

	"gorm.io/gorm"

	credential "task-processor/internal/aicapability/channelcredential"
	"task-processor/internal/aicapability/websearch"
	"task-processor/internal/integration/aicapability/bravesearch"
	channelstore "task-processor/internal/integration/persistence/channelcredential"
	websearchstore "task-processor/internal/integration/persistence/websearch"
)

func (c *Config) webSearchEnabled() bool {
	return c != nil && c.AIWorkbench != nil && c.AIWorkbench.PublicWebSearch
}

func (c *Config) prepareWebSearch(ctx context.Context, routeWriter, credentialReader, credentialWriter *gorm.DB, secrets credential.SecretReader) (*websearch.Service, error) {
	if routeWriter == nil || credentialReader == nil || credentialWriter == nil || secrets == nil || c.ChannelCredentials == nil {
		return nil, errors.New("public web search requires the admitted credential and route pools")
	}
	repo := websearchstore.New(routeWriter)
	if err := repo.VerifyRuntime(ctx); err != nil {
		return nil, err
	}
	protection, err := c.ChannelCredentials.protection()
	if err != nil {
		return nil, err
	}
	meta, err := channelstore.New(credentialReader, nil)
	if err != nil {
		return nil, err
	}
	writer, err := channelstore.New(credentialWriter, protection)
	if err != nil {
		return nil, err
	}
	var versionStoreReady bool
	if err := credentialWriter.WithContext(ctx).Raw("SELECT has_table_privilege(current_user, 'public.ai_search_credential_versions', 'SELECT') AND has_column_privilege(current_user, 'public.ai_search_credential_versions', 'organization_id', 'INSERT') AND has_column_privilege(current_user, 'public.ai_search_credential_versions', 'revision', 'INSERT')").Row().Scan(&versionStoreReady); err != nil || !versionStoreReady {
		return nil, errors.New("public web search requires the credential revision metadata grant")
	}
	root, err := base64.StdEncoding.Strict().DecodeString(c.ChannelCredentials.ProtectionKey)
	if err != nil {
		return nil, err
	}
	signing, err := hkdf.Key(sha256.New, root, nil, "brave-web-search-request-v1", 32)
	if err != nil {
		return nil, err
	}
	return &websearch.Service{Repo: repo, Metadata: meta, Writer: writer, Secrets: secrets, Provider: bravesearch.New(nil), FingerprintKey: signing}, nil
}
