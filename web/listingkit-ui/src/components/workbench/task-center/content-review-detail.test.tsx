import { afterEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { TitleReviewControls, TitleReviewDetail } from "./title-review-detail";
import { contentProposalFixture } from "@/test/product-content-review-fixture";
import { parseProductTitleProposal } from "@/lib/api/product-title-review";

afterEach(cleanup);
it("identifies the review target and distinguishes captured evidence", () => {
  const p = parseProductTitleProposal(contentProposalFixture())!;
  render(<TitleReviewDetail proposal={p} />);
  expect(screen.getByText(p.input.product_key)).toBeInTheDocument();
  expect(screen.getByText("采集的资料")).toBeInTheDocument();
});
it("presents source and comparison without internal identifiers", () => {
  const f = contentProposalFixture();
  const p = parseProductTitleProposal({ ...f, evidence: [{ id: "evidence-secret", reference_type: "user_input", reference_id: "ref-secret", snapshot_id: "snapshot-secret", checksum: "sha256:secret" }] })!;
  render(<TitleReviewDetail proposal={p} />);
  expect(screen.getByText("用户提供的资料")).toBeInTheDocument();
  expect(screen.getByText("Original description")).toBeInTheDocument();
  expect(screen.getByText("Suggested description")).toBeInTheDocument();
  expect(screen.queryByText(/evidence-secret|ref-secret|snapshot-secret|sha256:secret|product-content-review-v1/)).not.toBeInTheDocument();
});
it("compares all selected text fields and sends an edit of the whole frozen set", () => {
  const p = parseProductTitleProposal(contentProposalFixture())!;
  const decide = vi.fn();
  render(<><TitleReviewDetail proposal={p} /><TitleReviewControls proposal={p} permissions={["listingkit.admin.write"]} roles={[]} userId="owner" pending={false} decide={decide} apply={vi.fn()} /></>);
  expect(screen.getByText("Original description")).toBeInTheDocument();
  expect(screen.getByText("Suggested description")).toBeInTheDocument();
  expect(screen.getByText("ja · 标题、描述")).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "编辑内容" }));
  fireEvent.change(screen.getByLabelText("编辑描述"), { target: { value: "Reviewed description" } });
  fireEvent.click(screen.getByRole("button", { name: "保存为待审核提案" }));
  expect(decide).toHaveBeenCalledWith("edit", undefined, [{ field: "title", value: "Suggested title" }, { field: "description", value: "Reviewed description" }]);
});
it.each(["read_only", "unavailable"])("keeps content %s suggestions readable without adoption controls", mode => {
  const f = contentProposalFixture();
  const p = parseProductTitleProposal({ ...f, execution_mode: mode === "unavailable" ? undefined : { ...f.execution_mode, mode } })!;
  render(<TitleReviewControls proposal={p} roles={[]} permissions={["listingkit.admin.write", "local_agent.write"]} userId="owner" pending={false} decide={vi.fn()} apply={vi.fn()} />);
  expect(screen.queryByRole("button")).toBeNull();
  expect(screen.getByText(mode === "read_only" ? /本次为只读权限/ : /无法读取本次冻结权限/)).toBeInTheDocument();
});
