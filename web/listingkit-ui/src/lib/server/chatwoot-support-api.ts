import { NextRequest } from "next/server";
import { serverAuth } from "@/auth";
import { readBoundedStrictJSON } from "@/lib/api/strict-json-response";
import type { SupportConversation, SupportMessage } from "@/lib/contracts/platform-support";
import { readChatwootConfig, supportSubject } from "./chatwoot-support";
import { hasTrustedSameOriginWrite } from "./same-origin-write";
import { authorizeZitadelIdentity, getZitadelAuthOptions, readZitadelIdentityFromSession, readZitadelSessionClientID, readZitadelSessionError, readZitadelSessionIssuerURL } from "./zitadel-auth";
import { readZitadelServerAccessToken } from "./zitadel-server-token";

const imageLimit = 5 * 1024 * 1024;
const noStore = { "Cache-Control": "no-store" };
class SupportFailure extends Error { constructor(readonly status: number, readonly code: string) { super(code); } }
const object = (value: unknown): Record<string, unknown> => { if (!value || typeof value !== "object" || Array.isArray(value)) throw new SupportFailure(502, "SUPPORT_UNAVAILABLE"); return value as Record<string, unknown>; };
const list = (value: unknown) => { if (!Array.isArray(value)) throw new SupportFailure(502, "SUPPORT_UNAVAILABLE"); return value.map(object); };
const id = (value: unknown) => Number.isSafeInteger(Number(value)) && Number(value) > 0 ? Number(value) : 0;
const json = (value: unknown, status = 200) => Response.json(value, { status, headers: noStore });

async function boundedBytes(response: Response, limit: number, signal: AbortSignal) {
  if (!response.body) return new Uint8Array();
  const reader = response.body.getReader(); const chunks: Uint8Array[] = []; let size = 0;
  const abort = () => { void reader.cancel().catch(() => undefined); };
  signal.addEventListener("abort", abort, { once: true });
  try {
    while (true) { if (signal.aborted) throw new SupportFailure(504, "SUPPORT_TIMEOUT"); const result = await reader.read(); if (result.done) break; size += result.value.length; if (size > limit) { abort(); throw new SupportFailure(413, "SUPPORT_TOO_LARGE"); } chunks.push(result.value); }
    if (signal.aborted) throw new SupportFailure(504, "SUPPORT_TIMEOUT");
    const bytes = new Uint8Array(size); let offset = 0; for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.length; } return bytes;
  } finally { signal.removeEventListener("abort", abort); reader.releaseLock(); }
}

