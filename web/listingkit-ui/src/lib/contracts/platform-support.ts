export type SupportSession = { origin: string; websiteToken: string; subject: string };

export function parseSupportSession(value: unknown): SupportSession | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  const { origin, websiteToken, subject } = value as Record<string, unknown>;
  if (typeof origin !== "string" || typeof websiteToken !== "string" || typeof subject !== "string") return null;
  try {
    const url = new URL(origin);
    if (url.origin !== origin || !["http:", "https:"].includes(url.protocol) || url.username || url.password || !/^[A-Za-z0-9_-]{8,256}$/.test(websiteToken) || !/^platform:[a-f0-9]{64}$/.test(subject)) return null;
    return { origin, websiteToken, subject };
  } catch { return null; }
}
