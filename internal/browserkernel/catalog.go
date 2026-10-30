// Package browserkernel owns the platform's immutable kernel release metadata.
// It never fetches artifacts or grants permission to start a store browser.
package browserkernel

import (
	"context"
	"errors"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/mod/semver"
	"task-processor/internal/authidentity"
)

var (
	ErrInvalid     = errors.New("invalid kernel request")
	ErrNotFound    = errors.New("kernel release not found")
	ErrConflict    = errors.New("kernel release conflict")
	ErrWithdrawn   = errors.New("kernel release withdrawn")
	ErrUnavailable = errors.New("kernel catalog unavailable")
	strictVersion  = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)
	digest         = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

const MaxArtifactSize int64 = 2 << 30

type Publish struct {
	Version                   string `json:"version"`
	OS                        string `json:"os"`
	Arch                      string `json:"arch"`
	MinClientVersion          string `json:"minClientVersion"`
	MaxClientVersionExclusive string `json:"maxClientVersionExclusive"`
	ArchiveFormat             string `json:"archiveFormat"`
	Size                      int64  `json:"size"`
	SHA256                    string `json:"sha256"`
	DownloadURL               string `json:"downloadUrl"`
}
type Release struct {
	Publish
	ReleaseID               string
	Recommended             bool
	PublishedAt             time.Time
	PublishedBy             string
	WithdrawnAt             *time.Time
	WithdrawnBy             *string
	RecommendationUpdatedAt *time.Time
	RecommendationUpdatedBy *string
}
type Summary struct {
	ReleaseID                 string `json:"releaseId"`
	Version                   string `json:"version"`
	OS                        string `json:"os"`
	Arch                      string `json:"arch"`
	MinClientVersion          string `json:"minClientVersion"`
	MaxClientVersionExclusive string `json:"maxClientVersionExclusive"`
	Recommended               bool   `json:"recommended"`
}
type Artifact struct {
	Summary
	ArchiveFormat   string   `json:"archiveFormat"`
	Size            int64    `json:"size"`
	SHA256          string   `json:"sha256"`
	DownloadURL     string   `json:"downloadUrl"`
	DownloadOrigins []string `json:"downloadOrigins"`
}
type Result struct {
	ReleaseID string `json:"releaseId"`
	State     string `json:"state"`
}
type Page struct {
	Releases   []Summary `json:"releases"`
	NextCursor string    `json:"nextCursor"`
}
type Query struct {
	OS, Arch, ClientVersion, Cursor string
	Limit                           int
}
type Repository interface {
	Candidates(context.Context, string, string, string, int) ([]Release, error)
	Find(context.Context, string) (Release, error)
	Publish(context.Context, string, Publish, string) (Release, error)
	Withdraw(context.Context, string, string) (Release, error)
	Recommend(context.Context, string, bool, string) (Release, error)
}
type Service struct {
	repository Repository
	origins    []string
}

func New(repo Repository, origins []string) (*Service, error) {
	if repo == nil {
		return nil, ErrUnavailable
	}
	if ValidateOrigins(origins) != nil {
		return nil, ErrInvalid
	}
	return &Service{repository: repo, origins: append([]string(nil), origins...)}, nil
}

// ValidateOrigins requires exact canonical origins so native redirect checks use
// the same spelling as the initial URL (including non-default ports).
func ValidateOrigins(origins []string) error {
	if len(origins) < 1 || len(origins) > 8 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, s := range origins {
		u, e := url.Parse(s)
		if e != nil || len(s) > 256 || u.Scheme != "https" || u.Hostname() == "" || u.Opaque != "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(s, "#") || u.Host != strings.ToLower(u.Host) || u.Port() == "443" || strings.HasSuffix(u.Host, ":") || "https://"+u.Host != s || seen[s] {
			return ErrInvalid
		}
		if port := u.Port(); port != "" {
			n, err := strconv.Atoi(port)
			if err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != port {
				return ErrInvalid
			}
		}
		if net.ParseIP(u.Hostname()) == nil {
			for _, label := range strings.Split(u.Hostname(), ".") {
				if len(label) == 0 || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
					return ErrInvalid
				}
				for _, c := range label {
					if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
						return ErrInvalid
					}
				}
			}
		}
		for _, c := range u.Host {
			if c > 127 || unicode.IsSpace(c) || unicode.IsControl(c) {
				return ErrInvalid
			}
		}
		seen[s] = true
	}
	return nil
}
func ValidID(id string) bool { v, e := uuid.Parse(id); return e == nil && v.String() == id }
func validVersion(v string) bool {
	return len(v) <= 64 && strictVersion.MatchString(v) && semver.IsValid("v"+v)
}
func validPlatform(os, arch string) bool {
	return (os == "windows" || os == "linux" || os == "darwin") && (arch == "amd64" || arch == "arm64")
}
func (s *Service) validPublish(p Publish) bool {
	if !validPlatform(p.OS, p.Arch) || !validVersion(p.MinClientVersion) || (p.MaxClientVersionExclusive != "" && (!validVersion(p.MaxClientVersionExclusive) || semver.Compare("v"+p.MaxClientVersionExclusive, "v"+p.MinClientVersion) <= 0)) || p.Size < 1 || p.Size > MaxArtifactSize || !digest.MatchString(p.SHA256) || (p.ArchiveFormat != "zip" && p.ArchiveFormat != "tar.gz" && p.ArchiveFormat != "tar.xz") || p.Version == "" || len(p.Version) > 64 || !utf8.ValidString(p.Version) || strings.TrimSpace(p.Version) != p.Version {
		return false
	}
	for _, c := range p.Version {
		if unicode.IsControl(c) {
			return false
		}
	}
	if len(p.DownloadURL) > 2048 {
		return false
	}
	u, e := url.Parse(p.DownloadURL)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(p.DownloadURL, "#") || u.Path == "" || u.Path == "/" {
		return false
	}
	for _, o := range s.origins {
		if "https://"+u.Host == o {
			return true
		}
	}
	return false
}
func validQuery(q Query) bool {
	return validPlatform(q.OS, q.Arch) && validVersion(q.ClientVersion) && (q.Cursor == "" || ValidID(q.Cursor)) && q.Limit >= 0 && q.Limit <= 100
}
func compatible(r Release, q Query) bool {
	return r.WithdrawnAt == nil && r.OS == q.OS && r.Arch == q.Arch && semver.Compare("v"+q.ClientVersion, "v"+r.MinClientVersion) >= 0 && (r.MaxClientVersionExclusive == "" || semver.Compare("v"+q.ClientVersion, "v"+r.MaxClientVersionExclusive) < 0)
}
func summary(r Release) Summary {
	return Summary{r.ReleaseID, r.Version, r.OS, r.Arch, r.MinClientVersion, r.MaxClientVersionExclusive, r.Recommended}
}
func result(r Release) Result {
	state := "published"
	if r.WithdrawnAt != nil {
		state = "withdrawn"
	}
	return Result{r.ReleaseID, state}
}

