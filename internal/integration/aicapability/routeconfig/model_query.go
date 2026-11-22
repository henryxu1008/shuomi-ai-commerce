package routeconfiguration

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	credential "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/channelcredential"
	d "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/routeconfig"
	governed "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/aicapability/einomodel"
	strictjson "sigs.k8s.io/json"
)

const (
	OpenAIQueryModel        = "gpt-5.4-mini-2026-03-17"
	openAIQueryBase         = "https://api.openai.com/v1"
	modelQueryDeadline      = 5 * time.Second
	modelQueryResponseLimit = 16 << 10
)

var ErrModelQueryForbidden = errors.New("model query authorization changed")

type ModelQueryResult struct {
	Model      string     `json:"model"`
	Status     string     `json:"status"`
	CheckedAt  time.Time  `json:"checkedAt"`
	RetryAfter *time.Time `json:"retryAfter,omitempty"`
}

// ModelQuery uses the configuration catalog and the original credential owner.
// It never invokes an execution policy, price service or ledger.
type ModelQuery struct {
	managed   *ManagedDefinitions
	catalog   *Catalog
	metadata  credential.MetadataReader
	secrets   credential.SecretReader
	transport http.RoundTripper
}

func (q *ModelQuery) WithManagedDefinitions(definitions *ManagedDefinitions) *ModelQuery {
	q.managed = definitions
	return q
}

func NewModelQuery(catalog *Catalog, metadata credential.MetadataReader, secrets credential.SecretReader) *ModelQuery {
	if catalog == nil || metadata == nil || secrets == nil {
		return nil
	}
	transport := governed.NewSingleSendHTTPTransport()
	transport.Proxy = nil
	return &ModelQuery{catalog: catalog, metadata: metadata, secrets: secrets, transport: transport}
}

// registered uses exact text references from trusted settings definitions.
// Every matching scope must agree on the identity of the official connection.
func (q *ModelQuery) registered(ctx context.Context, org, id string, ref d.ModelBindingReference) (Definition, bool) {
	if q == nil || !credential.ValidID(org) || !credential.ValidID(id) || (!d.ValidFact(ref.ID, 64) || !d.ValidFact(ref.Version, 128)) {
		return Definition{}, false
	}
	var binding Definition
	digest := governed.EndpointIdentityDigest(openAIQueryBase)
	definitions := []Definition{}
	if d.IsManagedBindingID(ref.ID) {
		resolved, err := q.managed.read(ctx, org, ref)
		if err != nil || resolved.reason != "" {
			return Definition{}, false
		}
		definitions = resolved.values
	} else {
		for _, scope := range q.catalog.Scopes(org) {
			def, ok := q.catalog.definitions[scope][ref.ID]
			if ok && def.ReferenceVersion == ref.Version && sharedDefinitionCategory(def) == d.ModelCategoryText {
				definitions = append(definitions, def)
			}
		}
	}
	for _, def := range definitions {
		profile := def.Text.ShapeProfile()
		if def.ConnectionID != id || def.Manufacturer != "openai" || def.ConnectionProvider != "openai" ||
			def.Channel != "openai-direct" || profile.ProviderID != "openai" || profile.ModelID != OpenAIQueryModel ||
			profile.EndpointIdentityDigest != digest || (def.ConnectionEndpoint != "" && def.ConnectionEndpoint != openAIQueryBase) ||
			(binding.ConnectionID != "" && binding.Region != def.Region) {
			return Definition{}, false
		}
		binding = def
	}
	return binding, binding.ConnectionID != ""
}

func (q *ModelQuery) Target(connection credential.ManagedConnection) *d.ModelBindingReference {
	for _, model := range connection.Models {
		if model.Category != string(d.ModelCategoryText) || model.Model != OpenAIQueryModel ||
			connection.Channel != "openai-direct" || connection.Provider != "openai" {
			continue
		}
		ref := d.ModelBindingReference{ID: model.ID, Version: model.Version}
		if connection.Registration != nil && connection.Registration.Template == credential.OfficialMiniTemplate &&
			connection.BaseURL == openAIQueryBase && credential.IsRegisteredConnection(connection.ConnectionID) && d.IsManagedBindingID(ref.ID) {
			return &ref
		}
		if !d.IsManagedBindingID(ref.ID) {
			if _, ok := q.registered(context.Background(), connection.OrganizationID, connection.ConnectionID, ref); ok {
				return &ref
			}
		}
	}
	return nil
}

