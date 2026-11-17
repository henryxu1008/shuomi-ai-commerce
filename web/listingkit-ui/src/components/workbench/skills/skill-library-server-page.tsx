import {requireAuthenticatedUserId} from "@/lib/server/referral-page-auth";
import {SkillLibraryPage, type SkillCategory} from "./skill-library-page";

export async function SkillLibraryServerPage({category, returnTo, maintenance=false, initialSkillId}: {category: SkillCategory; returnTo: string; maintenance?: boolean; initialSkillId?: string}) {
  const expectedUserId = await requireAuthenticatedUserId(returnTo);
  return <SkillLibraryPage expectedUserId={expectedUserId} category={category} maintenance={maintenance} initialSkillId={initialSkillId}/>;
}
