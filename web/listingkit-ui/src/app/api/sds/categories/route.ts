import { NextResponse } from "next/server";

import { serverAuth } from "@/auth";
import type { AuthenticatedUpstreamRequest } from "@/lib/server/upstream-identity";
import { fetchSDSJSON, sdsAPIErrorPayload } from "@/app/api/sds/shared";
import type { SDSCategory } from "@/lib/types/sds";

export const dynamic = "force-dynamic";

const authenticatedGET = serverAuth(async (request: AuthenticatedUpstreamRequest) => {
  const shipmentArea = request.nextUrl.searchParams.get("shipmentArea") ?? "US";

  try {
    const { payload } = await fetchSDSJSON<SDSCategory[]>(
      request,
      request.auth,
      "/categories",
      new URLSearchParams({ shipmentArea }),
    );
    return NextResponse.json(payload);
  } catch (error) {
    const payload = sdsAPIErrorPayload(error, "sds_category_query_failed");
    return NextResponse.json(payload.body, { status: payload.status });
  }
});

export { authenticatedGET as GET };
