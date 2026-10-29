package openai

import (
	"context"

	"task-processor/internal/ai"
)

type contextualImageClient struct {
	manager *Manager
	name    string
}

func (c *contextualImageClient) GenerateImage(ctx context.Context, req *ai.ImageGenerateRequest) (*ai.ImageResponse, error) {
	client, err := c.manager.resolveClient(ctx, c.name)
	if err != nil {
		return nil, err
	}
	return client.GenerateImage(ctx, req)
}

func (c *contextualImageClient) EditImage(ctx context.Context, req *ai.ImageEditRequest) (*ai.ImageResponse, error) {
	client, err := c.manager.resolveClient(ctx, c.name)
	if err != nil {
		return nil, err
	}
	return client.EditImage(ctx, req)
}

func (c *contextualImageClient) EditImageWithRoute(ctx context.Context, req *ai.ImageEditRequest, selection ai.ImageRouteSelection) (*ai.ImageResponse, error) {
	client, err := c.manager.resolveClientWithSelection(ctx, c.name, &selection)
	if err != nil {
		return nil, err
	}
	return client.EditImage(ctx, req)
}

func (c *contextualImageClient) GetDefaultModel() string {
	client, err := c.manager.resolveClient(context.Background(), c.name)
	if err != nil {
		return ""
	}
	return client.GetDefaultModel()
}

func (c *contextualImageClient) SupportsAsyncImageGeneration() bool {
	client, err := c.manager.resolveClient(context.Background(), c.name)
	if err != nil {
		return false
	}
	return client.SupportsAsyncImageGeneration()
}

func (c *contextualImageClient) SubmitImageGeneration(ctx context.Context, req *ai.ImageGenerateRequest) (*ai.ImageAsyncSubmitResponse, error) {
	client, err := c.manager.resolveClient(ctx, c.name)
	if err != nil {
		return nil, err
	}
	return client.SubmitImageGeneration(ctx, req)
}

func (c *contextualImageClient) SubmitImageEdit(ctx context.Context, req *ai.ImageEditRequest) (*ai.ImageAsyncSubmitResponse, error) {
	client, err := c.manager.resolveClient(ctx, c.name)
	if err != nil {
		return nil, err
	}
	return client.SubmitImageEdit(ctx, req)
}

func (c *contextualImageClient) QueryImageGeneration(ctx context.Context, jobID string) (*ai.ImageAsyncQueryResponse, error) {
	client, err := c.manager.resolveClient(ctx, c.name)
	if err != nil {
		return nil, err
	}
	return client.QueryImageGeneration(ctx, jobID)
}
