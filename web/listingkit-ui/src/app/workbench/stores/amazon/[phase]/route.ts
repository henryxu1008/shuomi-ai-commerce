import type { NextRequest } from "next/server";

import { serverAuth } from "@/auth";
import { proxyAmazonAuthorization } from "@/lib/server/amazon-authorization-proxy";
import { readZitadelIdentityFromSession, readZitadelSessionError } from "@/lib/server/zitadel-auth";
import { readZitadelServerAccessToken } from "@/lib/server/zitadel-server-token";

export const dynamic = "force-dynamic";

type RouteContext = { params: Promise<{ phase: string }> };

const route = serverAuth(async (request, context) => {
  const { phase } = await (context as unknown as RouteContext).params;
  const identity = readZitadelSessionError(request.auth) ? null : readZitadelIdentityFromSession(request.auth);
  return proxyAmazonAuthorization(request, phase, {
    actor: typeof identity?.userId === "string" ? identity.userId : "",
    accessToken: readZitadelServerAccessToken(request.auth as never),
  });
}) as (request: NextRequest, context: RouteContext) => Response | void | Promise<Response | void>;

export const GET = route;
export const POST = route;
