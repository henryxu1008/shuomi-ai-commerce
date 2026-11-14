package einoplanner

import (
	"context"
	"strings"
	"task-processor/internal/aiworkbench"
	rp "task-processor/internal/product/review/proposal"
	"testing"
	"time"
)

func TestV4FitsExistingPlanningBudgetWithProductAndHistory(t *testing.T) {
	p := testProfile()
	p.PromptVersion = ProductContextPromptVersion
	p.MaximumPromptTokens = 4096
	p.MaximumCompletionTokens = 512
	s := aiworkbench.Scope{OrganizationID: "111111111111111111", ActorID: "222222222222222222"}
	r := aiworkbench.ScopeReader{}
	f, e := r.Freeze(context.Background(), s, "22222222-2222-4222-8222-222222222222", "222222222222222222", aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"})
	if e != nil {
		t.Fatal(e)
	}
	sub := &aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: "11111111-1111-4111-8111-111111111111", CatalogVersion: "1", PublicationID: "own-33333333-3333-4333-8333-333333333333"}, Capability: aiworkbench.ContentTaskKind, Parameters: rp.ContentSelection{TargetLanguage: "en-US", Description: true}}
	history := []aiworkbench.Message{}
	for i, content := range []string{"测试上下文", "测试回复", strings.Repeat("合成资料整理需求", 7), strings.Repeat("Synthetic planning response. ", 8), strings.Repeat("请整理已选商品描述", 7)} {
		author := aiworkbench.AuthorUser
		if i%2 == 1 {
			author = aiworkbench.AuthorAssistant
		}
		history = append(history, aiworkbench.Message{ID: "11111111-1111-4111-8111-111111111111", ConversationID: f.ConversationID, Sequence: uint64(i + 1), Author: author, Content: content, CreatedAt: time.Now()})
	}
	req := Request{Scope: s, MemberID: f.MemberID, InvocationID: strings.Repeat("a", 64), TaskSelection: sub, Profile: p, FrozenScope: &f, History: history, ProductContext: &ProductContext{Title: "【测试商品 20261108-示例01】单图黑白T恤", Description: "这是一段合成的商品描述，用于验证较长历史消息和商品资料能在既有限额中合理整理。商品存在多个规格，颜色和图像仅为测试占位，文案中不应推断未提供的材质、重量或尺寸。"}}
	q, e := Prepare(req)
	t.Logf("error=%v max=%d prompt_bytes=%d", e, q.Quote.MaximumTokens, len(q.Identity.Prompt))
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(q.Identity.Prompt, req.ProductContext.Description) {
		t.Fatal("normal product facts should fit alongside a representative conversation")
	}
}

func TestV4TruncatesProductFactsDeterministicallyWithinBudget(t *testing.T) {
	p := testProfile()
	p.PromptVersion = ProductContextPromptVersion
	p.MaximumPromptTokens = 4096
	p.MaximumCompletionTokens = 512
	s := aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}
	reader := aiworkbench.ScopeReader{}
	f, err := reader.Freeze(context.Background(), s, "conversation", "member", aiworkbench.BusinessScope{Version: aiworkbench.BusinessScopeVersion, Kind: "general"})
	if err != nil {
		t.Fatal(err)
	}
	selection := &aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: "11111111-1111-4111-8111-111111111111", CatalogVersion: "1", PublicationID: "publication"}, Capability: aiworkbench.ContentTaskKind, Parameters: rp.ContentSelection{TargetLanguage: "en-US", Description: true}}
	original := &ProductContext{Title: "商品", Description: strings.Repeat("a", 4096)}
	request := Request{Scope: s, MemberID: "member", InvocationID: "invocation", TaskSelection: selection, Profile: p, FrozenScope: &f, History: []aiworkbench.Message{{ID: "message", ConversationID: "conversation", Sequence: 1, Author: aiworkbench.AuthorUser, Content: "优化描述"}}, ProductContext: original}
	first, err := Prepare(request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Prepare(request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Quote.InputHash != second.Quote.InputHash || !strings.Contains(first.Identity.Prompt, `"truncated":true`) || len(original.Description) != 4096 {
		t.Fatal("truncation must be stable, visible and preserve original")
	}
}
