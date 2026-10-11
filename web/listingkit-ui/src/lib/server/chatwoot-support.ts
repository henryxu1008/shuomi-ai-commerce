import { createHash } from "node:crypto";
import { closeSync, openSync, readSync } from "node:fs";
import type { ZitadelVerifiedIdentity } from "./zitadel-auth";
import { resolvePublicAppOrigin } from "./zitadel-auth";

export function readChatwootConfig() {
  try {
    const raw = process.env.CHATWOOT_ORIGIN?.trim() ?? "";
    const url = new URL(raw);
    const loopback = ["localhost", "127.0.0.1", "[::1]"].includes(url.hostname);
    if (raw !== url.origin || url.username || url.password || !(url.protocol === "https:" || url.protocol === "http:" && loopback) || url.origin === new URL(resolvePublicAppOrigin()).origin) return null;
    const websiteToken = process.env.CHATWOOT_WEBSITE_TOKEN?.trim() ?? "";
    if (!/^[A-Za-z0-9_-]{8,256}$/.test(websiteToken)) return null;
    const secretFile = process.env.CHATWOOT_IDENTITY_SECRET_FILE?.trim();
    const inlineSecret = process.env.CHATWOOT_IDENTITY_SECRET?.trim();
    if (secretFile && inlineSecret) return null;
    let secret = inlineSecret ?? "";
    if (secretFile) {
      const fd = openSync(secretFile, "r");
      try {
        const bytes = Buffer.alloc(4097);
        const size = readSync(fd, bytes, 0, bytes.length, null);
        if (size > 4096) return null;
        secret = bytes.subarray(0, size).toString("utf8").trim();
      } finally { closeSync(fd); }
    }
    // Chatwoot has_secure_token generates 24-character inbox HMAC secrets.
    if (secret.length < 24 || secret.length > 4096 || /[\r\n\u0000]/.test(secret)) return null;
    return { origin: url.origin, websiteToken, secret };
  } catch { return null; }
}

export function isChatwootSupportAvailable() { return readChatwootConfig() !== null; }

export function supportSubject(issuer: string, identity: ZitadelVerifiedIdentity) {
  return `platform:${createHash("sha256").update(JSON.stringify([issuer, String(identity.tenantId), String(identity.userId)])).digest("hex")}`;
}
