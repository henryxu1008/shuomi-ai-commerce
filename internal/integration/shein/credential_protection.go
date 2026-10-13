package shein

import (
	"encoding/json"
	"task-processor/internal/integration/credentialprotection"

	"task-processor/internal/storecenter"
)

type CredentialProtection struct {
	cipher *credentialprotection.AESGCM
}

func NewCredentialProtection(keyID string, key []byte) (*CredentialProtection, error) {
	cipher, err := credentialprotection.NewAESGCM(keyID, key)
	if err != nil {
		return nil, err
	}
	return &CredentialProtection{cipher: cipher}, nil
}

type sealedCredential struct{ AppID, OpenKeyID, SecretKey, SupplierID string }

func associatedData(a storecenter.OfficialConnectionAttempt, keyID string) []byte {
	data, _ := json.Marshal([]any{"shein-merchant-v1", a.OrganizationID, a.StoreID, a.AttemptID, a.AppID, a.AppVersion, a.ActorID, a.MemberID, a.ConnectionVersion, keyID})
	return data
}
func (p *CredentialProtection) Seal(a storecenter.OfficialConnectionAttempt, c storecenter.OfficialMerchantCredential) (string, string, error) {
	if c.Temu != nil || c.AppID != a.AppID || c.OpenKeyID == "" || c.SecretKey == "" || c.SupplierID == "" {
		return "", "", storecenter.ErrOfficialExchangeUnknown
	}
	plain, err := json.Marshal(sealedCredential{c.AppID, c.OpenKeyID, c.SecretKey, c.SupplierID})
	if err != nil {
		return "", "", err
	}
	defer clear(plain)
	key := p.cipher.KeyID()
	encrypted, err := p.cipher.Seal(plain, associatedData(a, key))
	if err != nil {
		return "", "", storecenter.ErrOfficialConnectionUnavailable
	}
	return key, encrypted, nil
}
func (p *CredentialProtection) Open(a storecenter.OfficialConnectionAttempt, keyID, encrypted string) (storecenter.OfficialMerchantCredential, error) {
	plain, err := p.cipher.Open(keyID, encrypted, associatedData(a, keyID))
	if err != nil {
		return storecenter.OfficialMerchantCredential{}, storecenter.ErrOfficialConnectionUnavailable
	}
	defer clear(plain)
	var c sealedCredential
	if err := json.Unmarshal(plain, &c); err != nil || c.AppID != a.AppID || c.OpenKeyID == "" || c.SecretKey == "" || c.SupplierID == "" {
		return storecenter.OfficialMerchantCredential{}, storecenter.ErrOfficialConnectionUnavailable
	}
	return storecenter.OfficialMerchantCredential{AppID: c.AppID, OpenKeyID: c.OpenKeyID, SecretKey: c.SecretKey, SupplierID: c.SupplierID}, nil
}
