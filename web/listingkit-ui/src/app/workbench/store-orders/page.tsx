import { ObservationPage } from "@/components/workbench/store-observations/observation-page";
import { FulfillmentEntry } from "@/components/workbench/fulfillment/fulfillment-page";
export default function StoreOrdersPage() {
  return <><FulfillmentEntry/><ObservationPage kind="orders" /></>;
}
