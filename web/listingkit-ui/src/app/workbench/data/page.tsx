import { DataServicesOverview } from "@/components/workbench/data-services/data-services-overview";
import { consolePrimaryDeploymentCapabilities } from "@/lib/server/console-primary-availability";

export default function Page() {
  return <DataServicesOverview capabilities={consolePrimaryDeploymentCapabilities()} />;
}
