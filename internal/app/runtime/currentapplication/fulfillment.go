package currentapplication

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	app "task-processor/internal/app/fulfillment"
	"task-processor/internal/authidentity"
	f "task-processor/internal/fulfillment"
	"task-processor/internal/integration/erpnext"
)

type FulfillmentSiteConfig struct {
	Alias          string `json:"alias"`
	Organization   string `json:"organization"`
	Origin         string `json:"origin"`
	APIKey         string `json:"apiKey"`
	APISecretFile  string `json:"apiSecretFile"`
	SupplyMovement string `json:"supplyMovement"`
}
type FulfillmentConfig struct {
	TemporalAddress   string                  `json:"temporalAddress"`
	TemporalNamespace string                  `json:"temporalNamespace"`
	Sites             []FulfillmentSiteConfig `json:"sites"`
}

func (c *Config) validateFulfillment() error {
	if c.Fulfillment == nil {
		return nil
	}
	cfg := c.Fulfillment
	if c.StoreCenter == nil || !c.StoreCenter.Enabled || len(c.StoreCenter.OfficialApplications) == 0 || c.Identity.TenantDirectoryToken == "" || len(cfg.Sites) < 1 || len(cfg.Sites) > 20 {
		return errors.New("fulfillment requires current Store authority, exact membership and isolated ERP sites")
	}
	host, port, err := net.SplitHostPort(cfg.TemporalAddress)
	p, e := strconv.Atoi(port)
	if err != nil || e != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || p < 1 || p > 65535 || !boundedValue(cfg.TemporalNamespace, 128) {
		return errors.New("fulfillment requires bounded current Temporal configuration")
	}
	seen := map[string]bool{}
	for _, site := range cfg.Sites {
		if !authidentity.IsBoundedIdentifier(site.Alias) || !authidentity.IsBoundedIdentifier(site.Organization) || !filepath.IsAbs(site.APISecretFile) || site.SupplyMovement != "owned_transfer" && site.SupplyMovement != "commercial_sale" {
			return errors.New("fulfillment site binding is incomplete")
		}
		if _, err = erpnext.New(erpnext.Config{Origin: site.Origin, Organization: site.Organization, APIKey: site.APIKey, APISecret: "validation-only"}, nil); err != nil {
			return errors.New("fulfillment requires a fixed HTTPS ERP origin")
		}
		for _, key := range []string{"alias:" + site.Alias, "org:" + site.Organization, "origin:" + strings.ToLower(site.Origin), "file:" + strings.ToLower(filepath.Clean(site.APISecretFile))} {
			if seen[key] {
				return errors.New("fulfillment enterprises require separate aliases, sites and private credentials")
			}
			seen[key] = true
		}
	}
	return nil
}
func (c *FulfillmentConfig) prepare(ctx context.Context) (*app.Configuration, error) {
	if c == nil {
		return nil, nil
	}
	result := &app.Configuration{ERPs: map[string]f.ERP{}, Aliases: map[string]string{}, SupplyMovement: map[string]string{}}
	for _, site := range c.Sites {
		secret, err := readOfficialPrivateFile(ctx, site.APISecretFile)
		if err != nil {
			return nil, errors.New("fulfillment ERP credential file is unavailable or not private")
		}
		client, err := erpnext.New(erpnext.Config{Origin: site.Origin, Organization: site.Organization, APIKey: site.APIKey, APISecret: strings.TrimSpace(string(secret))}, nil)
		if err != nil {
			return nil, err
		}
		result.ERPs[site.Organization] = client
		result.Aliases[site.Organization] = site.Alias
		result.SupplyMovement[site.Organization] = site.SupplyMovement
	}
	return result, nil
}
