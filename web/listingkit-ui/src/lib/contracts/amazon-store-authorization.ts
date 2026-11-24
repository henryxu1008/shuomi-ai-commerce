export const AMAZON_AUTHORIZATION_PATH = "/workbench/stores/amazon";

export const AMAZON_AUTHORIZATION_PRIVACY_HEADERS = {
  "Cache-Control": "no-store",
  "Referrer-Policy": "no-referrer",
} as const;

export function isAmazonAuthorizationCallback(pathname: string) {
  const exact = pathname.endsWith("/") ? pathname.slice(0, -1) : pathname;
  return exact === `${AMAZON_AUTHORIZATION_PATH}/login` || exact === `${AMAZON_AUTHORIZATION_PATH}/redirect`;
}