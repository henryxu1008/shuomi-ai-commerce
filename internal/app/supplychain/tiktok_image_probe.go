package supplychainapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"time"

	"task-processor/internal/integration/httpimage"
	record "task-processor/internal/listing/record/target"
	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/product/asset"
)

func (p *PublicImageProbe) ProbeTikTok(ctx context.Context, approved asset.ApprovedAsset, useCase string) (marketplace.ImageObservation, error) {
	observation, content, err := p.readTikTokImage(ctx, approved, useCase)
	clear(content)
	return observation, err
}
func (p *PublicImageProbe) readTikTokImage(ctx context.Context, approved asset.ApprovedAsset, useCase string) (marketplace.ImageObservation, []byte, error) {
	if p == nil || p.client == nil || ctx == nil || ctx.Err() != nil || approved.ID == "" {
		return marketplace.ImageObservation{}, nil, record.ErrNotReady
	}
	if validated, err := httpimage.ValidatePublicHTTPSURL(approved.URL); err != nil || validated != approved.URL {
		return marketplace.ImageObservation{}, nil, record.ErrNotReady
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	content, err := httpimage.Download(ctx, p.client, approved.URL, 10<<20)
	if err != nil {
		return marketplace.ImageObservation{}, nil, record.ErrNotReady
	}
	media, width, height, err := httpimage.InspectGeneratedArtifact(content)
	if err != nil || width > 10000 || height > 10000 || int64(width)*int64(height) > 20_000_000 {
		clear(content)
		return marketplace.ImageObservation{}, nil, record.ErrNotReady
	}
	hash := sha256.Sum256(content)
	observation := marketplace.ImageObservation{AssetID: approved.ID, UseCase: useCase, SourceURL: approved.URL, ContentHash: hex.EncodeToString(hash[:]), Width: width, Height: height, Bytes: int64(len(content)), MediaType: media}
	if !marketplace.ValidImageObservation(observation) {
		clear(content)
		return marketplace.ImageObservation{}, nil, record.ErrNotReady
	}
	if _, _, err = image.Decode(bytes.NewReader(content)); err != nil || ctx.Err() != nil {
		clear(content)
		return marketplace.ImageObservation{}, nil, record.ErrNotReady
	}
	return observation, content, nil
}
func (p *PublicImageProbe) ReadTikTokContent(ctx context.Context, approved asset.ApprovedAsset, useCase string) (marketplace.ImageObservation, []byte, error) {
	return p.readTikTokImage(ctx, approved, useCase)
}

var _ record.TargetTikTokImageReader = (*PublicImageProbe)(nil)