async function run(request: NextRequest, config: NonNullable<ReturnType<typeof readChatwootConfig>>, subject: string) {
  const url = new URL(request.url); const path = url.pathname.slice("/api/support/".length);
  if (request.method === "GET" && path === "session" && !url.search) return json({ subject });
  const allowedRead = path === "conversations" || /^conversations\/[1-9]\d*\/messages(?:\/[1-9]\d*\/attachments\/[1-9]\d*)?$/.test(path);
  if (!allowedRead || request.method === "POST" && path !== "conversations") throw new SupportFailure(404, "SUPPORT_NOT_FOUND");
  if ([...url.searchParams.keys()].some(key => key !== "before") || url.searchParams.has("before") && (!id(url.searchParams.get("before")) || !/^conversations\/[1-9]\d*\/messages$/.test(path))) throw new SupportFailure(400, "INVALID_REQUEST");

  const api = `/api/v1/accounts/${config.accountId}`;
  async function provider(route: string, init: RequestInit = {}) {
    if (request.signal.aborted) throw new SupportFailure(504, "SUPPORT_TIMEOUT");
    const response = await fetch(`${config.origin}${api}${route}`, { ...init, cache: "no-store", redirect: "manual", signal: request.signal,
      headers: { Accept: "application/json", api_access_token: config.token, ...(typeof init.body === "string" ? { "Content-Type": "application/json" } : {}), ...init.headers } });
    if (!response.ok) { void response.body?.cancel().catch(() => undefined); throw new SupportFailure(response.status === 422 ? 422 : 502, "SUPPORT_UNAVAILABLE"); }
    return object(await readBoundedStrictJSON(response, 2 * 1024 * 1024, request.signal));
  }
  const inbox = await provider(`/inboxes/${config.inboxId}`);
  if (id(inbox.id) !== config.inboxId || inbox.channel_type !== "Channel::Api") throw new SupportFailure(503, "SUPPORT_NOT_CONFIGURED");
  async function findContact() {
    const result = await provider(`/contacts/search?q=${encodeURIComponent(subject)}`);
    const matches = list(result.payload).filter(contact => contact.identifier === subject);
    if (matches.length > 1 || matches.some(contact => !id(contact.id))) throw new SupportFailure(502, "SUPPORT_UNAVAILABLE");
    return matches[0] ?? null;
  }
  let contact = await findContact();
  const owned = (conversation: Record<string, unknown>) => Boolean(contact && id(conversation.id) && conversation.account_id === config.accountId && conversation.inbox_id === config.inboxId && object(conversation.meta).sender && object(object(conversation.meta).sender).id === contact.id && object(conversation.custom_attributes).platform_support_subject === subject);
  const toConversation = (value: Record<string, unknown>): SupportConversation => ({ id: id(value.id), status: String(value.status), createdAt: Number(value.created_at) || 0 });
  async function conversations() { if (!contact) return []; const result = await provider(`/contacts/${id(contact.id)}/conversations`); return list(result.payload).filter(owned); }
  async function ownConversation(conversationId: number) {
    if (!contact) throw new SupportFailure(404, "SUPPORT_NOT_FOUND");
    const result = await provider(`/conversations/${conversationId}`);
    if (!owned(result) || result.id !== conversationId) throw new SupportFailure(404, "SUPPORT_NOT_FOUND"); return result;
  }
  if (request.method === "GET" && path === "conversations") return json({ subject, conversations: (await conversations()).map(toConversation) });

  if (request.method === "POST") {
    const contentType = request.headers.get("content-type") ?? "";
    if (!contentType.startsWith("multipart/form-data;")) throw new SupportFailure(400, "INVALID_REQUEST");
    const bytes = await boundedBytes(new Response(request.body), imageLimit + 32768, request.signal);
    const form = await new Response(bytes, { headers: { "Content-Type": contentType } }).formData();
    if ([...form.keys()].some(key => !["subject", "conversationId", "content", "attachment"].includes(key)) || [...new Set(form.keys())].some(key => form.getAll(key).length !== 1)) throw new SupportFailure(400, "INVALID_REQUEST");
    if (form.get("subject") !== subject) throw new SupportFailure(409, "IDENTITY_CONTEXT_CHANGED");
    const content = form.get("content"); const attachment = form.get("attachment"); const selected = form.get("conversationId");
    if (typeof content !== "string" || content.length > 4000 || selected !== null && (typeof selected !== "string" || !id(selected))) throw new SupportFailure(400, "INVALID_REQUEST");
    if (attachment !== null && (!(attachment instanceof File) || !attachment.size || attachment.size > imageLimit || !["image/png", "image/jpeg", "image/webp"].includes(attachment.type))) throw new SupportFailure(400, "INVALID_IMAGE");
    if (!content.trim() && !attachment) throw new SupportFailure(400, "INVALID_REQUEST");
    if (!contact) {
      if (selected) throw new SupportFailure(404, "SUPPORT_NOT_FOUND");
      try { const result = await provider("/contacts", { method: "POST", body: JSON.stringify({ identifier: subject, name: `平台用户 ${subject.slice(-8)}`, inbox_id: config.inboxId, source_id: subject }) }); contact = object(object(result.payload).contact); }
      catch (cause) { if (!(cause instanceof SupportFailure) || cause.status !== 422) throw cause; contact = await findContact(); }
      if (!contact || contact.identifier !== subject || !id(contact.id)) throw new SupportFailure(502, "SUPPORT_UNAVAILABLE");
    }
    let conversation = selected ? await ownConversation(id(selected)) : (await conversations())[0];
    if (!conversation) {
      const created = await provider("/conversations", { method: "POST", body: JSON.stringify({ inbox_id: config.inboxId, contact_id: contact.id, source_id: subject, custom_attributes: { platform_support_subject: subject } }) });
      if (!id(created.id)) throw new SupportFailure(502, "SUPPORT_UNAVAILABLE"); conversation = await ownConversation(id(created.id));
    }
    const outgoing = new FormData(); outgoing.set("content", content.trim()); outgoing.set("message_type", "incoming"); outgoing.set("private", "false");
    if (attachment instanceof File) outgoing.set("attachments[]", attachment, attachment.name);
    const sent = await provider(`/conversations/${id(conversation.id)}/messages`, { method: "POST", body: outgoing });
    if (!id(sent.id)) throw new SupportFailure(502, "SUPPORT_UNAVAILABLE");
    return json({ subject, conversation: toConversation(conversation), messageId: id(sent.id) });
  }

  const segments = path.split("/"); const conversationId = id(segments[1]);
  await ownConversation(conversationId);
  const messageId = segments.length > 3 ? id(segments[3]) : 0;
  const result = await provider(`/conversations/${conversationId}/messages${messageId ? `?before=${messageId + 1}` : url.search}`);
  const rawMessages = list(result.payload);
  const publicMessages = rawMessages.filter(message => message.private === false && [0, 1].includes(Number(message.message_type)) && message.conversation_id === conversationId);
  if (messageId) {
    const message = publicMessages.find(item => item.id === messageId); const attachmentId = id(segments[5]);
    const attachment = message && list(message.attachments ?? []).find(item => item.id === attachmentId);
    if (!attachment || typeof attachment.data_url !== "string") throw new SupportFailure(404, "SUPPORT_NOT_FOUND");
    let location = new URL(attachment.data_url);
    for (let step = 0; step < 3; step++) {
      if (!location.pathname.startsWith("/rails/active_storage/")) throw new SupportFailure(502, "SUPPORT_UNAVAILABLE");
      const response = await fetch(`${config.origin}${location.pathname}${location.search}`, { signal: request.signal, redirect: "manual", cache: "no-store" });
      if ([301, 302, 303, 307, 308].includes(response.status)) { const target = response.headers.get("location"); void response.body?.cancel().catch(() => undefined); if (!target) break; location = new URL(target, config.origin); continue; }
      if (!response.ok) { void response.body?.cancel().catch(() => undefined); break; }
      const mime = response.headers.get("content-type")?.split(";")[0];
      if (!mime || !["image/png", "image/jpeg", "image/webp"].includes(mime)) { void response.body?.cancel().catch(() => undefined); break; }
      const bytes = await boundedBytes(response, imageLimit, request.signal);
      const extension = mime === "image/png" ? "png" : mime === "image/jpeg" ? "jpg" : "webp";
      return new Response(bytes, { headers: { ...noStore, "Content-Type": mime, "X-Content-Type-Options": "nosniff", "Content-Disposition": `attachment; filename="support-image-${attachmentId}.${extension}"` } });
    }
    throw new SupportFailure(502, "SUPPORT_UNAVAILABLE");
  }
  const messages: SupportMessage[] = publicMessages.map(message => ({ id: id(message.id), content: typeof message.content === "string" ? message.content : "", type: Number(message.message_type) === 0 ? "incoming" : "outgoing", createdAt: Number(message.created_at) || 0,
    attachments: list(message.attachments ?? []).filter(attachment => id(attachment.id) && attachment.file_type === "image").map(attachment => ({ id: id(attachment.id), name: "截图", url: `/api/support/conversations/${conversationId}/messages/${id(message.id)}/attachments/${id(attachment.id)}` })) }));
  return json({ subject, messages, nextBefore: rawMessages.length >= 20 ? Math.min(...rawMessages.map(message => id(message.id)).filter(Boolean)) : null });
}

