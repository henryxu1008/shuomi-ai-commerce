package shopify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"
)

const (
	TopicAppUninstalled         = "app/uninstalled"
	TopicAppScopesUpdate        = "app/scopes_update"
	MaxAuthorizationNoticeBytes = 64 << 10
)

var ErrNoticeSignature = errors.New("invalid Shopify notification signature")

type AuthorizationNotice struct {
	ShopID, BodyHash string
	RequiresCheck    bool
}

// VerifyAuthorizationNotice returns only the signed Shop identity and a digest.
// Shop contact details and request headers never identify a local authorization.
func (c *OAuthClient) VerifyAuthorizationNotice(topic, signature string, raw []byte) (AuthorizationNotice, error) {
	if err := c.verifyNoticeSignature(signature, raw); err != nil {
		return AuthorizationNotice{}, err
	}
	var fields map[string]json.RawMessage
	if !decodeAdminJSON(raw, &fields) || !canonicalAuthorizationFields(fields) {
		return AuthorizationNotice{}, ErrInvalidRequest
	}
	var notice AuthorizationNotice
	switch topic {
	case TopicAppUninstalled:
		if fields["shop_id"] != nil || fields["previous"] != nil || fields["current"] != nil {
			return AuthorizationNotice{}, ErrInvalidRequest
		}
		notice.ShopID = "gid://shopify/Shop/" + string(fields["id"])
		if rawGID := fields["admin_graphql_api_id"]; rawGID != nil && string(rawGID) != "null" {
			var gid string
			if json.Unmarshal(rawGID, &gid) != nil || gid != notice.ShopID {
				return AuthorizationNotice{}, ErrInvalidRequest
			}
		}
		notice.RequiresCheck = true
	case TopicAppScopesUpdate:
		var previous, current []string
		if json.Unmarshal(fields["shop_id"], &notice.ShopID) != nil ||
			json.Unmarshal(fields["previous"], &previous) != nil || json.Unmarshal(fields["current"], &current) != nil ||
			!validNoticeScopes(previous) || !validNoticeScopes(current) {
			return AuthorizationNotice{}, ErrInvalidRequest
		}
		notice.RequiresCheck = slices.Contains(previous, productWriteScope) && !slices.Contains(current, productWriteScope)
	default:
		return AuthorizationNotice{}, ErrInvalidRequest
	}
	if !validGID(notice.ShopID, "Shop") {
		return AuthorizationNotice{}, ErrInvalidRequest
	}
	digest := sha256.Sum256(raw)
	notice.BodyHash = hex.EncodeToString(digest[:])
	return notice, nil
}

func (c *OAuthClient) verifyNoticeSignature(signature string, raw []byte) error {
	if c == nil || len(raw) == 0 || len(raw) > MaxAuthorizationNoticeBytes {
		return ErrInvalidRequest
	}
	mac := hmac.New(sha256.New, []byte(c.clientSecret))
	mac.Write(raw)
	decoded, err := base64.StdEncoding.Strict().DecodeString(signature)
	if err != nil || base64.StdEncoding.EncodeToString(decoded) != signature || !hmac.Equal(decoded, mac.Sum(nil)) {
		return ErrNoticeSignature
	}
	return nil
}

func canonicalAuthorizationFields(fields map[string]json.RawMessage) bool {
	for name := range fields {
		switch strings.ToLower(name) {
		case "id", "shop_id", "admin_graphql_api_id", "previous", "current":
			if name != strings.ToLower(name) {
				return false
			}
		}
	}
	return true
}

func validNoticeScopes(scopes []string) bool {
	if scopes == nil || len(scopes) > 256 {
		return false
	}
	seen := make(map[string]bool, len(scopes))
	for _, scope := range scopes {
		if !oauthParameterPattern.MatchString(scope) || seen[scope] {
			return false
		}
		seen[scope] = true
	}
	return true
}
