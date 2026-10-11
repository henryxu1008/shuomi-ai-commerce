import { createHash } from "node:crypto";
import { closeSync, openSync, readSync } from "node:fs";
import type { ZitadelVerifiedIdentity } from "./zitadel-auth";
import { resolvePublicAppOrigin } from "./zitadel-auth";

export function readChatwootConfig() {
  try {
    const raw = process.env.CHATWOOT_ORIGIN?.trim() ?? "";
    const url = new URL(raw);
    const privateHost = ["localhost", "127.0.0.1", "[::1]", "chatwoot-rails"].includes(url.hostname);
    if (raw !== url.origin || url.username || url.password || !(url.protocol === "https:" || url.protocol === "http:" && privateHost) || url.origin === new URL(resolvePublicAppOrigin()).origin) return null;
    const accountId = Number(process.env.CHATWOOT_ACCOUNT_ID);
    const inboxId = Number(process.env.CHATWOOT_INBOX_ID);
    if (![accountId, inboxId].every(id => Number.isSafeInteger(id) && id > 0)) return null;
    const secretFile = process.env.CHATWOOT_API_TOKEN_FILE?.trim();
    const inlineSecret = process.env.CHATWOOT_API_TOKEN?.trim();
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
    if (secret.length < 16 || secret.length > 4096 || !/^[A-Za-z0-9._-]+$/.test(secret)) return null;
    return { origin: url.origin, accountId, inboxId, token: secret };
  } catch { return null; }
}

export function isChatwootSupportAvailable() { return readChatwootConfig() !== null; }

export function supportSubject(issuer: string, identity: ZitadelVerifiedIdentity) {
  return `platform:${createHash("sha256").update(JSON.stringify([issuer, String(identity.tenantId), String(identity.userId)])).digest("hex")}`;
}
