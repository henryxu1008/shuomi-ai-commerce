import {afterEach, beforeEach, expect, it, vi} from "vitest";
import {cleanup, fireEvent, render, screen, waitFor} from "@testing-library/react";
import {skillRequest} from "@/lib/api/skill";
import type {SkillProjection} from "@/lib/contracts/skill";
import {SkillLibraryPage} from "./skill-library-page";

const context = vi.hoisted(() => ({
  user: {id: "alice"},
  effectiveOrganization: null as {id: string; name: string} | null,
  permissions: [] as string[], selectionRequired: true, isLoading: false, isSwitching: false,
  error: null as {code: string} | null, blockingError: null as {code: string} | null,
  retry: vi.fn(),
}));
vi.mock("@/components/providers/workbench-context-provider", () => ({useWorkbenchContext: () => context}));
vi.mock("@/lib/api/skill", async importOriginal => ({...await importOriginal<typeof import("@/lib/api/skill")>(), skillRequest: vi.fn()}));

beforeEach(() => {
  Object.assign(context, {user: {id: "alice"}, effectiveOrganization: null, permissions: [], selectionRequired: true, isLoading: false, isSwitching: false, error: null, blockingError: null});
  vi.mocked(skillRequest).mockImplementation(async (_scope, path) => path.endsWith("/info")
    ? {available: true, canCreate: !path.startsWith("official"), reason: ""}
    : path.endsWith("/admission") ? {policy: {allowPersonal: false, revision: "0"}, canConfigure: false}
    : {items: [], nextCursor: ""});
});
afterEach(() => {cleanup(); vi.clearAllMocks()});

it("manages personal skills without an organization or a modal", async () => {
  render(<SkillLibraryPage expectedUserId="alice" category="mine"/>);
  expect(await screen.findByRole("button", {name: "新建技能"})).toBeEnabled();
  expect(screen.getByRole("heading", {level: 1, name: "技能库"})).toBeInTheDocument();
  expect(screen.getByRole("link", {name: "我的技能"})).toHaveAttribute("aria-current", "page");
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(vi.mocked(skillRequest).mock.calls.every(([scope, path]) => scope.userId === "alice" && !scope.organizationId && path.startsWith("personal"))).toBe(true);
});

it("reads official templates without an organization and shows an honest empty state", async () => {
  render(<SkillLibraryPage expectedUserId="alice" category="official"/>);
  expect(await screen.findByText(/官方模板由获授权维护者/)).toBeInTheDocument();
  expect(screen.queryByRole("button", {name: "新建技能"})).not.toBeInTheDocument();
  expect(vi.mocked(skillRequest).mock.calls.every(([, path]) => path.startsWith("official"))).toBe(true);
});

it("keeps the team category closed without a current organization", () => {
  render(<SkillLibraryPage expectedUserId="alice" category="team"/>);
  expect(screen.getByRole("heading", {name: "请先选择企业"})).toBeInTheDocument();
  expect(skillRequest).not.toHaveBeenCalled();
});

it("uses current team permissions and removes the old request subtree during a switch", async () => {
  Object.assign(context, {effectiveOrganization: {id: "org-a", name: "企业 A"}, selectionRequired: false});
  const view = render(<SkillLibraryPage expectedUserId="alice" category="team"/>);
  expect(await screen.findByRole("button", {name: "待审批的个人提交"})).toBeDisabled();
  const oldSignals = vi.mocked(skillRequest).mock.calls.map(([, , , init]) => init?.signal).filter(Boolean);
  context.isSwitching = true;
  view.rerender(<SkillLibraryPage expectedUserId="alice" category="team"/>);
  expect(screen.queryByRole("button", {name: "新建技能"})).not.toBeInTheDocument();
  expect(oldSignals.every(signal => signal?.aborted)).toBe(true);
  Object.assign(context, {isSwitching: false, effectiveOrganization: {id: "org-b", name: "企业 B"}});
  view.rerender(<SkillLibraryPage expectedUserId="alice" category="team"/>);
  await waitFor(() => expect(vi.mocked(skillRequest).mock.calls.some(([scope, path]) => scope.organizationId === "org-b" && path === "organization/info")).toBe(true));
});

