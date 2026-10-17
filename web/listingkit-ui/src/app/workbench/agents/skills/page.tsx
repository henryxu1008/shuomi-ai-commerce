import {PrimaryIntroduction} from "@/components/workbench/console/primary-introduction";
import {requireAuthenticatedUserId} from "@/lib/server/referral-page-auth";

export default async function Page() {
  await requireAuthenticatedUserId("/workbench/agents/skills");
  return <PrimaryIntroduction kind="skills"/>;
}
