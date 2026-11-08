// Package research owns private evidence relationships and human decisions.
// Projects, source evidence, products, model runs and transfers retain their owners.
package research

import (
	"context"
	"errors"
	"task-processor/internal/product/collection"
	"time"
)

var (
	ErrUnsupported = errors.New("SOURCE_NOT_SUPPORTED")
	ErrInvalid     = errors.New("INVALID_REQUEST")
	ErrForbidden   = errors.New("FORBIDDEN")
	ErrNotFound    = errors.New("NOT_FOUND")
	ErrConflict    = errors.New("REVISION_CONFLICT")
	ErrArchived    = errors.New("PROJECT_ARCHIVED")
	ErrUnavailable = errors.New("DEPENDENCY_UNAVAILABLE")
	ErrUnknown     = errors.New("OUTCOME_UNKNOWN")
)

type Selection struct {
	ItemID        string `json:"itemId"`
	Revision      int64  `json:"revision"`
	PublicationID string `json:"publicationId"`
	Version       uint64 `json:"version,string"`
}
type EvidenceRef struct {
	Selection                  Selection         `json:"selection"`
	Source                     collection.Source `json:"source"`
	JobID                      string            `json:"jobId"`
	Checksum                   string            `json:"checksum"`
	QueryRef                   string            `json:"queryRef,omitempty"`
	ObservationVersion         string            `json:"observationVersion,omitempty"`
	ObservationEncodingVersion string            `json:"observationEncodingVersion,omitempty"`
}

func (r EvidenceRef) Digest() string { return collection.Digest(r) }

type QuerySummary struct {
	Mode         string   `json:"mode"`
	Keyword      string   `json:"keyword,omitempty"`
	CategoryNode string   `json:"categoryNode,omitempty"`
	CategoryRef  string   `json:"categoryRef,omitempty"`
	ASINCount    int      `json:"asinCount"`
	Limit        int      `json:"limit"`
	Fields       []string `json:"fields"`
}
type Observation struct {
	Query          QuerySummary    `json:"query"`
	Item           collection.Item `json:"item"`
	Ref            EvidenceRef     `json:"ref"`
	BatchID        string          `json:"batchId"`
	Site           string          `json:"site"`
	ASIN           string          `json:"asin"`
	Title          string          `json:"title"`
	SourceURL      string          `json:"sourceUrl"`
	CapturedAt     string          `json:"capturedAt"`
	ParserVersion  string          `json:"parserVersion"`
	Currency       string          `json:"currency,omitempty"`
	Price          *float64        `json:"price,omitempty"`
	PriceBasis     string          `json:"priceBasis,omitempty"`
	FixtureVersion string          `json:"fixtureVersion,omitempty"`
	Missing        []string        `json:"missing"`
	SHEIN          *SHEINCard      `json:"shein,omitempty"`
}

type SHEINCard struct {
	Market        string          `json:"market"`
	NativeGoodsID string          `json:"nativeGoodsId"`
	Price         *SHEINCardPrice `json:"price"`
}

type SHEINCardPrice struct {
	Amount        string  `json:"amount"`
	Currency      string  `json:"currency"`
	Basis         string  `json:"basis"`
	PromotionNote *string `json:"promotionNote"`
	TaxIncluded   *bool   `json:"taxIncluded"`
	Unit          *string `json:"unit"`
}

type ProductIdentity struct{ Platform, Market, NativeProductID string }

func (o Observation) Identity() ProductIdentity {
	if o.SHEIN != nil {
		return ProductIdentity{"shein", o.SHEIN.Market, o.SHEIN.NativeGoodsID}
	}
	return ProductIdentity{"amazon", o.Site, o.ASIN}
}

func (i ProductIdentity) CandidateID(scope collection.Scope, project string) string {
	if i.Platform == "amazon" {
		return collection.StableID(scope.OrganizationID, scope.ActorID, project, "opportunity", i.Market, i.NativeProductID)
	}
	return collection.StableID(scope.OrganizationID, scope.ActorID, project, "opportunity", i.Platform, i.Market, i.NativeProductID)
}

