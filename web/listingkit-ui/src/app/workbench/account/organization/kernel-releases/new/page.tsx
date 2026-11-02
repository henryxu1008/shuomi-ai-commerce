import { requireAccountUserId } from "@/lib/server/account-page-auth";
import { KernelReleasesPage } from "@/components/workbench/account/kernel-releases";
export default async function Page() { const user = await requireAccountUserId("/workbench/account/organization/kernel-releases/new"); return <KernelReleasesPage expectedUserId={user} create />; }
