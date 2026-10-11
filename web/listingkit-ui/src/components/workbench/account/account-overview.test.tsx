import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, expect, it, vi } from "vitest";
import { AccountOverview } from "./account-overview";
import { getReferralEarnings } from "@/lib/api/account-referral-economics";
import type { AccountOrganization, AccountProfile } from "@/lib/api/account";
vi.mock("@/lib/api/account-referral-economics", () => ({ getReferralEarnings: vi.fn() }));
const profile: AccountProfile = { schemaVersion: "account-v1", userId: "user-a", homeOrganizationId: "org-a", displayName: null, email: null, emailVerified: null, phoneNumber: null, phoneNumberVerified: null, readAt: "2026-10-11T00:00:00Z", source: "zitadel_userinfo" };
const organization: AccountOrganization = { schemaVersion: "account-v1", userId: "user-a", homeOrganizationId: "org-a", effectiveOrganizationId: "org-a", name: null, roles: [], readAt: profile.readAt, source: "zitadel_project_authorizations", authorizationMaxAgeSeconds: 60 };
const earnings = { schemaVersion: "referral-earnings-v1" as const, referrer: "user-a", currency: "CNY" as const, pendingMinor: "158000", availableMinor: "326000", reservedMinor: "0", adjustmentMinor: "0", version: "1", updatedAt: null, source: "referral_earnings_projection" as const, entryLimit: 100 as const };
const clients: QueryClient[] = [];
function mount() { const client = new QueryClient({ defaultOptions: { queries: { retry: false } } }); clients.push(client); return render(<QueryClientProvider client={client}><AccountOverview profile={profile} organization={organization} business={null} /></QueryClientProvider>); }
afterEach(() => { cleanup(); clients.splice(0).forEach(c => c.clear()); vi.mocked(getReferralEarnings).mockReset(); });
it("shows personal earnings without claiming KYC or withdrawal eligibility", async () => {
  vi.mocked(getReferralEarnings).mockResolvedValue(earnings); mount();
  expect(await screen.findByText("可用 ¥3,260")).toBeVisible();
  expect(screen.getByText("待结算 ¥1,580")).toBeVisible();
  expect(screen.queryByText(/已实名认证|企业已认证|可提现 ¥/)).not.toBeInTheDocument();
  expect(getReferralEarnings).toHaveBeenCalledWith("user-a", expect.any(AbortSignal));
});
it("rejects another person's earnings", async () => {
  vi.mocked(getReferralEarnings).mockResolvedValue({ ...earnings, referrer: "user-b" }); mount();
  await waitFor(() => expect(screen.getByText("收益读取暂不可用，请进入收益页重试")).toBeVisible());
  expect(screen.queryByText("可用 ¥3,260")).not.toBeInTheDocument();
});
