package channelcredentialstore

import (
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	credential "task-processor/internal/aicapability/channelcredential"
	"task-processor/internal/integration/credentialprotection"
)

type Protection struct {
	cipher         *credentialprotection.AESGCM
	fingerprintKey []byte
}

func NewProtection(id string, root []byte) (*Protection, error) {
	c, e := credentialprotection.NewAESGCM(id, root)
	if e != nil {
		return nil, credential.ErrUnavailable
	}
	k, e := hkdf.Key(sha256.New, root, nil, "ai-channel-credential-command-v1", 32)
	if e != nil {
		return nil, credential.ErrUnavailable
	}
	return &Protection{cipher: c, fingerprintKey: k}, nil
}

func (p *Protection) Fingerprint(c credential.Command) (string, error) {
	if p == nil || c.Validate() != nil {
		return "", credential.ErrUnavailable
	}
	// Command JSON intentionally omits the secret; the private canonical envelope
	// explicitly includes it before HMAC. Only the keyed digest is persisted.
	raw, e := json.Marshal(struct {
		Version string
		Command credential.Command
		Secret  string
	}{"channel-command-v1", c, c.Secret})
	if e != nil {
		return "", credential.ErrUnavailable
	}
	h := hmac.New(sha256.New, p.fingerprintKey)
	_, _ = h.Write(raw)
	return "channel-command-v1:" + hex.EncodeToString(h.Sum(nil)), nil
}

func associatedData(m credential.Metadata) []byte {
	raw, _ := json.Marshal(struct {
		Org, ID, Provider, Channel, Account, Region, Origin string
		Revision                                            int64
	}{m.OrganizationID, m.ConnectionID, m.Provider, m.Channel, m.Account, m.Region, m.BaseURL, m.Revision})
	return raw
}
