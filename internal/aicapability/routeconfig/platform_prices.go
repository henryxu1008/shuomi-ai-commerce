package routeconfig

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"task-processor/internal/aicapability"
	"time"
)

const PlatformPricesCapability = "ai.models.official-prices"
const PlatformPricesOrganization = "platform-ai-models"

func PlatformPricesScope() Scope {
	return Scope{OrganizationID: PlatformPricesOrganization, Capability: PlatformPricesCapability}
}

type OfficialImagePriceScope struct {
	ReferenceEstimate aicapability.ReferenceImageEstimate `json:"referenceEstimate,omitzero"`
	Width             int                                 `json:"width"`
	Height            int                                 `json:"height"`
	Quality           string                              `json:"quality"`
}
type PublishedModelPrice struct {
	Region         string                          `json:"region"`
	ServiceType    string                          `json:"serviceType"`
	Image          *OfficialImagePriceScope        `json:"image,omitempty"`
	ID             string                          `json:"id"`
	Version        string                          `json:"version,omitempty"`
	Provider       string                          `json:"provider"`
	Channel        string                          `json:"channel"`
	Model          string                          `json:"model"`
	Currency       string                          `json:"currency"`
	UsageVersion   string                          `json:"usageVersion"`
	Rates          aicapability.OfficialModelRates `json:"rates"`
	SourceURL      string                          `json:"sourceUrl"`
	VerifiedAt     time.Time                       `json:"verifiedAt"`
	EffectiveFrom  time.Time                       `json:"effectiveFrom"`
	EffectiveUntil time.Time                       `json:"effectiveUntil"`
}
type PublishedFXRate struct {
	Version                  string    `json:"version,omitempty"`
	Currency                 string    `json:"currency"`
	CNYMicrosPerCurrencyUnit int64     `json:"cnyMicrosPerCurrencyUnit"`
	SourceURL                string    `json:"sourceUrl"`
	RateDate                 string    `json:"rateDate"`
	VerifiedAt               time.Time `json:"verifiedAt"`
	EffectiveFrom            time.Time `json:"effectiveFrom"`
	EffectiveUntil           time.Time `json:"effectiveUntil"`
}
type PlatformPricesSnapshot struct {
	SchemaVersion int                   `json:"schemaVersion"`
	UnitVersion   string                `json:"unitVersion"`
	Prices        []PublishedModelPrice `json:"prices"`
	FX            []PublishedFXRate     `json:"fx"`
}
type PlatformPricesInput struct {
	ExpectedRevision string                `json:"expectedRevision"`
	Prices           []PublishedModelPrice `json:"prices"`
	FX               []PublishedFXRate     `json:"fx"`
}

