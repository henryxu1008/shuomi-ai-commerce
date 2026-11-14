import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { ResearchWorkflowNav } from "./research-workflow-nav";
afterEach(cleanup);

it("keeps the current research project and filters while switching steps", () => {
  const id="11111111-1111-4111-8111-111111111111";
  render(<ResearchWorkflowNav center="research" projectId={id} step="evidence" mode="archived" search="收纳" />);
  expect(screen.getByRole("link",{name:"查看商品资料"})).toHaveAttribute("aria-current","page");
  const url=new URL(screen.getByRole("link",{name:"比较重点商品"}).getAttribute("href")!,"https://example.test");
  expect(url.pathname).toBe(`/workbench/data/research/${id}`);
  expect(Object.fromEntries(url.searchParams)).toEqual({step:"summary",mode:"archived",search:"收纳"});
});

it("keeps a validated selection sample and cursor when switching steps", () => {
  const id="11111111-1111-4111-8111-111111111111",other="22222222-2222-4222-8222-222222222222";
  const sample=JSON.stringify([id,other].map(candidateId=>({candidateId,expectedRevision:"2",linkId:candidateId,evidenceDigest:"a".repeat(64),selection:{itemId:candidateId,expectedRevision:"1",originalPublicationId:candidateId,originalVersion:"1"}})));
  render(<ResearchWorkflowNav center="selection" projectId={id} step="analysis" mode="active" search="" sample={sample} candidateAfter={other} />);
  const url=new URL(screen.getByRole("link",{name:"评估与决定"}).getAttribute("href")!,"https://example.test");
  expect(url.pathname).toBe(`/workbench/supply/selection/${id}`);
  expect(Object.fromEntries(url.searchParams)).toEqual({step:"decisions",mode:"active",search:"",sample,candidateAfter:other});
});

it("retains the handoff token on the same project so the original gate remains active", () => {
  const id="11111111-1111-4111-8111-111111111111";
  render(<ResearchWorkflowNav center="selection" projectId={id} step="analysis" mode="active" search="" researchToken="original-token" />);
  const url=new URL(screen.getByRole("link",{name:"评估与决定"}).getAttribute("href")!,"https://example.test");
  expect(url.pathname).toBe(`/workbench/supply/selection/${id}`);
  expect(url.searchParams.get("handoff")).toBe("original-token");
});
