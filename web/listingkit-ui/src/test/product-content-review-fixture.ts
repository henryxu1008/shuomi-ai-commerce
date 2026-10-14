import { titleProposalFixture } from "./product-title-review-fixture";
/** UI contract fixture only; never mounted as a model or runtime fallback. */
export function contentProposalFixture() {
  const title = titleProposalFixture();
  const selection = { targetLanguage: "ja", title: true, description: true };
  const mode = { Kind: "agent-execution-mode-v1" as const, ID: title.proposal_id, Digest: "a".repeat(64) };
  return { ...title, schema_version: 2 as const, coverage: "product-content-proposals-only" as const, policy: "product-content-review-v1" as const, before: "", after: "", original_title: "",
    content: { selection, before: [{ field: "title" as const, value: "Original title" }, { field: "description" as const, value: "Original description" }], after: [{ field: "title" as const, value: "Suggested title" }, { field: "description" as const, value: "Suggested description" }], decisions: [] },
    execution: { ContentSelection: selection, scope: { OrganizationID: "B", ActorID: "owner" }, runId: title.proposal_id, requestKey: title.proposal_id, binding: { ContextKind: "acquisition", ContextID: title.proposal_id, ProductKey: title.input.product_key, CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "product" as const }, configuration: { Kind: "agent-configuration-v1" as const, ID: title.proposal_id, Digest: "b".repeat(64) }, mode },
    execution_mode: { mode: "default" as const, revision: "0", snapshot: mode },
  };
}
