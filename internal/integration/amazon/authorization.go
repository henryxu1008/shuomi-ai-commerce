package amazon

import (
	"net/url"
	"strings"

	"github.com/google/uuid"
)

const (
	usSellerCentral = "sellercentral.amazon.com"
	appIDPrefix     = "amzn1.sellerapps.app."
)

func ConsentURL(appID, state string, beta bool) (string, error) {
	if !validAppID(appID) || !visibleASCII(state, 256) {
		return "", ErrInvalid
	}
	query := url.Values{"application_id": {appID}, "state": {state}}
	if beta {
		query.Set("version", "beta")
	}
	target := url.URL{Scheme: "https", Host: usSellerCentral, Path: "/apps/authorize/consent", RawQuery: query.Encode()}
	return target.String(), nil
}

// ContinueAuthorization validates the provider-supplied destination and builds
// the next URL from fixed components. The Store owner validates the caller,
// claims the pending attempt and issues the fresh state before using it.
func ContinueAuthorization(appID, redirectURI, callbackURI, amazonState, state string, beta bool) (string, error) {
	if !validAppID(appID) || !validRedirectURI(redirectURI) ||
		!visibleASCII(amazonState, 8192) || !visibleASCII(state, 256) || len(callbackURI) > 512 {
		return "", ErrInvalid
	}
	path := "/apps/authorize/confirm/" + appID
	callback, err := url.Parse(callbackURI)
	if err != nil || (callback.Host != "amazon.com" && callback.Host != usSellerCentral) ||
		callback.Scheme != "https" || callback.User != nil || callback.Path != path || callback.RawPath != "" ||
		callback.RawQuery != "" || callback.ForceQuery || strings.Contains(callbackURI, "#") {
		return "", ErrInvalid
	}
	query := url.Values{"amazon_state": {amazonState}, "state": {state}, "redirect_uri": {redirectURI}}
	if beta {
		query.Set("version", "beta")
	}
	target := url.URL{Scheme: "https", Host: callback.Host, Path: path, RawQuery: query.Encode()}
	return target.String(), nil
}

func validAppID(value string) bool {
	suffix, found := strings.CutPrefix(value, appIDPrefix)
	if !found || len(suffix) != 36 {
		return false
	}
	parsed, err := uuid.Parse(suffix)
	return err == nil && parsed.String() == suffix
}

func validRedirectURI(value string) bool {
	callback, err := url.Parse(value)
	return err == nil && len(value) <= 1024 && callback.Scheme == "https" && callback.Hostname() != "" &&
		callback.User == nil && callback.Path != "" && callback.RawQuery == "" &&
		!callback.ForceQuery && !strings.Contains(value, "#")
}