func validPriceInterval(verified, from, until, now time.Time) bool {
	return !verified.IsZero() && !verified.After(now) && !from.IsZero() && until.After(from)
}
func officialPriceHost(raw, provider string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || len(raw) > 2048 {
		return false
	}
	h := strings.ToLower(u.Hostname())
	switch provider {
	case "openai":
		return h == "developers.openai.com" || h == "openai.com"
	case "google":
		return h == "ai.google.dev" || h == "aistudio.google.com" || h == "cloud.google.com"
	case "qwen":
		return h == "help.aliyun.com"
	case "deepseek":
		return h == "api-docs.deepseek.com" || h == "deepseek.com" || h == "www.deepseek.com"
	case "cfets":
		return h == "www.chinamoney.com.cn" || h == "chinamoney.com.cn"
	}
	return false
}
func FreezePlatformPrices(in PlatformPricesInput, revision string, previous *PlatformPricesSnapshot, now time.Time) (PlatformPricesSnapshot, error) {
	out := PlatformPricesSnapshot{SchemaVersion: 1, UnitVersion: aicapability.AIPointMicrounitV1, Prices: append([]PublishedModelPrice{}, in.Prices...), FX: append([]PublishedFXRate{}, in.FX...)}
	if !ValidFact(revision, 96) || len(out.Prices) > 32 || len(out.FX) > 8 || now.IsZero() {
		return out, ErrInvalid
	}
	if previous != nil && (previous.SchemaVersion != 1 || previous.UnitVersion != out.UnitVersion) {
		return out, ErrInvalid
	}
	for i := range out.Prices {
		p := &out.Prices[i]
		if p.Image != nil {
			copy := *p.Image
			p.Image = &copy
			if p.UsageVersion == aicapability.ReferenceImagesTotalsV1 {
				if p.Model != aicapability.FlareImageModel || p.Provider != "openai" || p.Channel != "openai-direct" || p.ServiceType != "image-sync" ||
					p.Image.ReferenceEstimate.Validate() != nil || p.Image.Width != 1024 || p.Image.Height != 1024 || p.Image.Quality != "auto" {
					return out, ErrInvalid
				}
			} else if p.Image.ReferenceEstimate != (aicapability.ReferenceImageEstimate{}) {
				return out, ErrInvalid
			}
		}
		if p.Version != "" || !ValidFact(p.Region, 64) || !ValidFact(p.ServiceType, 64) || !ValidFact(p.ID, 64) || !ValidFact(p.Model, 128) || !ValidFact(p.Channel, 128) || !officialPriceHost(p.SourceURL, p.Provider) || !validPriceInterval(p.VerifiedAt, p.EffectiveFrom, p.EffectiveUntil, now) {
			return out, ErrInvalid
		}
		imagePrice := p.UsageVersion == "qwen-image-count-v1" || strings.Contains(p.UsageVersion, "images")
		if imagePrice != (p.Image != nil) {
			return out, ErrInvalid
		}
		if p.Image != nil && (p.Image.Width < 1 || p.Image.Width > 8192 || p.Image.Height < 1 || p.Image.Height > 8192 || !ValidFact(p.Image.Quality, 64)) {
			return out, ErrInvalid
		}
		// This validates dimensions, not a real FX publication. Only Resolve can
		// construct an executable price with the separately published applicable FX.
		shape := aicapability.OfficialModelTariff{UnitVersion: out.UnitVersion, PriceVersion: "shape", FXVersion: "shape", Currency: p.Currency, CNYMicrosPerCurrencyUnit: 1000000, Model: p.Model, Channel: p.Channel, UsageVersion: p.UsageVersion, Rates: p.Rates}
		if shape.Validate() != nil {
			return out, ErrInvalid
		}
		for j := 0; j < i; j++ {
			q := out.Prices[j]
			if (q.ID == p.ID || q.Model == p.Model && q.Channel == p.Channel) && p.EffectiveFrom.Before(q.EffectiveUntil) && q.EffectiveFrom.Before(p.EffectiveUntil) {
				return out, ErrInvalid
			}
		}
		digest, _ := sharedDigest(*p)
		p.Version = revision + ":" + digest[:16]
		if previous != nil {
			for _, old := range previous.Prices {
				version := old.Version
				old.Version = ""
				oldDigest, _ := sharedDigest(old)
				if oldDigest == digest {
					p.Version = version
					break
				}
			}
		}
	}
	for i := range out.FX {
		f := &out.FX[i]
		day, err := time.Parse("2006-01-02", f.RateDate)
		if err != nil || day.Format("2006-01-02") > f.VerifiedAt.In(time.FixedZone("Asia/Shanghai", 8*3600)).Format("2006-01-02") || f.Version != "" || len(f.Currency) != 3 || f.Currency == "CNY" || !officialPriceHost(f.SourceURL, "cfets") || f.CNYMicrosPerCurrencyUnit <= 0 || f.CNYMicrosPerCurrencyUnit > 1000000000000 || !validPriceInterval(f.VerifiedAt, f.EffectiveFrom, f.EffectiveUntil, now) {
			return out, ErrInvalid
		}
		for _, r := range f.Currency {
			if r < 'A' || r > 'Z' {
				return out, ErrInvalid
			}
		}
		for j := 0; j < i; j++ {
			q := out.FX[j]
			if q.Currency == f.Currency && f.EffectiveFrom.Before(q.EffectiveUntil) && q.EffectiveFrom.Before(f.EffectiveUntil) {
				return out, ErrInvalid
			}
		}
		digest, _ := sharedDigest(*f)
		f.Version = revision + ":" + digest[:16]
		if previous != nil {
			for _, old := range previous.FX {
				version := old.Version
				old.Version = ""
				oldDigest, _ := sharedDigest(old)
				if oldDigest == digest {
					f.Version = version
					break
				}
			}
		}
	}
	return out, nil
}

