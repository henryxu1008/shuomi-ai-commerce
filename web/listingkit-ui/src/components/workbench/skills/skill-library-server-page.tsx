import {requireAuthenticatedUserId} from "@/lib/server/referral-page-auth";
import {SkillLibraryPage, type SkillCategory} from "./skill-library-page";

export async function SkillLibraryServerPage({category, returnTo}: {category: SkillCategory; returnTo: string}) {
  const expectedUserId = await requireAuthenticatedUserId(returnTo);
  return <SkillLibraryPage expectedUserId={expectedUserId} category={category}/>;
}