it("clears management when the logged-in identity differs from the server identity", async () => {
  const view = render(<SkillLibraryPage expectedUserId="alice" category="mine"/>);
  await screen.findByRole("button", {name: "新建技能"});
  context.user = {id: "bob"};
  view.rerender(<SkillLibraryPage expectedUserId="alice" category="mine"/>);
  expect(screen.getByRole("heading", {name: "登录身份已变化"})).toBeInTheDocument();
  expect(screen.queryByRole("button", {name: "新建技能"})).not.toBeInTheDocument();
});

it("keeps personal management available when only the organization context fails", async () => {
  context.error = {code: "ORGANIZATION_ACCESS_REVOKED"};
  render(<SkillLibraryPage expectedUserId="alice" category="mine"/>);
  expect(await screen.findByRole("button", {name: "新建技能"})).toBeEnabled();
  expect(vi.mocked(skillRequest).mock.calls.every(([scope]) => !scope.organizationId)).toBe(true);
});

it("shows a service failure instead of an empty successful catalog", async () => {
  vi.mocked(skillRequest).mockRejectedValue(new Error("offline"));
  render(<SkillLibraryPage expectedUserId="alice" category="official"/>);
  expect(await screen.findByRole("alert")).toHaveTextContent("技能服务尚未配置或暂不可用");
  expect(screen.queryByText(/当前范围没有可读取的技能/)).not.toBeInTheDocument();
});

it("stops personal reads when authentication is lost", async () => {
  const view = render(<SkillLibraryPage expectedUserId="alice" category="mine"/>);
  await screen.findByRole("button", {name: "新建技能"});
  context.error = {code: "AUTHENTICATION_REQUIRED"};
  view.rerender(<SkillLibraryPage expectedUserId="alice" category="mine"/>);
  expect(screen.getByRole("heading", {name: "登录已失效"})).toBeInTheDocument();
  expect(screen.getByRole("link", {name: "重新登录"})).toHaveAttribute("href", "/login?returnTo=%2Fworkbench%2Fagents%2Fskills%2Fmine");
  expect(screen.queryByRole("button", {name: "新建技能"})).not.toBeInTheDocument();
});

it.each([["复制到本人", "personal/copy"], ["复制到当前企业", "organization/copy"]])("copies an official exact version through the existing %s action", async (label, path) => {
  Object.assign(context, {effectiveOrganization: {id: "org-a", name: "企业 A"}, selectionRequired: false});
  const item: SkillProjection = {
    version: {skill: {id: "11111111-1111-4111-8111-111111111111", owner: {kind: "PLATFORM", id: "official"}, creator: "maintainer", headVersion: 2, revision: 4, createdAt: "2026-10-17T00:00:00Z", updatedAt: "2026-10-17T00:00:00Z"}, version: 2,
      package: {name: "official-method", description: "Official method", digest: "a".repeat(64), files: []}, state: "ENABLED", visibility: "SHARED", allowCopy: true, allowExport: true, createdAt: "2026-10-17T00:00:00Z"},
    canEdit: false, canPublish: false, canUse: true, canCopy: true, canExport: true,
  };
  vi.mocked(skillRequest).mockImplementation(async (_scope, requestPath) => requestPath.endsWith("/info") ? {available: true, canCreate: false, reason: ""}
    : requestPath.endsWith("/versions/2") ? item : requestPath.endsWith("/versions") ? {items: [item.version]}
    : requestPath.endsWith("/copy") ? {} : {items: [item], nextCursor: ""});
  render(<SkillLibraryPage expectedUserId="alice" category="official"/>);
  fireEvent.click(await screen.findByText("official-method"));
  fireEvent.click(await screen.findByRole("button", {name: label}));
  await waitFor(() => expect(vi.mocked(skillRequest).mock.calls.some(([, requestPath]) => requestPath === path)).toBe(true));
  const call = vi.mocked(skillRequest).mock.calls.find(([, requestPath]) => requestPath === path)!;
  expect(call[0]).toEqual({userId: "alice", organizationId: "org-a"});
  expect(call[3]?.method).toBe("POST");
  expect(call[3]?.headers).toHaveProperty("Idempotency-Key", expect.any(String));
  expect(JSON.parse(String(call[3]?.body))).toEqual({source: {kind: "skill-version-v1", id: item.version.skill.id, ownerKind: "PLATFORM", ownerId: "official", version: 2, digest: "a".repeat(64), policyVersion: "skill-method-v1"}, allowCopy: false, allowExport: false});
});
