package dataservicehttpapi

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/dataservice"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/ledger/orgresource"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/dataacquisition"
)

// The HTTP adapter consumes owner services/read projections. It owns no schema,
// funds, Product facts or startup lifecycle.
type ReadRepository interface {
	Usage(context.Context, collection.Scope) (dataacquisition.Usage, error)
	KeyQuotas(context.Context, collection.Scope) ([]dataacquisition.KeyQuota, error)
	List(context.Context, collection.Scope, int) ([]dataacquisition.Job, error)
	Cancel(context.Context, collection.Scope, string, string) (dataacquisition.Job, error)
}
type AcquisitionReader interface {
	Read(context.Context, dataacquisition.Principal, string) (dataacquisition.Job, error)
	Results(context.Context, dataacquisition.Principal, string, string, int, dataacquisition.CapturedResultReader) (dataacquisition.ResultPage, error)
}

type Dependencies struct {
	ExecutionEnabled bool
	Reader           AcquisitionReader
	Keys             *dataservice.CredentialService
	Custom           *dataservice.CustomService
	Acquisition      *dataacquisition.Service
	Access           dataservice.Access
	Live             dataacquisition.LiveAccess
	Funding          interface {
		Funding(context.Context, collection.Scope) (orgresource.ResourceFunding, error)
	}
	Provider interface {
		Ready(context.Context) error
		Sites() []dataacquisition.Site
	}
	Results           dataacquisition.CapturedResultReader
	Repository        ReadRepository
	Options           func(context.Context) Options
	TrustedProxyCIDRs []string
}
type handler struct{ Dependencies }
type Options struct {
	Sites             []dataacquisition.Site         `json:"sites"`
	CustomSites       []dataacquisition.Site         `json:"customSites"`
	Fields            []string                       `json:"fields"`
	Price             *dataacquisition.PriceSnapshot `json:"price,omitempty"`
	MaximumRows       int                            `json:"maximumRows"`
	Formats           []string                       `json:"formats"`
	AcquisitionReady  bool                           `json:"acquisitionReady"`
	UnavailableReason string                         `json:"unavailableReason,omitempty"`
}

func ValidateProxyCIDRs(values []string) error { return validateProxyCIDRs(values) }
