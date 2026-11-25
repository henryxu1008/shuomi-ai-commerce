package shopify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const (
	appendDraftMediaQuery = `mutation AppendDraftMedia($product: ProductUpdateInput!, $media: [CreateMediaInput!]!, $limit: Int!) {
  productUpdate(product: $product, media: $media) {
    product { id status media(first: $limit) { pageInfo { hasNextPage } nodes { id mediaContentType } } }
    userErrors { field message }
  }
}`
	associateDraftMediaQuery = `mutation AssociateDraftMedia($productId: ID!, $variants: [ProductVariantsBulkInput!]!) {
  productVariantsBulkUpdate(productId: $productId, variants: $variants, allowPartialUpdates: false) {
    product { id status }
    productVariants { id }
    userErrors { field message }
  }
}`
	maxStagedResponseBytes = 8192
	stagedBucket           = "shopify-staged-uploads"
)

// ErrDraftMediaRejected identifies a complete, atomic Shopify rejection.
var ErrDraftMediaRejected = errors.New("Shopify rejected the draft media request")

// DraftMediaWrite binds one request to a verified development installation and
// product. The caller still needs the Store authorization and Listing permit.
type DraftMediaWrite struct {
	client       *AdminClient
	installation Installation
	product      string
	token        string
	expiresAt    time.Time
	used         *atomic.Bool
}

func (*DraftMediaWrite) String() string   { return "Shopify media write [redacted]" }
func (*DraftMediaWrite) GoString() string { return "Shopify media write [redacted]" }

func (c *AdminClient) VerifyDraftMediaWrite(ctx context.Context, expected Installation, token, product string) (*DraftMediaWrite, error) {
	if c == nil || !c.developmentStoresOnly || !validGID(product, "Product") || !currencyPattern.MatchString(expected.Currency) {
		return nil, ErrInvalidRequest
	}
	actual, err := c.matchInstallation(ctx, expected, token)
	if err != nil || actual != expected {
		return nil, ErrInstallationUnavailable
	}
	return &DraftMediaWrite{client: c, installation: actual, product: product, token: token,
		expiresAt: time.Now().Add(adminTimeout), used: new(atomic.Bool)}, nil
}

func (w *DraftMediaWrite) take(ctx context.Context) bool {
	return ctx != nil && ctx.Err() == nil && w != nil && w.used != nil && time.Now().Before(w.expiresAt) &&
		w.used.CompareAndSwap(false, true)
}

// StagedDraftMedia contains signed temporary storage fields. It is scoped to a
// product and content digest and can be used only within the current process.
type StagedDraftMedia struct {
	target       stagedImageTarget
	filename     string
	installation Installation
	product      string
	digest       [32]byte
	attempted    *atomic.Bool
	uploaded     *atomic.Bool
}

func (*StagedDraftMedia) String() string   { return "Shopify staged media [redacted]" }
func (*StagedDraftMedia) GoString() string { return "Shopify staged media [redacted]" }

func (w *DraftMediaWrite) Prepare(ctx context.Context, hashes [][32]byte) ([]*StagedDraftMedia, error) {
	if len(hashes) == 0 || len(hashes) > maxDraftImages {
		return nil, ErrInvalidRequest
	}
	seen := make(map[[32]byte]bool, len(hashes))
	prefix := uuid.NewString()
	inputs := make([]stagedImageInput, len(hashes))
	for i, hash := range hashes {
		if hash == ([32]byte{}) || seen[hash] {
			return nil, ErrInvalidRequest
		}
		seen[hash] = true
		inputs[i] = stagedImageInput{Filename: fmt.Sprintf("%s-%d.png", prefix, i+1), MIMEType: stagedMimeType, Method: "POST", Resource: "IMAGE"}
	}
	if !w.take(ctx) {
		return nil, ErrInvalidRequest
	}
	targets, err := w.client.prepareStagedImages(ctx, w.installation, w.token, inputs)
	if err != nil {
		return nil, ErrDraftOutcomeUnknown
	}
	prepared := make([]*StagedDraftMedia, len(targets))
	for i, target := range targets {
		prepared[i] = &StagedDraftMedia{target: target, filename: inputs[i].Filename, installation: w.installation,
			product: w.product, digest: hashes[i], attempted: new(atomic.Bool), uploaded: new(atomic.Bool)}
	}
	return prepared, nil
}

