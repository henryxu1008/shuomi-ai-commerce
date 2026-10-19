import {expect,it} from "vitest";
import {parseSkillLaunch,skillLaunchHref,skillLaunchMatches} from "./skill-launch";
const launch={userId:"alice",organizationId:"org",target:"product-image" as const,selection:{skillId:"11111111-1111-4111-8111-111111111111",version:2,digest:"a".repeat(64)}};
it("round trips exact selection and scope without touching original request fields",()=>{
 const url=new URL(skillLaunchHref("/workbench/example?agent_key=original&preparation=source",launch),"https://test");
 expect(url.searchParams.get("agent_key")).toBe("original");expect(parseSkillLaunch(url.searchParams)).toEqual(launch);
 expect(skillLaunchMatches(launch,{userId:"alice",organizationId:"org"},"product-image")).toBe(true);
 expect(skillLaunchMatches(launch,{userId:"alice",organizationId:"other"},"product-image")).toBe(false);
 expect(skillLaunchMatches(launch,{userId:"other",organizationId:"org"},"product-image")).toBe(false);
 expect(skillLaunchMatches(launch,{userId:"alice",organizationId:"org"},"product-title")).toBe(false);
});
it("fails closed on partial, duplicate, malformed or ambiguous launch hints",()=>{
 expect(parseSkillLaunch(new URLSearchParams())).toBeUndefined();
 for(const suffix of ["&skill_version=3","&skill_id=bad"]){const params=new URL(skillLaunchHref("/",launch),"https://test").searchParams;params.append(...suffix.slice(1).split("=") as [string,string]);expect(parseSkillLaunch(params)).toEqual({invalid:true})}
 const params=new URL(skillLaunchHref("/",launch),"https://test").searchParams;params.delete("skill_digest");expect(parseSkillLaunch(params)).toEqual({invalid:true});
});
