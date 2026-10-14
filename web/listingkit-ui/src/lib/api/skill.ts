import {z} from "zod";
import {readBoundedStrictJSON} from "./strict-json-response";
export type SkillScope={userId:string;organizationId?:string};
export class SkillError extends Error {constructor(public code:string,public unknown=false){super(code)}}
export async function skillRequest<T>(scope:SkillScope,path:string,schema:z.ZodType<T>,init:RequestInit={}):Promise<T>{
 const write=!!init.method&&init.method!=="GET",headers=new Headers(init.headers);headers.set("Accept","application/json");headers.set("X-Expected-User-ID",scope.userId);if(path.startsWith("organization/")||path==="organization")headers.set("X-Expected-Organization-ID",scope.organizationId??"");
 try{const r=await fetch("/api/workbench/skills/"+path,{...init,headers,cache:"no-store",credentials:"same-origin",redirect:"error"});const value=await readBoundedStrictJSON(r,262144,init.signal??undefined);if(!r.ok){const e=z.strictObject({code:z.string().regex(/^[A-Z_]{1,80}$/)}).safeParse(value);throw new SkillError(e.success?e.data.code:write?"OUTCOME_UNKNOWN":"SKILL_UNAVAILABLE",write&&(!e.success||e.data.code==="OUTCOME_UNKNOWN"))}return schema.parse(value)}
 catch(e){if(e instanceof SkillError)throw e;throw new SkillError(write?"OUTCOME_UNKNOWN":"SKILL_UNAVAILABLE",write)}
}
export function skillError(e:unknown){const code=e instanceof SkillError?e.code:"SKILL_UNAVAILABLE";return ({SKILL_UNAVAILABLE:"技能服务尚未配置或暂不可用。",SKILL_FORBIDDEN:"当前身份无权操作该技能。",SKILL_NOT_FOUND:"该技能版本当前不可用。",SKILL_NOT_ENABLED:"该技能版本未启用或已撤销。",SKILL_INVALID:"格式不符合要求，请检查 SKILL.md 和文本材料。",SKILL_UNSUPPORTED_FIELD:"包含不支持的执行字段、脚本、链接或文件。",SKILL_REVISION_MISMATCH:"技能已变化，请重新读取后确认。",SKILL_IDEMPOTENCY_CONFLICT:"同一编号已绑定其他操作，请读取原操作结果。",OUTCOME_UNKNOWN:"结果尚未确认，请保留本次编号并核实原操作。",IDENTITY_CONTEXT_CHANGED:"登录身份已变化，请重新登录。",ORGANIZATION_CONTEXT_CHANGED:"当前企业已变化，请重新选择。"} as Record<string,string>)[code]??"操作未完成，请核对当前权限后重试。"}