type Link struct {
	ID      string
	Ref     EvidenceRef
	Removed bool
}
type Decision struct {
	State, Reason, EvidenceDigest string
	Ref                           EvidenceRef
	Revision                      int64
	At                            time.Time
	AssessmentRef                 *AssessmentRef `json:"assessmentRef,omitempty"`
}
type Candidate struct {
	ID, ProjectID, SelectedLinkID, State, Reason string
	Revision                                     int64
	Identity                                     ProductIdentity
	Links                                        []Link
	History                                      []Decision
	ReasonDecision                               *Decision
}
type EvidenceView struct {
	ID          string       `json:"id"`
	Observation *Observation `json:"observation,omitempty"`
}
type DecisionView struct {
	State          string         `json:"state"`
	Reason         string         `json:"reason,omitempty"`
	EvidenceDigest string         `json:"evidenceDigest,omitempty"`
	Revision       int64          `json:"revision"`
	At             time.Time      `json:"at"`
	AssessmentRef  *AssessmentRef `json:"assessmentRef,omitempty"`
}
type CandidateView struct {
	ID             string         `json:"id"`
	State          string         `json:"state"`
	Reason         string         `json:"reason,omitempty"`
	Revision       int64          `json:"revision"`
	SelectedLinkID string         `json:"selectedLinkId,omitempty"`
	EvidenceDigest string         `json:"evidenceDigest,omitempty"`
	Observation    *Observation   `json:"observation,omitempty"`
	Evidence       []EvidenceView `json:"evidence"`
	SupplierReview bool           `json:"supplierReview,omitempty"`
	Assessment     bool           `json:"assessment,omitempty"`
	History        []DecisionView `json:"history"`
}
type Page struct {
	Items      []CandidateView `json:"items"`
	NextCursor string          `json:"nextCursor,omitempty"`
	Incomplete bool            `json:"incomplete"`
}
type Command struct {
	Action           string           `json:"action"`
	Supplier         *SupplierInput   `json:"supplier,omitempty"`
	Assessment       *AssessmentInput `json:"assessment,omitempty"`
	AssessmentRef    *AssessmentRef   `json:"assessmentRef,omitempty"`
	Selections       []Selection      `json:"selections,omitempty"`
	CandidateID      string           `json:"candidateId,omitempty"`
	ExpectedRevision int64            `json:"expectedRevision,omitempty"`
	LinkID           string           `json:"linkId,omitempty"`
	EvidenceDigest   string           `json:"evidenceDigest,omitempty"`
	State            string           `json:"state,omitempty"`
	Reason           string           `json:"reason,omitempty"`
}
type Receipt struct {
	Key           string           `json:"key"`
	CandidateIDs  []string         `json:"candidateIds"`
	Replayed      bool             `json:"replayed"`
	Revisions     map[string]int64 `json:"revisions"`
	AssessmentRef *AssessmentRef   `json:"assessmentRef,omitempty"`
}
type Projects interface {
	Authorize(context.Context, collection.Scope, string, bool) error
	AuthorizeAnalysis(context.Context, collection.Scope, string, bool) error
}
type Sources interface {
	Observe(context.Context, collection.Scope, string, Selection) (Observation, error)
}
type Repository interface {
	Replay(context.Context, collection.Scope, string, string, Command) (Receipt, bool, error)
	Lookup(context.Context, collection.Scope, string, string) (Receipt, error)
	Load(context.Context, collection.Scope, string, string) (Candidate, error)
	ReadAnalysisEvidence(context.Context, collection.Scope, string, string, string) (AnalysisEvidence, error)
	List(context.Context, collection.Scope, string, string) ([]Candidate, string, error)
	Commit(context.Context, collection.Scope, string, string, Command, []Observation, error) (Receipt, error)
}
type Service struct {
	Store            Repository
	Projects         Projects
	Sources          Sources
	SupplierSources  SupplierSources
	SupplierExpiry   func(context.Context) time.Time
	AssessmentExpiry func(context.Context) time.Time
}

func New(store Repository, projects Projects, sources Sources) *Service {
	return &Service{Store: store, Projects: projects, Sources: sources}
}
