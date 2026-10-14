import { expect, it } from "vitest";
import { parseProductTitleProposal, parseProductTitleProposalList, productTitleDecisionSchema } from "./product-title-review";
import { contentProposalFixture } from "@/test/product-content-review-fixture";

it("reads content v2 only with the exact selection, field set and actual execution provenance", () => {
  const p = contentProposalFixture();
  expect(parseProductTitleProposal(p)).toEqual(p);
  for (const value of [
    { ...p, schema_version: 1 }, { ...p, coverage: "product-title-proposals-only" }, { ...p, policy: "title-review-v1" },
    { ...p, execution: undefined }, { ...p, content: { ...p.content, after: p.content.after.slice(0, 1) } },
    { ...p, content: { ...p.content, after: [p.content.after[0], p.content.after[0]] } },
    { ...p, execution: { ...p.execution, ContentSelection: { ...p.content.selection, targetLanguage: "en-US" } } },
    { ...p, execution_mode: { ...p.execution_mode, snapshot: { ...p.execution.mode, Digest: "c".repeat(64) } } },
  ]) expect(parseProductTitleProposal(value)).toBeNull();
});
it("accepts bounded text-only edits and rejects field paths, clears and malformed arrays", () => {
  const p = contentProposalFixture();
  expect(productTitleDecisionSchema.safeParse({ action: "edit", expected_revision: "1", content: p.content.after }).success).toBe(true);
  for (const content of [[{ field: "price", value: "1" }], [{ field: "title", value: "" }], [{ field: "selling_points", value: "[]" }], [{ field: "seo_keywords", value: '["one",null]' }], [p.content.after[0], p.content.after[0]]]) expect(productTitleDecisionSchema.safeParse({ action: "edit", expected_revision: "1", content }).success).toBe(false);
});
it("keeps content selection visible in the explicitly versioned mixed text collection",()=>{
 const p=contentProposalFixture();
 const item={proposal_id:p.proposal_id,product_key:p.input.product_key,base_version:"1",proposal_revision:"1",state:"pending",policy:p.policy,content_selection:p.content.selection};
 const list={schema_version:2,coverage:"product-text-proposals",items:[item],next_cursor:null};
 expect(parseProductTitleProposalList(list)).toEqual(list);
 expect(parseProductTitleProposalList({...list,schema_version:1,coverage:"product-title-proposals-only"})).toBeNull();
 expect(parseProductTitleProposalList({...list,items:[{...item,policy:"title-review-v1"}]})).toBeNull();
});
