"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronRight, SquarePen } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { AIWorkbenchError, requestAIWorkbench, type AIScope } from "@/lib/api/ai-workbench";
import { type AISkillAuthoringChoice, type AISkillAuthoringJob, type AITitleSkillDraft } from "@/lib/contracts/ai-workbench";
import { AccountDialog } from "../account/account-dialog";
import styles from "./chat-skill-authoring.module.css";
import type {SkillCategory} from "../skills/skill-library-page";
import type {SkillVersion} from "@/lib/contracts/skill";
import {ChatLibrarySkillOptions} from "./chat-library-skill";

function useCatalog(scope: AIScope, authorizationKey: string, enabled: boolean) {
  return useQuery({
    queryKey: ["chat-skill-authoring-catalog", scope.userId, scope.organizationId, authorizationKey],
    queryFn: ({ signal }) => requestAIWorkbench({ route: "skill-authoring-catalog", method: "GET", path: "chat/conversations/skill-authoring", scope, signal }),
    enabled, retry: false, staleTime: 0,
  });
}

export function ChatSkillDrawer({ scope, authorizationKey, disabled, onSelect, onResume, onSelectLibrary }: {
  scope: AIScope;
  authorizationKey: string;
  disabled?: boolean;
  onSelect: () => void;
  onResume?: () => void;
  onSelectLibrary?: (v:SkillVersion) => void;
}) {
  const [open, setOpen] = useState(false);
  const [category,setCategory]=useState<"all"|SkillCategory>("all");
  const official=category==="all"||category==="official";
  const catalog = useCatalog(scope, authorizationKey, open);
  const trigger = useRef<HTMLButtonElement>(null);
  const skill = catalog.isSuccess ? catalog.data.skill : null;
  return <>
    <Button ref={trigger} variant="outline" disabled={disabled} aria-haspopup="dialog" aria-expanded={open} onClick={() => setOpen(true)}>选择技能</Button>
    {open ? <AccountDialog title="选择技能" drawer className={styles.skillDrawer} onClose={() => setOpen(false)} returnFocusRef={trigger}>
      <div className={styles.drawerContent}>
        <p className={styles.drawerIntro}>选择已有技能处理商品，或在 Chat 中创建新技能。</p>
        <div role="group" aria-label="技能分类" className="flex flex-wrap gap-2">{([["all","全部"],["official","官方模板"],["mine","我的技能"],["team","团队技能"]] as const).map(([value,label])=><Button key={value} size="sm" variant={category===value?"default":"outline"} aria-pressed={category===value} onClick={()=>setCategory(value)}>{label}</Button>)}</div>
        {official && <>{catalog.isFetching ? <p role="status">正在读取可用技能…</p>
          : catalog.isError ? <><p role="alert">技能暂不可用，请重试。</p><Button variant="outline" onClick={() => void catalog.refetch()}>重新读取</Button></>
          : skill ? <><Button variant="outline" className={styles.skillChoice}
            disabled={!catalog.data?.ready} onClick={() => { onSelect(); setOpen(false); }}>
            <span className={styles.skillIcon}><SquarePen size={20} aria-hidden="true" /></span>
            <span className={styles.skillChoiceText}><span><strong>创建技能</strong><small>官方 v{skill.version}</small></span><span>在 Chat 里描述你想创建的技能。</span></span>
            <ChevronRight size={16} aria-hidden="true" />
          </Button>{!catalog.data?.ready && <p role="status">当前企业的技能创作模型尚未就绪。</p>}</>
          : <p>当前没有可用的创作技能。</p>}</>}
        {onSelectLibrary && <ChatLibrarySkillOptions category={category} scope={scope} authorizationKey={authorizationKey} onSelect={v=>{onSelectLibrary(v);setOpen(false);}}/>}
        <div className={styles.drawerFooter}>
          {onResume && <Button variant="outline" onClick={() => { onResume(); setOpen(false); }}>查看本会话的技能草稿</Button>}
          <Button asChild variant="link"><Link href="/workbench/agents/skills/mine">管理我的技能</Link></Button>
        </div>
      </div>
    </AccountDialog> : null}
  </>;
}