func (q *ModelQuery) Check(ctx context.Context, org, id string, revision int64, ref d.ModelBindingReference, authorized func(context.Context) bool) (ModelQueryResult, error) {
	var zero ModelQueryResult
	if q == nil {
		return zero, credential.ErrUnavailable
	}
	if ctx == nil || ctx.Err() != nil || authorized == nil || !authorized(ctx) {
		return zero, ErrModelQueryForbidden
	}
	binding, ok := q.registered(ctx, org, id, ref)
	if !ok || revision <= 0 {
		return zero, credential.ErrInvalid
	}
	row, err := q.metadata.Metadata(ctx, org, id)
	if err != nil {
		return zero, err
	}
	if row.Revision != revision {
		return zero, credential.ErrConflict
	}
	if row.OrganizationID != org || row.ConnectionID != id || row.Region != binding.Region ||
		row.Provider != binding.ConnectionProvider || row.Channel != binding.Channel || row.BaseURL != openAIQueryBase ||
		row.EndpointDigest != binding.Text.ShapeProfile().EndpointIdentityDigest ||
		row.Reference != row.WithIdentity().Reference || !row.Enabled || !row.KeyPresent {
		return zero, credential.ErrUnavailable
	}
	secret, err := q.secrets.Resolve(ctx, org, id, row.Reference)
	if err != nil || secret.Metadata != row || strings.TrimSpace(secret.APIKey) == "" {
		return zero, credential.ErrConflict
	}
	recheck := func() error {
		if ctx.Err() != nil || !authorized(ctx) {
			return ErrModelQueryForbidden
		}
		fresh, readErr := q.metadata.Metadata(ctx, org, id)
		if readErr != nil {
			return readErr
		}
		if fresh != row {
			return credential.ErrConflict
		}
		if d.IsManagedBindingID(ref.ID) {
			if _, ok := q.registered(ctx, org, id, ref); !ok {
				return credential.ErrConflict
			}
		}
		return ctx.Err()
	}
	if err = recheck(); err != nil {
		return zero, err
	}
	transport := q.transport
	if credential.IsRegisteredConnection(id) {
		transport = governed.NewRegisteredChannelTransport()
	}
	result := q.retrieve(ctx, secret.APIKey, transport)
	if err = recheck(); err != nil {
		return zero, err
	}
	return result, nil
}

func (q *ModelQuery) retrieve(ctx context.Context, key string, transport http.RoundTripper) ModelQueryResult {
	ctx, cancel := context.WithTimeout(ctx, modelQueryDeadline)
	defer cancel()
	// The fixed URL and HTTP/1.1 fresh connection constrain credential egress.
	// RoundTrip makes exactly one attempt and never follows a redirect.
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, openAIQueryBase+"/models/"+OpenAIQueryModel, nil)
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")
	resp, err := transport.RoundTrip(req)
	result := ModelQueryResult{Model: OpenAIQueryModel, Status: "unavailable", CheckedAt: time.Now().UTC()}
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.Status = "timeout"
		}
		return result
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, modelQueryResponseLimit+1))
	result.CheckedAt = time.Now().UTC()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.Status = "timeout"
		return result
	}
	validSize := err == nil && len(raw) <= modelQueryResponseLimit
	var body struct {
		ID     string `json:"id"`
		Object string `json:"object"`
		Error  struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	parsed := false
	if validSize {
		violations, parseErr := strictjson.UnmarshalStrict(raw, &body, strictjson.DisallowDuplicateFields)
		parsed = parseErr == nil && len(violations) == 0
	}
	errorCode := ""
	if parsed {
		errorCode = body.Error.Code
	}

	switch resp.StatusCode {
	case http.StatusOK:
		result.Status = "unconfirmed"
		if parsed && body.ID == OpenAIQueryModel && body.Object == "model" {
			result.Status = "found"
		}
	case http.StatusUnauthorized:
		result.Status = "authentication-rejected"
		if errorCode == "invalid_api_key" {
			result.Status = "invalid-key"
		}
	case http.StatusForbidden:
		result.Status = "rejected"
		switch errorCode {
		case "insufficient_permissions", "missing_scope":
			result.Status = "read-forbidden"
		case "unsupported_country_region_territory":
			result.Status = "region-restricted"
		}
	case http.StatusNotFound:
		result.Status = "not-found"
	case http.StatusTooManyRequests:
		retryAfter := modelQueryRetryAfter(resp.Header.Get("Retry-After"), result.CheckedAt)
		switch {
		case errorCode == "insufficient_quota" || errorCode == "credit_balance_exhausted":
			result.Status = "unavailable"
		case errorCode == "rate_limit_exceeded" || retryAfter != nil:
			result.Status = "rate-limited"
			result.RetryAfter = retryAfter
		default:
			result.Status = "unconfirmed"
		}
	}
	return result
}

func modelQueryRetryAfter(header string, now time.Time) *time.Time {
	secs, err := strconv.ParseUint(header, 10, 31)
	if err == nil && secs > 0 {
		at := now.Add(time.Duration(secs) * time.Second)
		return &at
	}
	if at, err := http.ParseTime(header); err == nil && at.After(now) {
		return &at
	}
	return nil
}
