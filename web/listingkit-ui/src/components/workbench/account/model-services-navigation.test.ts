import { describe, expect, it } from "vitest";
import { modelServicesHref, readModelServicesNavigation } from "@/lib/contracts/model-services-navigation";

describe("model settings return navigation", () => {
  it("keeps the original permitted product path and query", () => {
    const original = "/workbench/data/mine/own/item-1/images?source=source-1&preparation=prep-1";
    const href = modelServicesHref("multimodal", "org-a", original);
    const search = Object.fromEntries(new URL(href, "https://app.test").searchParams);
    expect(readModelServicesNavigation(search)).toEqual({ category: "multimodal", organizationId: "org-a", returnTo: original });
  });

  it("returns to the original pending-listing draft opened from a preparation", () => {
    const returnTo = "/workbench/listing/missing?preparation=22222222-2222-4222-8222-222222222222&source=11111111-1111-4111-8111-111111111111";
    const business = { purpose: "image-generation", org: "org-a" };
    expect(readModelServicesNavigation({ ...business, back: returnTo })?.returnTo).toBe(returnTo);
    expect(readModelServicesNavigation({ ...business, back: returnTo + "&mode=platform" })?.returnTo).toBeUndefined();
    expect(readModelServicesNavigation({ ...business, back: "/workbench/listing/missing" })?.returnTo).toBeUndefined();
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
