// Shared current public-origin resolution, independent of authentication setup.
export function resolvePublicAppOrigin() {
  const configuredOrigin =
    process.env.LISTINGKIT_PUBLIC_BASE_URL?.trim() ||
    process.env.TASK_PROCESSOR_LISTINGKIT_PUBLIC_BASE_URL?.trim() ||
    process.env.NEXT_PUBLIC_APP_URL?.trim() ||
    process.env.APP_URL?.trim();

  if (configuredOrigin) {
    return configuredOrigin.replace(/\/+$/, "");
  }

  return "http://localhost:3000";
}
