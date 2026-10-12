package temu

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"task-processor/internal/storecenter"
)

const (
	maxOfficialBody      = 2 * 1024 * 1024
	supplierInfoAPI      = "bg.open.accesstoken.info.get.global"
	supplierModeAPI      = "bg.mall.info.get"
	localInfoAPI         = "bg.open.accesstoken.info.get"
	verificationRevision = "temu-identity-E08-E10-E11-E26-v1"
)

type OfficialClientOptions struct {
	Application storecenter.OfficialApplication
	AppSecret   string
	HTTPClient  *http.Client
}

type OfficialClient struct {
	application storecenter.OfficialApplication
	appSecret   string
	http        *http.Client
	now         func() time.Time
}

func NewOfficialClient(options OfficialClientOptions) (*OfficialClient, error) {
	app := options.Application
	if app.Platform != storecenter.PlatformTemu || !boundedParameter(app.AppID, 128) || !boundedParameter(app.Version, 128) || !boundedParameter(options.AppSecret, 512) || len(options.AppSecret) < 16 || app.CallbackURL != "" || app.AuthorizationMethod != storecenter.AuthorizationManualToken {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	switch app.IdentityPartition {
	case storecenter.PartitionCN:
		if app.SupportedModes != storecenter.ModesFullyManaged|storecenter.ModesSemiManaged {
			return nil, storecenter.ErrOfficialConnectionUnavailable
		}
	case storecenter.PartitionUS, storecenter.PartitionEU, storecenter.PartitionGlobal:
		if app.SupportedModes != storecenter.ModesLocal {
			return nil, storecenter.ErrOfficialConnectionUnavailable
		}
	default:
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	client := http.Client{Timeout: 10 * time.Second}
	if options.HTTPClient != nil {
		client = *options.HTTPClient
	}
	if client.Timeout <= 0 || client.Timeout > 10*time.Second {
		client.Timeout = 10 * time.Second
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &OfficialClient{application: app, appSecret: options.AppSecret, http: &client, now: time.Now}, nil
}

func (c *OfficialClient) Application() storecenter.OfficialApplication { return c.application }
func (c *OfficialClient) AuthorizationURL(state string) (string, error) {
	if !boundedParameter(state, 256) {
		return "", storecenter.ErrOfficialConnectionUnavailable
	}
	return "", nil // Manual input in the original member-authorized Store attempt.
}
func (c *OfficialClient) Exchange(context.Context, string, string) (storecenter.OfficialMerchantCredential, error) {
	return storecenter.OfficialMerchantCredential{}, storecenter.ErrOfficialAuthorizationRejected
}
func (c *OfficialClient) QueryStore(ctx context.Context, credential storecenter.OfficialMerchantCredential) (storecenter.OfficialStoreInformation, error) {
	if !c.validCredential(credential) {
		return storecenter.OfficialStoreInformation{}, storecenter.ErrOfficialConnectionUnavailable
	}
	// Both supplier reads share one total deadline and one exact token. The
	// second request cannot receive a fresh five-second budget or another route.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return storecenter.OfficialStoreInformation{}, storecenter.ErrOfficialConnectionUnavailable
	}
	if c.application.IdentityPartition == storecenter.PartitionCN {
		return c.querySupplier(ctx, credential.Temu.AccessToken)
	}
	return c.queryLocal(ctx, credential.Temu.AccessToken)
}

var _ storecenter.OfficialConnectionProvider = (*OfficialClient)(nil)

func boundedParameter(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && utf8.ValidString(value) && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\r\n\x00")
}

func (c *OfficialClient) validCredential(credential storecenter.OfficialMerchantCredential) bool {
	return credential.AppID == c.application.AppID && credential.Temu != nil && credential.OpenKeyID == "" && credential.SecretKey == "" && credential.SupplierID == "" && boundedParameter(credential.Temu.AccessToken, 4096)
}

func (c *OfficialClient) querySupplier(ctx context.Context, token string) (storecenter.OfficialStoreInformation, error) {
	var identity struct {
		UniqueID    string   `json:"uniqueId"`
		MallName    *string  `json:"mallName"`
		ExpiredTime *int64   `json:"expiredTime"`
		Scopes      []string `json:"apiScopeList"`
	}
	if err := c.readIdentity(ctx, "https://openapi-b-partner.temu.com", supplierInfoAPI, token, &identity); err != nil {
		return storecenter.OfficialStoreInformation{}, err
	}
	if !boundedParameter(identity.UniqueID, 200) || identity.MallName != nil && len(*identity.MallName) > 1000 {
		return storecenter.OfficialStoreInformation{}, storecenter.ErrOfficialConnectionUnavailable
	}
	facts, err := c.verification(identity.UniqueID, storecenter.MerchantSupplierNamespace, 0, identity.ExpiredTime, identity.Scopes)
	if err != nil {
		return storecenter.OfficialStoreInformation{}, err
	}
	var mode struct {
		SemiManaged *bool `json:"semiManagedMall"`
	}
	if err := c.readIdentity(ctx, "https://openapi.kuajingmaihuo.com", supplierModeAPI, token, &mode); err != nil {
		return storecenter.OfficialStoreInformation{}, err
	}
	if mode.SemiManaged == nil {
		return storecenter.OfficialStoreInformation{}, storecenter.ErrOfficialConnectionUnavailable
	}
	facts.Mode = storecenter.MerchantFullyManaged
	if *mode.SemiManaged {
		facts.Mode = storecenter.MerchantSemiManaged
	}
	if _, err := facts.MerchantIdentityHash(); err != nil {
		return storecenter.OfficialStoreInformation{}, err
	}
	return storecenter.OfficialStoreInformation{StoreName: identity.MallName, Verification: &facts}, nil
}

func (c *OfficialClient) queryLocal(ctx context.Context, token string) (storecenter.OfficialStoreInformation, error) {
	var identity struct {
		MallID      *int64   `json:"mallId"`
		MallType    *int     `json:"mallType"`
		RegionID    *int64   `json:"regionId"`
		ExpiredTime *int64   `json:"expiredTime"`
		Scopes      []string `json:"apiScopeList"`
	}
	origin := localAPIOrigin(c.application.IdentityPartition)
	if origin == "" {
		return storecenter.OfficialStoreInformation{}, storecenter.ErrOfficialConnectionUnavailable
	}
	if err := c.readIdentity(ctx, origin, localInfoAPI, token, &identity); err != nil {
		return storecenter.OfficialStoreInformation{}, err
	}
	if identity.MallID == nil || *identity.MallID < 1 || identity.MallType == nil || *identity.MallType != 100 || identity.RegionID == nil || *identity.RegionID < 1 {
		return storecenter.OfficialStoreInformation{}, storecenter.ErrOfficialConnectionUnavailable
	}
	facts, err := c.verification(strconv.FormatInt(*identity.MallID, 10), storecenter.MerchantLocalNamespace, *identity.RegionID, identity.ExpiredTime, identity.Scopes)
	if err != nil {
		return storecenter.OfficialStoreInformation{}, err
	}
	facts.Mode = storecenter.MerchantLocal
	if _, err := facts.MerchantIdentityHash(); err != nil {
		return storecenter.OfficialStoreInformation{}, err
	}
	return storecenter.OfficialStoreInformation{Verification: &facts}, nil
}

func (c *OfficialClient) verification(merchant string, namespace storecenter.OfficialMerchantNamespace, region int64, expiry *int64, scopes []string) (storecenter.OfficialVerificationFacts, error) {
	expires, now, err := c.validScopeExpiry(expiry, scopes)
	if err != nil {
		return storecenter.OfficialVerificationFacts{}, err
	}
	return storecenter.OfficialVerificationFacts{Platform: storecenter.PlatformTemu, IdentityPartition: c.application.IdentityPartition, Namespace: namespace, MerchantID: merchant, RegionID: region, ExpiresAt: expires, Scopes: append([]string{}, scopes...), ContractRevision: verificationRevision, VerifiedAt: now}, nil
}

func (c *OfficialClient) validScopeExpiry(expiry *int64, scopes []string) (time.Time, time.Time, error) {
	if expiry == nil || *expiry <= 0 || *expiry > 253402300799 || scopes == nil || len(scopes) > 4096 {
		return time.Time{}, time.Time{}, storecenter.ErrOfficialConnectionUnavailable
	}
	now := c.now().UTC()
	expires := time.Unix(*expiry, 0).UTC()
	if !now.Before(expires) {
		return time.Time{}, time.Time{}, storecenter.ErrOfficialCredentialExpired
	}
	for _, scope := range scopes {
		if !boundedParameter(scope, 256) {
			return time.Time{}, time.Time{}, storecenter.ErrOfficialConnectionUnavailable
		}
	}
	return expires, now, nil
}

func (c *OfficialClient) readIdentity(ctx context.Context, origin, api, token string, result any) error {
	return c.readOfficialResponse(ctx, origin, api, token, nil, result)
}

func (c *OfficialClient) readOfficialResponse(ctx context.Context, origin, api, token string, fields map[string]json.RawMessage, result any) error {
	_, err := c.callOfficialResponse(ctx, origin, api, token, fields, result)
	return err
}

func (c *OfficialClient) callOfficialResponse(ctx context.Context, origin, api, token string, fields map[string]json.RawMessage, result any) (string, error) {
	if ctx.Err() != nil {
		return "", storecenter.ErrOfficialConnectionUnavailable
	}
	body, err := c.encodeRequest(api, token, fields)
	if err != nil {
		return "", err
	}
	defer clear(body)
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, origin+"/openapi/router", bytes.NewReader(body))
	if err != nil {
		return "", storecenter.ErrOfficialConnectionUnavailable
	}
	r.Header.Set("Content-Type", "application/json;charset=UTF-8")
	response, err := c.http.Do(r)
	if err != nil {
		return "", storecenter.ErrOfficialConnectionUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", storecenter.ErrOfficialConnectionUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxOfficialBody+1))
	if err != nil || len(data) > maxOfficialBody {
		return "", storecenter.ErrOfficialConnectionUnavailable
	}
	defer clear(data)
	var envelope struct {
		Success *bool           `json:"success"`
		Result  json.RawMessage `json:"result"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Success == nil || !*envelope.Success || len(envelope.Result) == 0 || bytes.Equal(envelope.Result, []byte("null")) || json.Unmarshal(envelope.Result, result) != nil {
		// No provider message/raw body is included: either can echo secrets.
		return "", storecenter.ErrOfficialConnectionUnavailable
	}
	if ctx.Err() != nil {
		return "", storecenter.ErrOfficialConnectionUnavailable
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
