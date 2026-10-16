import { z } from "zod";
import { readBoundedStrictJSON } from "./strict-json-response";
import {
  maxToolFileBytes,
  type toolAttachment,
} from "@/lib/contracts/tool-market";
export type ToolScope = { userId: string; organizationId: string };
export class ToolMarketError extends Error {
  constructor(
    public code: string,
    public status = 500,
  ) {
    super(code);
  }
}
export async function toolRequest<T>(
  scope: ToolScope,
  path: string,
  schema: z.ZodType<T>,
  init: RequestInit = {},
): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("X-Expected-User-ID", scope.userId);
  headers.set("X-Expected-Organization-ID", scope.organizationId);
  headers.set("Accept", "application/json");
  const write = !!init.method && init.method !== "GET";
  try {
    const r = await fetch("/api/tool-market/" + path, {
      ...init,
      headers,
      credentials: "same-origin",
      cache: "no-store",
      redirect: "error",
    });
    const payload = await readBoundedStrictJSON(
      r,
      256 << 10,
      init.signal ?? undefined,
    );
    if (!r.ok) {
      const error = z
        .strictObject({ code: z.string().regex(/^[A-Z_]{1,80}$/) })
        .safeParse(payload);
      throw new ToolMarketError(
        error.success
          ? error.data.code
          : write
            ? "OUTCOME_UNKNOWN"
            : "DEPENDENCY_UNAVAILABLE",
        r.status,
      );
    }
    const parsed = schema.safeParse(payload);
    if (r.status !== 200 || !parsed.success) throw new Error();
    return parsed.data;
  } catch (e) {
    if (e instanceof ToolMarketError) throw e;
    throw new ToolMarketError(
      write ? "OUTCOME_UNKNOWN" : "DEPENDENCY_UNAVAILABLE",
    );
  }
}
export async function downloadTool(scope: ToolScope) {
  const response = await fetch("/api/tool-market/plugin", {
    headers: {
      "X-Expected-User-ID": scope.userId,
      "X-Expected-Organization-ID": scope.organizationId,
    },
    credentials: "same-origin",
    cache: "no-store",
    redirect: "error",
    signal: AbortSignal.timeout(15000),
  });
  if (
    response.status !== 200 ||
    response.headers.get("Content-Type") !== "application/zip"
  )
    throw new ToolMarketError("DEPENDENCY_UNAVAILABLE");
  const { readToolPackage } = await import("./tool-package");
  const bytes = await readToolPackage(response, AbortSignal.timeout(15000));
  const url = URL.createObjectURL(
    new Blob([bytes], { type: "application/zip" }),
  );
  const link = document.createElement("a");
  link.href = url;
  link.download = "shuomi-1688-capture.zip";
  link.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export async function downloadToolReference(
  scope: ToolScope,
  admin: boolean,
  requestId: string,
  file: z.infer<typeof toolAttachment>,
  signal?: AbortSignal,
) {
  const controller = new AbortController(),
    abort = () => controller.abort(),
    timer = setTimeout(() => controller.abort(), 15000);
  signal?.addEventListener("abort", abort, { once: true });
  if (signal?.aborted) abort();
  try {
    controller.signal.throwIfAborted();
    const response = await fetch(
      "/api/tool-market/" +
        (admin ? "admin/" : "") +
        "requests/" +
        requestId +
        "/files/" +
        file.id,
      {
        headers: {
          "X-Expected-User-ID": scope.userId,
          "X-Expected-Organization-ID": scope.organizationId,
        },
        credentials: "same-origin",
        cache: "no-store",
        redirect: "error",
        signal: controller.signal,
      },
    );
    if (
      !response.ok ||
      response.status !== 200 ||
      response.headers.get("content-type") !== file.contentType
    )
      throw new ToolMarketError("DEPENDENCY_UNAVAILABLE");
    const reader = response.body?.getReader();
    if (!reader) throw new ToolMarketError("DEPENDENCY_UNAVAILABLE");
    const chunks: Uint8Array[] = [];
    let size = 0;
    const cancel = () => {
      void reader.cancel().catch(() => undefined);
    };
    controller.signal.addEventListener("abort", cancel, { once: true });
    try {
      while (true) {
        controller.signal.throwIfAborted();
        const next = await reader.read();
        controller.signal.throwIfAborted();
        if (next.done) break;
        size += next.value.length;
        if (size > maxToolFileBytes || size > file.size)
          throw new Error("Invalid file size");
        chunks.push(next.value);
      }
    } catch (e) {
      cancel();
      throw e;
    } finally {
      controller.signal.removeEventListener("abort", cancel);
      reader.releaseLock();
    }
    if (size !== file.size) throw new Error("Invalid file size");
    const bytes = new Uint8Array(size);
    let at = 0;
    for (const chunk of chunks) {
      bytes.set(chunk, at);
      at += chunk.length;
    }
    const digest = Array.from(
      new Uint8Array(await crypto.subtle.digest("SHA-256", bytes)),
      (v) => v.toString(16).padStart(2, "0"),
    ).join("");
    controller.signal.throwIfAborted();
    if (digest !== file.sha256) throw new Error("Invalid file digest");
    const href = URL.createObjectURL(
      new Blob([bytes], { type: file.contentType }),
    );
    const link = document.createElement("a");
    link.href = href;
    link.download = file.name;
    link.click();
    setTimeout(() => URL.revokeObjectURL(href), 1000);
  } finally {
    clearTimeout(timer);
    signal?.removeEventListener("abort", abort);
  }
}
