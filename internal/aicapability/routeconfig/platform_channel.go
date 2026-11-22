package routeconfig

import (
	"slices"
	"strings"

	credential "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/channelcredential"
)

type PlatformChannel struct {
	ConnectionID  string   `json:"connectionId"`
	PriceID       string   `json:"priceId"`
	Organizations []string `json:"organizations"`
}

func (r PlatformChannel) Validate(connection string) error {
	if connection != r.ConnectionID || !credential.IsRegisteredConnection(connection) || len(r.Organizations) > 128 ||
		(r.PriceID != "" || len(r.Organizations) > 0) && !ValidFact(r.PriceID, 128) {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, org := range r.Organizations {
		if !ValidFact(org, 128) || org == credential.PlatformOwner || seen[org] {
			return ErrInvalid
		}
		seen[org] = true
	}
	return nil
}

func PlatformConnection(binding string) string {
	if !strings.HasPrefix(binding, "pm-") {
		return ""
	}
	connection := "uc-" + strings.TrimPrefix(binding, "pm-")
	if !credential.IsRegisteredConnection(connection) {
		return ""
	}
	return connection
}

func (r PlatformChannel) Reference() ModelBindingReference {
	digest, _ := sharedDigest(r.PriceID)
	return ModelBindingReference{ID: "pm-" + strings.TrimPrefix(r.ConnectionID, "uc-"), Version: digest}
}

func (r PlatformChannel) Allows(org string, ref ModelBindingReference) bool {
	return ref == r.Reference() && slices.Contains(r.Organizations, org)
}

func PlatformChannelScope(connection string) Scope {
	return Scope{OrganizationID: credential.PlatformOwner, Capability: "ai.models.platform-channel." + connection}
}
