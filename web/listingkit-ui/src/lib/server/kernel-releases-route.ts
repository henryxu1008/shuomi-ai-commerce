import { NextRequest } from "next/server";
import { serverAuth } from "@/auth";
import { readZitadelIdentityFromSession } from "./zitadel-auth";
import { readZitadelServerAccessToken } from "./zitadel-server-token";
import { kernelFailure, proxyKernelReleases } from "./kernel-releases-proxy";

const authenticated = serverAuth(async (request: NextRequest & { auth?: unknown }) => {
  const identity = readZitadelIdentityFromSession(request.auth as never);
  return proxyKernelReleases(request, readZitadelServerAccessToken(request.auth as never), String(identity?.userId ?? ""));
});
export async function handleKernelReleases(request: NextRequest) {
  const controller = new AbortController(), abort = () => controller.abort();
  const failure = () => kernelFailure(504, request.method === "PUT" ? "OUTCOME_UNKNOWN" : "UNAVAILABLE");
  request.signal.addEventListener("abort", abort, { once: true }); if (request.signal.aborted) abort();
  const timer = setTimeout(abort, 20000); let finish = () => {};
  const ended = new Promise<Response>(resolve => { finish = () => resolve(failure()); controller.signal.addEventListener("abort", finish, { once: true }); if (controller.signal.aborted) finish(); });
  try { return await Promise.race([authenticated(new NextRequest(request, { signal: controller.signal }), { params: Promise.resolve({}) }), ended]) ?? kernelFailure(503, "UNAVAILABLE"); }
  catch { return kernelFailure(503, request.method === "PUT" ? "OUTCOME_UNKNOWN" : "UNAVAILABLE"); }
  finally { clearTimeout(timer); request.signal.removeEventListener("abort", abort); controller.signal.removeEventListener("abort", finish); }
}
export const rejectKernelReleases = () => kernelFailure(405, "INVALID_REQUEST");
