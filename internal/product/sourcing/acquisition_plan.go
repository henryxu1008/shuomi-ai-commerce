package sourcing

import (
	"context"
	"encoding/json"
	"math/big"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DiscoveryProduct = "product"
	DiscoveryPage    = "page"
	DiscoveryKeyword = "keyword"
	MaxPlanInputs    = 20
	MaxPlanProducts  = 200
	MaxPlanBodyBytes = 64 << 10
	RuleMatched      = "matched"
	RuleNotMatched   = "not_matched"
	RuleFactsUnknown = "unknown"
)

var discoveryShopHost = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}\.1688\.com$`)

type AcquisitionRule struct {
	MinPrice string `json:"minPrice,omitempty"`
	MaxPrice string `json:"maxPrice,omitempty"`
	MaxMOQ   int64  `json:"maxMOQ,omitempty"`
}

type RuleEvaluation struct {
	State           string `json:"state"`
	Reason          string `json:"reason,omitempty"`
	StartingPrice   string `json:"startingPrice,omitempty"`
	MinimumQuantity int64  `json:"minimumQuantity,omitempty,string"`
}

func (r AcquisitionRule) Validate() error {
	if r.MaxMOQ < 0 || r.MaxMOQ > 1000000000 {
		return ErrInvalidAcquisition
	}
	for _, value := range []string{r.MinPrice, r.MaxPrice} {
		if value != "" && !acquisitionDecimal.MatchString(value) {
			return ErrInvalidAcquisition
		}
	}
	if r.MinPrice != "" && r.MaxPrice != "" {
		min, _ := new(big.Rat).SetString(r.MinPrice)
		max, _ := new(big.Rat).SetString(r.MaxPrice)
		if min.Cmp(max) > 0 {
			return ErrInvalidAcquisition
		}
	}
	return nil
}

// Evaluate reads only original public tier facts, never mutable Catalog or SKU prices.
func (r AcquisitionRule) Evaluate(facts []AcquisitionPrice) (RuleEvaluation, error) {
	if err := r.Validate(); err != nil {
		return RuleEvaluation{}, err
	}
	result := RuleEvaluation{State: RuleMatched}
	priceNeeded := r.MinPrice != "" || r.MaxPrice != ""
	if !priceNeeded && r.MaxMOQ == 0 {
		return result, nil
	}
	unknown := func(reason string) (RuleEvaluation, error) {
		return RuleEvaluation{State: RuleFactsUnknown, Reason: reason}, nil
	}
	if len(facts) == 0 || len(facts) > 64 {
		return unknown("source_facts_unavailable")
	}
	var lowest *big.Rat
	for _, fact := range facts {
		if _, err := validateAcquisitionPrice(fact); err != nil {
			return unknown("source_facts_invalid")
		}
		if priceNeeded {
			if fact.Currency == nil || *fact.Currency != "CNY" {
				return unknown("source_currency_unavailable")
			}
			amount, ok := new(big.Rat).SetString(fact.Amount)
			if !ok || amount.Sign() < 0 {
				return unknown("source_price_unavailable")
			}
			if lowest == nil || amount.Cmp(lowest) < 0 {
				lowest = amount
				result.StartingPrice = fact.Amount
			}
		}
		if r.MaxMOQ > 0 {
			if fact.MinQuantity == nil {
				return unknown("source_moq_unavailable")
			}
			quantity, err := strconv.ParseInt(*fact.MinQuantity, 10, 64)
			if err != nil || quantity <= 0 {
				return unknown("source_moq_unavailable")
			}
			if result.MinimumQuantity == 0 || quantity < result.MinimumQuantity {
				result.MinimumQuantity = quantity
			}
		}
	}
	if r.MinPrice != "" {
		min, _ := new(big.Rat).SetString(r.MinPrice)
		if lowest.Cmp(min) < 0 {
			result.State = RuleNotMatched
			result.Reason = "price_below_minimum"
		}
	}
	if r.MaxPrice != "" {
		max, _ := new(big.Rat).SetString(r.MaxPrice)
		if lowest.Cmp(max) > 0 {
			result.State = RuleNotMatched
			result.Reason = "price_above_maximum"
		}
	}
	if r.MaxMOQ > 0 && result.MinimumQuantity > r.MaxMOQ {
		result.State = RuleNotMatched
		result.Reason = "moq_above_maximum"
	}
	return result, nil
}

type DiscoveryInput struct {
	Mode    string `json:"mode"`
	URL     string `json:"url"`
	Keyword string `json:"keyword,omitempty"`
}

type DiscoveryEvidence struct {
	SourceURL string   `json:"sourceURL"`
	Products  []string `json:"products"`
	Truncated bool     `json:"truncated"`
}

type PublicProductDiscoverer interface {
	Discover(context.Context, DiscoveryInput) (DiscoveryEvidence, error)
}

func Canonical1688Discovery(mode, input string) (DiscoveryInput, error) {
	input = strings.TrimSpace(input)
	if !utf8.ValidString(input) || strings.ContainsAny(input, "\r\n\x00") {
		return DiscoveryInput{}, ErrInvalidAcquisition
	}
	if mode == DiscoveryKeyword {
		if len(input) == 0 || len(input) > 200 {
			return DiscoveryInput{}, ErrInvalidAcquisition
		}
		q := url.Values{"keywords": {input}}
		return DiscoveryInput{Mode: mode, Keyword: input, URL: "https://s.1688.com/selloffer/offer_search.htm?" + q.Encode()}, nil
	}
	if mode != DiscoveryPage || len(input) > 2048 {
		return DiscoveryInput{}, ErrInvalidAcquisition
	}
	u, err := url.Parse(input)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Opaque != "" || u.Fragment != "" || strings.Contains(input, "#") || u.RawPath != "" || (u.Port() != "" && u.Port() != "443") {
		return DiscoveryInput{}, ErrInvalidAcquisition
	}
	host := strings.ToLower(u.Hostname())
	search := host == "s.1688.com" && u.Path == "/selloffer/offer_search.htm"
	shop := discoveryShopHost.MatchString(host) && u.Path == "/page/offerlist.htm"
	for _, reserved := range []string{"s", "detail", "login", "passport", "www", "mobile", "m", "work", "open", "api", "service", "member", "account"} {
		if host == reserved+".1688.com" {
			shop = false
		}
	}
	if !search && !shop {
		return DiscoveryInput{}, ErrInvalidAcquisition
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return DiscoveryInput{}, ErrInvalidAcquisition
	}
	for key, values := range q {
		if len(values) != 1 || values[0] == "" || (key != "beginPage" && !(search && key == "keywords")) {
			return DiscoveryInput{}, ErrInvalidAcquisition
		}
		if key == "beginPage" {
			n, e := strconv.Atoi(values[0])
			if e != nil || n < 1 || n > 1000 {
				return DiscoveryInput{}, ErrInvalidAcquisition
			}
			q.Set(key, strconv.Itoa(n))
		}
	}
	if search {
		keyword := strings.TrimSpace(q.Get("keywords"))
		if len(keyword) == 0 || len(keyword) > 200 || strings.ContainsAny(keyword, "\r\n\x00") || !utf8.ValidString(keyword) {
			return DiscoveryInput{}, ErrInvalidAcquisition
		}
		q.Set("keywords", keyword)
	}
	u.Host = host
	u.RawQuery = q.Encode()
	u.ForceQuery = false
	return DiscoveryInput{Mode: mode, URL: u.String()}, nil
}

// AllowsDiscoveryNavigation validates every top-level navigation, not just its origin.
func AllowsDiscoveryNavigation(input DiscoveryInput, target string) bool {
	canonical, err := Canonical1688Discovery(DiscoveryPage, target)
	return err == nil && canonical.URL == input.URL
}

type AcquisitionPlanInput struct {
	Mode             string          `json:"mode"`
	Inputs           []string        `json:"inputs"`
	Rule             AcquisitionRule `json:"rule"`
	AutoTransfer     bool            `json:"autoTransfer"`
	TemplateID       string          `json:"templateId,omitempty"`
	TemplateRevision int64           `json:"templateRevision,omitempty"`
}

func NewAcquisitionPlanInput(mode string, inputs []string, rule AcquisitionRule, autoTransfer bool) (AcquisitionPlanInput, error) {
	result := AcquisitionPlanInput{Mode: mode, Rule: rule, AutoTransfer: autoTransfer, Inputs: []string{}}
	if rule.Validate() != nil || len(inputs) == 0 || len(inputs) > MaxPlanInputs {
		return result, ErrInvalidAcquisition
	}
	seen := map[string]bool{}
	for _, raw := range inputs {
		value := ""
		if mode == DiscoveryProduct {
			source, err := Canonical1688Source(strings.TrimSpace(raw))
			if err != nil || strings.ContainsAny(raw, "\r\n\x00") {
				return result, ErrInvalidAcquisition
			}
			value = source.URL
		} else {
			source, err := Canonical1688Discovery(mode, raw)
			if err != nil {
				return result, err
			}
			value = source.URL
			if mode == DiscoveryKeyword {
				value = source.Keyword
			}
		}
		if !seen[value] {
			result.Inputs = append(result.Inputs, value)
			seen[value] = true
		}
	}
	encoded, _ := json.Marshal(result)
	if len(encoded) > MaxPlanBodyBytes {
		return result, ErrSourcePublicationTooLarge
	}
	return result, nil
}

type AcquisitionPlanScope struct {
	OrganizationID string `json:"organizationId"`
	ActorID        string `json:"actorId"`
	MemberID       string `json:"memberId"`
}
type AcquisitionRuleTemplate struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Rule      AcquisitionRule `json:"rule"`
	Revision  int64           `json:"revision"`
	Archived  bool            `json:"archived"`
	CreatedAt time.Time       `json:"createdAt"`
}
type AcquisitionPlanItem struct {
	Source             string          `json:"source"`
	ChildKey           string          `json:"-"`
	OperationID        string          `json:"operationId,omitempty"`
	State              string          `json:"state"`
	FailureCode        string          `json:"failureCode,omitempty"`
	Evaluation         *RuleEvaluation `json:"evaluation,omitempty"`
	CollectionItemID   string          `json:"collectionItemId,omitempty"`
	CollectionBatchID  string          `json:"-"`
	CollectionRevision int64           `json:"-"`
}
type AcquisitionPlan struct {
	ID                string                `json:"id"`
	Key               string                `json:"-"`
	Scope             AcquisitionPlanScope  `json:"-"`
	Input             AcquisitionPlanInput  `json:"input"`
	Revision          int64                 `json:"revision"`
	DiscoveryIndex    int                   `json:"discoveryIndex"`
	Items             []AcquisitionPlanItem `json:"items"`
	State             string                `json:"state"`
	Truncated         bool                  `json:"truncated"`
	FailureCode       string                `json:"failureCode,omitempty"`
	DiscoveryFailures []DiscoveryFailure    `json:"discoveryFailures"`
	TransferKey       string                `json:"-"`
	TransferInputJSON []byte                `json:"-"`
	PreparationIDs    []string              `json:"preparationIds"`
	CreatedAt         time.Time             `json:"createdAt"`
	Fence             int64                 `json:"-"`
	LeaseUntil        time.Time             `json:"-"`
}

type DiscoveryFailure struct {
	InputIndex int    `json:"inputIndex"`
	Code       string `json:"code"`
}

type PlanAuthorizer interface {
	AuthorizePlan(context.Context, bool) (AcquisitionPlanScope, error)
}

type PlanAuthorityCheck func(context.Context) error

type AcquisitionPlanRepository interface {
	Create(context.Context, AcquisitionPlan, PlanAuthorityCheck) (AcquisitionPlan, error)
	ByKey(context.Context, AcquisitionPlanScope, string) (AcquisitionPlan, error)
	Read(context.Context, AcquisitionPlanScope, string) (AcquisitionPlan, error)
	List(context.Context, AcquisitionPlanScope, string) ([]AcquisitionPlan, error)
	Claim(context.Context, AcquisitionPlanScope, string, int64, PlanAuthorityCheck) (AcquisitionPlan, error)
	Save(context.Context, AcquisitionPlan, PlanAuthorityCheck) (AcquisitionPlan, error)
}

type RuleTemplateMutation struct {
	Action           string          `json:"action"`
	ID               string          `json:"id,omitempty"`
	Name             string          `json:"name,omitempty"`
	Rule             AcquisitionRule `json:"rule"`
	ExpectedRevision int64           `json:"expectedRevision,omitempty"`
}

type AcquisitionRuleRepository interface {
	ListRules(context.Context, AcquisitionPlanScope) ([]AcquisitionRuleTemplate, error)
	ReadRule(context.Context, AcquisitionPlanScope, string) (AcquisitionRuleTemplate, error)
	MutateRule(context.Context, AcquisitionPlanScope, string, RuleTemplateMutation, PlanAuthorityCheck) (AcquisitionRuleTemplate, error)
}