func (w *DraftMediaWrite) Upload(ctx context.Context, image *StagedDraftMedia, content []byte) error {
	client := newStagedUploadClient()
	defer client.CloseIdleConnections()
	return w.upload(ctx, client, image, content)
}

func (w *DraftMediaWrite) upload(ctx context.Context, client *http.Client, image *StagedDraftMedia, content []byte) error {
	if !w.owns(image) || len(content) > MaxStagedDraftImageBytes || !bytes.HasPrefix(content, []byte("\x89PNG\r\n\x1a\n")) ||
		sha256.Sum256(content) != image.digest || !w.take(ctx) || !image.attempted.CompareAndSwap(false, true) {
		return ErrInvalidRequest
	}
	reply, err := image.target.post(ctx, client, image.filename, content)
	if err != nil {
		return ErrDraftOutcomeUnknown
	}
	defer reply.Body.Close()
	if !image.target.completeUpload(reply) {
		return ErrDraftOutcomeUnknown
	}
	image.uploaded.Store(true)
	return nil
}

func (w *DraftMediaWrite) owns(image *StagedDraftMedia) bool {
	return w != nil && image != nil && image.attempted != nil && image.uploaded != nil &&
		image.installation == w.installation && image.product == w.product
}

func (t stagedImageTarget) completeUpload(reply *http.Response) bool {
	want := "204"
	for _, parameter := range t.Parameters {
		if parameter.Name == "success_action_status" {
			want = parameter.Value
		}
	}
	if fmt.Sprint(reply.StatusCode) != want {
		return false
	}
	raw, err := io.ReadAll(io.LimitReader(reply.Body, maxStagedResponseBytes+1))
	if err != nil || len(raw) > maxStagedResponseBytes {
		return false
	}
	if reply.StatusCode != http.StatusCreated {
		return len(raw) == 0
	}
	var result struct {
		XMLName  xml.Name `xml:"PostResponse"`
		Bucket   []string `xml:"Bucket"`
		Key      []string `xml:"Key"`
		Location []string `xml:"Location"`
		ETag     []string `xml:"ETag"`
	}
	decoder := xml.NewDecoder(bytes.NewReader(bytes.TrimSpace(raw)))
	if decoder.Decode(&result) != nil || len(result.Bucket) != 1 || result.Bucket[0] != stagedBucket ||
		len(result.Key) != 1 || result.Key[0] != strings.TrimPrefix(t.ResourceURL, stagedPostURL) ||
		len(result.Location) != 1 || len(result.ETag) != 1 || result.ETag[0] == "" {
		return false
	}
	if _, err := decoder.Token(); err != io.EOF {
		return false
	}
	return result.Location[0] == t.ResourceURL ||
		result.Location[0] == "https://storage.googleapis.com/"+stagedBucket+"/"+result.Key[0]
}

