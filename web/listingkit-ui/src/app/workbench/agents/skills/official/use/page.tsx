import {connection} from "next/server";
import {notFound} from "next/navigation";
import {parseSkillLaunchRecord} from "@/lib/contracts/skill-launch";
import {SkillUsePage} from "@/components/workbench/skills/skill-use-page";
export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){
 await connection();if(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true")notFound();
 return <SkillUsePage launch={parseSkillLaunchRecord(await searchParams)} titleAvailable={process.env.LISTINGKIT_PRODUCT_AGENT_ENABLED==="true"} imageAvailable={process.env.LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED==="true"}/>;
}
