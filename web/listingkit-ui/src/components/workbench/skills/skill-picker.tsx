"use client";
import {useCallback,useEffect,useId,useRef,useState} from "react";
import {Button} from "@/components/ui/button";
import {skillRequest,skillError,type SkillScope} from "@/lib/api/skill";
import {skillCandidatesSchema,type SkillVersion,type SkillSelection} from "@/lib/contracts/skill";
const publisher=(v:SkillVersion)=>v.skill.owner.kind==="USER"?"本人":v.skill.owner.kind==="ORGANIZATION"?"当前团队":"官方";
const name=(v:SkillVersion)=>v.package.displayName||v.package.name;
type Changed=(selection:SkillSelection|undefined,ready:boolean)=>void;

type PickerProps={scope:SkillScope;onChange:Changed;target?:"product-title"|"product-image";disabled?:boolean;initialSelection?:SkillSelection};
export function SkillPicker(props:PickerProps){return <ScopedSkillPicker key={JSON.stringify([props.scope.userId,props.scope.organizationId,props.target,props.initialSelection])} {...props}/>}
function ScopedSkillPicker({scope,onChange,target="product-title",disabled=false,initialSelection}:PickerProps){
 const id=useId(),request=useRef<AbortController|null>(null),notify=useRef(onChange),selection=useRef<SkillSelection|undefined>(initialSelection);
 const [use,setUse]=useState(!!initialSelection),[items,setItems]=useState<SkillVersion[]>([]),[selected,setSelected]=useState<SkillSelection|undefined>(initialSelection),[loading,setLoading]=useState(!!initialSelection),[failure,setFailure]=useState("");
 const image=target==="product-image",userId=scope.userId,organizationId=scope.organizationId;
 useEffect(()=>{notify.current=onChange},[onChange]);
 const load=useCallback(async()=>{
  request.current?.abort();const c=new AbortController();request.current=c;setLoading(true);setFailure("");
  const current=selection.current;notify.current(current,image&&!current);
  try{
   const r=await skillRequest({userId,organizationId},image?"organization/image-candidates":"organization/candidates",skillCandidatesSchema,{signal:c.signal});if(c.signal.aborted)return;
   setItems(r.items);const valid=!!current&&r.items.some(v=>v.skill.id===current.skillId&&v.version===current.version&&v.package.digest===current.digest);
   if(current&&!valid)setFailure(image?"所选版本不可用或缺少视觉 brief，请重新选择或明确不使用技能。":"所选版本已不可用，请重新选择或明确取消使用技能。");
   notify.current(current,valid||image&&!current);
  }catch(e){if(!c.signal.aborted){setItems([]);setFailure(skillError(e));notify.current(current,image&&!current)}}finally{if(!c.signal.aborted)setLoading(false)}
 },[userId,organizationId,image]);
 useEffect(()=>{let cancelled=false;if(image||initialSelection)queueMicrotask(()=>{if(!cancelled)void load()});return()=>{cancelled=true;request.current?.abort()}},[image,load,initialSelection]);
 function choose(value:SkillSelection|undefined){selection.current=value;setSelected(value);setFailure("");notify.current(value,image||!!value)}
 const value=items.find(v=>v.skill.id===selected?.skillId&&v.version===selected.version&&v.package.digest===selected.digest);
 return <div className="my-4 space-y-3">
  {!image&&<label><input type="checkbox" disabled={disabled} checked={use} onChange={e=>{setUse(e.target.checked);if(e.target.checked)void load();else{request.current?.abort();setLoading(false);setFailure("");selection.current=undefined;setSelected(undefined);notify.current(undefined,true)}}}/>使用技能 / 方法</label>}
  {(image||use)&&<>
   <label className="block text-sm font-medium" htmlFor={`${id}-version`}>{image?"使用的方法/技能（可选）":"准确技能版本"}</label>
   <select id={`${id}-version`} className="block w-full rounded-lg border border-border bg-background p-3 text-sm" disabled={disabled||loading} value={selected?`${selected.skillId}:${selected.version}`:""} onChange={e=>{const v=items.find(v=>`${v.skill.id}:${v.version}`===e.target.value);choose(v?{skillId:v.skill.id,version:v.version,digest:v.package.digest}:undefined)}}>
    <option value="">{image?"不使用技能":"请选择已启用版本"}</option>
    {selected&&!value&&<option value={`${selected.skillId}:${selected.version}`} disabled>原选版本（不可用） · v{selected.version}</option>}
    {items.map(v=><option key={`${v.skill.id}:${v.version}`} value={`${v.skill.id}:${v.version}`}>{name(v)} · {publisher(v)} · v{v.version}</option>)}
   </select>
   {image&&<p className="text-sm text-muted-foreground">仅列当前企业可执行的官方与团队图片版本。此设置用于新计划，原任务保留其准确版本。</p>}
   <Button variant="outline" disabled={disabled||loading} onClick={()=>void load()}>刷新技能版本</Button>
   {loading&&<p role="status">正在读取当前可用版本…</p>}{failure&&<p role="alert">{failure}</p>}
   {!loading&&!items.length&&!failure&&<p className="text-sm text-muted-foreground">{image?"暂无可用图片技能，可以选择不使用技能。":"没有已启用且获准使用的技能；个人技能需当前企业明确准入。"}</p>}
   {value&&image&&<div className="space-y-2 rounded-lg bg-success p-4 text-sm"><strong>{name(value)}</strong><p>{value.package.description}</p><p>技能不增加任务数量，也不替代平台规则、报价与审核。</p></div>}
   {selected&&<p className="break-all text-xs text-muted-foreground">准确引用：{selected.skillId} · v{selected.version} · {selected.digest}</p>}
  </>}
 </div>;
}

export function ImageMethodSummary({scope,method,refresh}:{scope:SkillScope;method:{Skill:{id:string;ownerKind:string;version:number;digest:string};PolicyVersion:string};refresh:unknown}){
 const {id,version,digest,ownerKind}=method.Skill,[label,setLabel]=useState<{key:string;text:string}>(),key=`${id}:${version}:${digest}`;
 useEffect(()=>{const c=new AbortController();void skillRequest(scope,"organization/image-candidates",skillCandidatesSchema,{signal:c.signal}).then(r=>{if(c.signal.aborted)return;const v=r.items.find(v=>v.skill.id===id&&v.version===version&&v.package.digest===digest);setLabel(v?{key,text:name(v)}:undefined)}).catch(()=>{if(!c.signal.aborted)setLabel(undefined)});return()=>c.abort()},[scope,id,version,digest,key,refresh]);
 return <div className="space-y-1 rounded-lg border border-border p-3 text-sm"><p>计划使用的方法：{label?.key===key?label.text:id} · {ownerKind==="PLATFORM"?"官方":"当前团队"} · v{version}</p><p className="break-all text-xs text-muted-foreground">版本摘要：{digest}</p><p className="text-xs text-muted-foreground">准确版本保留在计划与报价中，方法不改变任务、素材或点数上限。</p></div>;
}
