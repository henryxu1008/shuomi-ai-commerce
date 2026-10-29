// Package official owns release-published platform guides, separate from enterprise documents.
package official

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxBodyBytes = 64 << 10
const MaxListBytes = 128 << 10
const MaxArticleBytes = 256 << 10

var ErrInvalid = errors.New("OFFICIAL_KNOWLEDGE_INVALID_REQUEST")
var ErrNotFound = errors.New("OFFICIAL_KNOWLEDGE_NOT_FOUND")
var ErrUnavailable = errors.New("OFFICIAL_KNOWLEDGE_UNAVAILABLE")
var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
var revisionPattern = regexp.MustCompile(`^[1-9][0-9]{0,8}$`)

func ValidID(id string) bool             { return idPattern.MatchString(id) }
func ValidRevision(revision string) bool { return revisionPattern.MatchString(revision) }
func Digest(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

type Summary struct {
	ID           string `json:"id"`
	Revision     string `json:"revision"`
	Title        string `json:"title"`
	Summary      string `json:"summary"`
	Category     string `json:"category"`
	UpdatedAt    string `json:"updatedAt"`
	MaintainedBy string `json:"maintainedBy"`
	Digest       string `json:"digest"`
}
type Source struct {
	Title string `json:"title"`
}
type Article struct {
	Summary
	Body    string   `json:"body"`
	Sources []Source `json:"sources"`
}
type Catalog struct {
	entries map[string]Article
	latest  []Summary
}

func validText(s string, limit int) bool {
	return s != "" && len(s) <= limit && utf8.ValidString(s) && !strings.ContainsRune(s, 0)
}
func copyArticle(a Article) Article { a.Sources = append([]Source(nil), a.Sources...); return a }
func NewCatalog(articles []Article) (*Catalog, error) {
	if len(articles) > 100 {
		return nil, ErrInvalid
	}
	c := &Catalog{entries: map[string]Article{}, latest: []Summary{}}
	latest := map[string]Summary{}
	for _, a := range articles {
		if !ValidID(a.ID) || !ValidRevision(a.Revision) || !validText(a.Title, 512) || !validText(a.Summary.Summary, 2048) || !validText(a.Category, 256) || !validText(a.MaintainedBy, 256) || !validText(a.Body, MaxBodyBytes) || a.Digest != Digest(a.Body) || len(a.Sources) == 0 || len(a.Sources) > 10 {
			return nil, ErrInvalid
		}
		date, err := time.Parse("2006-01-02", a.UpdatedAt)
		if err != nil || date.Format("2006-01-02") != a.UpdatedAt {
			return nil, ErrInvalid
		}
		for _, s := range a.Sources {
			if !validText(s.Title, 512) {
				return nil, ErrInvalid
			}
		}
		data, err := json.Marshal(a)
		if err != nil || len(data) > MaxArticleBytes {
			return nil, ErrInvalid
		}
		key := a.ID + ":" + a.Revision
		if _, found := c.entries[key]; found {
			return nil, ErrInvalid
		}
		c.entries[key] = copyArticle(a)
		old := latest[a.ID]
		n, _ := strconv.Atoi(a.Revision)
		previous, _ := strconv.Atoi(old.Revision)
		if n > previous {
			latest[a.ID] = a.Summary
		}
	}
	for _, a := range latest {
		c.latest = append(c.latest, a)
	}
	sort.Slice(c.latest, func(i, j int) bool { return c.latest[i].ID < c.latest[j].ID })
	data, err := json.Marshal(struct {
		Items []Summary `json:"items"`
	}{c.latest})
	if err != nil || len(data) > MaxListBytes {
		return nil, ErrInvalid
	}
	return c, nil
}
func (c *Catalog) List() []Summary { return append([]Summary{}, c.latest...) }
func (c *Catalog) Get(id, revision string) (Article, error) {
	if !ValidID(id) || !ValidRevision(revision) {
		return Article{}, ErrInvalid
	}
	a, found := c.entries[id+":"+revision]
	if !found {
		return Article{}, ErrNotFound
	}
	return copyArticle(a), nil
}

//go:embed content/ai-commerce-guide-v2.txt
var guide string

func NewEmbeddedCatalog() (*Catalog, error) {
	// Canonical LF bytes give the same revision digest on Windows and Linux builds.
	body := strings.ReplaceAll(guide, "\r\n", "\n")
	// Keep the published revision's digest fixed; edits require a new revision.
	const guideV2Digest = "bd3cabcac27e3d0ca8f569afd646a1f2c7416f15e8d8f16e43da636f589ca3ec"
	return NewCatalog([]Article{{Summary: Summary{ID: "ai-commerce-guide", Revision: "2", Title: "AI电商应用指南", Summary: "了解企业知识选择、商品标题建议、Human Review 与显式应用的真实使用路径。", Category: "AI电商应用指南", UpdatedAt: "2026-10-22", MaintainedBy: "硕米", Digest: guideV2Digest}, Body: body, Sources: []Source{
		{Title: "企业知识与引用规则"},
		{Title: "知识选择与版本依据"},
		{Title: "商品标题建议与审核操作"},
	}}})
}
