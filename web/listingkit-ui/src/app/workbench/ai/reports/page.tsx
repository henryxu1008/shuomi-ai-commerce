import { ReportPage } from "@/components/workbench/reports/report-page";
import { ConsoleState } from "@/components/workbench/console/console-page";
import { isReportCenterAvailable, isReportTitleReviewAvailable, isReportSheinTargetAvailable } from "@/lib/server/report-center-availability";
import {isOfflineSheinRecordsAvailable} from "@/lib/server/offline-shein-records-availability";
import { reportTargetQuery,reportOfflineQuery } from "@/lib/contracts/report-center";
import { notFound } from "next/navigation";
export default async function Page({ searchParams }: { searchParams?: Promise<Record<string, string | string[] | undefined>> } = {}) {
  const query=await searchParams??{},initialOfflineRecord=reportOfflineQuery(query);
  const initialTargetRecord=initialOfflineRecord?undefined:reportTargetQuery(query);
  if (initialTargetRecord === null||initialOfflineRecord===null) notFound();
  return isReportCenterAvailable() ? <ReportPage titleReviewAvailable={isReportTitleReviewAvailable()} sheinTargetSourcesAvailable={isReportSheinTargetAvailable()} initialTargetRecord={initialTargetRecord} sheinOfflineSourcesAvailable={isOfflineSheinRecordsAvailable()} initialOfflineRecord={initialOfflineRecord} /> : <ConsoleState kind="unavailable" title="我的报告尚未启用">当前安装尚未启用报告服务。</ConsoleState>;
}
