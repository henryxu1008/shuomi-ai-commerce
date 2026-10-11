package temu

import (
	"encoding/json"
	"task-processor/internal/integration/credentialprotection"
	"task-processor/internal/storecenter"
)

type CredentialProtection struct{ cipher *credentialprotection.AESGCM }

func NewCredentialProtection(keyID string, key []byte) (*CredentialProtection, error) {
	cipher, err := credentialprotection.NewAESGCM(keyID, key)
	if err != nil {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	return &CredentialProtection{cipher: cipher}, nil
}

type sealedToken struct {
	Version     int    `json:"version"`
	AppID       string `json:"appId"`
	AccessToken string `json:"accessToken"`
}

func tokenAAD(a storecenter.OfficialConnectionAttempt, key string) []byte {
	data, _ := json.Marshal([]any{"temu-merchant-v1", a.OrganizationID, a.StoreID, a.AttemptID, a.ActorID, a.MemberID, a.Platform, a.IdentityPartition, a.AuthorizationMethod, a.AppID, a.AppVersion, a.ConnectionVersion, key})
	return data
}
func validTokenAttempt(a storecenter.OfficialConnectionAttempt) bool {
	if a.Platform != storecenter.PlatformTemu || a.AuthorizationMethod != storecenter.AuthorizationManualToken || a.ConnectionVersion < 1 {
		return false
	}
	for _, value := range []string{a.OrganizationID, a.StoreID, a.AttemptID, a.ActorID, a.MemberID, a.AppID, a.AppVersion} {
		if !boundedParameter(value, 200) {
			return false
		}
	}
	return a.IdentityPartition == storecenter.PartitionCN || a.IdentityPartition == storecenter.PartitionUS || a.IdentityPartition == storecenter.PartitionEU || a.IdentityPartition == storecenter.PartitionGlobal
}
func (p *CredentialProtection) Seal(a storecenter.OfficialConnectionAttempt, c storecenter.OfficialMerchantCredential) (string, string, error) {
	if !validTokenAttempt(a) || c.AppID != a.AppID || c.Temu == nil || c.OpenKeyID != "" || c.SecretKey != "" || c.SupplierID != "" || !boundedParameter(c.Temu.AccessToken, 4096) {
		return "", "", storecenter.ErrOfficialConnectionUnavailable
	}
	plain, err := json.Marshal(sealedToken{Version: 1, AppID: c.AppID, AccessToken: c.Temu.AccessToken})
	if err != nil {
		return "", "", storecenter.ErrOfficialConnectionUnavailable
	}
	defer clear(plain)
	key := p.cipher.KeyID()
	encrypted, err := p.cipher.Seal(plain, tokenAAD(a, key))
	if err != nil {
		return "", "", storecenter.ErrOfficialConnectionUnavailable
	}
	return key, encrypted, nil
}
func (p *CredentialProtection) Open(a storecenter.OfficialConnectionAttempt, keyID, encrypted string) (storecenter.OfficialMerchantCredential, error) {
	if !validTokenAttempt(a) {
		return storecenter.OfficialMerchantCredential{}, storecenter.ErrOfficialConnectionUnavailable
	}
	plain, err := p.cipher.Open(keyID, encrypted, tokenAAD(a, keyID))
	if err != nil {
		return storecenter.OfficialMerchantCredential{}, storecenter.ErrOfficialConnectionUnavailable
	}
	defer clear(plain)
	var token sealedToken
	if json.Unmarshal(plain, &token) != nil || token.Version != 1 || token.AppID != a.AppID || !boundedParameter(token.AccessToken, 4096) {
		return storecenter.OfficialMerchantCredential{}, storecenter.ErrOfficialConnectionUnavailable
	}
	return storecenter.OfficialMerchantCredential{AppID: token.AppID, Temu: &storecenter.TemuMerchantCredential{AccessToken: token.AccessToken}}, nil
}

var _ storecenter.OfficialCredentialProtection = (*CredentialProtection)(nil)
