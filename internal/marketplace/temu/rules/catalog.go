// Package rules owns versioned official TEMU specification facts and pure
// checks. A documentation snapshot is never merchant authorization.
package rules

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
)

//go:embed data/*.tsv
var officialData embed.FS

var ErrOfficialSnapshotInvalid = errors.New("TEMU official specification snapshot is invalid")

type Opening string

const (
	Unknown    Opening = "unknown"
	Open       Opening = "open"
	Closed     Opening = "closed"
	ComingSoon Opening = "coming_soon"
)

type IssueCode string

const (
	RegionMappingMissing          IssueCode = "region_mapping_missing"
	PartitionMismatch             IssueCode = "original_application_partition_mismatch"
	LocalOpeningUnverified        IssueCode = "local_opening_unverified"
	SpecificationsMissing         IssueCode = "local_specifications_missing"
	SemiSiteUnknown               IssueCode = "semi_site_unknown"
	SemiSiteClosed                IssueCode = "semi_site_closed"
	SitesMissing                  IssueCode = "sites_missing"
	DuplicateSite                 IssueCode = "duplicate_site"
	AssociatedSitesMissing        IssueCode = "associated_sites_missing"
	PanEuropeanEligibilityMissing IssueCode = "pan_european_eligibility_unverified"
)

type Source struct {
	EvidenceID   string `json:"evidenceId"`
	URL          string `json:"url"`
	UpdatedAt    string `json:"updatedAt"`
	ContentHash  string `json:"contentHash"`
	TotalRows    int    `json:"totalRows"`
	IncludedRows int    `json:"includedRows"`
}

type SemiSite struct {
	SiteID            int64   `json:"siteId"`
	Label             string  `json:"label"`
	Opening           Opening `json:"opening"`
	Association       string  `json:"association"`
	AssociatedSiteIDs []int64 `json:"associatedSiteIds"`
}

type Region struct {
	RegionID            int64   `json:"regionId"`
	SiteID              int64   `json:"siteId"`
	Label               string  `json:"label"`
	DeploymentPartition string  `json:"deploymentPartition"`
	Semi                Opening `json:"semi"`
	Local               Opening `json:"local"`
}

type UnitRule struct {
	Unit   string     `json:"unit"`
	Number NumberRule `json:"number"`
}

type LocalSpecification struct {
	SiteID            int64      `json:"siteId"`
	Label             string     `json:"label"`
	DocumentPartition string     `json:"documentPartition"`
	RawLanguages      string     `json:"rawLanguages"`
	Languages         []string   `json:"languages"`
	Weight            UnitRule   `json:"weight"`
	Dimension         UnitRule   `json:"dimension"`
	BaseCurrency      string     `json:"baseCurrency"`
	RetailCurrency    string     `json:"retailCurrency"`
	BasePrice         NumberRule `json:"basePrice"`
	RetailPrice       NumberRule `json:"retailPrice"`
}

type LocalTarget struct {
	ApplicationPartition string             `json:"applicationPartition"`
	Region               Region             `json:"region"`
	Specification        LocalSpecification `json:"specification"`
	SnapshotRevision     string             `json:"snapshotRevision"`
	Issues               []IssueCode        `json:"issues"`
}

type SemiSelection struct {
	SiteIDs                     []int64     `json:"siteIds"`
	RequiresPanEuropeanEvidence bool        `json:"requiresPanEuropeanEvidence"`
	SnapshotRevision            string      `json:"snapshotRevision"`
	Issues                      []IssueCode `json:"issues"`
}

type Catalog struct {
	semi     map[int64]SemiSite
	regions  map[int64]Region
	local    map[int64]LocalSpecification
	sources  []Source
	revision string
}

func NewOfficialCatalog() (*Catalog, error) {
	c := &Catalog{semi: map[int64]SemiSite{}, regions: map[int64]Region{}, local: map[int64]LocalSpecification{}}
	if err := c.readSemi(); err != nil {
		return nil, err
	}
	if err := c.readRegions(); err != nil {
		return nil, err
	}
	if err := c.readLocalSpecifications(); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(c.sources)
	if err != nil {
		return nil, ErrOfficialSnapshotInvalid
	}
	c.revision = contentHash(raw)
	return c, nil
}

