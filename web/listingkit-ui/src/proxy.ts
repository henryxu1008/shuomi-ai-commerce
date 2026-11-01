import { NextRequest, NextResponse } from "next/server";
import type { Session } from "next-auth";

import { serverAuth } from "@/auth";
import {
  isZitadelAuthConfigured,
  normalizeReturnTo,
  readZitadelIdentityFromSession,
  readZitadelSessionError,
} from "@/lib/server/zitadel-auth";
import { readZitadelServerAccessToken } from "@/lib/server/zitadel-server-token";
import { resolvePublicAppOrigin } from "@/lib/server/public-app-origin";

type AuthenticatedProxyRequest = NextRequest & {
  auth?: unknown;
};

const authenticatedProxy = serverAuth(async (request) =>
  handleProxy(request as AuthenticatedProxyRequest),
);

export { authenticatedProxy as proxy };
export default authenticatedProxy;

async function handleProxy(request: AuthenticatedProxyRequest) {
  if (!isProtectedPagePath(request.nextUrl.pathname)) {
    return NextResponse.next();
  }

  if (!isZitadelAuthConfigured()) {
    return NextResponse.json(
      { error: "ZITADEL auth is not configured" },
      { status: 503 },
    );
  }

  const session = (request.auth ?? null) as Session | null;
  const accessToken = readZitadelServerAccessToken(session);
  const sessionError = readZitadelSessionError(session);
  if (!accessToken || sessionError) {
    return redirectToZitadelLogin(request);
  }

  const identity = readZitadelIdentityFromSession(session);
  if (!identity) {
    return redirectToZitadelLogin(request);
  }

  return NextResponse.next();
}

function isWorkbenchPagePath(pathname: string) {
  return pathname === "/workbench" || pathname.startsWith("/workbench/");
}

function isProtectedPagePath(pathname: string) {
  return isWorkbenchPagePath(pathname);
}

function redirectToZitadelLogin(request: NextRequest) {
  const loginUrl = new URL(
    "/login",
    (process.env.AUTH_URL ?? process.env.NEXTAUTH_URL)?.trim() || resolvePublicAppOrigin(),
  );
  loginUrl.searchParams.set("returnTo", buildReturnTo(request));
  return NextResponse.redirect(loginUrl);
}

function buildReturnTo(request: NextRequest) {
  return normalizeReturnTo(
    `${request.nextUrl.pathname}${request.nextUrl.search}`,
  );
}

export const config = {
  matcher: ["/workbench/:path*"],
};
