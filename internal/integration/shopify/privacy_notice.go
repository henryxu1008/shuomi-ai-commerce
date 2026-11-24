package shopify

import (
	"bytes"
	"encoding/json"
)

const (
	TopicCustomerDataRequest = "customers/data_request"
	TopicCustomerRedact      = "customers/redact"
	TopicShopRedact          = "shop/redact"
)

// VerifyPrivacyNotice returns a signed Shop identity only for shop redaction.
// Buyer requests for the product-only app do not carry personal data past this boundary.
func (c *OAuthClient) VerifyPrivacyNotice(topic, signature string, raw []byte) (string, error) {
	if err := c.verifyNoticeSignature(signature, raw); err != nil {
		return "", err
	}
	var fields map[string]json.RawMessage
	var domain string
	if !decodeAdminJSON(raw, &fields) || json.Unmarshal(fields["shop_domain"], &domain) != nil ||
		!shopDomainPattern.MatchString(domain) {
		return "", ErrInvalidRequest
	}
	shop := "gid://shopify/Shop/" + string(fields["shop_id"])
	if !validGID(shop, "Shop") {
		return "", ErrInvalidRequest
	}
	switch topic {
	case TopicShopRedact:
		if len(fields) == 2 {
			return shop, nil
		}
	case TopicCustomerDataRequest:
		if hasNoticeFieldType(fields["data_request"], '{') && fields["orders_to_redact"] == nil {
			return "", nil
		}
	case TopicCustomerRedact:
		if hasNoticeFieldType(fields["orders_to_redact"], '[') && fields["data_request"] == nil && fields["orders_requested"] == nil {
			return "", nil
		}
	}
	return "", ErrInvalidRequest
}

func hasNoticeFieldType(field json.RawMessage, first byte) bool {
	field = bytes.TrimSpace(field)
	return len(field) > 0 && field[0] == first
}
