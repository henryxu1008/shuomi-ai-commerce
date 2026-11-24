export const AMAZON_AUTHORIZATION_PATH = "/workbench/stores/amazon";

export const AMAZON_AUTHORIZATION_PRIVACY_HEADERS = {
  "Cache-Control": "no-store",
  "Referrer-Policy": "no-referrer",
} as const;

export function isAmazonAuthorizationCallback(pathname: string) {
  const exact = pathname.endsWith("/") ? pathname.slice(0, -1) : pathname;
  return exact === `${AMAZON_AUTHORIZATION_PATH}/login` || exact === `${AMAZON_AUTHORIZATION_PATH}/redirect`;
}

export function readAmazonConsentURL(value: unknown) {
  if (typeof value !== "string" || value.length > 1024) return "";
  try {
    const url = new URL(value);
    if (url.origin !== "https://sellercentral.amazon.com" || url.username || url.password || url.hash ||
        url.pathname !== "/apps/authorize/consent") return "";
    const id = url.searchParams.get("application_id") ?? "";
    return /^amzn1\.sellerapps\.app\.[a-f0-9-]{36}$/.test(id) ? url.href : "";
  } catch {
    return "";
  }
}
