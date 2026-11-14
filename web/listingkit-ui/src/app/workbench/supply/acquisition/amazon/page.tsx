import { notFound } from "next/navigation";
import { connection } from "next/server";
import { OnlineAcquisitionHub } from "@/components/workbench/acquisition/online-acquisition-hub";
import { isProductAcquisitionAvailable } from "@/lib/server/product-acquisition-availability";

export default async function Page() {
  await connection();
  if (process.env.LISTINGKIT_DATA_SERVICES_ENABLED !== "true") notFound();
  return <OnlineAcquisitionHub has1688={isProductAcquisitionAvailable()} hasAmazon initial="amazon" plansEnabled={process.env.LISTINGKIT_ACQUISITION_PLANS_ENABLED === "true"} />;
}