func (w *DraftMediaWrite) Append(ctx context.Context, images []*StagedDraftMedia) ([]string, error) {
	if len(images) == 0 || len(images) > maxDraftImages {
		return nil, ErrInvalidRequest
	}
	type mediaInput struct {
		MediaContentType string `json:"mediaContentType"`
		OriginalSource   string `json:"originalSource"`
	}
	media := make([]mediaInput, 0, len(images))
	seen := make(map[string]bool, len(images))
	for _, image := range images {
		if !w.owns(image) || !image.uploaded.Load() || seen[image.target.ResourceURL] {
			return nil, ErrInvalidRequest
		}
		seen[image.target.ResourceURL] = true
		media = append(media, mediaInput{MediaContentType: "IMAGE", OriginalSource: image.target.ResourceURL})
	}
	if !w.take(ctx) {
		return nil, ErrInvalidRequest
	}
	var data struct {
		Result *struct {
			Product    *draftMediaMutationProduct `json:"product"`
			UserErrors []json.RawMessage          `json:"userErrors"`
		} `json:"productUpdate"`
	}
	variables := map[string]any{"product": map[string]string{"id": w.product}, "media": media, "limit": draftMediaLimit}
	if w.client.query(ctx, w.installation.Domain, w.token, appendDraftMediaQuery, variables, &data) != nil ||
		data.Result == nil || data.Result.UserErrors == nil || len(data.Result.UserErrors) != 0 ||
		!data.Result.Product.matches(w.product) || !data.Result.Product.Media.complete(draftMediaLimit) ||
		len(data.Result.Product.Media.Nodes) < len(images) {
		return nil, ErrDraftOutcomeUnknown
	}
	ids := make([]string, 0, len(data.Result.Product.Media.Nodes))
	clear(seen)
	for _, node := range data.Result.Product.Media.Nodes {
		if !validGID(node.ID, "MediaImage") || node.MediaContentType != "IMAGE" || seen[node.ID] {
			return nil, ErrDraftOutcomeUnknown
		}
		seen[node.ID] = true
		ids = append(ids, node.ID)
	}
	return ids, nil
}

type draftMediaMutationProduct struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Media  draftMediaPage[struct {
		ID               string `json:"id"`
		MediaContentType string `json:"mediaContentType"`
	}] `json:"media"`
}

func (p *draftMediaMutationProduct) matches(product string) bool {
	return p != nil && p.ID == product && p.Status == "DRAFT"
}

type DraftMediaAssociation struct {
	VariantID string `json:"id"`
	MediaID   string `json:"mediaId"`
}

func (w *DraftMediaWrite) Associate(ctx context.Context, changes []DraftMediaAssociation) error {
	if len(changes) == 0 || len(changes) > draftMediaVariantLimit {
		return ErrInvalidRequest
	}
	variants := make(map[string]bool, len(changes))
	for _, change := range changes {
		if !validGID(change.VariantID, "ProductVariant") || !validGID(change.MediaID, "MediaImage") || variants[change.VariantID] {
			return ErrInvalidRequest
		}
		variants[change.VariantID] = true
	}
	if !w.take(ctx) {
		return ErrInvalidRequest
	}
	var data struct {
		Result *struct {
			Product         json.RawMessage `json:"product"`
			ProductVariants json.RawMessage `json:"productVariants"`
			UserErrors      []struct {
				Field   []string `json:"field"`
				Message string   `json:"message"`
			} `json:"userErrors"`
		} `json:"productVariantsBulkUpdate"`
	}
	variables := map[string]any{"productId": w.product, "variants": changes}
	if w.client.query(ctx, w.installation.Domain, w.token, associateDraftMediaQuery, variables, &data) != nil ||
		data.Result == nil || data.Result.UserErrors == nil || len(data.Result.Product) == 0 || len(data.Result.ProductVariants) == 0 {
		return ErrDraftOutcomeUnknown
	}
	var product *draftMediaMutationProduct
	var returned []struct {
		ID string `json:"id"`
	}
	if !decodeAdminJSON(data.Result.Product, &product) || !decodeAdminJSON(data.Result.ProductVariants, &returned) ||
		product != nil && !product.matches(w.product) {
		return ErrDraftOutcomeUnknown
	}
	if len(data.Result.UserErrors) != 0 {
		for _, issue := range data.Result.UserErrors {
			if strings.TrimSpace(issue.Message) == "" {
				return ErrDraftOutcomeUnknown
			}
		}
		if len(returned) == 0 {
			return ErrDraftMediaRejected
		}
		return ErrDraftOutcomeUnknown
	}
	if product == nil || len(returned) != len(changes) {
		return ErrDraftOutcomeUnknown
	}
	for _, variant := range returned {
		if !variants[variant.ID] {
			return ErrDraftOutcomeUnknown
		}
		delete(variants, variant.ID)
	}
	return nil
}
