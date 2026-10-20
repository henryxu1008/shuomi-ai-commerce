import {z} from "zod";
import {skillSelectionSchema,skillTargetSchema} from "./skill";

// Navigation hints only. Candidates and execution still authorize the exact ref.
const schema=z.strictObject({userId:z.string().regex(/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/),organizationId:z.string().regex(/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/),target:skillTargetSchema,selection:skillSelectionSchema});
export type SkillLaunch=z.infer<typeof schema>;
export type SkillLaunchInput=SkillLaunch|{invalid:true}|undefined;
const fields=["skill_actor","skill_org","skill_target","skill_id","skill_version","skill_digest"] as const;
export function parseSkillLaunch(params:URLSearchParams):SkillLaunchInput{
 if(!fields.some(key=>params.has(key)))return;
 if(fields.some(key=>params.getAll(key).length!==1))return {invalid:true};
 const raw=params.get("skill_version")??"";
 if(!/^[1-9][0-9]{0,15}$/.test(raw))return {invalid:true};
 const parsed=schema.safeParse({userId:params.get("skill_actor"),organizationId:params.get("skill_org"),target:params.get("skill_target"),selection:{skillId:params.get("skill_id"),version:Number(raw),digest:params.get("skill_digest")}});
 return parsed.success?parsed.data:{invalid:true};
}
export function parseSkillLaunchRecord(record:Record<string,string|string[]|undefined>){
 const params=new URLSearchParams();for(const [key,value] of Object.entries(record))for(const item of Array.isArray(value)?value:value===undefined?[]:[value])params.append(key,item);return parseSkillLaunch(params);
}
export function skillLaunchMatches(launch:SkillLaunchInput,scope:{userId:string;organizationId:string},target?:SkillLaunch["target"]):launch is SkillLaunch{
 return !!launch&&!("invalid" in launch)&&launch.userId===scope.userId&&launch.organizationId===scope.organizationId&&(!target||target===launch.target);
}
export function skillLaunchHref(path:string,launch:SkillLaunch){
 const url=new URL(path,"https://navigation.invalid");const values=[launch.userId,launch.organizationId,launch.target,launch.selection.skillId,String(launch.selection.version),launch.selection.digest];fields.forEach((key,index)=>url.searchParams.set(key,values[index]));return url.pathname+url.search+url.hash;
}
