import { requireAccountUserId } from "@/lib/server/account-page-auth";
import { ModelRoutePricingPage } from "@/components/workbench/account/model-route-pricing";
import { modelServicesPath, readModelServicesNavigation } from "@/lib/contracts/model-services-navigation";

export default async function Page({ searchParams }: { searchParams: Promise<Record<string, string | string[] | undefined>> }) {
  const user = await requireAccountUserId(modelServicesPath);
  return <ModelRoutePricingPage expectedUserId={user} navigation={readModelServicesNavigation(await searchParams)} />;
}
