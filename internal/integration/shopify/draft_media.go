package shopify

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/httpimage"
)

const (
	draftOriginalHost      = "shopify-shop-assets.storage.googleapis.com"
	draftMediaLimit        = 60
	draftMediaVariantLimit = 30
	draftMediaProofLimit   = 30
)

const inspectDraftMediaQuery = `query InspectDraftMedia($id: ID!, $mediaLimit: Int!, $variantLimit: Int!) {
  product(id: $id) {
    id status title
    media(first: $mediaLimit) {
      pageInfo { hasNextPage }
      nodes { id status mediaContentType ... on MediaImage { originalSource { url fileSize } } }
    }
    variants(first: $variantLimit) {
      pageInfo { hasNextPage }
      nodes {
        id sku selectedOptions { name value }
        media(first: 10) { pageInfo { hasNextPage } nodes { id } }
      }
    }
  }
}`

type DraftMediaFingerprint struct {
	Size int64
	Hash [32]byte
}

type DraftMediaState string

const (
	DraftMediaPresent    DraftMediaState = "present"
	DraftMediaMissing    DraftMediaState = "missing"
	DraftMediaUnverified DraftMediaState = "unverified"
)

type DraftMediaMatch struct {
	State   DraftMediaState
	MediaID string
}

type DraftMediaOption struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DraftMediaVariant struct {
	ID       string
	SKU      string
	Options  []DraftMediaOption
	MediaIDs []string
}

type DraftMediaObservation struct {
	ID           string
	Title        string
	GalleryCount int
	Variants     []DraftMediaVariant
	// Images are ordered as the supplied fingerprints.
	Images []DraftMediaMatch
}

// ParseDraftMediaProduct turns a selected shop's own admin link or a numeric
// product ID into the only identifier that can reach the GraphQL client.
func ParseDraftMediaProduct(input, verifiedDomain string) (string, error) {
	if len(input) > 512 || !shopDomainPattern.MatchString(verifiedDomain) {
		return "", ErrInvalidRequest
	}
	number := strings.TrimSpace(input)
	if strings.HasPrefix(number, "https://") {
		parsed, err := url.Parse(number)
		if err != nil || parsed.User != nil || parsed.RawPath != "" {
			return "", ErrInvalidRequest
		}
		var prefix string
		switch parsed.Host {
		case verifiedDomain:
			prefix = "/admin/products/"
		case "admin.shopify.com":
			prefix = "/store/" + strings.TrimSuffix(verifiedDomain, ".myshopify.com") + "/products/"
		default:
			return "", ErrInvalidRequest
		}
		var ok bool
		if number, ok = strings.CutPrefix(parsed.Path, prefix); !ok {
			return "", ErrInvalidRequest
		}
	}
	id := "gid://shopify/Product/" + number
	if !validGID(id, "Product") {
		return "", ErrInvalidRequest
	}
	return id, nil
}

type draftMediaPage[T any] struct {
	PageInfo struct {
		HasNextPage *bool `json:"hasNextPage"`
	} `json:"pageInfo"`
	Nodes []T `json:"nodes"`
}

func (p draftMediaPage[T]) complete(limit int) bool {
	return p.PageInfo.HasNextPage != nil && !*p.PageInfo.HasNextPage && p.Nodes != nil && len(p.Nodes) <= limit
}

type draftGalleryMedia struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	MediaContentType string `json:"mediaContentType"`
	OriginalSource   struct {
		URL      string      `json:"url"`
		FileSize json.Number `json:"fileSize"`
	} `json:"originalSource"`
}

type remoteMediaVariant struct {
	ID              string             `json:"id"`
	SKU             string             `json:"sku"`
	SelectedOptions []DraftMediaOption `json:"selectedOptions"`
	Media           draftMediaPage[struct {
		ID string `json:"id"`
	}] `json:"media"`
}

type remoteDraftMedia struct {
	ID       string                             `json:"id"`
	Status   string                             `json:"status"`
	Title    string                             `json:"title"`
	Media    draftMediaPage[draftGalleryMedia]  `json:"media"`
	Variants draftMediaPage[remoteMediaVariant] `json:"variants"`
}

// InspectDraftMedia reads gallery membership and original byte evidence without
// exposing Shopify's temporary URLs or the application token to callers.
func (c *AdminClient) InspectDraftMedia(ctx context.Context, expected Installation, token, productID string, proofs []DraftMediaFingerprint) (DraftMediaObservation, error) {
	if c == nil || !c.developmentStoresOnly {
		return DraftMediaObservation{}, ErrInvalidRequest
	}
	images := httpimage.NewPublicImageHTTPClient()
	images.Timeout = 11 * time.Second
	images.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return c.inspectDraftMedia(ctx, expected, token, productID, proofs, images)
}

