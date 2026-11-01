import { ReportPage } from "@/components/workbench/reports/report-page";
import { ConsoleState } from "@/components/workbench/console/console-page";
import { isReportCenterAvailable, isReportTitleReviewAvailable, isReportSheinTargetAvailable } from "@/lib/server/report-center-availability";
import { reportTargetQuery } from "@/lib/contracts/report-center";
import { notFound } from "next/navigation";
export default async function Page({ searchParams }: { searchParams?: Promise<Record<string, string | string[] | undefined>> } = {}) {
  const initialTargetRecord = reportTargetQuery(await searchParams ?? {});
  if (initialTargetRecord === null) notFound();
  return isReportCenterAvailable() ? <ReportPage titleReviewAvailable={isReportTitleReviewAvailable()} sheinTargetSourcesAvailable={isReportSheinTargetAvailable()} initialTargetRecord={initialTargetRecord} /> : <ConsoleState kind="unavailable" title="我的报告尚未启用">当前安装尚未启用报告服务。</ConsoleState>;
}
