package shopify

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

const clientAssertionType = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

type AppTokenIssuer struct {
	http     *http.Client
	clientID string
	keyID    string
	key      *ecdsa.PrivateKey
}

func (*AppTokenIssuer) String() string   { return "Shopify application token issuer [redacted]" }
func (*AppTokenIssuer) GoString() string { return "Shopify application token issuer [redacted]" }

func NewAppTokenIssuer(transport *http.Client, clientID, keyID string, key *ecdsa.PrivateKey) (*AppTokenIssuer, error) {
	if !oauthParameterPattern.MatchString(clientID) || !oauthParameterPattern.MatchString(keyID) ||
		key == nil || key.Curve != elliptic.P256() || key.D == nil || key.D.Sign() <= 0 || key.D.Cmp(key.Params().N) >= 0 {
		return nil, ErrInvalidRequest
	}
	return &AppTokenIssuer{http: boundedClient(transport), clientID: clientID, keyID: keyID, key: key}, nil
}

type assertionClaims struct {
	jwt.RegisteredClaims
	Audience string `json:"aud"`
}

// Issue is called only for a store admitted by Store Center. The resulting
// token must be checked against the exact current installation before use.
func (c *AppTokenIssuer) Issue(ctx context.Context, shop string) (AccessToken, error) {
	if c == nil || ctx == nil || ctx.Err() != nil || !shopDomainPattern.MatchString(shop) {
		return AccessToken{}, ErrInvalidRequest
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return AccessToken{}, ErrTokenUnavailable
	}
	now := time.Now()
	claims := assertionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: c.clientID, Subject: c.clientID, ID: id.String(),
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
		},
		Audience: "https://" + shop + accessTokenPath,
	}
	assertion := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	assertion.Header["kid"] = c.keyID
	signed, err := assertion.SignedString(c.key)
	if err != nil {
		return AccessToken{}, ErrTokenUnavailable
	}
	body := url.Values{
		"grant_type": {"client_credentials"}, "client_id": {c.clientID},
		"client_assertion_type": {clientAssertionType}, "client_assertion": {signed},
	}
	raw, err := postShopify(ctx, c.http, shop, accessTokenPath, "", "application/x-www-form-urlencoded", []byte(body.Encode()), maxTokenResponseBytes)
	if err != nil {
		return AccessToken{}, ErrTokenUnavailable
	}
	var reply struct {
		Token     string `json:"access_token"`
		Scope     string `json:"scope"`
		ExpiresIn int64  `json:"expires_in"`
	}
	if !decodeAdminJSON(raw, &reply) || !validToken(reply.Token) || !writeScope(reply.Scope) ||
		reply.ExpiresIn <= 0 || reply.ExpiresIn > 24*60*60 {
		return AccessToken{}, ErrTokenUnavailable
	}
	return AccessToken{Value: reply.Token, ExpiresAt: now.Add(time.Duration(reply.ExpiresIn) * time.Second)}, nil
}