const authenticated = serverAuth(async request => {
  try {
    const config = readChatwootConfig(); const options = getZitadelAuthOptions();
    if (!config || !options) throw new SupportFailure(503, "SUPPORT_NOT_CONFIGURED");
    const identity = readZitadelIdentityFromSession(request.auth);
    if (!identity || !readZitadelServerAccessToken(request.auth) || readZitadelSessionError(request.auth) || readZitadelSessionIssuerURL(request.auth) !== options.issuerUrl || readZitadelSessionClientID(request.auth) !== options.clientId) throw new SupportFailure(401, "AUTHENTICATION_REQUIRED");
    if (!authorizeZitadelIdentity(identity).authorized) throw new SupportFailure(403, "PERMISSION_DENIED");
    if (request.signal.aborted) throw new SupportFailure(504, "SUPPORT_TIMEOUT");
    return await run(request, config, supportSubject(options.issuerUrl, identity));
  } catch (cause) { return json({ code: request.signal.aborted ? "SUPPORT_TIMEOUT" : cause instanceof SupportFailure ? cause.code : "SUPPORT_UNAVAILABLE" }, request.signal.aborted ? 504 : cause instanceof SupportFailure ? cause.status : 502); }
});

export async function handleSupportRequest(request: NextRequest) {
  if (request.method === "POST" && !hasTrustedSameOriginWrite(request)) return json({ code: "PERMISSION_DENIED" }, 403);
  const length = request.headers.get("content-length");
  if (length !== null && (!/^\d+$/.test(length) || Number(length) > imageLimit + 32768)) return json({ code: "SUPPORT_TOO_LARGE" }, 413);
  const controller = new AbortController(); const abort = () => controller.abort();
  request.signal.addEventListener("abort", abort, { once: true }); if (request.signal.aborted) abort();
  let finish = () => {}; const deadline = new Promise<Response>(resolve => { finish = () => resolve(json({ code: "SUPPORT_TIMEOUT" }, 504)); controller.signal.addEventListener("abort", finish, { once: true }); });
  const timer = setTimeout(abort, 20_000);
  try { return await Promise.race([authenticated(new NextRequest(request, { signal: controller.signal }), { params: Promise.resolve({}) }), deadline]) ?? json({ code: "SUPPORT_UNAVAILABLE" }, 502); }
  catch { return json({ code: "SUPPORT_UNAVAILABLE" }, 502); }
  finally { clearTimeout(timer); request.signal.removeEventListener("abort", abort); controller.signal.removeEventListener("abort", finish); }
}
