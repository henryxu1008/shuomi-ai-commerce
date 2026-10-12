package supplychainapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"net/url"
	"task-processor/internal/integration/httpimage"
	record "task-processor/internal/listing/record/target"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/product/asset"
	"time"
)

func (p *PublicImageProbe) ProbeTemu(ctx context.Context, approved asset.ApprovedAsset, catID, usage int64, mode string) (temugoods.ImageObservation, error) {
	observed, content, err := p.readTemuImage(ctx, approved, catID, usage, mode)
	clear(content)
	return observed, err
}

func (p *PublicImageProbe) readTemuImage(ctx context.Context, approved asset.ApprovedAsset, catID, usage int64, mode string) (temugoods.ImageObservation, []byte, error) {
	var zero temugoods.ImageObservation
	if p == nil || p.client == nil || ctx == nil || approved.ID == "" || catID < 1 || usage < 1 || usage > 7 || mode != "local" && mode != "fully_managed" && mode != "semi_managed" {
		return zero, nil, record.ErrNotReady
	}
	validated, err := httpimage.ValidatePublicHTTPSURL(approved.URL)
	parsed, parseErr := url.Parse(approved.URL)
	if err != nil || parseErr != nil || validated != approved.URL || parsed.User != nil || parsed.Fragment != "" || parsed.Port() != "" && parsed.Port() != "443" {
		return zero, nil, record.ErrNotReady
	}
	maximum := int64(3 << 20)
	if mode != "local" {
		maximum = 2 << 20
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	content, err := httpimage.Download(ctx, p.client, approved.URL, maximum)
	if err != nil {
		return zero, nil, record.ErrNotReady
	}
	media, w, h, err := httpimage.InspectGeneratedArtifact(content)
	if err != nil || media != "image/jpeg" && media != "image/png" || w > 10000 || h > 10000 || int64(w)*int64(h) > 20_000_000 {
		clear(content)
		return zero, nil, record.ErrNotReady
	}
	if _, _, err = image.Decode(bytes.NewReader(content)); err != nil || ctx.Err() != nil {
		clear(content)
		return zero, nil, record.ErrNotReady
	}
	hash := sha256.Sum256(content)
	return temugoods.ImageObservation{AssetID: approved.ID, SourceURL: approved.URL, CatID: catID, Usage: usage, ContentHash: hex.EncodeToString(hash[:]), Width: w, Height: h, Bytes: int64(len(content)), MediaType: media}, content, nil
}

var _ record.TargetTemuImageReader = (*PublicImageProbe)(nil)