const editorFields = [["displayName","技能名称"],["name","技能标识（小写英文字母、数字和连字符）"],["description","用途"],["instructions","正文"]] as const;
const generationStatus={QUOTED:"尚未开始生成",READY:"生成结果待核实",COMPLETE:"已生成草稿",FAILED:"本次未能生成草稿，请稍后再试。",INVALID:"草稿格式未通过，请修改需求后再试。",UNKNOWN:"本次生成结果未能确认。"} as const;
function authoringError(error:unknown){
 const code=error instanceof AIWorkbenchError?error.code:"";
 if(code==="PROPOSAL_STALE"||code==="IDEMPOTENCY_CONFLICT")return "草稿已在其他页面更新，请读取原结果后再操作。";
 if(code==="INVALID_REQUEST")return "请检查填写内容和格式后重试。";
 return "本次操作暂未确认，请核实原请求。";
}
export type ChatSkillRevision={skill:{version:1|2;digest:string};revision:{root:string;baseRevision:string;snapshot:AITitleSkillDraft}};

export function ChatSkillTurn({scope,authorizationKey,conversation,turnKey,readOnly,onRevise,onRecover}:{scope:AIScope;authorizationKey:string;conversation:string;turnKey:string;readOnly?:boolean;onRevise:(revision:ChatSkillRevision)=>void;onRecover?:()=>Promise<void>}){
 const path=`chat/conversations/${conversation}/turns/${turnKey}`;
 const result=useQuery({queryKey:["chat-turn",scope.userId,scope.organizationId,authorizationKey,conversation,turnKey],queryFn:({signal})=>requestAIWorkbench({route:"chat-turn-read",method:"GET",scope,path,signal}),retry:false,staleTime:0});
 const [busy,setBusy]=useState(false),[notice,setNotice]=useState("");
 const action=useRef<AbortController|null>(null);
 useEffect(()=>()=>action.current?.abort(),[]);
 async function recover(){
  if(action.current)return;const controller=new AbortController();action.current=controller;setBusy(true);setNotice("");
  try{if(onRecover)await onRecover();else await requestAIWorkbench({route:"chat-turn-recover",method:"POST",scope,path:path+"/recover",signal:controller.signal});if(!controller.signal.aborted)await result.refetch();}
  catch(error){if(!controller.signal.aborted)setNotice(authoringError(error));}
  finally{action.current=null;if(!controller.signal.aborted)setBusy(false);}
 }
 const job=result.data?.authoring,skill=result.data?.skill;
 return <div>
  {result.isFetching&&!result.data?<p role="status">正在读取技能草稿…</p>:null}
  {result.isError?<p role="status">暂时无法读取原请求。 <Button variant="outline" onClick={()=>void result.refetch()}>重新读取草稿</Button></p>:null}
  {notice&&<p role="status">{notice}</p>}
  {result.data&&result.data.state!=="COMPLETE"?<p role="status">{result.data.state==="NOT_STARTED"?"本次尚未开始生成，请核对技能后重新提出需求。":result.data.state==="FAILED"?"本次未能生成草稿。":result.data.state==="UNKNOWN"?"生成结果暂时无法确认。":"正在生成技能草稿。"}</p>:null}
  {!readOnly&&result.data&&["PROCESSING","UNKNOWN"].includes(result.data.state)?<Button variant="outline" disabled={busy} onClick={()=>void recover()}>{busy?"正在核实…":"核实原生成请求"}</Button>:null}
  {job?<ChatSkillAuthoring scope={scope} authorizationKey={authorizationKey} conversation={conversation} initialJob={job} readOnly={readOnly} onRevise={skill?.version===2?draft=>onRevise({skill,revision:{root:job.key,baseRevision:job.revision??job.key,snapshot:draft}}):undefined}/>:null}
 </div>;
}