func contentHash(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func (c *Catalog) Revision() string  { return c.revision }
func (c *Catalog) Sources() []Source { return slices.Clone(c.sources) }

func (c *Catalog) SemiSite(id int64) (SemiSite, bool) {
	s, ok := c.semi[id]
	s.AssociatedSiteIDs = slices.Clone(s.AssociatedSiteIDs)
	return s, ok
}
func (c *Catalog) SemiSites() []SemiSite {
	result := make([]SemiSite, 0, len(c.semi))
	for id := range c.semi {
		s, _ := c.SemiSite(id)
		result = append(result, s)
	}
	slices.SortFunc(result, func(a, b SemiSite) int { return compareID(a.SiteID, b.SiteID) })
	return result
}
func (c *Catalog) Region(id int64) (Region, bool) { r, ok := c.regions[id]; return r, ok }
func (c *Catalog) Regions() []Region {
	result := make([]Region, 0, len(c.regions))
	for _, r := range c.regions {
		result = append(result, r)
	}
	slices.SortFunc(result, func(a, b Region) int { return compareID(a.RegionID, b.RegionID) })
	return result
}
func (c *Catalog) LocalSpecification(id int64) (LocalSpecification, bool) {
	s, ok := c.local[id]
	s.Languages = slices.Clone(s.Languages)
	return s, ok
}
func (c *Catalog) LocalSpecifications() []LocalSpecification {
	result := make([]LocalSpecification, 0, len(c.local))
	for id := range c.local {
		s, _ := c.LocalSpecification(id)
		result = append(result, s)
	}
	slices.SortFunc(result, func(a, b LocalSpecification) int { return compareID(a.SiteID, b.SiteID) })
	return result
}
func compareID(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// LocalTarget joins exact identifiers without using a country name or a
// specification's historical partition label as a credential route. No
// ready-for-upload or merchant-eligibility conclusion is returned here.
func (c *Catalog) LocalTarget(partition string, regionID int64) LocalTarget {
	view := LocalTarget{ApplicationPartition: partition, SnapshotRevision: c.revision, Issues: []IssueCode{}}
	r, ok := c.Region(regionID)
	if !ok {
		view.Issues = append(view.Issues, RegionMappingMissing)
		return view
	}
	view.Region = r
	if partition != r.DeploymentPartition || partition != "US" && partition != "EU" && partition != "GLOBAL" {
		view.Issues = append(view.Issues, PartitionMismatch)
	}
	if r.Local != Open {
		view.Issues = append(view.Issues, LocalOpeningUnverified)
	}
	view.Specification, ok = c.LocalSpecification(r.SiteID)
	if !ok {
		view.Issues = append(view.Issues, SpecificationsMissing)
	}
	return view
}

func (c *Catalog) SemiSelection(ids []int64) SemiSelection {
	view := SemiSelection{SiteIDs: []int64{}, SnapshotRevision: c.revision, Issues: []IssueCode{}}
	if len(ids) == 0 || len(ids) > len(c.semi) {
		view.Issues = append(view.Issues, SitesMissing)
		return view
	}
	view.SiteIDs = slices.Clone(ids)
	slices.Sort(view.SiteIDs)
	for index, id := range view.SiteIDs {
		if index > 0 && view.SiteIDs[index-1] == id {
			view.Issues = appendUniqueIssue(view.Issues, DuplicateSite)
		}
		s, ok := c.semi[id]
		if !ok {
			view.Issues = appendUniqueIssue(view.Issues, SemiSiteUnknown)
			continue
		}
		if s.Opening != Open {
			view.Issues = appendUniqueIssue(view.Issues, SemiSiteClosed)
		}
		for _, associated := range s.AssociatedSiteIDs {
			if !slices.Contains(view.SiteIDs, associated) {
				view.Issues = appendUniqueIssue(view.Issues, AssociatedSitesMissing)
			}
		}
		if s.Association == "泛欧" {
			view.RequiresPanEuropeanEvidence = true
			view.Issues = appendUniqueIssue(view.Issues, PanEuropeanEligibilityMissing)
		}
	}
	return view
}

func appendUniqueIssue(values []IssueCode, value IssueCode) []IssueCode {
	if !slices.Contains(values, value) {
		values = append(values, value)
	}
	return values
}
