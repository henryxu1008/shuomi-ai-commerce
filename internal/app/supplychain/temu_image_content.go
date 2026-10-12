package supplychainapp

import (
	"context"
	record "task-processor/internal/listing/record/target"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
)

type TemuImageContentReader interface {
	ReadTemuImage(context.Context, temugoods.ImageObservation, string) ([]byte, error)
	ProbeTemuArtifact(context.Context, string) (temugoods.ImageArtifact, error)
}

func (p *PublicImageProbe) ReadTemuImage(ctx context.Context, expected temugoods.ImageObservation, mode string) ([]byte, error) {
	if len(expected.RemoteURLs) != 0 || expected.ResponseHash != "" || !temugoods.ValidImageObservation(expected) {
		return nil, record.ErrNotReady
	}
	actual, content, err := p.readTemuImage(ctx, asset.ApprovedAsset{ID: expected.AssetID, URL: expected.SourceURL}, expected.CatID, expected.Usage, mode)
	if err != nil || collection.Digest(actual) != collection.Digest(expected) {
		clear(content)
		return nil, record.ErrNotReady
	}
	return content, nil
}
func (p *PublicImageProbe) ProbeTemuArtifact(ctx context.Context, url string) (temugoods.ImageArtifact, error) {
	// An official returned URL proves an artifact, never AssetDB approval.
	o, content, err := p.readTemuImage(ctx, asset.ApprovedAsset{ID: "official-artifact", URL: url}, 1, 2, "fully_managed")
	clear(content)
	if err != nil {
		return temugoods.ImageArtifact{}, err
	}
	return temugoods.ImageArtifact{URL: url, ContentHash: o.ContentHash, Width: o.Width, Height: o.Height, Bytes: o.Bytes, MediaType: o.MediaType}, nil
}
