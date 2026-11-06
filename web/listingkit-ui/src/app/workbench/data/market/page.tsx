import { connection } from "next/server";
import { notFound, redirect } from "next/navigation";
import { DataServicesPage } from "@/components/workbench/data-services/data-services-page";
import { amazonRealtimeAcquisitionAnchor, amazonRealtimeAcquisitionHref } from "@/lib/workbench/console-navigation";

export default async function Page({ searchParams }: { searchParams: Promise<{ focus?: string | string[] }> }) {
  await connection();
  if (process.env.LISTINGKIT_DATA_SERVICES_ENABLED !== "true") notFound();
  const { focus } = await searchParams;
  if (focus === amazonRealtimeAcquisitionAnchor) redirect(amazonRealtimeAcquisitionHref);
  return <DataServicesPage mode="market" />;
}
