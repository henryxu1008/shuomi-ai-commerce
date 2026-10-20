"use client";
import {useRef,useState,useEffect} from "react";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {skillRequest,skillError} from "@/lib/api/skill";
import {skillProjectionSchema,skillRefSchema,type SkillProjection} from "@/lib/contracts/skill";
import type {z} from "zod";
export function SkillMethodSummary({method}:{method?:z.infer<typeof skillRefSchema>}){
	if(!method)return null;
	return <SummaryWithContext method={method}/>;
}
function SummaryWithContext({method}:{method:z.infer<typeof skillRefSchema>}){
 const c=useWorkbenchContext(),user=c.user?.id,org=c.effectiveOrganization?.id;
 return <section><p className="break-all">采用技能：{method.ownerKind==="USER"?"个人":method.ownerKind==="ORGANIZATION"?"组织":"官方"} · {method.id} · v{method.version} · 摘要 {method.digest}</p>{user&&!c.isSwitching&&!c.isLoading&&!c.error&&!c.blockingError&&(method.ownerKind!=="USER"||method.ownerId===user)&&(method.ownerKind!=="ORGANIZATION"||method.ownerId===org)&&<MethodBody key={`${user}:${org}:${c.permissions.join(",")}:${method.id}:${method.version}`} user={user} org={org} method={method}/>}</section>;
}
function MethodBody({user,org,method}:{user:string;org?:string;method:z.infer<typeof skillRefSchema>}){
 const [value,setValue]=useState<SkillProjection|null>(null),[failure,setFailure]=useState(""),[busy,setBusy]=useState(false),request=useRef<AbortController|null>(null);
 useEffect(()=>()=>request.current?.abort(),[]);
 async function read(){const c=new AbortController();request.current?.abort();request.current=c;setBusy(true);setFailure("");try{const space=method.ownerKind==="USER"?"personal":method.ownerKind==="ORGANIZATION"?"organization":"official",v=await skillRequest({userId:user,organizationId:org},`${space}/${method.id}/versions/${method.version}`,skillProjectionSchema,{signal:c.signal});if(!c.signal.aborted){if(v.version.package.digest!==method.digest)throw Error();setValue(v)}}catch(e){if(!c.signal.aborted){setValue(null);setFailure(skillError(e))}}finally{if(!c.signal.aborted)setBusy(false)}}
 return <><Button variant="outline" disabled={busy} onClick={()=>void read()}>按当前权限读取技能正文</Button>{failure&&<p role="alert">{failure}</p>}{value?.version.package.files?.map(f=><details key={f.path}><summary>{f.path}</summary><pre className="whitespace-pre-wrap break-words p-3 text-sm">{f.text}</pre></details>)}</>;
}
