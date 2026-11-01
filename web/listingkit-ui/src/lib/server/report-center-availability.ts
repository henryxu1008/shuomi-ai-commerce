import { configuredProductReviewOrigin } from "./product-title-review-request";

// Deployment availability is independent of the live Go authorization decision.
export function isReportCenterAvailable() {
  return process.env.LISTINGKIT_REPORT_CENTER_ENABLED === "true";
}

export function isReportSheinTargetAvailable() {
  if (!isReportCenterAvailable() || process.env.LISTINGKIT_REPORT_SHEIN_TARGET_SOURCES_ENABLED !== "true" || process.env.LISTINGKIT_SUPPLY_CHAIN_ENABLED !== "true") return false;
  try {
    const base = new URL(process.env.LISTINGKIT_SERVICE_API_BASE ?? "");
    return ["http:", "https:"].includes(base.protocol) && !base.username && !base.password && !base.search && !base.hash && ["/api/v1", "/api/v1/"].includes(base.pathname);
  } catch { return false; }
}

// The normal Report reader consumes the Review owner mounted in this application.
export function isReportTitleReviewAvailable() {
  const reviewOrigin = configuredProductReviewOrigin();
  if (!reviewOrigin) return false;
  try {
    const base = new URL(process.env.LISTINGKIT_SERVICE_API_BASE ?? "");
    return !base.username && !base.password && !base.search && !base.hash &&
      ["/api/v1", "/api/v1/"].includes(base.pathname) && base.origin === reviewOrigin;
  } catch { return false; }
}
