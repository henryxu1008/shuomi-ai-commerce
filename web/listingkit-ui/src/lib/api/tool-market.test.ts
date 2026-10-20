import { afterEach, expect, it, vi } from "vitest";
import { downloadToolReference } from "./tool-market";

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

it("does not download private reference bytes after their scope is abandoned during digest verification", async () => {
  const digest = Promise.withResolvers<ArrayBuffer>();
  const hash = vi.fn(() => digest.promise);
  vi.stubGlobal("crypto", { subtle: { digest: hash } });
  vi.stubGlobal(
    "fetch",
    vi.fn(
      async () =>
        new Response("test", {
          headers: { "Content-Type": "text/plain; charset=utf-8" },
        }),
    ),
  );
  const click = vi
    .spyOn(HTMLAnchorElement.prototype, "click")
    .mockImplementation(() => undefined);
  vi.stubGlobal(
    "URL",
    class extends URL {
      static createObjectURL = vi.fn(() => "blob:reference");
      static revokeObjectURL = vi.fn();
    },
  );
  const controller = new AbortController();
  const result = downloadToolReference(
    { userId: "actor", organizationId: "org-a" },
    false,
    "aa1043df-c64c-499d-9c99-57598f7bff18",
    {
      id: "bb1043df-c64c-499d-9c99-57598f7bff18",
      name: "flow.txt",
      contentType: "text/plain; charset=utf-8",
      size: 4,
      sha256:
        "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
    },
    controller.signal,
  );
  await vi.waitFor(() => expect(hash).toHaveBeenCalledOnce());
  controller.abort();
  digest.resolve(
    Uint8Array.from(
      "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08".match(
        /../g,
      )!,
      (v) => parseInt(v, 16),
    ).buffer,
  );
  await expect(result).rejects.toBeDefined();
  expect(click).not.toHaveBeenCalled();
});
