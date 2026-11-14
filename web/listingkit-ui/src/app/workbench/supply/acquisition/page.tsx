import { connection } from "next/server";
import { notFound } from "next/navigation";
import { OnlineAcquisitionHub } from "@/components/workbench/acquisition/online-acquisition-hub";
import { isProductAcquisitionAvailable } from "@/lib/server/product-acquisition-availability";

export default async function Page() {
  await connection();
  const has1688=isProductAcquisitionAvailable(),hasAmazon=process.env.LISTINGKIT_DATA_SERVICES_ENABLED === "true";
  if (!has1688 && !hasAmazon) notFound();
  return <OnlineAcquisitionHub has1688={has1688} hasAmazon={hasAmazon} initial={has1688?"1688":"amazon"} plansEnabled={process.env.LISTINGKIT_ACQUISITION_PLANS_ENABLED === "true"} />;
}
