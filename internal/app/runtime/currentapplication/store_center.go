package currentapplication

import (
	"errors"
	"path/filepath"
	"strings"
)

// StoreCenterConfig uses the independently owned native record database.
type StoreCenterConfig struct {
	OperationsCockpit    bool                            `json:"operationsCockpit,omitempty"`
	Observations         *StoreObservationsConfig        `json:"observations,omitempty"`
	Enabled              bool                            `json:"enabled"`
	Database             DatabaseConfig                  `json:"database"`
	OfficialApplications []OfficialStoreConnectionConfig `json:"officialApplications,omitempty"`
	TikTokShop           *TikTokShopConfig               `json:"tiktokShop,omitempty"`
	WalmartUS            *WalmartUSConfig                `json:"walmartUS,omitempty"`
}

func (c *Config) validateStoreCenter() error {
	if c.StoreCenter != nil && c.StoreCenter.OperationsCockpit && !c.StoreCenter.Enabled {
		return errors.New("operations cockpit requires enabled Store center")
	}
	if c.StoreCenter == nil || !c.StoreCenter.Enabled {
		return nil
	}
	s := c.StoreCenter
	if err := s.WalmartUS.validate(); err != nil {
		return err
	}
	if err := s.TikTokShop.validate(); err != nil {
		return err
	}
	if s.WalmartUS != nil {
		type privateApp struct {
			key   string
			files []string
		}
		others := []privateApp{}
		for _, app := range s.OfficialApplications {
			others = append(others, privateApp{app.CredentialKeyID, []string{app.AppSecretFile, app.CredentialKeyFile}})
		}
		if s.TikTokShop != nil {
			others = append(others, privateApp{s.TikTokShop.CredentialKeyID, []string{s.TikTokShop.AppSecretFile, s.TikTokShop.CredentialKeyFile}})
		}
		for _, app := range others {
			if app.key == s.WalmartUS.CredentialKeyID {
				return errors.New("Store application credential key identities must be unique")
			}
			for _, old := range app.files {
				for _, next := range []string{s.WalmartUS.ClientSecretFile, s.WalmartUS.CredentialKeyFile} {
					if strings.EqualFold(filepath.Clean(old), filepath.Clean(next)) {
						return errors.New("Store applications require separate private files")
					}
				}
			}
		}
	}
	if s.TikTokShop != nil {
		for _, app := range s.OfficialApplications {
			if app.CredentialKeyID == s.TikTokShop.CredentialKeyID {
				return errors.New("Store application credential key identities must be unique")
			}
			for _, existing := range []string{app.AppSecretFile, app.CredentialKeyFile} {
				for _, proposed := range []string{s.TikTokShop.AppSecretFile, s.TikTokShop.CredentialKeyFile} {
					if strings.EqualFold(filepath.Clean(existing), filepath.Clean(proposed)) {
						return errors.New("Store applications require separate private files")
					}
				}
			}
		}
	}
	if err := validateOfficialApplications(s.OfficialApplications); err != nil {
		return err
	}
	if err := s.Database.validate("storeCenter.database"); err != nil {
		return err
	}
	if s.Database.User != "store_center_runtime" || s.Database.MaxConnections > 8 {
		return errors.New("store center requires narrow runtime roles and at most 8 connections per pool")
	}
	if c.CommercialOwnerDatabase == nil {
		return errors.New("store service requires its resource owner database")
	}
	others := []*DatabaseConfig{&c.SourceAccountDatabase, c.CommercialOwnerDatabase, c.MoneyOwnerDatabase, c.ProductAcquisitionDatabase}
	if c.Referrals.Enabled {
		others = append(others, &c.Referrals.Database)
	}
	if c.Membership != nil {
		others = append(others, &c.Membership.Database)
	}
	if c.ImageAgent != nil {
		others = append(others, &c.ImageAgent.Database)
	}
	if c.ProductAgent != nil && c.ProductAgent.Enabled {
		others = append(others, &c.ProductAgent.Database, &c.ProductAgent.ReviewDatabase, &c.ProductAgent.AssetDatabase)
	}
	for _, other := range others {
		if other != nil && sameDatabaseTarget(s.Database, *other) {
			return errors.New("store center requires a dedicated record database")
		}
	}
	return nil
}

func sameDatabaseTarget(a, b DatabaseConfig) bool {
	return a.Host == b.Host && a.Port == b.Port && a.Database == b.Database
}
