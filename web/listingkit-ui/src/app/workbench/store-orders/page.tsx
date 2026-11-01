import { OrderHomepage } from "@/components/workbench/fulfillment/order-homepage";
import { configuredOrderBusinessCase } from "@/lib/server/order-business-case";
export const dynamic = "force-dynamic";
export default function StoreOrdersPage() {
  return <OrderHomepage businessCase={configuredOrderBusinessCase()} />;
}
