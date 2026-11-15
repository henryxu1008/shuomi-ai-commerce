import { NextRequest } from "next/server";
import { serverAuth } from "@/auth";
import { readZitadelIdentityFromSession } from "@/lib/server/zitadel-auth";
import { readZitadelServerAccessToken } from "@/lib/server/zitadel-server-token";
import { proxyOfflineRecords, offlineFailure } from "@/lib/server/offline-shein-records-proxy";
export const dynamic="force-dynamic";
const authenticated = serverAuth(async (request: NextRequest & { auth?: unknown }) => proxyOfflineRecords(request, readZitadelServerAccessToken(request.auth as never), String(readZitadelIdentityFromSession(request.auth as never)?.userId ?? "")));
async function handle(request: NextRequest) {
  const controller = new AbortController(), abort = () => controller.abort();
  const failure = () => offlineFailure(504, request.method === "GET" ? "DEPENDENCY_UNAVAILABLE" : "OUTCOME_UNKNOWN");
  request.signal.addEventListener("abort", abort, { once: true }); if (request.signal.aborted) abort();
  const timer = setTimeout(abort, 15000); let finish = () => {};
  const ended = new Promise<Response>(resolve => { finish = () => resolve(failure()); controller.signal.addEventListener("abort", finish, { once: true }); if (controller.signal.aborted) finish(); });
  try { return await Promise.race([authenticated(new NextRequest(request, { signal: controller.signal }), { params: Promise.resolve({}) }), ended]) ?? offlineFailure(503, "DEPENDENCY_UNAVAILABLE"); }
  catch { return offlineFailure(503, request.method === "GET" ? "DEPENDENCY_UNAVAILABLE" : "OUTCOME_UNKNOWN"); }
  finally { clearTimeout(timer); request.signal.removeEventListener("abort", abort); controller.signal.removeEventListener("abort", finish); }
}
export const GET = handle, POST = handle;
export const PUT = () => offlineFailure(405, "INVALID_REQUEST");
export const PATCH = PUT, DELETE = PUT, OPTIONS = PUT, HEAD = PUT;
