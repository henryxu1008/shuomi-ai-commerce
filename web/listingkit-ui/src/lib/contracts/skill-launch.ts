import {z} from "zod";
import {isAcquisitionUUID} from "./product-acquisition";
import {sameSkill,skillSelectionSchema,skillTargetSchema} from "./skill";

function isChatSkillReturn(path:string){
 if(!path.startsWith("/")||path.startsWith("//")||path.length>1000)return false;
 let url:URL;try{url=new URL(path,"https://navigation.invalid")}catch{return false;}
 const tail=url.pathname.slice("/workbench/ai/chat/".length);
 if(url.origin!=="https://navigation.invalid"||!url.pathname.startsWith("/workbench/ai/chat/")||tail!=="new"&&!isAcquisitionUUID(tail)||url.hash)return false;
 for(const [key,value] of url.searchParams){if(url.searchParams.getAll(key).length!==1||!(["projectId","storeId","operationId"].includes(key)?isAcquisitionUUID(value):["projectTask","recheckStore"].includes(key)&&value==="1"))return false;}
 return !(url.searchParams.has("projectId")&&url.searchParams.has("storeId"));
}
// Navigation hints only. Candidates and execution still authorize the exact ref.
const schema=z.strictObject({userId:z.string().regex(/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/),organizationId:z.string().regex(/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/),target:skillTargetSchema,selection:skillSelectionSchema,chatReturn:z.string().refine(isChatSkillReturn).optional()});
export type SkillLaunch=z.infer<typeof schema>;
export type SkillLaunchInput=SkillLaunch|{invalid:true}|undefined;
const fields=["skill_actor","skill_org","skill_target","skill_id","skill_version","skill_digest"] as const;
export function parseSkillLaunch(params:URLSearchParams):SkillLaunchInput{
 if(!fields.some(key=>params.has(key))&&!params.has("skill_chat"))return;
 if(params.getAll("skill_chat").length>1)return {invalid:true};
 if(fields.some(key=>params.getAll(key).length!==1))return {invalid:true};
 const raw=params.get("skill_version")??"";
 if(!/^[1-9][0-9]{0,15}$/.test(raw))return {invalid:true};
 const parsed=schema.safeParse({userId:params.get("skill_actor"),organizationId:params.get("skill_org"),target:params.get("skill_target"),...(params.has("skill_chat")?{chatReturn:params.get("skill_chat")}:{ }),selection:{skillId:params.get("skill_id"),version:Number(raw),digest:params.get("skill_digest")}});
 return parsed.success?parsed.data:{invalid:true};
}
export function parseSkillLaunchRecord(record:Record<string,string|string[]|undefined>){
 const params=new URLSearchParams();for(const [key,value] of Object.entries(record))for(const item of Array.isArray(value)?value:value===undefined?[]:[value])params.append(key,item);return parseSkillLaunch(params);
}
export function skillLaunchMatches(launch:SkillLaunchInput,scope:{userId:string;organizationId:string},target?:SkillLaunch["target"]):launch is SkillLaunch{
 return !!launch&&!("invalid" in launch)&&launch.userId===scope.userId&&launch.organizationId===scope.organizationId&&(!target||target===launch.target);
}
export function skillLaunchHref(path:string,launch:SkillLaunch){
 const url=new URL(path,"https://navigation.invalid");const values=[launch.userId,launch.organizationId,launch.target,launch.selection.skillId,String(launch.selection.version),launch.selection.digest];fields.forEach((key,index)=>url.searchParams.set(key,values[index]));url.searchParams.delete("skill_chat");if(launch.chatReturn)url.searchParams.set("skill_chat",launch.chatReturn);return url.pathname+url.search+url.hash;
}

const locationKey=(launch:SkillLaunch)=>"chat-skill-title-location:"+JSON.stringify([launch.userId,launch.organizationId,launch.chatReturn,launch.target,launch.selection]);
function validChatLocation(launch:SkillLaunch,path:string){
 if(!launch.chatReturn||launch.target!=="product-title"||!path.startsWith("/")||path.startsWith("//")||path.length>16384)return false;
 let url:URL;try{url=new URL(path,"https://navigation.invalid")}catch{return false;}
 const m=/^\/workbench\/data\/mine\/own\/([^/]+)\/optimize$/.exec(url.pathname),parsed=parseSkillLaunch(url.searchParams);
 return url.origin==="https://navigation.invalid"&&!!m&&isAcquisitionUUID(m[1])&&!!parsed&&!("invalid" in parsed)&&parsed.userId===launch.userId&&parsed.organizationId===launch.organizationId&&parsed.target===launch.target&&parsed.chatReturn===launch.chatReturn&&sameSkill(parsed.selection,launch.selection)&&isAcquisitionUUID(url.searchParams.get("agent_key")??"")&&url.searchParams.get("agent_actor")===launch.userId&&url.searchParams.get("agent_org")===launch.organizationId;
}
const chatSkillLocations=z.array(z.string());

export function readChatSkillLocations(launch:SkillLaunch){
 const raw=sessionStorage.getItem(locationKey(launch));
 if(raw===null)return [];
 const locations=raw.startsWith("/")?[raw]:chatSkillLocations.parse(JSON.parse(raw));
 if(locations.some(path=>!validChatLocation(launch,path)))throw Error("Original title location is unreadable");
 return locations;
}

export function rememberChatSkillLocation(launch:SkillLaunch,path:string){
 if(!validChatLocation(launch,path))throw Error("Invalid original title location");
 const previous=readChatSkillLocations(launch);
 if(!previous.includes(path))sessionStorage.setItem(locationKey(launch),JSON.stringify([path,...previous]));
}
