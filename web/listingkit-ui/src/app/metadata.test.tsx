import { headers } from "next/headers";
import { describe, expect, it, vi } from "vitest";

vi.mock("next/headers", () => ({ headers: vi.fn() }));
// Metadata is inspected outside Next's Server Component runtime.
vi.mock("server-only", () => ({}));

import { generateMetadata } from "./layout";

describe("application metadata", () => {
  it("brands the current application shell as Sumi", async () => {
    vi.mocked(headers).mockResolvedValue(new Headers({ host: "listingkit.example" }));

    const metadata = await generateMetadata();

    expect(metadata.title).toBe("硕米智能引擎 | AI 电商智能操作系统");
    expect(metadata.openGraph).toMatchObject({
      title: "硕米智能引擎 | AI 电商智能操作系统",
      images: ["https://listingkit.example/og-v2.png"],
    });
  });

  it("keeps the approved public home page metadata", async () => {
    const homePage = await import("./page");

    expect(homePage.metadata).toMatchObject({
      title: "硕米智能引擎 | 新一代 AI 电商智能操作系统",
      openGraph: {
        title: "硕米智能引擎 | 新一代 AI 电商智能操作系统",
        images: ["/sumi/fd824975-1e65-4585-9ebf-212d68cb1507.png"],
      },
    });
  });

  it("keeps Sumi branding in the social previews for every public legal document", async () => {
    const legalPages = await Promise.all([
      import("./privacy-policy/page"),
      import("./user-agreement/page"),
      import("./ai-compute-billing/page"),
      import("./service-agreement/page"),
    ]);

    for (const page of legalPages) {
      expect(page.metadata).toMatchObject({
        openGraph: {
          siteName: "硕米智能引擎",
          images: ["/sumi/fd824975-1e65-4585-9ebf-212d68cb1507.png"],
        },
        twitter: {
          card: "summary_large_image",
          images: ["/sumi/fd824975-1e65-4585-9ebf-212d68cb1507.png"],
        },
      });
    }
  });
});
