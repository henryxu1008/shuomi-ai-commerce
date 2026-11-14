import {requireAuthenticatedUserId} from "@/lib/server/referral-page-auth";
import {SkillLibraryPage, type SkillCategory} from "./skill-library-page";

export async function SkillLibraryServerPage({category, returnTo, maintenance=false}: {category: SkillCategory; returnTo: string; maintenance?: boolean}) {
  const expectedUserId = await requireAuthenticatedUserId(returnTo);
  return <SkillLibraryPage expectedUserId={expectedUserId} category={category} maintenance={maintenance}/>;
}
