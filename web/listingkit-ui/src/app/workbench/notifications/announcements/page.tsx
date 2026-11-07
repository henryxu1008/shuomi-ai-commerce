import { OfficialAnnouncementsPage } from "@/components/workbench/notifications/official-announcements";
import { ConsolePage, ConsoleState } from "@/components/workbench/console/console-page";
import { isNotificationCenterAvailable } from "@/lib/server/notification-availability";
import { requireAuthenticatedSubject } from "@/lib/server/referral-page-auth";

export default async function Page() {
  const subject = await requireAuthenticatedSubject("/workbench/notifications/announcements");
  if (!isNotificationCenterAvailable() || !subject.realm) {
    return <ConsolePage title="官方公告管理" breadcrumbs={[{ label: "通知中心", href: "/workbench/notifications" }, { label: "官方公告管理" }]}>
      <ConsoleState kind="unavailable" title="当前环境尚未启用官方公告管理" />
    </ConsolePage>;
  }
  return <OfficialAnnouncementsPage key={JSON.stringify(subject)} scope={{ ...subject, channel: "platform/official" }} />;
}
