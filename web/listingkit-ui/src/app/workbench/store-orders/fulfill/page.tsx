import { Suspense } from "react";
import { FulfillmentPage } from "@/components/workbench/fulfillment/fulfillment-page";
export default function Page() {
  return (
    <Suspense fallback={<p>正在加载履约工作页…</p>}>
      <FulfillmentPage />
    </Suspense>
  );
}
