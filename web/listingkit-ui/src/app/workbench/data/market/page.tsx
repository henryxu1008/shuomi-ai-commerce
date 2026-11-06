import { connection } from "next/server";
import { notFound } from "next/navigation";
import { DataServicesPage } from "@/components/workbench/data-services/data-services-page";
import { amazonRealtimeAcquisitionAnchor } from "@/lib/workbench/console-navigation";

export default async function Page({ searchParams }: { searchParams: Promise<{ focus?: string | string[] }> }) {
  await connection();
  if (process.env.LISTINGKIT_DATA_SERVICES_ENABLED !== "true") notFound();
  const { focus } = await searchParams;
  return <DataServicesPage mode="market" focusRealtime={focus === amazonRealtimeAcquisitionAnchor} />;
}
