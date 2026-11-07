import { NotificationCenterPage } from "@/components/workbench/notifications/notification-center";
import { requireAuthenticatedSubject } from "@/lib/server/referral-page-auth";
import { isNotificationCenterAvailable } from "@/lib/server/notification-availability";
export default async function Page() {
  const subject = await requireAuthenticatedSubject("/workbench/notifications");
  return isNotificationCenterAvailable() ? <NotificationCenterPage expectedUserId={subject.userId} realm={subject.realm} /> : <section className="console-page"><h1 className="text-3xl font-semibold">通知中心</h1><p className="mt-4 text-muted-foreground">当前环境尚未启用通知中心。</p></section>;
}
