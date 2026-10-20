package supplychainapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"task-processor/internal/integration/httpimage"
	record "task-processor/internal/listing/record/target"
	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/product/asset"
	"time"
)

func (p *PublicImageProbe) ProbeWalmart(ctx context.Context, a asset.ApprovedAsset) (marketplace.ImageObservation, error) {
	var out marketplace.ImageObservation
	if p == nil || p.client == nil || ctx == nil || a.ID == "" {
		return out, record.ErrNotReady
	}
	valid, e := httpimage.ValidatePublicHTTPSURL(a.URL)
	u, pe := url.Parse(a.URL)
	if e != nil || pe != nil || valid != a.URL || u.User != nil || u.Fragment != "" || u.Port() != "" && u.Port() != "443" || u.RawQuery != "" {
		return out, record.ErrNotReady
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	content, e := httpimage.Download(ctx, p.client, a.URL, 5<<20)
	if e != nil {
		return out, record.ErrNotReady
	}
	defer clear(content)
	media, w, h, e := httpimage.InspectGeneratedArtifact(content)
	if e != nil || media != "image/jpeg" && media != "image/png" || w < 1 || h < 1 || w > 10000 || h > 10000 || int64(w)*int64(h) > 20_000_000 {
		return out, record.ErrNotReady
	}
	hash := sha256.Sum256(content)
	return marketplace.ImageObservation{AssetID: a.ID, URL: a.URL, ContentHash: hex.EncodeToString(hash[:]), Width: w, Height: h, Bytes: int64(len(content)), MediaType: strings.ToLower(media)}, nil
}
