import { SheinDiagnosticPage } from "@/components/workbench/shein-diagnostic/diagnostic-page";

import {OfflineDiagnosticPanel} from "@/components/workbench/shein-diagnostic/offline-panel";
import {isOfflineSheinRecordsAvailable} from "@/lib/server/offline-shein-records-availability";
import {reportID} from "@/lib/contracts/report-center";
import {ConsoleState} from "@/components/workbench/console/console-page";
import {notFound} from "next/navigation";
export default async function Page({ params }: { params: Promise<{ record_id: string }> }) {
  const { record_id } = await params;
  if(!reportID.safeParse(record_id).success)notFound();
  if(process.env.LISTINGKIT_SHEIN_OFFLINE_RECORDS_ENABLED==="true")return isOfflineSheinRecordsAvailable()?<OfflineDiagnosticPanel recordId={record_id}/>:<ConsoleState kind="unavailable" title="离线诊断尚未就绪"/>;
  return <SheinDiagnosticPage recordId={record_id} />;
}
