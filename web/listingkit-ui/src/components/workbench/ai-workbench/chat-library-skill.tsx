"use client";
import Link from "next/link";
import {useEffect,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {skillRequest,type SkillScope} from "@/lib/api/skill";
import {skillCandidatesSchema,skillSelectionSchema,type SkillVersion} from "@/lib/contracts/skill";
import {parseSkillLaunch,readChatSkillLocation,skillLaunchHref,type SkillLaunch} from "@/lib/contracts/skill-launch";
import styles from "./chat-skill-authoring.module.css";
const label=(v:SkillVersion)=>v.package.displayName||v.package.name;
const source=(v:SkillVersion)=>v.skill.owner.kind==="PLATFORM"?"官方":v.skill.owner.kind==="USER"?"本人":"当前团队";
type Target="product-title"|"product-image";
function useCandidates(scope:SkillScope,authorizationKey:string,target:Target,enabled=true){return useQuery({queryKey:["chat-library-skills",scope.userId,scope.organizationId,authorizationKey,target],queryFn:({signal})=>skillRequest(scope,target==="product-title"?"organization/candidates":"organization/image-candidates",skillCandidatesSchema,{signal}),enabled,retry:false,staleTime:0});}
export function ChatLibrarySkillOptions({scope,authorizationKey,onSelect}:{scope:SkillScope;authorizationKey:string;onSelect:(v:SkillVersion)=>void}){
 const [search,setSearch]=useState("");return <><label className={styles.field}>查找已有技能<Input value={search} maxLength={100} onChange={e=>setSearch(e.target.value)} placeholder="按技能名称或用途查找"/></label><p className={styles.drawerIntro}>仅显示已启用且当前企业允许使用的版本。新保存的个人技能需先启用并取得企业使用许可。</p>{(["product-title","product-image"] as const).map(target=><LibraryGroup key={target} scope={scope} authorizationKey={authorizationKey} target={target} search={search} onSelect={onSelect}/>)}</>;
}
function LibraryGroup({scope,authorizationKey,target,search,onSelect}:{scope:SkillScope;authorizationKey:string;target:Target;search:string;onSelect:(v:SkillVersion)=>void}){
 const result=useCandidates(scope,authorizationKey,target),title=target==="product-title"?"标题":"图片";
 const items=result.data?.items.filter(v=>(v.package.target??"product-title")===target&&(!search||[label(v),v.package.description].some(t=>t.toLocaleLowerCase().includes(search.toLocaleLowerCase()))))??[];
 return <section className="grid gap-3" aria-label={title+"技能"}><h3 className="font-medium">{title}处理</h3>{result.isFetching?<p role="status">正在读取{title}技能…</p>:result.isError?<><p role="alert">{title}技能读取失败，请重试。</p><Button variant="outline" onClick={()=>void result.refetch()}>重新读取{title}技能</Button></>:!items.length?<p>{search?"没有匹配的技能。":"当前企业暂无可用的"+title+"技能。"}</p>:items.map(v=><Button key={v.skill.id+":"+v.version} variant="outline" className={styles.skillChoice} onClick={()=>onSelect(v)}><span className={styles.skillChoiceText}><span><strong>{label(v)}</strong><small>{source(v)} · v{v.version}{v.package.execution?" · Python":""}</small></span><span>{v.package.description}</span></span></Button>)}</section>;
}
export function useChatLibrarySkill(scope:{userId:string;organizationId:string},entry:string,authorizationKey:string){
 const key="chat-library-choice:"+JSON.stringify([scope.userId,scope.organizationId,entry]),identity=JSON.stringify([key,authorizationKey]);
 const [state,setState]=useState<{identity:string;launch?:SkillLaunch;error?:string}>();
 useEffect(()=>{let live=true;queueMicrotask(()=>{if(!live)return;try{const raw=sessionStorage.getItem(key);let launch:SkillLaunch|undefined;if(raw){const url=new URL(raw,"https://navigation.invalid"),parsed=parseSkillLaunch(url.searchParams);if(!parsed||"invalid" in parsed||parsed.userId!==scope.userId||parsed.organizationId!==scope.organizationId||parsed.chatReturn!==entry||!["product-title","product-image"].includes(parsed.target)||!skillSelectionSchema.safeParse(parsed.selection).success)throw Error();launch=parsed;}setState({identity,launch});}catch{setState({identity,error:"无法读取原技能选择，请重新选择。"})}});return()=>{live=false}},[key,identity,scope.userId,scope.organizationId,entry]);
 function choose(v?:SkillVersion){try{const launch:SkillLaunch|undefined=v?{userId:scope.userId,organizationId:scope.organizationId,target:(v.package.target??"product-title") as Target,selection:{skillId:v.skill.id,version:v.version,digest:v.package.digest},chatReturn:entry}:undefined;if(launch)sessionStorage.setItem(key,skillLaunchHref("/workbench/agents/skills/official/use",launch));else sessionStorage.removeItem(key);setState({identity,launch});}catch{setState(previous=>({...((previous?.identity===identity)?previous:{}),identity,error:"无法保留技能选择，请检查浏览器存储后重试。"}))}}
 return {launch:state?.identity===identity?state.launch:undefined,error:state?.identity===identity?state.error:undefined,choose};
}
export function ChatLibrarySkillSelection({launch,authorizationKey,onClear,disabled=false}:{launch:SkillLaunch;authorizationKey:string;onClear:()=>void;disabled?:boolean}){
 const read=useCandidates({userId:launch.userId,organizationId:launch.organizationId},authorizationKey,launch.target as Target);
 const version=!read.isFetching&&!read.isError?read.data?.items.find(v=>v.skill.id===launch.selection.skillId&&v.version===launch.selection.version&&v.package.digest===launch.selection.digest):undefined;
 const [resume,setResume]=useState<{key:string;href?:string;error?:boolean}>(),key=JSON.stringify(launch);
 useEffect(()=>{let live=true;queueMicrotask(()=>{if(!live)return;try{setResume({key,href:readChatSkillLocation(launch)})}catch{setResume({key,error:true})}});return()=>{live=false}},[key,launch]);
 const original=resume?.key===key?resume:undefined;
 return <div className="my-3 grid gap-3 rounded-xl border border-border p-4"><p>{version?label(version):"已选择的技能"} · v{launch.selection.version}{version?" · "+source(version):""}</p><p className="text-sm text-muted-foreground">聊天草稿保留，选择商品后在处理页确认。</p>{read.isFetching?<p role="status">正在核对技能版本…</p>:read.isError?<><p role="alert">暂时无法核对技能。</p><Button variant="outline" onClick={()=>void read.refetch()}>重新核对技能</Button></>:!version?<p role="alert">所选版本已停用或当前不可用，请重新选择。原请求仍可从下方核实。</p>:null}{original?.error?<p role="alert">原操作入口无法读取，请保留当前页面并从原商品页核实。</p>:original?.href?<Button asChild disabled={disabled} variant="outline"><Link prefetch={false} href={original.href} onClick={e=>{if(disabled)e.preventDefault()}}>核实或查看原标题操作</Link></Button>:version&&original?<Button asChild disabled={disabled}><Link prefetch={false} href={skillLaunchHref("/workbench/agents/skills/official/use",launch)} onClick={e=>{if(disabled)e.preventDefault()}}>{launch.target==="product-image"?"前往图片处理页":"前往标题处理页"}</Link></Button>:null}<Button variant="ghost" disabled={disabled} onClick={onClear}>取消选择</Button></div>;
}
