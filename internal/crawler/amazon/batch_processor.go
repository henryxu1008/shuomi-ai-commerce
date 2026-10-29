// Package amazon 提供Amazon批量处理功能
package amazon

import (
	"context"

	amazonresult "task-processor/internal/crawler/amazon/capturedata"
)

type batchRequestProcessor interface {
	ProcessWithContext(ctx context.Context, url, zipcode string) (*amazonresult.Product, error)
}

// BatchProcessor 批量处理器。
// 这里只负责批量请求的顺序编排，实例获取、超时和重建统一交给单请求处理链路。
type BatchProcessor struct{}

// NewBatchProcessor 创建批量处理器
func NewBatchProcessor() *BatchProcessor {
	return &BatchProcessor{}
}

// Process 兼容不传 context 的旧调用方。
func (bp *BatchProcessor) Process(requests []amazonresult.ProductRequest, processor batchRequestProcessor) []amazonresult.ProductResult {
	return bp.ProcessWithContext(context.Background(), requests, processor)
}

// ProcessWithContext 顺序处理批量请求，复用单请求处理链路。
func (bp *BatchProcessor) ProcessWithContext(ctx context.Context, requests []amazonresult.ProductRequest, processor batchRequestProcessor) []amazonresult.ProductResult {
	if len(requests) == 0 {
		return []amazonresult.ProductResult{}
	}

	results := make([]amazonresult.ProductResult, len(requests))
	if processor == nil {
		for i := range requests {
			results[i] = amazonresult.ProductResult{Error: context.Canceled}
		}
		return results
	}

	for i, req := range requests {
		if err := ctx.Err(); err != nil {
			for j := i; j < len(requests); j++ {
				results[j] = amazonresult.ProductResult{Error: err}
			}
			return results
		}

		product, err := processor.ProcessWithContext(ctx, req.URL, req.Zipcode)
		results[i] = amazonresult.ProductResult{
			Product: product,
			Error:   err,
		}
	}

	return results
}