// Storage errors are deliberately collapsed; command outcomes may have committed.
func storageError(e error) error {
	for _, v := range []error{ErrInvalid, ErrNotFound, ErrConflict, ErrWithdrawn} {
		if errors.Is(e, v) {
			return v
		}
	}
	if e != nil {
		return ErrUnavailable
	}
	return nil
}
func (s *Service) List(ctx context.Context, q Query) (Page, error) {
	out := Page{Releases: []Summary{}}
	if !validQuery(q) {
		return out, ErrInvalid
	}
	if q.Limit == 0 {
		q.Limit = 50
	}
	rows, e := s.repository.Candidates(ctx, q.OS, q.Arch, q.Cursor, q.Limit+1)
	if e != nil {
		return out, storageError(e)
	}
	if len(rows) > q.Limit+1 {
		return out, ErrUnavailable
	}
	more := len(rows) > q.Limit
	if more {
		rows = rows[:q.Limit]
	}
	last := q.Cursor
	for _, r := range rows {
		if !ValidID(r.ReleaseID) || r.ReleaseID <= last || !s.validPublish(r.Publish) || r.OS != q.OS || r.Arch != q.Arch || r.WithdrawnAt != nil {
			return Page{}, ErrUnavailable
		}
		last = r.ReleaseID
		if compatible(r, q) {
			out.Releases = append(out.Releases, summary(r))
		}
	}
	if more {
		out.NextCursor = last
	}
	return out, nil
}
func (s *Service) Artifact(ctx context.Context, id string, q Query) (Artifact, error) {
	if !ValidID(id) || !validQuery(q) || q.Cursor != "" || q.Limit != 0 {
		return Artifact{}, ErrInvalid
	}
	r, e := s.repository.Find(ctx, id)
	if e != nil {
		return Artifact{}, storageError(e)
	}
	if r.ReleaseID != id || !s.validPublish(r.Publish) {
		return Artifact{}, ErrUnavailable
	}
	if !compatible(r, q) {
		return Artifact{}, ErrNotFound
	}
	return Artifact{summary(r), r.ArchiveFormat, r.Size, r.SHA256, r.DownloadURL, append([]string(nil), s.origins...)}, nil
}
func (s *Service) Publish(ctx context.Context, id string, p Publish, actor string) (Result, error) {
	if !ValidID(id) || !s.validPublish(p) || !authidentity.IsBoundedIdentifier(actor) {
		return Result{}, ErrInvalid
	}
	r, e := s.repository.Publish(ctx, id, p, actor)
	return result(r), storageError(e)
}
func (s *Service) Withdraw(ctx context.Context, id, actor string) (Result, error) {
	if !ValidID(id) || !authidentity.IsBoundedIdentifier(actor) {
		return Result{}, ErrInvalid
	}
	r, e := s.repository.Withdraw(ctx, id, actor)
	return result(r), storageError(e)
}
func (s *Service) Recommend(ctx context.Context, id string, value bool, actor string) (Result, error) {
	if !ValidID(id) || !authidentity.IsBoundedIdentifier(actor) {
		return Result{}, ErrInvalid
	}
	r, e := s.repository.Recommend(ctx, id, value, actor)
	return result(r), storageError(e)
}
