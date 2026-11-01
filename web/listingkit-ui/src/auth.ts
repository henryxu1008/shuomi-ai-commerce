import { Auth } from "@auth/core";
import NextAuth from "next-auth";

import { buildAuthConfig, buildServerAuthConfig } from "@/auth.config";

const publicConfig = buildAuthConfig();
// NextAuth applies its environment/basePath defaults to this same config.
const publicAuth = NextAuth(publicConfig);
const serverOnlyAuth = NextAuth(buildServerAuthConfig());

// NextAuth's reqWithEnvURL constructs another NextRequest, which normalizes
// 127/8 to localhost even when AUTH_URL explicitly selects a different host.
// Use the official Auth.js Web Request entry point so authorization and token
// exchange use the same configured callback URI. Native checks/cookies remain
// owned by Auth.js; request headers never override a configured origin.
function publicHandler(request: Request) {
  const configuredURL = process.env.AUTH_URL ?? process.env.NEXTAUTH_URL;
  if (!configuredURL) {
    return Auth(request, publicConfig);
  }
  const url = new URL(request.url);
  const canonicalURL = new URL(configuredURL);
  canonicalURL.pathname = url.pathname;
  canonicalURL.search = url.search;
  canonicalURL.hash = "";
  return Auth(new Request(canonicalURL, request), publicConfig);
}

export const handlers = { GET: publicHandler, POST: publicHandler };
export const { signIn, signOut } = publicAuth;
export const { auth: serverAuth } = serverOnlyAuth;
