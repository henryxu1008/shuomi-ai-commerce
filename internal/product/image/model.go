package image

const (
	MaxInlineArtifactBytes          = 32 << 20
	MaxInlineArtifactAggregateBytes = 64 << 20
)

type Role string

const (
	RoleSource          Role = "source"
	RoleSubject         Role = "subject"
	RoleWhiteBackground Role = "white_background"
	RoleScene           Role = "scene"
)

type Asset struct {
	URL           string
	Bytes         []byte
	MediaType     string
	SourceURL     string
	SourceAssetID string
	Role          Role
	Width         int
	Height        int
	Operations    []string
}

type ProductContext struct {
	ProductKey  string
	Title       string
	ProductType string
	Attributes  map[string]string
}

type GenerationMetadata struct {
	Capability      string
	ModelFamily     string
	InvocationID    string
	PromptReference string
	PromptVersion   string
	Values          map[string]string
}

type Candidate struct {
	Asset    Asset
	Metadata GenerationMetadata
}

type UsageQuoteRequest struct {
	Operation        string
	InputFingerprint string
	MaximumOutputs   int64
}

type UsageQuote struct {
	Operation            string
	Provider             string
	RouteReference       string
	Model                string
	CredentialReference  string
	ConfigurationVersion string
	PricingVersion       string
	Fingerprint          string
	MaximumOutputs       int64
	MaximumModelCalls    int64
	// MaximumTokens is a provider-specific conservative upper bound for one
	// invocation. It is not observed usage.
	MaximumTokens       int64
	MaximumCostMicros   int64
	CostUpperBoundKnown bool
}
