import {cleanup, render, screen, within} from "@testing-library/react";
import {afterEach, beforeEach, expect, it, vi} from "vitest";
import SkillsPage from "@/app/workbench/agents/skills/page";
import {requireAuthenticatedUserId} from "@/lib/server/referral-page-auth";

vi.mock("@/lib/server/referral-page-auth", () => ({requireAuthenticatedUserId: vi.fn()}));
vi.mock("next/navigation", () => ({redirect: () => {throw new Error("unexpected redirect to a category")}}));

beforeEach(() => {vi.mocked(requireAuthenticatedUserId).mockResolvedValue("alice")});
afterEach(() => {cleanup(); vi.resetAllMocks(); vi.unstubAllGlobals()});

it("shows an identity-only introduction with three existing destinations without reading a skill catalog", async () => {
  const fetch = vi.fn();
  vi.stubGlobal("fetch", fetch);
  render(await SkillsPage());
  expect(requireAuthenticatedUserId).toHaveBeenCalledWith("/workbench/agents/skills");
  expect(screen.getByRole("heading", {level: 1, name: "把业务经验沉淀为可复用的技能"})).toBeVisible();
  for (const [name, category] of [["浏览官方模板", "official"], ["管理我的技能", "mine"], ["管理团队技能", "team"]]) {
    expect(screen.getByRole("link", {name: new RegExp(name)})).toHaveAttribute("href", `/workbench/agents/skills/${category}`);
  }
  expect(fetch).not.toHaveBeenCalled();
  expect(screen.queryByRole("button", {name: /新建|启用|提交/})).not.toBeInTheDocument();
});

it("keeps the existing server identity gate before rendering the introduction", async () => {
  const denied = new Error("authentication required");
  vi.mocked(requireAuthenticatedUserId).mockRejectedValue(denied);
  await expect(SkillsPage()).rejects.toBe(denied);
});

it("explains the four-step path and consent/execution boundaries separately from navigation", async () => {
  render(await SkillsPage());
  const steps = within(screen.getByRole("list")).getAllByRole("listitem");
  expect(steps.map(step => within(step).getByRole("heading").textContent)).toEqual(["选择来源", "完善方法", "保存并启用", "业务中使用"]);
  expect(screen.getByText(/本人同意并提交指定版本后/)).toBeVisible();
  expect(screen.getByText(/现有智能体的权限、预算与审核/)).toBeVisible();
});
