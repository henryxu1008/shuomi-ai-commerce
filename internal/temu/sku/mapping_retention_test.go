package sku

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"task-processor/internal/ai"
	"task-processor/internal/model"
	"task-processor/internal/prompt"
	temutemplate "task-processor/internal/temu/api/template"
	temucontext "task-processor/internal/temu/context"

	"github.com/sirupsen/logrus"
)

type mappingClientStub struct {
	calls int
	err   error
	req   *ai.ChatCompletionRequest
}

func (c *mappingClientStub) GetDefaultModel() string { return "mapping-test" }
func (c *mappingClientStub) Generate(context.Context, string) (string, error) {
	return "", errors.New("unexpected Generate call")
}
func (c *mappingClientStub) AnalyzeImage(context.Context, string, string) (string, error) {
	return "", errors.New("unexpected image call")
}
func (c *mappingClientStub) CreateChatCompletion(_ context.Context, req *ai.ChatCompletionRequest) (*ai.ChatCompletionResponse, error) {
	c.calls++
	c.req = req
	if c.err != nil {
		return nil, c.err
	}
	return &ai.ChatCompletionResponse{Choices: []ai.ChatCompletionChoice{{
		Message: ai.ChatCompletionMessage{Content: `{"sku_list":[{"asin":"A","spec":[{"spec_id":"blue","spec_name":"Blue","parent_spec_id":"color"}]}]}`},
	}}}, nil
}

func mappingTestProcessor(t *testing.T, client ai.ChatCompleter) *SkuVariantProcessor {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	entry := logrus.NewEntry(logger)
	previous := prompt.GlobalRegistry
	prompt.GlobalRegistry = prompt.NewRegistry(entry)
	t.Cleanup(func() { prompt.GlobalRegistry = previous })
	return NewSkuVariantProcessor(entry, client, NewSkuSpecHandler(entry), nil)
}

func TestRetainedAIMappingRunsWithoutSKUConstruction(t *testing.T) {
	client := &mappingClientStub{}
	processor := mappingTestProcessor(t, client)
	product := &model.Product{Asin: "A", Title: "Example"}
	ctx := &temucontext.TemuTaskContext{}
	ctx.AmazonProduct = product
	ctx.UserInputParentSpecList = []temutemplate.UserInputParentSpec{{ParentSpecID: "color", ParentSpecName: "Color"}}
	response, err := processor.GenerateAISkuMapping(ctx, []*model.Product{product})
	if err != nil {
		t.Fatal(err)
	}
	if client.calls != 1 || client.req.Model != "mapping-test" || response.SkuCount() != 1 {
		t.Fatalf("mapping request/result lost: calls=%d response=%+v", client.calls, response)
	}
	sku, _ := response.FirstSKU()
	if sku.Asin != "A" || len(sku.Spec) != 1 || sku.Spec[0].SpecID != "TEMP_Blue" || sku.Spec[0].ParentSpecName != "Color" {
		t.Fatalf("user-input spec conversion/normalization lost: %+v", sku)
	}
	if err := processor.validateAIMappingSpecs(response); err != nil {
		t.Fatal(err)
	}
}

func TestRetainedAIMappingFailureBoundaries(t *testing.T) {
	client := &mappingClientStub{}
	processor := mappingTestProcessor(t, client)
	if _, err := processor.GenerateAISkuMapping(nil, make([]*model.Product, 101)); err == nil || client.calls != 0 {
		t.Fatal("variant cap must reject before calling the client")
	}
	if _, err := processor.GenerateAISkuMappingSingleBatch(nil, nil); err == nil || client.calls != 0 {
		t.Fatal("missing context must reject before calling the client")
	}
	if _, err := mappingTestProcessor(t, nil).GenerateAISkuMapping(nil, nil); err == nil {
		t.Fatal("missing client must still reject")
	}
	client.err = errors.New("stub failure")
	if _, err := processor.callAIAPI(temucontext.VariantMappingRequest{}); !errors.Is(err, client.err) {
		t.Fatalf("client failure lost: %v", err)
	}
	if err := processor.validateAIMappingSpecs(temucontext.NewAISkuMappingResponse([]temucontext.AIGeneratedSku{{Spec: []temucontext.SpecInfo{{}}}})); err == nil {
		t.Fatal("invalid mapping specs must still reject")
	}
	if err := processor.resolveAIMappingSpecIDs(nil, temucontext.NewEmptyAISkuMappingResponse()); err == nil || !strings.Contains(err.Error(), "temu context is nil") {
		t.Fatalf("spec runtime validation lost: %v", err)
	}
	if err := processor.ensureAIMappingMatchesVariants(temucontext.NewAISkuMappingResponse(make([]temucontext.AIGeneratedSku, 4)), nil); err == nil {
		t.Fatal("excess mapping count must still reject")
	}
}

func TestRetainedMappingBatchesAndSpecValues(t *testing.T) {
	variants := make([]*model.Product, 21)
	for i := range variants {
		variants[i] = &model.Product{Asin: "A"}
	}
	input, err := buildAIBatchInput(variants, 20)
	if err != nil || input.TotalBatches() != 2 {
		t.Fatalf("batch input lost: %v", err)
	}
	batch, start, end, ok := input.BatchVariants(1)
	if !ok || start != 20 || end != 21 || len(batch) != 1 || batch[0] != variants[20] {
		t.Fatal("last batch values lost")
	}
	if _, _, _, ok := input.BatchVariants(2); ok {
		t.Fatal("invalid batch must reject")
	}
	if _, err := buildAIBatchInput(variants, 0); err == nil {
		t.Fatal("invalid batch size must reject")
	}
	source := []temucontext.SpecInfo{{ParentSpecID: "color", SpecID: "blue", SpecName: "Blue", ParentSpecName: "Color", ParentID: "parent"}}
	converted := convertSpecInfos(source)
	if len(converted) != 1 || converted[0].ParentSpecID != "color" || converted[0].SpecID != "blue" || converted[0].SpecName != "Blue" || converted[0].ParentSpecName != "Color" || converted[0].ParentID != "parent" {
		t.Fatalf("spec conversion dropped metadata: %+v", converted)
	}
	processor := mappingTestProcessor(t, nil)
	response := temucontext.NewAISkuMappingResponse([]temucontext.AIGeneratedSku{{Spec: source}, {Spec: append([]temucontext.SpecInfo(nil), source...)}})
	if err := processor.normalizeGeneratedAIMapping(response); err != nil {
		t.Fatal(err)
	}
	response.ForEachSKU(func(sku *temucontext.AIGeneratedSku) {
		if !reflect.DeepEqual(sku.Spec, source) {
			t.Fatalf("retained dimension values changed: %+v", sku.Spec)
		}
	})
}
