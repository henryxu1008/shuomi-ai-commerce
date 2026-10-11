package currentapplication

import (
	"errors"
	"net"
	"strconv"
)

type StoreObservationsConfig struct {
	TemporalAddress   string `json:"temporalAddress"`
	TemporalNamespace string `json:"temporalNamespace"`
	PreparationOnly   bool   `json:"preparationOnly"`
}

func (c *Config) validateStoreObservations() error {
	if c.StoreCenter == nil || c.StoreCenter.Observations == nil {
		return nil
	}
	s := c.StoreCenter.Observations
	if !c.StoreCenter.Enabled || c.Identity.TenantDirectoryToken == "" {
		return errors.New("Store observations require current Store, official applications and exact membership authorization")
	}
	if s.PreparationOnly && len(c.StoreCenter.OfficialApplications) != 0 || !s.PreparationOnly && len(c.StoreCenter.OfficialApplications) == 0 {
		return errors.New("Store observations require official applications in full mode and no applications in explicit preparation mode")
	}
	host, port, err := net.SplitHostPort(s.TemporalAddress)
	p, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || p < 1 || p > 65535 || !boundedValue(s.TemporalNamespace, 128) {
		return errors.New("Store observations require literal loopback Temporal address and bounded namespace")
	}
	return nil
}