// Validate verifies stored facts without replacing their frozen versions.
func (s PlatformPricesSnapshot) Validate(now time.Time) error {
	if s.SchemaVersion != 1 || s.UnitVersion != aicapability.AIPointMicrounitV1 {
		return ErrInvalid
	}
	in := PlatformPricesInput{Prices: append([]PublishedModelPrice(nil), s.Prices...), FX: append([]PublishedFXRate(nil), s.FX...)}
	for i := range in.Prices {
		if !ValidFact(in.Prices[i].Version, 128) {
			return ErrInvalid
		}
		in.Prices[i].Version = ""
	}
	for i := range in.FX {
		if !ValidFact(in.FX[i].Version, 128) {
			return ErrInvalid
		}
		in.FX[i].Version = ""
	}
	_, err := FreezePlatformPrices(in, "validation", nil, now)
	return err
}
func (s PlatformPricesSnapshot) Resolve(id, model, channel string, now time.Time) (aicapability.OfficialModelTariff, error) {
	invalid := aicapability.OfficialModelTariff{}
	if s.Validate(now) != nil {
		return invalid, ErrNeedsConfiguration
	}
	var found *PublishedModelPrice
	for i := range s.Prices {
		p := &s.Prices[i]
		if p.ID == id && p.Model == model && p.Channel == channel && !now.Before(p.EffectiveFrom) && now.Before(p.EffectiveUntil) {
			if found != nil {
				return invalid, ErrNeedsConfiguration
			}
			found = p
		}
	}
	if found == nil || !ValidFact(found.Version, 128) {
		return invalid, ErrNeedsConfiguration
	}
	p := *found
	// This model uses the admitted basic-totals estimate, not a supplier clock
	// or a generic cache-partition claim.
	if p.Model == "deepseek-v4.1-flash" && p.UsageVersion != aicapability.ReferenceTextTotalsV1 {
		return invalid, ErrNeedsConfiguration
	}
	result := aicapability.OfficialModelTariff{UnitVersion: s.UnitVersion, PriceVersion: p.Version, Currency: p.Currency, Model: p.Model, Channel: p.Channel, UsageVersion: p.UsageVersion, Rates: p.Rates}
	if p.Currency == "CNY" {
		result.CNYMicrosPerCurrencyUnit = 1000000
		result.FXVersion = "cny-identity-v1"
	} else {
		count := 0
		for _, f := range s.FX {
			if f.Currency == p.Currency && !now.Before(f.EffectiveFrom) && now.Before(f.EffectiveUntil) {
				count++
				result.FXVersion = f.Version
				result.CNYMicrosPerCurrencyUnit = f.CNYMicrosPerCurrencyUnit
			}
		}
		if count != 1 {
			return invalid, ErrNeedsConfiguration
		}
	}
	if result.Validate() != nil {
		return invalid, ErrNeedsConfiguration
	}
	return result, nil
}

type PlatformPricesService struct {
	Repository Repository
	Now        func() time.Time
}

func (s *PlatformPricesService) Current(ctx context.Context) (Revision, error) {
	if s == nil || s.Repository == nil {
		return Revision{}, ErrUnavailable
	}
	r, err := s.Repository.Current(ctx, PlatformPricesScope())
	if err != nil {
		return r, err
	}
	p := r.Payload
	if r.Scope != PlatformPricesScope() || r.SchemaVersion != 1 || p.PlatformPrices == nil || p.SharedModels != nil || p.Text != nil || p.Image != nil || p.Optional || p.BindingID != "" || p.CredentialVersion != "" || p.Rates != (Rates{}) || p.PlatformPrices.SchemaVersion != 1 || p.PlatformPrices.UnitVersion != aicapability.AIPointMicrounitV1 {
		return Revision{}, ErrInvalid
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	if p.PlatformPrices.Validate(now) != nil {
		return Revision{}, ErrInvalid
	}
	return r, nil
}
func (s *PlatformPricesService) Publish(ctx context.Context, actor, key string, input PlatformPricesInput) (Revision, error) {
	if s == nil || s.Repository == nil || ctx == nil || ctx.Err() != nil || !ValidFact(actor, 128) || !ValidFact(key, 128) || !ValidFact(input.ExpectedRevision, 128) {
		return Revision{}, ErrInvalid
	}
	scope := PlatformPricesScope()
	fingerprint, err := sharedDigest(struct {
		Scope Scope
		Actor string
		Input PlatformPricesInput
	}{scope, actor, input})
	if err != nil {
		return Revision{}, err
	}
	// Only build depends on mutable current data. Original receipt replay still
	// takes priority inside Repository.Publish, including after an ACK loss.
	previous, readErr := s.Current(ctx)
	var old *PlatformPricesSnapshot
	if readErr == nil {
		old = previous.Payload.PlatformPrices
	}
	cmd := Command{Scope: scope, ActorID: actor, Key: key, Fingerprint: fingerprint, Input: Input{ExpectedRevision: input.ExpectedRevision}}
	return s.Repository.Publish(ctx, cmd, func(_ CredentialReader, revision string) (Payload, error) {
		if readErr != nil && !errors.Is(readErr, ErrNotFound) {
			return Payload{}, readErr
		}
		expected := previous.ID
		if errors.Is(readErr, ErrNotFound) {
			expected = NoRevision
		}
		if input.ExpectedRevision != expected {
			return Payload{}, ErrConflict
		}
		now := time.Now().UTC()
		if s.Now != nil {
			now = s.Now().UTC()
		}
		snapshot, err := FreezePlatformPrices(input, revision, old, now)
		if err != nil {
			return Payload{}, err
		}
		return Payload{PlatformPrices: &snapshot}, nil
	})
}
