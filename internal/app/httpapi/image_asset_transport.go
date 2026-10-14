package httpapi

import (
	"context"
	"fmt"
	"strings"
	imageapp "task-processor/internal/app/imageagent"
	"task-processor/internal/core/config"
	"task-processor/internal/imageagent"
	"task-processor/internal/integration/httpimage"
	s3integration "task-processor/internal/integration/s3"
	"task-processor/internal/product/catalog"
)

func imageAgentDurableAssetPublicURLResolver(cfg *config.Config) imageagent.DurableAssetPublicURLResolver {
	if cfg == nil {
		return nil
	}
	store := cfg.ImageAgent.ArtifactStore
	if !store.Enabled || !strings.EqualFold(strings.TrimSpace(store.Provider), "s3") || strings.TrimSpace(store.PublicBase) == "" || strings.TrimSpace(store.S3.Bucket) == "" {
		return nil
	}
	return imageAgentObjectURLResolver{
		bucket: store.S3.Bucket, publicBase: store.PublicBase,
		endpoint: store.S3.Endpoint, usePathStyle: store.S3.UsePathStyle,
	}
}

type imageAgentObjectURLResolver struct {
	bucket       string
	publicBase   string
	endpoint     string
	usePathStyle bool
}

func (r imageAgentObjectURLResolver) PublicURL(key string) string {
	fallbackBase := s3integration.BuildS3PublicBase(r.endpoint, r.bucket, r.usePathStyle)
	fallbackURL := ""
	if fallbackBase != "" {
		fallbackURL = strings.TrimRight(fallbackBase, "/") + "/" + strings.TrimLeft(strings.TrimSpace(key), "/")
	} else {
		fallbackURL = fmt.Sprintf("https://%s.s3.amazonaws.com/%s", strings.TrimSpace(r.bucket), strings.TrimLeft(strings.TrimSpace(key), "/"))
	}
	return s3integration.ResolveObjectURL(r.publicBase, key, fallbackURL)
}

func newImageSetSourceByteReader() imageapp.SourceByteReader {
	client := httpimage.NewPublicImageHTTPClient()
	return func(ctx context.Context, asset imageagent.AuthorizedAsset, maximum int64) ([]byte, error) {
		return httpimage.Download(ctx, client, asset.URL, maximum)
	}
}

func buildAuthorizedAssetsFromCatalogImages(images []catalog.Image) []imageagent.AuthorizedAsset {
	assets := make([]imageagent.AuthorizedAsset, 0, len(images))
	for _, item := range catalog.IdentifyImages(images) {
		url, err := imageagent.ValidateSafeImageURL(item.URL)
		if err != nil {
			continue
		}
		assetType := imageagent.AuthorizedAssetSource
		if strings.EqualFold(strings.TrimSpace(item.Role), "style") {
			assetType = imageagent.AuthorizedAssetStyle
		}
		assets = append(assets, imageagent.AuthorizedAsset{
			ID: item.ID, Type: assetType,
			URL: url, SourceURL: url, DisplayURL: url, Label: "Product image",
			Width: item.Width, Height: item.Height,
		})
	}
	return assets
}

func selectAuthorizedAssets(assets []imageagent.AuthorizedAsset, selectedSourceID string, selectedStyleIDs []string, styleSelectionProvided bool) ([]imageagent.AuthorizedAsset, error) {
	selectedSourceID = strings.TrimSpace(selectedSourceID)
	selectedStyles := make(map[string]struct{}, len(selectedStyleIDs))
	for _, id := range selectedStyleIDs {
		if id = strings.TrimSpace(id); id != "" {
			selectedStyles[id] = struct{}{}
		}
	}
	out := make([]imageagent.AuthorizedAsset, 0, len(assets))
	foundSource := selectedSourceID == ""
	foundStyles := make(map[string]struct{}, len(selectedStyles))
	for _, item := range assets {
		if item.Type == imageagent.AuthorizedAssetSource {
			if selectedSourceID == "" || item.ID == selectedSourceID {
				out = append(out, item)
				foundSource = true
			}
			continue
		}
		if !styleSelectionProvided {
			out = append(out, item)
			continue
		}
		if _, ok := selectedStyles[item.ID]; ok {
			out = append(out, item)
			foundStyles[item.ID] = struct{}{}
		}
	}
	if !foundSource {
		return nil, fmt.Errorf("%w: unknown source asset %q", imageagent.ErrValidation, selectedSourceID)
	}
	for id := range selectedStyles {
		if _, ok := foundStyles[id]; !ok {
			return nil, fmt.Errorf("%w: unknown style asset %q", imageagent.ErrValidation, id)
		}
	}
	return out, nil
}

func catalogAttributes(attributes []catalog.Attribute) map[string]string {
	if len(attributes) == 0 {
		return nil
	}
	out := make(map[string]string, len(attributes))
	for _, attribute := range attributes {
		if name := strings.TrimSpace(attribute.Name); name != "" {
			out[name] = strings.TrimSpace(attribute.Value)
		}
	}
	return out
}
