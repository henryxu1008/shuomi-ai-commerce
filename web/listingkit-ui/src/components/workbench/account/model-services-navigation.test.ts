import { describe, expect, it } from "vitest";
import { modelServicesHref, readModelServicesNavigation } from "@/lib/contracts/model-services-navigation";

describe("model settings return navigation", () => {
  it("keeps the original permitted product path and query", () => {
    const original = "/workbench/data/mine/own/item-1/images?source=source-1&preparation=prep-1";
    const href = modelServicesHref("multimodal", "org-a", original);
    const search = Object.fromEntries(new URL(href, "https://app.test").searchParams);
    expect(readModelServicesNavigation(search)).toEqual({ category: "multimodal", organizationId: "org-a", returnTo: original });
  });

  it.each([
    "https://other.test/workbench/ai/chat",
    "//other.test/workbench/ai/chat",
    "/workbench/ai/chat/../tasks",
    "/workbench/ai/chat/%2e%2e",
    "/workbench/account/organization",
    "/workbench/ai/chat\\external",
  ])("rejects unauthorized return paths %s", (target) => {
    expect(readModelServicesNavigation({ purpose: "text", org: "org-a", back: target })).toEqual({
      category: "text", organizationId: "org-a", returnTo: undefined,
    });
  });
});
