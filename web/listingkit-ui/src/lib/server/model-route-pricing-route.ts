import { NextRequest } from "next/server";
import { serverAuth } from "@/auth";
import { readZitadelIdentityFromSession } from "./zitadel-auth";
import { readZitadelServerAccessToken } from "./zitadel-server-token";
import {
  proxyModelRoutePricing,
  routeFailure,
} from "./model-route-pricing-proxy";
const authenticated = serverAuth(
  async (request: NextRequest & { auth?: unknown }) => {
    const id = readZitadelIdentityFromSession(request.auth as never);
    return proxyModelRoutePricing(
      request,
      readZitadelServerAccessToken(request.auth as never),
      String(id?.userId ?? ""),
    );
  },
);
export async function handleModelRoutePricing(request: NextRequest) {
  const controller = new AbortController(),
    abort = () => controller.abort();
  request.signal.addEventListener("abort", abort, { once: true });
  if (request.signal.aborted) abort();
  const timer = setTimeout(abort, 28000);
  let finish = () => {};
  const failure = () =>
    routeFailure(
      504,
      request.method === "GET" ? "DEPENDENCY_UNAVAILABLE" : "OUTCOME_UNKNOWN",
    );
  const ended = new Promise<Response>((resolve) => {
    finish = () => resolve(failure());
    controller.signal.addEventListener("abort", finish, { once: true });
    if (controller.signal.aborted) finish();
  });
  try {
    return (
      (await Promise.race([
        authenticated(new NextRequest(request, { signal: controller.signal }), {
          params: Promise.resolve({}),
        }),
        ended,
      ])) ?? routeFailure(503, "DEPENDENCY_UNAVAILABLE")
    );
  } catch {
    return routeFailure(
      503,
      request.method === "GET" ? "DEPENDENCY_UNAVAILABLE" : "OUTCOME_UNKNOWN",
    );
  } finally {
    clearTimeout(timer);
    request.signal.removeEventListener("abort", abort);
    controller.signal.removeEventListener("abort", finish);
  }
}
export const rejectModelRoutePricing = () =>
  routeFailure(405, "INVALID_REQUEST");
