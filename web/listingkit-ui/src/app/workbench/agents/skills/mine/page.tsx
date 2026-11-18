import {SkillLibraryServerPage} from "@/components/workbench/skills/skill-library-server-page";
import {isAcquisitionUUID} from "@/lib/contracts/product-acquisition";

export default async function Page({searchParams}: {searchParams: Promise<{skillId?: string | string[]}>}) {
  const {skillId} = await searchParams;
  const initialSkillId = typeof skillId === "string" && isAcquisitionUUID(skillId) ? skillId : undefined;
  return <SkillLibraryServerPage category="mine" returnTo="/workbench/agents/skills/mine" initialSkillId={initialSkillId}/>;
}
