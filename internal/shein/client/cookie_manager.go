package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"task-processor/internal/core/logger"
	"task-processor/internal/pkg/jsonx"

	"github.com/sirupsen/logrus"
)

type CookieManager struct {
	storeID             int64
	cookieProvider      CookieProvider
	storeConfigProvider StoreConfigProvider
	logger              *logrus.Entry
	resolvedTenantID    int64
}

func NewCookieManager(storeID int64, cookieProvider CookieProvider, storeConfigProvider StoreConfigProvider) *CookieManager {
	return &CookieManager{
		storeID:             storeID,
		cookieProvider:      cookieProvider,
		storeConfigProvider: storeConfigProvider,
		logger:              logger.GetGlobalLogger("SheinCookieManager").WithField("storeID", storeID),
	}
}

func (cm *CookieManager) LoadCookies() ([]*http.Cookie, error) {
	cm.logger.Debug("loading SHEIN cookies from runtime provider")

	if cm.cookieProvider == nil {
		cm.logger.Error("cookie provider is nil")
		return nil, fmt.Errorf("cookie provider is nil")
	}

	cookieStr, tenantID, err := cm.loadCookieJSONFromSource()
	if err != nil {
		cm.logger.WithError(err).Warn("failed to read SHEIN cookie payload from runtime provider")
		return nil, err
	}
	if strings.TrimSpace(cookieStr) == "" {
		cm.logger.Warn("cookie provider returned empty cookie payload")
		return nil, nil
	}

	cm.resolvedTenantID = tenantID
	cookies, parseErr := cm.parseCookieString(cookieStr)
	if parseErr != nil {
		cm.logger.WithError(parseErr).Error("failed to parse SHEIN cookie payload")
		return nil, fmt.Errorf("parse SHEIN cookie payload: %w", parseErr)
	}
	return cookies, nil
}

type CookieData struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HttpOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
}

type cookiePayload struct {
	Cookies []CookieData `json:"cookies"`
}

func (cm *CookieManager) ParseCookieString(cookieStr string) ([]*http.Cookie, error) {
	return cm.parseCookieString(cookieStr)
}

func (cm *CookieManager) parseCookieString(cookieStr string) ([]*http.Cookie, error) {
	if cookieStr == "" {
		return nil, nil
	}

	var cookieDataList []CookieData
	if err := jsonx.UnmarshalString(cookieStr, &cookieDataList, ""); err == nil {
		return buildCookiesFromJSON(cookieDataList), nil
	}

	var payload cookiePayload
	if err := jsonx.UnmarshalString(cookieStr, &payload, ""); err == nil && len(payload.Cookies) > 0 {
		return buildCookiesFromJSON(payload.Cookies), nil
	}

	cookiePairs := strings.Split(cookieStr, ";")
	cookies := make([]*http.Cookie, 0, len(cookiePairs))
	for _, pair := range cookiePairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			cm.logger.Warnf("skip invalid cookie format: %s", pair)
			continue
		}

		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if name == "" {
			continue
		}

		domain := ".shein.com"
		switch {
		case strings.Contains(name, "sso"), strings.Contains(name, "geiwohuo"):
			domain = ".geiwohuo.com"
		}

		cookies = append(cookies, &http.Cookie{
			Name:   name,
			Value:  value,
			Domain: domain,
			Path:   "/",
		})
	}

	return cookies, nil
}

func buildCookiesFromJSON(cookieDataList []CookieData) []*http.Cookie {
	cookies := make([]*http.Cookie, 0, len(cookieDataList))
	for _, cookieData := range cookieDataList {
		cookie := &http.Cookie{
			Name:     cookieData.Name,
			Value:    cookieData.Value,
			Domain:   cookieData.Domain,
			Path:     cookieData.Path,
			HttpOnly: cookieData.HttpOnly,
			Secure:   cookieData.Secure,
		}
		if cookieData.Expires > 0 {
			cookie.Expires = time.Unix(int64(cookieData.Expires), 0)
		}

		switch strings.ToLower(cookieData.SameSite) {
		case "strict":
			cookie.SameSite = http.SameSiteStrictMode
		case "lax":
			cookie.SameSite = http.SameSiteLaxMode
		case "none":
			cookie.SameSite = http.SameSiteNoneMode
		default:
			cookie.SameSite = http.SameSiteDefaultMode
		}

		cookies = append(cookies, cookie)
	}
	return cookies
}

func (cm *CookieManager) RefreshCookies() ([]*http.Cookie, error) {
	cm.logger.Info("refreshing cookies")
	return cm.LoadCookies()
}

func (cm *CookieManager) ForceRefreshCookies() ([]*http.Cookie, error) {
	return nil, fmt.Errorf("SHEIN session expired; explicit authorized login required")
}

func (cm *CookieManager) loadCookieJSONFromSource() (string, int64, error) {
	if cm.cookieProvider != nil {
		for _, storeID := range cm.cookieLookupStoreIDs() {
			result, err := cm.cookieProvider.GetCookie(context.Background(), storeID)
			if err != nil {
				return "", 0, err
			}
			if result != nil && strings.TrimSpace(result.CookieJSON) != "" {
				return result.CookieJSON, result.TenantID, nil
			}
		}
	}
	return "", 0, nil
}

func (cm *CookieManager) cookieLookupStoreIDs() []int64 {
	if cm.storeID > 0 {
		return []int64{cm.storeID}
	}
	return nil
}

func (cm *CookieManager) TestConnection() error {
	cm.logger.Debug("testing store config connection")
	if cm.storeConfigProvider == nil {
		return fmt.Errorf("store config provider is nil")
	}

	if _, err := cm.storeConfigProvider.GetStoreConfig(context.Background(), cm.storeID); err != nil {
		cm.logger.WithError(err).Error("store config connection test failed")
		return fmt.Errorf("store config connection test failed: %w", err)
	}

	cm.logger.Debug("store config connection test succeeded")
	return nil
}

func (cm *CookieManager) GetResolvedTenantID() int64 {
	return cm.resolvedTenantID
}
