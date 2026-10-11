export type SupportConversation = { id: number; status: string; createdAt: number };
export type SupportMessage = { id: number; content: string; type: "incoming" | "outgoing"; createdAt: number; attachments: { id: number; name: string; url: string }[] };
export type SupportHistory = { subject: string; messages: SupportMessage[]; nextBefore: number | null };
export type SupportCollection = { subject: string; conversations: SupportConversation[] };
export function isSupportSubject(value: unknown): value is string { return typeof value === "string" && /^platform:[a-f0-9]{64}$/.test(value); }
