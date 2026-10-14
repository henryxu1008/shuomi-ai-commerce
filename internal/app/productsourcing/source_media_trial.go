package productsourcing

import (
	"net/url"
	"regexp"
	"strconv"
	"task-processor/internal/product/collection"
)

// IsolatedTrialSourceMediaURLPolicy only validates server-created source object
// references. It is not permission to download private URLs or generated images.
type IsolatedTrialSourceMediaURLPolicy struct{ base string }

var sourceTrialBucket = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)
var sourceTrialKey = regexp.MustCompile(`^product-source/[a-f0-9]{64}/[a-f0-9]{64}$`)

func NewIsolatedTrialSourceMediaURLPolicy(base, bucket, endpoint string) (*IsolatedTrialSourceMediaURLPolicy, error) {
	validPort := func(raw string) bool {
		port, err := strconv.Atoi(raw)
		return err == nil && port > 0 && port <= 65535 && strconv.Itoa(port) == raw
	}
	clean := func(u *url.URL, raw string) bool {
		return u != nil && u.User == nil && u.RawPath == "" && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && u.Opaque == "" && u.String() == raw && validPort(u.Port())
	}
	u, err := url.Parse(base)
	if err != nil || !sourceTrialBucket.MatchString(bucket) || !clean(u, base) || u.Scheme != "https" || u.Hostname() != "localhost" || u.Path != "/source-media-assets/"+bucket {
		return nil, collection.ErrUnavailable
	}
	s, err := url.Parse(endpoint)
	if err != nil || !clean(s, endpoint) || (s.Scheme != "http" && s.Scheme != "https") || s.Hostname() != "127.0.0.1" || s.Path != "" {
		return nil, collection.ErrUnavailable
	}
	return &IsolatedTrialSourceMediaURLPolicy{base: base}, nil
}

func (p *IsolatedTrialSourceMediaURLPolicy) ValidateURL(key, raw string) error {
	if p == nil || !sourceTrialKey.MatchString(key) || raw != p.base+"/"+key {
		return collection.ErrUnavailable
	}
	return nil
}
