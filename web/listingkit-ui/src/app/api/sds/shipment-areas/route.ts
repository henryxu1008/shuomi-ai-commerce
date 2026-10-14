import { NextResponse } from "next/server";

import { serverAuth } from "@/auth";
import type { AuthenticatedUpstreamRequest } from "@/lib/server/upstream-identity";
import { fetchSDSJSON, sdsAPIErrorPayload } from "@/app/api/sds/shared";
import type { SDSShipmentArea } from "@/lib/types/sds";

export const dynamic = "force-dynamic";

const authenticatedGET = serverAuth(async (request: AuthenticatedUpstreamRequest) => {
  try {
    const { payload } = await fetchSDSJSON<SDSShipmentArea[]>(
      request,
      request.auth,
      "/shipment-areas",
    );
    return NextResponse.json(payload);
  } catch (error) {
    const payload = sdsAPIErrorPayload(error, "sds_shipment_area_query_failed");
    return NextResponse.json(payload.body, { status: payload.status });
  }
});

export { authenticatedGET as GET };
