package routeconfiguration

import (
	"net/url"
	"strconv"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	governed "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/aicapability/einomodel"
)

// PrivateTrialPrice is deployment input for one short-lived, local fixture.
// It is never a platform price publication or an alternate provider route.
type PrivateTrialPrice struct {
	NotBefore time.Time                        `json:"notBefore"`
	NotAfter  time.Time                        `json:"notAfter"`
	Tariff    aicapability.OfficialModelTariff `json:"tariff"`
}

func (price PrivateTrialPrice) validFor(def Definition) bool {
	if def.Scope.Capability != string(aicapability.OperationProductContentDecision) || def.Text == nil ||
		def.Image != nil || def.ManufacturerModel != "" || price.NotBefore.IsZero() ||
		!price.NotAfter.After(price.NotBefore) || price.NotAfter.Sub(price.NotBefore) > time.Hour {
		return false
	}
	endpoint, err := url.Parse(def.ConnectionEndpoint)
	if err != nil || endpoint.Scheme != "http" || endpoint.Hostname() != "127.0.0.1" || endpoint.User != nil ||
		endpoint.Path != "/v1" || endpoint.RawPath != "" || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return false
	}
	port, err := strconv.Atoi(endpoint.Port())
	if err != nil || port < 1 || port > 65535 {
		return false
	}
	p := def.Text.ShapeProfile()
	tariff := price.Tariff
	rates := tariff.Rates
	return p.AdapterKind == string(governed.AdapterOpenAICompatible) && def.ConnectionProvider == p.ProviderID &&
		p.EndpointIdentityDigest == governed.EndpointIdentityDigest(def.ConnectionEndpoint) &&
		tariff.Validate() == nil && tariff.Currency == "CNY" && tariff.Model == p.ModelID && tariff.WireModel == "" &&
		tariff.Channel == def.Channel && tariff.UsageVersion == "openai-text-v1" && rates.InputText > 0 && rates.OutputText > 0 &&
		rates.InputImage == 0 && rates.CachedInputImage == 0 && rates.OutputImage == 0 && rates.OutputImagePerItem == 0
}
