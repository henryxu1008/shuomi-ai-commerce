import { SupplyOverview } from "@/components/workbench/supply-market/supply-overview";
import { consolePrimaryDeploymentCapabilities } from "@/lib/server/console-primary-availability";

export default function Page() {
  const capabilities = consolePrimaryDeploymentCapabilities();
  return <SupplyOverview marketAvailable={capabilities.supplyMarketAvailable} preparationAvailable={capabilities.supplyPreparationAvailable} />;
}