// The editor and original save/recovery endpoints remain shared by history and
// a message card. Creation and revisions use the one Chat composer.
export function ChatSkillAuthoring({scope,authorizationKey,conversation,initialJob,readOnly,onRevise,onClose}:{scope:AIScope;authorizationKey:string;conversation:string;initialJob?:AISkillAuthoringJob;readOnly?:boolean;onRevise?:(draft:AITitleSkillDraft)=>void;onClose?:()=>void}){
 const queryClient=useQueryClient();
 const path=`chat/conversations/${conversation}/skill-authoring`;
 const [job,setJob]=useState(initialJob),[draft,setDraft]=useState(initialJob?.draft??undefined);
 const [busy,setBusy]=useState(false),[notice,setNotice]=useState(""),[saveCheck,setSaveCheck]=useState(initialJob?.savePending??false);
 const [validation,setValidation]=useState<{draft:AITitleSkillDraft;valid:boolean;unavailable?:boolean}>(),[validationRetry,setValidationRetry]=useState(0);
 const action=useRef<AbortController|null>(null);
 const dirty=useRef(false);
 const [edited,setEdited]=useState(false);
 const list=useQuery({queryKey:["chat-skill-authoring-list",scope.userId,scope.organizationId,authorizationKey,conversation],queryFn:({signal})=>requestAIWorkbench({route:"skill-authoring-list",method:"GET",scope,path,signal}),enabled:!initialJob,retry:false,staleTime:0});
 useEffect(()=>()=>action.current?.abort(),[]);
 const editable=Boolean(draft&&job?.state==="COMPLETE"&&!readOnly&&!job.savedSkillId&&!job.savePending&&!saveCheck);
 const stale=Boolean(initialJob&&job&&(initialJob.revision!==job.revision||initialJob.generationKey!==job.generationKey));
 useEffect(()=>{
  if(!initialJob)return;let live=true;
  queueMicrotask(()=>{
   if(!live)return;
   if(!dirty.current){setJob(initialJob);setDraft(initialJob.draft??undefined);setValidation(undefined);}
   else if(initialJob.savedSkillId){setJob(initialJob);}
   setSaveCheck(current=>Boolean(initialJob.savedSkillId)?false:current||initialJob.savePending);
  });
  return()=>{live=false;};
 },[initialJob]);
 function updateDraft(next:AITitleSkillDraft){dirty.current=true;setEdited(true);setDraft(next);}

 useEffect(()=>{
  if(!editable||!draft||busy)return;const controller=new AbortController();
  const timer=window.setTimeout(()=>{void requestAIWorkbench({route:"skill-authoring-validate",method:"POST",scope,path:path+"/validate",body:draft,signal:controller.signal}).then(()=>{if(!controller.signal.aborted)setValidation({draft,valid:true});}).catch(error=>{if(!controller.signal.aborted)setValidation({draft,valid:false,unavailable:!(error instanceof AIWorkbenchError)||error.code!=="INVALID_REQUEST"});});},250);
  return()=>{window.clearTimeout(timer);controller.abort();};
 },[editable,draft,busy,scope,path,validationRetry]);
 function display(result:AISkillAuthoringJob){dirty.current=false;setEdited(false);setJob(result);setDraft(result.draft??undefined);setValidation(undefined);setSaveCheck(result.savePending);}
 async function perform(run:(signal:AbortSignal)=>Promise<AISkillAuthoringJob>,saving=false){
  if(action.current)return;const controller=new AbortController();action.current=controller;setBusy(true);setNotice("");
  try{const result=await run(controller.signal);if(!controller.signal.aborted){display(result);if(!initialJob)void list.refetch();void queryClient.invalidateQueries({queryKey:["chat-turn",scope.userId,scope.organizationId]});}}
  catch(error){if(!controller.signal.aborted){setNotice(authoringError(error));if(saving&&(!(error instanceof AIWorkbenchError)||!["INVALID_REQUEST","NOT_FOUND"].includes(error.code)))setSaveCheck(true);}}
  finally{action.current=null;if(!controller.signal.aborted)setBusy(false);}
 }
 const checked=Boolean(validation&&validation.draft===draft&&validation.valid);
 return <Card className={styles.authoringCard}>
  <header className={styles.authoringHeader}><div><h2>技能草稿</h2><p>编辑后确认保存到我的技能。</p></div>{onClose&&<Button variant="ghost" onClick={onClose}>收起</Button>}</header>
  {notice&&<p role="status">{notice}</p>}
  {list.isError&&<p role="status">暂时无法读取原草稿。 <Button variant="outline" onClick={()=>void list.refetch()}>重新读取</Button></p>}
  {list.data?.drafts.map(item=><Button key={item.generationKey??item.key} variant="outline" disabled={busy||saveCheck} onClick={()=>display(item)}>{item.draft?.displayName||generationStatus[item.state]}</Button>)}
  {!initialJob&&!job&&list.isSuccess&&!list.data.drafts.length?<p>暂无草稿。可以直接在聊天中描述要创建的标题技能。</p>:null}
  {job&&<div className={styles.jobContent}>
   <p role="status">{job.superseded?"这次修改未覆盖当前草稿。":generationStatus[job.state]}</p>
   {!readOnly&&(job.state==="READY"||job.state==="UNKNOWN"||job.state==="QUOTED")&&!initialJob?<Button variant="outline" disabled={busy} onClick={()=>void perform(signal=>requestAIWorkbench({route:(job.generationKey??job.key)[14]==="7"?"skill-authoring-recover-generation":"skill-authoring-generate",method:"POST",scope,path:path+"/"+(job.generationKey??job.key)+((job.generationKey??job.key)[14]==="7"?"/recover-generation":"/generate"),signal}))}>核实原生成请求</Button>:null}
   {job.savedSkillId?<p className={styles.savedNotice}>已保存到我的技能。 <Link href={"/workbench/agents/skills/mine?skillId="+job.savedSkillId}>查看我的技能</Link></p>:!readOnly&&(saveCheck||job.savePending)?<Button variant="outline" disabled={busy} onClick={()=>void perform(signal=>requestAIWorkbench({route:"skill-authoring-save-recover",method:"POST",scope,path:path+"/"+job.key+"/recover-save",signal}))}>核实原保存</Button>:draft&&<div className={styles.draftEditor}>
    <div className={styles.editorFields}>{editorFields.map(([key,label])=><label className={key==="displayName"||key==="name"?styles.halfField:styles.field} key={key}>{label}{key==="instructions"?<Textarea rows={7} value={draft[key]} disabled={busy||!editable} onChange={event=>updateDraft({...draft,[key]:event.target.value})}/>:<Input value={draft[key]} disabled={busy||!editable} onChange={event=>updateDraft({...draft,[key]:event.target.value})}/>}</label>)}</div>
    {draft.script&&<div className={styles.scriptPanel}><div className={styles.scriptHeader}><p>包含 Python 脚本</p>{editable&&<Button variant="ghost" disabled={busy} onClick={()=>updateDraft({...draft,script:undefined})}>移除脚本</Button>}</div><details><summary>查看代码</summary><label className={styles.field}>脚本用途<Input value={draft.script.description} disabled={busy||!editable} onChange={event=>{if(draft.script)updateDraft({...draft,script:{...draft.script,description:event.target.value}});}}/></label><label className={styles.field}>Python 代码<Textarea className={styles.codeInput} rows={7} spellCheck={false} value={draft.script.source} disabled={busy||!editable} onChange={event=>{if(draft.script)updateDraft({...draft,script:{...draft.script,source:event.target.value}});}}/></label></details></div>}
    {stale&&edited?<p role="status">当前草稿已更新，你的人工编辑仍保留。 <Button variant="outline" disabled={busy} onClick={()=>{if(initialJob)display(initialJob);}}>使用最新草稿并替换当前编辑</Button></p>:null}
    {editable&&onRevise&&<Button variant="outline" disabled={busy||stale} onClick={()=>onRevise(draft)}>让 Chat 修改</Button>}
    {validation?.draft===draft&&!validation.valid?<p role="status">{validation.unavailable?"暂时无法检查格式。":"格式未通过，请检查名称、正文或脚本长度。"}{validation.unavailable&&<Button variant="outline" onClick={()=>setValidationRetry(value=>value+1)}>重新检查</Button>}</p>:null}
    {editable&&<footer className={styles.saveFooter}><p>确认后保存为个人私有草稿。</p><Button disabled={busy||!checked||stale} onClick={()=>void perform(signal=>requestAIWorkbench({route:"skill-authoring-save-package",method:"POST",scope,path:path+"/"+job.key+"/save-package",body:{revision:job.revision??job.key,draft},key:crypto.randomUUID(),signal}),true)}>保存到我的技能</Button></footer>}
   </div>}
  </div>}
 </Card>;
}