func (c *AdminClient) inspectDraftMedia(ctx context.Context, expected Installation, token, productID string, proofs []DraftMediaFingerprint, images *http.Client) (DraftMediaObservation, error) {
	if ctx == nil || !validGID(productID, "Product") || len(proofs) > draftMediaProofLimit {
		return DraftMediaObservation{}, ErrInvalidRequest
	}
	for _, proof := range proofs {
		if proof.Size < 1 || proof.Size > MaxStagedDraftImageBytes || proof.Hash == ([32]byte{}) {
			return DraftMediaObservation{}, ErrInvalidRequest
		}
	}
	if _, err := c.matchInstallation(ctx, expected, token); err != nil {
		return DraftMediaObservation{}, err
	}
	var data struct {
		Product *remoteDraftMedia `json:"product"`
	}
	variables := map[string]any{"id": productID, "mediaLimit": draftMediaLimit, "variantLimit": draftMediaVariantLimit}
	if c.query(ctx, expected.Domain, token, inspectDraftMediaQuery, variables, &data) != nil {
		return DraftMediaObservation{}, ErrDraftUnverified
	}
	product := data.Product
	if product == nil || product.ID != productID || product.Status != "DRAFT" || product.Title == "" ||
		!product.Media.complete(draftMediaLimit) || !product.Variants.complete(draftMediaVariantLimit) || len(product.Variants.Nodes) == 0 {
		return DraftMediaObservation{}, ErrDraftUnverified
	}
	observation, ok := product.observation()
	if !ok {
		return DraftMediaObservation{}, ErrDraftUnverified
	}
	observation.Images = matchDraftOriginals(ctx, images, product.Media.Nodes, proofs)
	return observation, nil
}

func (p *remoteDraftMedia) observation() (DraftMediaObservation, bool) {
	result := DraftMediaObservation{ID: p.ID, Title: p.Title, GalleryCount: len(p.Media.Nodes)}
	gallery := make(map[string]bool, len(p.Media.Nodes))
	for _, media := range p.Media.Nodes {
		var kind string
		switch media.MediaContentType {
		case "IMAGE":
			kind = "MediaImage"
		case "VIDEO":
			kind = "Video"
		case "EXTERNAL_VIDEO":
			kind = "ExternalVideo"
		case "MODEL_3D":
			kind = "Model3d"
		}
		if kind == "" || !validGID(media.ID, kind) || gallery[media.ID] {
			return DraftMediaObservation{}, false
		}
		gallery[media.ID] = true
	}
	seen := make(map[string]bool, len(p.Variants.Nodes))
	for _, v := range p.Variants.Nodes {
		if !validGID(v.ID, "ProductVariant") || seen[v.ID] || !v.Media.complete(10) || v.SelectedOptions == nil {
			return DraftMediaObservation{}, false
		}
		seen[v.ID] = true
		resultVariant := DraftMediaVariant{ID: v.ID, SKU: v.SKU, Options: v.SelectedOptions, MediaIDs: make([]string, 0, len(v.Media.Nodes))}
		for _, media := range v.Media.Nodes {
			if !gallery[media.ID] {
				return DraftMediaObservation{}, false
			}
			resultVariant.MediaIDs = append(resultVariant.MediaIDs, media.ID)
		}
		result.Variants = append(result.Variants, resultVariant)
	}
	return result, true
}

type draftOriginal struct {
	mediaID string
	size    int64
	hash    [32]byte
}

func matchDraftOriginals(ctx context.Context, client *http.Client, gallery []draftGalleryMedia, requested []DraftMediaFingerprint) []DraftMediaMatch {
	matches := make([]DraftMediaMatch, len(requested))
	if len(requested) == 0 {
		return matches
	}
	sizes := make(map[int64]bool, len(requested))
	for _, proof := range requested {
		sizes[proof.Size] = true
	}
	candidates := make([]draftGalleryMedia, 0, len(gallery))
	for _, media := range gallery {
		size, _ := media.OriginalSource.FileSize.Int64()
		if media.MediaContentType == "IMAGE" && (size <= 0 || sizes[size]) {
			candidates = append(candidates, media)
		}
	}
	originals := make([]draftOriginal, len(candidates))
	for i, candidate := range candidates {
		size, _ := candidate.OriginalSource.FileSize.Int64()
		originals[i] = draftOriginal{mediaID: candidate.ID, size: size}
	}
	if client != nil && len(candidates) <= draftMediaProofLimit {
		ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
		defer cancel()
		var downloads sync.WaitGroup
		slots := make(chan struct{}, 4)
		for i, media := range candidates {
			if media.Status != "READY" || !validDraftOriginalURL(media.OriginalSource.URL) {
				continue
			}
			downloads.Add(1)
			go func() {
				defer downloads.Done()
				select {
				case slots <- struct{}{}:
					defer func() { <-slots }()
				case <-ctx.Done():
					return
				}
				raw, err := httpimage.Download(ctx, client, media.OriginalSource.URL, MaxStagedDraftImageBytes)
				if err == nil && len(raw) > 0 && (originals[i].size <= 0 || originals[i].size == int64(len(raw))) {
					originals[i].size = int64(len(raw))
					originals[i].hash = sha256.Sum256(raw)
				}
			}()
		}
		downloads.Wait()
	}
	for i, proof := range requested {
		matches[i] = matchDraftOriginal(proof, originals)
	}
	return matches
}

func matchDraftOriginal(proof DraftMediaFingerprint, originals []draftOriginal) DraftMediaMatch {
	var match string
	for _, original := range originals {
		if original.size > 0 && original.size != proof.Size {
			continue
		}
		if original.hash == ([32]byte{}) {
			return DraftMediaMatch{State: DraftMediaUnverified}
		}
		if original.hash == proof.Hash {
			if match != "" {
				return DraftMediaMatch{State: DraftMediaUnverified}
			}
			match = original.mediaID
		}
	}
	if match == "" {
		return DraftMediaMatch{State: DraftMediaMissing}
	}
	return DraftMediaMatch{State: DraftMediaPresent, MediaID: match}
}

func validDraftOriginalURL(raw string) bool {
	if len(raw) > 4096 {
		return false
	}
	parsed, err := url.ParseRequestURI(raw)
	return err == nil && parsed.Scheme == "https" && parsed.Host == draftOriginalHost &&
		parsed.User == nil && parsed.Fragment == "" && parsed.Path != ""
}
