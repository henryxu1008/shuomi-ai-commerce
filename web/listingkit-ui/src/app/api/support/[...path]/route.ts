import { handleSupportRequest } from "@/lib/server/chatwoot-support-api";
export const dynamic = "force-dynamic";
export const runtime = "nodejs";
export const GET = handleSupportRequest;
export const POST = handleSupportRequest;
