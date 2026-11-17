import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { commercialResourcesFixture } from "@/test/fixtures/commercial-resources";
import { EnterpriseResources } from "./enterprise-resources";

const read = vi.hoisted(() => vi.fn());
vi.mock("@/lib/api/commercial-billing", async original => ({ ...await original<typeof import("@/lib/api/commercial-billing")>(), getCommercialResources: read }));
let client: QueryClient;
const tree = (org = "org-A", actor = "actor", roles = "operator") => <QueryClientProvider client={client}><EnterpriseResources key={`${org}:${actor}:${roles}`} userId={actor} organizationId={org} scope={JSON.stringify([actor, org, roles])} sequence={0} showSummary /></QueryClientProvider>;
beforeEach(() => { client = new QueryClient(); read.mockReset().mockImplementation((_user, org) => Promise.resolve(commercialResourcesFixture(org))); });
afterEach(() => { cleanup(); client.clear(); });
it("distinguishes enterprise usable resources from administrator unallocated resources",async()=>{
 const value=commercialResourcesFixture("org-A");read.mockResolvedValueOnce({...value,resources:value.resources.map(v=>v.resource_type==="store_renewal_period"?{...v,allocated:"2"}:v)});
 render(tree());const periods=await screen.findByRole("region",{name:"店铺续费期数余额"});
 expect(await within(periods).findByText("3 期")).toBeVisible();
 expect(screen.getAllByText("3 期")).toHaveLength(1);
 expect(within(periods).getByText("未分配（管理员可用）").nextElementSibling).toHaveTextContent("1 期");
 expect(within(periods).getByText("成员可用").nextElementSibling).toHaveTextContent("2 期");
});
it("displays exact balances, missing records and renewal periods with owner units", async () => {
  render(tree());
  const points = await screen.findByRole("region", { name: "AI 点数余额" });
  expect(await within(points).findByText("9,007,199,254.740993 点")).toBeVisible();
  expect(screen.getAllByText("9,007,199,254.740993 点")).toHaveLength(1);
  expect(within(points).getByText("预留").nextElementSibling).toHaveTextContent("0.000012 点");
  expect(within(points).getByText("已用").nextElementSibling).toHaveTextContent("0.000003 点");
  expect(within(screen.getByRole("region", { name: "数据额度余额" })).getByText("尚无资源记录")).toBeVisible();
  expect(within(screen.getByRole("region", { name: "店铺续费期数余额" })).getByText("企业合计可用").previousElementSibling).toHaveTextContent("1 期");
  expect(screen.queryByText(/1 家|充值暂未开放|未接入/)).not.toBeInTheDocument();
  expect(screen.getByRole("link", { name: "查看钱包与充值状态" })).toHaveAttribute("href", "/workbench/plans/top-up");
  expect(read).toHaveBeenCalledWith("actor", "org-A", expect.any(AbortSignal));
});
it("shows recorded zero and debt, and does not turn denied reads into zero", async () => {
  const value = commercialResourcesFixture("org-A");
  read.mockResolvedValueOnce({ ...value, resources: value.resources.map(v => v.resource_type === "ai_point" ? { ...v, available: "0", debt: "7" } : v) });
  const view = render(tree());
  const points = await screen.findByRole("region", { name: "AI 点数余额" });
  expect(await within(points).findByText("0 点")).toBeVisible();
  expect(within(points).getByText("待偿还").nextElementSibling).toHaveTextContent("7 点");
  read.mockRejectedValueOnce({ code: "PERMISSION_DENIED" }); view.rerender(tree("org-A", "actor", "viewer"));
  expect(await screen.findByRole("alert")).toHaveTextContent("无资源查看权限");
  expect(screen.queryByText("0 点")).not.toBeInTheDocument();
});
it("never displays a late A response after changing org or identity", async () => {
  let resolve!: (value: unknown) => void;
  read.mockReturnValueOnce(new Promise(r => { resolve = r; }));
  const view = render(tree());
  view.rerender(tree("org-B", "other"));
  await screen.findByText("9,007,199,254.740993 点");
  await act(async () => resolve({ ...commercialResourcesFixture("org-A"), resources: commercialResourcesFixture("org-A").resources.map(v => v.resource_type === "ai_point" ? { ...v, available: "998877" } : v) }));
  expect(screen.queryByText("998,877 点")).not.toBeInTheDocument();
  expect(read).toHaveBeenLastCalledWith("other", "org-B", expect.any(AbortSignal));
});
