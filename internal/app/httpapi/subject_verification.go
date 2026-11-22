package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	zitadel "github.com/henryxu1008/shuomi-ai-commerce/internal/authruntime/zitadel"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/core/config"
	store "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/subjectverification"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/tencentesign"
	kernelmodule "github.com/henryxu1008/shuomi-ai-commerce/internal/kernel/module"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/supplymarket"
	domain "github.com/henryxu1008/shuomi-ai-commerce/internal/subjectverification"
	verificationhttp "github.com/henryxu1008/shuomi-ai-commerce/internal/subjectverification/httpapi"
	"gorm.io/gorm"
)

var _ personalKYCReader = (*domain.PersonalService)(nil)

type subjectVerificationModule struct {
	handler     verificationhttp.Handler
	personal    verificationhttp.PersonalHandler
	personalKYC personalKYCReader
	merchants   supplymarket.VerifiedMerchantReader
}

func (subjectVerificationModule) Name() string { return "subject-verification" }
func (subjectVerificationModule) Enabled(cfg *config.Config) bool {
	return cfg != nil && cfg.Workbench.Enabled
}
func (m subjectVerificationModule) Register(reg *kernelmodule.Registry) error {
	reg.AddRoutes(m.handler.Routes(accountOrganizationTarget)...)
	reg.AddRoutes(m.personal.Routes()...)
	return nil
}

// This opt-in private file is absent by default, so unconfigured installations
// expose an explicit unavailable capability and make no provider requests.
func buildSubjectVerificationModule(ctx context.Context, db *gorm.DB, cfg *config.Config) (kernelmodule.Module, error) {
	m := subjectVerificationModule{}
	personal, err := buildPersonalVerification(ctx, db, cfg)
	if err != nil {
		return nil, err
	}
	m.personal = personal
	if reader, ok := personal.Service.(personalKYCReader); ok {
		m.personalKYC = reader
	}
	path := strings.TrimSpace(os.Getenv("TENCENT_ESIGN_VERIFICATION_CONFIG_FILE"))
	if path == "" {
		return m, nil
	}
	invalid := errors.New("subject verification private configuration invalid")
	if config.VerifyPrivateFiles(ctx, []string{path}) != nil {
		return nil, invalid
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, invalid
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, 8193))
	if err != nil || len(raw) > 8192 {
		return nil, invalid
	}
	var values struct{ Scope, SecretID, SecretKey, OperatorID, CallbackSigningKey, CallbackEncryptionKey, DataEncryptionKey string }
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&values) != nil {
		return nil, invalid
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return nil, invalid
	}
	if !authidentity.IsBoundedIdentifier(values.Scope) || !authidentity.IsBoundedIdentifier(values.OperatorID) || values.SecretID == "" || values.SecretKey == "" {
		return nil, invalid
	}
	key, err := base64.StdEncoding.DecodeString(values.DataEncryptionKey)
	if err != nil {
		return nil, invalid
	}
	if string(key) == values.CallbackEncryptionKey || values.CallbackSigningKey == values.CallbackEncryptionKey {
		return nil, invalid
	}
	protection, err := store.NewProtection(key)
	if err != nil {
		return nil, invalid
	}
	callback, err := tencentesign.NewCallback([]byte(values.CallbackSigningKey), []byte(values.CallbackEncryptionKey))
	if err != nil {
		return nil, invalid
	}
	client, err := tencentesign.NewClient(values.SecretID, values.SecretKey, values.OperatorID)
	if err != nil {
		return nil, invalid
	}
	repository, err := store.NewRepository(ctx, db)
	if err != nil {
		return nil, err
	}
	service := &domain.Service{Store: repository, Provider: client, Protection: protection, Scope: values.Scope}
	m.handler = verificationhttp.Handler{Service: service, Callback: callback, Profile: zitadel.NewUserInfoClient(cfg.ListingKit.Zitadel.IssuerURL, &http.Client{})}
	m.merchants = verifiedMarketMerchant{service}
	return m, nil
}
