import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { NewConversationIntroduction } from "./chat-introduction";

const profile = vi.hoisted(() => vi.fn());
vi.mock("@/lib/api/account", () => ({ getAccountProfile: profile }));
afterEach(() => { cleanup(); profile.mockReset(); });

it("uses the current account display name in the greeting", async () => {
  profile.mockResolvedValue({ displayName: "小林" });
  render(<QueryClientProvider client={new QueryClient()}><NewConversationIntroduction userId="user-a" directCompose canUse createAction={null} /></QueryClientProvider>);
  expect(await screen.findByRole("heading", { name: "小林，今天想做什么？" })).toBeVisible();
  expect(profile).toHaveBeenCalledWith(expect.objectContaining({ expectedUserId: "user-a" }));
});

it("uses a generic greeting when the name is missing or is the account identifier", async () => {
  profile.mockResolvedValue({ displayName: "user-a" });
  render(<QueryClientProvider client={new QueryClient()}><NewConversationIntroduction userId="user-a" directCompose canUse createAction={null} /></QueryClientProvider>);
  await waitFor(() => expect(profile).toHaveBeenCalled());
  expect(screen.getByRole("heading", { name: "今天想做什么？" })).toBeVisible();
});

it("does not show the previous account name while switching accounts", async () => {
  profile.mockImplementation(({ expectedUserId }: { expectedUserId: string }) => expectedUserId === "user-a" ? Promise.resolve({ displayName: "小林" }) : new Promise(() => {}));
  const client = new QueryClient();
  const view = render(<QueryClientProvider client={client}><NewConversationIntroduction userId="user-a" directCompose canUse createAction={null} /></QueryClientProvider>);
  await screen.findByRole("heading", { name: "小林，今天想做什么？" });
  view.rerender(<QueryClientProvider client={client}><NewConversationIntroduction userId="user-b" directCompose canUse createAction={null} /></QueryClientProvider>);
  expect(screen.queryByText("小林，今天想做什么？")).toBeNull();
  expect(screen.getByRole("heading", { name: "今天想做什么？" })).toBeVisible();
});
