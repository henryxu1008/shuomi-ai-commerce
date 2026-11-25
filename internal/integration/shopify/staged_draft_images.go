package shopify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/httpimage"
)

const (
	stagedHost               = "shopify-staged-uploads.storage.googleapis.com"
	stagedPostURL            = "https://" + stagedHost + "/"
	stagedMimeType           = "image/png"
	MaxStagedDraftImageBytes = 3 << 20
	stagedUploadsQuery       = `mutation StageShopifyDraftImages($input: [StagedUploadInput!]!) {
  stagedUploadsCreate(input: $input) {
    stagedTargets { url resourceUrl parameters { name value } }
    userErrors { field message }
  }
}`
)

type stagedImageInput struct {
	Filename string `json:"filename"`
	MIMEType string `json:"mimeType"`
	Method   string `json:"httpMethod"`
	Resource string `json:"resource"`
}

type stagedImageTarget struct {
	URL         string `json:"url"`
	ResourceURL string `json:"resourceUrl"`
	Parameters  []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"parameters"`
}

// StageImages uploads approved PNG bytes to temporary storage for development
// stores. It cannot create products or media entries, and it does not grant
// authorization to send a draft after staging.
func (w *DraftWrite) StageImages(ctx context.Context, contents [][]byte) ([]string, error) {
	if ctx == nil || ctx.Err() != nil || w == nil || w.client == nil || !w.client.developmentStoresOnly ||
		!time.Now().Before(w.expiresAt) || len(contents) == 0 || len(contents) > maxDraftImages {
		return nil, ErrInvalidRequest
	}
	ctx, cancel := context.WithTimeout(ctx, 22*time.Second)
	defer cancel()

	prefix := uuid.NewString()
	inputs := make([]stagedImageInput, len(contents))
	for i, content := range contents {
		if len(content) > MaxStagedDraftImageBytes || !bytes.HasPrefix(content, []byte("\x89PNG\r\n\x1a\n")) {
			return nil, ErrInvalidRequest
		}
		inputs[i] = stagedImageInput{Filename: fmt.Sprintf("%s-%d.png", prefix, i+1), MIMEType: stagedMimeType, Method: "POST", Resource: "PRODUCT_IMAGE"}
	}
	var data struct {
		Result *struct {
			Targets    []stagedImageTarget `json:"stagedTargets"`
			UserErrors []json.RawMessage   `json:"userErrors"`
		} `json:"stagedUploadsCreate"`
	}
	if w.client.query(ctx, w.installation.Domain, w.token, stagedUploadsQuery, map[string][]stagedImageInput{"input": inputs}, &data) != nil ||
		data.Result == nil || data.Result.UserErrors == nil || len(data.Result.UserErrors) != 0 || len(data.Result.Targets) != len(contents) {
		return nil, errAdminResponse
	}

	resources := make([]string, len(contents))
	for i, target := range data.Result.Targets {
		if !target.valid(inputs[i].Filename) {
			return nil, errAdminResponse
		}
		resources[i] = target.ResourceURL
	}
	client := httpimage.NewPublicImageHTTPClient()
	client.Timeout = 10 * time.Second
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	defer client.CloseIdleConnections()
	group, uploadContext := errgroup.WithContext(ctx)
	group.SetLimit(3)
	for i, target := range data.Result.Targets {
		group.Go(func() error {
			return target.upload(uploadContext, client, inputs[i].Filename, contents[i])
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return resources, nil
}

func stagedResource(value string) (*url.URL, bool) {
	u, err := url.Parse(value)
	if err != nil || len(value) > 2048 || u.Scheme != "https" || u.Host != stagedHost || u.User != nil ||
		u.ForceQuery || u.RawQuery != "" || u.Fragment != "" || u.EscapedPath() != u.Path ||
		!strings.HasPrefix(u.Path, "/tmp/") || path.Clean(u.Path) != u.Path {
		return nil, false
	}
	return u, true
}

func (t stagedImageTarget) valid(filename string) bool {
	resource, ok := stagedResource(t.ResourceURL)
	if !ok || (t.URL != stagedPostURL && t.URL != "https://"+stagedHost) ||
		path.Base(resource.Path) != filename || len(t.Parameters) < 7 || len(t.Parameters) > 9 {
		return false
	}
	seen := make(map[string]string, len(t.Parameters))
	for _, parameter := range t.Parameters {
		switch parameter.Name {
		case "Content-Type", "success_action_status", "acl", "key", "x-goog-date", "x-goog-credential", "x-goog-algorithm", "x-goog-signature", "policy":
		default:
			return false
		}
		if seen[parameter.Name] != "" || parameter.Value == "" || len(parameter.Value) > 8192 || strings.ContainsAny(parameter.Value, "\x00\r\n") {
			return false
		}
		seen[parameter.Name] = parameter.Value
	}
	return seen["Content-Type"] == stagedMimeType &&
		seen["key"] == strings.TrimPrefix(resource.Path, "/") &&
		seen["x-goog-algorithm"] == "GOOG4-RSA-SHA256" &&
		seen["x-goog-date"] != "" && seen["x-goog-credential"] != "" && seen["x-goog-signature"] != "" && seen["policy"] != "" &&
		(seen["acl"] == "" || seen["acl"] == "private" || seen["acl"] == "public-read") &&
		(seen["success_action_status"] == "" || seen["success_action_status"] == "200" ||
			seen["success_action_status"] == "201" || seen["success_action_status"] == "204")
}

func (t stagedImageTarget) upload(ctx context.Context, client *http.Client, filename string, contents []byte) error {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for _, parameter := range t.Parameters {
		if form.WriteField(parameter.Name, parameter.Value) != nil {
			return errAdminResponse
		}
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "file", "filename": filename}))
	header.Set("Content-Type", stagedMimeType)
	file, err := form.CreatePart(header)
	if err != nil {
		return errAdminResponse
	}
	if _, err := file.Write(contents); err != nil || form.Close() != nil {
		return errAdminResponse
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, stagedPostURL, &body)
	if err != nil {
		return errAdminResponse
	}
	request.GetBody = nil
	request.Header.Set("Content-Type", form.FormDataContentType())
	response, err := client.Do(request)
	if err != nil {
		return errAdminResponse
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return nil
	default:
		return errAdminResponse
	}
}
