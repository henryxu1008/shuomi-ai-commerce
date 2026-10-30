import {nativeRunDestinationAvailability} from "@/lib/server/native-run-destination-availability";
import {isAgentConfigurationAvailable} from "@/lib/server/agent-configuration-origin";
import { BusinessTaskPage } from "@/components/workbench/ai-workbench/task-page";
import { isSheinRecordsAvailable } from "@/lib/server/shein-records-availability";
import { connection } from "next/server";

export default async function Page() {
  await connection();
  return <BusinessTaskPage nativeRunDestinations={nativeRunDestinationAvailability()} nativeRunsAvailable={isAgentConfigurationAvailable()} mode="completed" sheinRecordsAvailable={isSheinRecordsAvailable()} />;
}
