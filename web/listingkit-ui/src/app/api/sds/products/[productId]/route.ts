import { NextResponse } from "next/server";

import { serverAuth } from "@/auth";
import type { AuthenticatedUpstreamRequest } from "@/lib/server/upstream-identity";
import { fetchSDSJSON, sdsAPIErrorPayload } from "@/app/api/sds/shared";
import type { SDSProductDetail } from "@/lib/types/sds";

export const dynamic = "force-dynamic";

const authenticatedGET = serverAuth(async (
  request: AuthenticatedUpstreamRequest,
  { params }: { params: Promise<{ productId: string }> },
) => {
  const { productId } = await params;

  try {
    const { payload } = await fetchSDSJSON<SDSProductDetail>(
      request,
      request.auth,
      `/products/${productId}`,
    );
    return NextResponse.json(payload);
  } catch (error) {
    const payload = sdsAPIErrorPayload(error, "sds_product_detail_failed");
    return NextResponse.json(payload.body, { status: payload.status });
  }
});

export { authenticatedGET as GET };
