import { ConsoleState } from "@/components/workbench/console/console-page";
import { EcoservicesOverview } from "@/components/workbench/ecoservices/overview";
import { isEcoservicesAvailable, isEcoservicesNonPaymentOnly } from "@/lib/server/ecoservices-availability";

export default function Page() {
  return isEcoservicesAvailable()
    ? <EcoservicesOverview nonPaymentOnly={isEcoservicesNonPaymentOnly()} />
    : <ConsoleState kind="unavailable" title="生态服务尚未开放" />;
}
