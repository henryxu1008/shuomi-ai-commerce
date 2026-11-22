package main

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/app/runtime/currentapplication"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/s3"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/supplymarket"
)

func prepareSupplyMarket(ctx context.Context, cfg *currentapplication.SupplyMarketConfig) (supplymarket.PrivateFileStorage, error) {
	c := cfg.Storage
	client, e := s3.NewKnowledgeClient(s3.ClientConfig{Region: c.Region, Endpoint: c.Endpoint, AccessKeyID: c.AccessKeyID, SecretAccessKey: c.SecretAccessKey, UsePathStyle: true})
	if e != nil {
		return nil, e
	}
	uploader, e := s3.NewUploaderWithOptions(client, s3.UploaderOptions{Bucket: c.Bucket, ArtifactCapabilities: s3.ArtifactStorageCapabilities{Mode: s3.ArtifactStorageMode(c.Mode), COSImmutableNonVersionedBucketPolicy: c.COSImmutableNonVersionedBucketPolicy}})
	if e != nil {
		return nil, e
	}
	return s3.NewSupplyMarketFiles(uploader)
}
