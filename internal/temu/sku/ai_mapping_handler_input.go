package sku

import (
	"fmt"

	"task-processor/internal/model"
)

type AIBatchInput struct {
	Variants  []*model.Product
	BatchSize int
}

func (input *AIBatchInput) TotalBatches() int {
	if input == nil || input.BatchSize <= 0 || len(input.Variants) == 0 {
		return 0
	}
	return (len(input.Variants) + input.BatchSize - 1) / input.BatchSize
}

func (input *AIBatchInput) BatchVariants(batchIndex int) ([]*model.Product, int, int, bool) {
	if input == nil || input.BatchSize <= 0 || batchIndex < 0 {
		return nil, 0, 0, false
	}

	start := batchIndex * input.BatchSize
	if start >= len(input.Variants) {
		return nil, 0, 0, false
	}

	end := start + input.BatchSize
	if end > len(input.Variants) {
		end = len(input.Variants)
	}

	return input.Variants[start:end], start, end, true
}

func buildAIBatchInput(variants []*model.Product, batchSize int) (*AIBatchInput, error) {
	if batchSize <= 0 {
		return nil, fmt.Errorf("batch size must be greater than 0")
	}

	return &AIBatchInput{
		Variants:  variants,
		BatchSize: batchSize,
	}, nil
}
