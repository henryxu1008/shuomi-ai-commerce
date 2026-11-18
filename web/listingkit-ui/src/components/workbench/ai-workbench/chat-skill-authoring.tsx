"use client";

import Link from "next/link";
import { useEffect, useMemo, useRef, useState } from "react";
import { useInfiniteQuery, useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronRight, SquarePen } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { AIWorkbenchError, requestAIWorkbench, type AIScope } from "@/lib/api/ai-workbench";
import { type AIAuthoringPreview, type AISkillAuthoringChoice, type AISkillAuthoringJob, type AITextSkillRef, type AITitleSkillDraft } from "@/lib/contracts/ai-workbench";
import { skillRequest } from "@/lib/api/skill";
import { skillPageSchema } from "@/lib/contracts/skill";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";
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

export type SelectedTextSkill = AITextSkillRef & { name: string };
const generalAuthoringVersion = 3;

export function ContinueTextSkill({scope,authorizationKey,onContinue}:{scope:AIScope;authorizationKey:string;onContinue:(choice:AISkillAuthoringChoice)=>void}) {
  const catalog=useCatalog(scope,authorizationKey,true);
  const choice=catalog.data?.skill;
  if (!catalog.isSuccess || !catalog.data.ready || choice?.version!==generalAuthoringVersion) return null;
  return <Button type="button" variant="outline" onClick={()=>onContinue(choice)}>用这段需求创建通用文字技能</Button>;
}

export function ChatSkillDrawer({ scope, authorizationKey, disabled, onSelect, onResume, onResumeText, onUseTextSkill, onSelectLibrary }: {
  scope: AIScope;
  authorizationKey: string;
  disabled?: boolean;
  onSelect?: () => void;
  onResume?: () => void;
  onResumeText?: (skill: AISkillAuthoringChoice) => void;
  onUseTextSkill?: (skill: SelectedTextSkill) => void;
  onSelectLibrary?: (v:SkillVersion) => void;
}) {
  const [open, setOpen] = useState(false);
  const [category,setCategory]=useState<"all"|SkillCategory>("all");
  const official=category==="all"||category==="official";
  const catalog = useCatalog(scope, authorizationKey, open && Boolean(onSelect));
  const trigger = useRef<HTMLButtonElement>(null);
  const skill = catalog.isSuccess ? catalog.data.skill : null;
  const personal = useInfiniteQuery({
    queryKey: ["chat-personal-text-skills", scope.userId, scope.organizationId, authorizationKey],
    enabled: open && Boolean(onUseTextSkill && catalog.data?.textReady),
    initialPageParam: "",
    queryFn: ({ pageParam, signal }) => skillRequest(scope, "personal?limit=50" + (pageParam ? "&cursor=" + encodeURIComponent(pageParam) : ""), skillPageSchema, {signal}),
    getNextPageParam: page => page.nextCursor || undefined,
    retry: false,
  });
  return <>
    <Button ref={trigger} variant="outline" disabled={disabled} aria-haspopup="dialog" aria-expanded={open} onClick={() => setOpen(true)}>选择技能</Button>
    {open ? <AccountDialog title="选择技能" drawer className={styles.skillDrawer} onClose={() => setOpen(false)} returnFocusRef={trigger}>
      <div className={styles.drawerContent}>
        <p className={styles.drawerIntro}>{onSelect?"选择已有技能处理商品，或在 Chat 中创建新技能。":"选择已有技能处理商品。"}</p>
        <div role="group" aria-label="技能分类" className="flex flex-wrap gap-2">{([["all","全部"],["official","官方模板"],["mine","我的技能"],["team","团队技能"]] as const).map(([value,label])=><Button key={value} size="sm" variant={category===value?"default":"outline"} aria-pressed={category===value} onClick={()=>setCategory(value)}>{label}</Button>)}</div>
        {official && onSelect && <>{catalog.isFetching ? <p role="status">正在读取可用技能…</p>
          : catalog.isError ? <><p role="alert">技能暂不可用，请重试。</p><Button variant="outline" onClick={() => void catalog.refetch()}>重新读取</Button></>
          : skill ? <><Button variant="outline" className={styles.skillChoice}
            disabled={!catalog.data?.ready} onClick={() => { onSelect(); setOpen(false); }}>
            <span className={styles.skillIcon}><SquarePen size={20} aria-hidden="true" /></span>
            <span className={styles.skillChoiceText}><span><strong>创建技能</strong><small>官方 v{skill.version}</small></span><span>在 Chat 里描述你想创建的技能。</span></span>
            <ChevronRight size={16} aria-hidden="true" />
          </Button>{!catalog.data?.ready && <p role="status">当前企业的技能创作模型尚未就绪。</p>}</>
          : <p>当前没有可用的创作技能。</p>}</>}
        {(category==="all"||category==="mine") && onUseTextSkill && catalog.data?.textReady && <section aria-label="我的文字技能">
          <h3>我的文字技能</h3>
          {personal.isPending ? <p role="status">正在读取我的技能…</p> : null}
          {personal.isError ? <Button variant="outline" onClick={() => void personal.refetch()}>重新读取我的技能</Button> : null}
          {personal.isSuccess && !personal.hasNextPage && !personal.data.pages.some(page => page.items.some(item => item.canUse && item.version.visibility === "PRIVATE" && item.version.package.target === "chat-text")) && <p>还没有可以选用的文字技能。</p>}
          {personal.data?.pages.flatMap(page => page.items).filter(item => item.canUse && item.version.visibility === "PRIVATE" && item.version.package.target === "chat-text").map(({version}) => {
            const name = version.package.displayName || version.package.name;
            return <Button className={styles.skillChoice} variant="outline" key={version.skill.id} onClick={() => {onUseTextSkill({id:version.skill.id, version:version.version, digest:version.package.digest, name}); setOpen(false);}}>{name}</Button>;
          })}
          {personal.hasNextPage && <Button variant="outline" disabled={personal.isFetchingNextPage} onClick={() => void personal.fetchNextPage()}>加载更多技能</Button>}
        </section>}
        {onSelectLibrary && <ChatLibrarySkillOptions category={category} scope={scope} authorizationKey={authorizationKey} onSelect={v=>{onSelectLibrary(v);setOpen(false);}}/>}
        <div className={styles.drawerFooter}>
          {onResume && <Button variant="outline" onClick={() => { onResume(); setOpen(false); }}>查看本会话的标题技能草稿</Button>}
          {onResumeText && skill?.version === generalAuthoringVersion && <Button variant="outline" onClick={() => {onResumeText(skill); setOpen(false);}}>查看本会话的文字技能草稿</Button>}
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
 const [recoveredRevision,setRecoveredRevision]=useState<{job:AISkillAuthoringJob;observedRevision?:string;observedGeneration?:string}>();
 const [validation,setValidation]=useState<{draft:AITitleSkillDraft;valid:boolean;unavailable?:boolean}>(),[validationRetry,setValidationRetry]=useState(0);
 const action=useRef<AbortController|null>(null);
 const dirty=useRef(false);
 const [edited,setEdited]=useState(false);
 const list=useQuery({queryKey:["chat-skill-authoring-list",scope.userId,scope.organizationId,authorizationKey,conversation],queryFn:({signal})=>requestAIWorkbench({route:"skill-authoring-list",method:"GET",scope,path,signal}),enabled:!initialJob,retry:false,staleTime:0});
 useEffect(()=>()=>action.current?.abort(),[]);
 const editable=Boolean(draft&&job?.state==="COMPLETE"&&!readOnly&&!job.savedSkillId&&!job.savePending&&!saveCheck);
 const latest=!initialJob?.savedSkillId && recoveredRevision && recoveredRevision.job.key===job?.key
  && recoveredRevision.observedRevision===initialJob?.revision
  && recoveredRevision.observedGeneration===initialJob?.generationKey
  ? recoveredRevision.job : initialJob;
 const stale=Boolean(latest&&job&&(latest.revision!==job.revision||latest.generationKey!==job.generationKey));
 useEffect(()=>{
  if(!initialJob||!latest)return;let live=true;
  queueMicrotask(()=>{
   if(!live)return;
   if(!dirty.current){setJob(latest);setDraft(latest.draft??undefined);setValidation(undefined);}
   else if(initialJob.savedSkillId){setJob(initialJob);}
   setSaveCheck(current=>Boolean(initialJob.savedSkillId)?false:current||initialJob.savePending||latest.savePending);
  });
  return()=>{live=false;};
 },[initialJob,latest]);
 function updateDraft(next:AITitleSkillDraft){dirty.current=true;setEdited(true);setDraft(next);}

 useEffect(()=>{
  if(!editable||!draft||busy)return;const controller=new AbortController();
  const timer=window.setTimeout(()=>{void requestAIWorkbench({route:"skill-authoring-validate",method:"POST",scope,path:path+"/validate",body:draft,signal:controller.signal}).then(()=>{if(!controller.signal.aborted)setValidation({draft,valid:true});}).catch(error=>{if(!controller.signal.aborted)setValidation({draft,valid:false,unavailable:!(error instanceof AIWorkbenchError)||error.code!=="INVALID_REQUEST"});});},250);
  return()=>{window.clearTimeout(timer);controller.abort();};
 },[editable,draft,busy,scope,path,validationRetry]);
 function display(result:AISkillAuthoringJob){if(result.key!==job?.key)setRecoveredRevision(undefined);dirty.current=false;setEdited(false);setJob(result);setDraft(result.draft??undefined);setValidation(undefined);setSaveCheck(result.savePending);}
 async function perform(run:(signal:AbortSignal)=>Promise<AISkillAuthoringJob>,mode?:"save"|"recover-save"){
  if(action.current)return;const controller=new AbortController();action.current=controller;setBusy(true);setNotice("");
  try{
   const result=await run(controller.signal);
   if(!controller.signal.aborted){
    if(mode==="recover-save"&&dirty.current&&result.key===job?.key&&result.state==="COMPLETE"&&!result.savePending&&!result.savedSkillId){
     setSaveCheck(false);
     if(result.revision!==job.revision||result.generationKey!==job.generationKey)setRecoveredRevision({job:result,observedRevision:initialJob?.revision,observedGeneration:initialJob?.generationKey});
    }else display(result);
    if(!initialJob)void list.refetch();
    void queryClient.invalidateQueries({queryKey:["chat-turn",scope.userId,scope.organizationId]});
   }
  }
  catch(error){if(!controller.signal.aborted){setNotice(authoringError(error));if(mode==="save"&&(!(error instanceof AIWorkbenchError)||!["INVALID_REQUEST","NOT_FOUND"].includes(error.code)))setSaveCheck(true);}}
  finally{action.current=null;if(!controller.signal.aborted)setBusy(false);}
 }
 const checked=Boolean(validation&&validation.draft===draft&&validation.valid);
 return <Card className={styles.authoringCard}>
  <header className={styles.authoringHeader}><div><h2>技能草稿</h2><p>编辑后确认保存到我的技能。</p></div>{onClose&&<Button variant="ghost" onClick={onClose}>收起</Button>}</header>
  {notice&&<p role="status">{notice}</p>}
  {list.isError&&<p role="status">暂时无法读取原草稿。 <Button variant="outline" onClick={()=>void list.refetch()}>重新读取</Button></p>}
  {list.data?.drafts.filter(item=>item.purpose!=="chat-text").map(item=><Button key={item.generationKey??item.key} variant="outline" disabled={busy||saveCheck} onClick={()=>display(item)}>{item.draft?.displayName||generationStatus[item.state]}</Button>)}
  {!initialJob&&!job&&list.isSuccess&&!list.data.drafts.length?<p>暂无草稿。可以直接在聊天中描述要创建的标题技能。</p>:null}
  {job&&<div className={styles.jobContent}>
   <p role="status">{job.superseded?"这次修改未覆盖当前草稿。":generationStatus[job.state]}</p>
   {!readOnly&&(job.state==="READY"||job.state==="UNKNOWN"||job.state==="QUOTED")&&!initialJob?<Button variant="outline" disabled={busy} onClick={()=>void perform(signal=>requestAIWorkbench({route:(job.generationKey??job.key)[14]==="7"?"skill-authoring-recover-generation":"skill-authoring-generate",method:"POST",scope,path:path+"/"+(job.generationKey??job.key)+((job.generationKey??job.key)[14]==="7"?"/recover-generation":"/generate"),signal}))}>核实原生成请求</Button>:null}
   {job.savedSkillId?<p className={styles.savedNotice}>已保存到我的技能。 <Link href={"/workbench/agents/skills/mine?skillId="+job.savedSkillId}>查看我的技能</Link></p>:!readOnly&&(saveCheck||job.savePending)?<Button variant="outline" disabled={busy} onClick={()=>void perform(signal=>requestAIWorkbench({route:"skill-authoring-save-recover",method:"POST",scope,path:path+"/"+job.key+"/recover-save",signal}),"recover-save")}>核实原保存</Button>:draft&&<div className={styles.draftEditor}>
    <div className={styles.editorFields}>{editorFields.map(([key,label])=><label className={key==="displayName"||key==="name"?styles.halfField:styles.field} key={key}>{label}{key==="instructions"?<Textarea rows={7} value={draft[key]} disabled={busy||!editable} onChange={event=>updateDraft({...draft,[key]:event.target.value})}/>:<Input value={draft[key]} disabled={busy||!editable} onChange={event=>updateDraft({...draft,[key]:event.target.value})}/>}</label>)}</div>
    {draft.script&&<div className={styles.scriptPanel}><div className={styles.scriptHeader}><p>包含 Python 脚本</p>{editable&&<Button variant="ghost" disabled={busy} onClick={()=>updateDraft({...draft,script:undefined})}>移除脚本</Button>}</div><details><summary>查看代码</summary><label className={styles.field}>脚本用途<Input value={draft.script.description} disabled={busy||!editable} onChange={event=>{if(draft.script)updateDraft({...draft,script:{...draft.script,description:event.target.value}});}}/></label><label className={styles.field}>Python 代码<Textarea className={styles.codeInput} rows={7} spellCheck={false} value={draft.script.source} disabled={busy||!editable} onChange={event=>{if(draft.script)updateDraft({...draft,script:{...draft.script,source:event.target.value}});}}/></label></details></div>}
    {stale&&edited?<p role="status">当前草稿已更新，你的人工编辑仍保留。 <Button variant="outline" disabled={busy} onClick={()=>{if(latest)display(latest);}}>使用最新草稿并替换当前编辑</Button></p>:null}
    {editable&&onRevise&&<Button variant="outline" disabled={busy||stale} onClick={()=>onRevise(draft)}>让 Chat 修改</Button>}
    {validation?.draft===draft&&!validation.valid?<p role="status">{validation.unavailable?"暂时无法检查格式。":"格式未通过，请检查名称、正文或脚本长度。"}{validation.unavailable&&<Button variant="outline" onClick={()=>setValidationRetry(value=>value+1)}>重新检查</Button>}</p>:null}
    {editable&&<footer className={styles.saveFooter}><p>确认后保存为个人私有草稿。</p><Button disabled={busy||!checked||stale} onClick={()=>void perform(signal=>requestAIWorkbench({route:"skill-authoring-save-package",method:"POST",scope,path:path+"/"+job.key+"/save-package",body:{revision:job.revision??job.key,draft},key:crypto.randomUUID(),signal}),"save")}>保存到我的技能</Button></footer>}
   </div>}
  </div>}
 </Card>;
}

type Quote = {key: string; quote: string; expiresAt: string};
type PendingGeneration = {key: string};
const generalEditorFields = [
  ["displayName", "技能名称"], ["name", "技能标识（小写英文字母、数字和连字符）"],
  ["description", "用途"], ["instructions", "正文"],
] as const;
const generalGenerationStatus = {
  QUOTED: "待确认生成", READY: "生成结果待核实", COMPLETE: "已生成草稿",
  FAILED: "本次未能生成草稿，请检查模型设置或可用点数后重试。",
  INVALID: "草稿格式未通过，可以修改需求后重试。",
  UNKNOWN: "本次生成结果未能确认。",
} as const;

function generalAuthoringError(error: unknown) {
  const code = error instanceof AIWorkbenchError ? error.code : "";
  if (code === "PROPOSAL_STALE") return "原确认已失效，请查看当前草稿后再试。";
  if (code === "INVALID_REQUEST") return "请检查填写内容和格式后重试。";
  if (code === "IDEMPOTENCY_CONFLICT") return "草稿已在其他页面更新或保存，请读取原结果。";
  return "本次操作暂未确认，请核实原请求。";
}

function readGenerationPointer(pointer: string): PendingGeneration | undefined {
  const raw = sessionStorage.getItem(pointer);
  if (raw && isAcquisitionUUID(raw)) return {key: raw};
}

function quoteExpired(quote: Quote): boolean {
  return Date.now() >= Date.parse(quote.expiresAt);
}

export function GeneralSkillAuthoring({ scope, memberId, authorizationKey, conversation, selected, initialRequirement, expectedVersion, expectedDigest, onClose, onUseTextSkill }: {
  scope: AIScope;
  memberId: string;
  authorizationKey: string;
  conversation: string;
  selected?: AISkillAuthoringChoice;
  expectedVersion: string | null;
  expectedDigest: string | null;
  initialRequirement: string;
  onClose: () => void;
  onUseTextSkill?: (skill: SelectedTextSkill) => void;
}) {
  const catalog = useCatalog(scope, authorizationKey, true);
  const choices = catalog.data?.ready ? [catalog.data.skill, catalog.data.titleSkill] : [];
  const candidate = !expectedVersion && !expectedDigest ? choices[0] : choices.find(choice => choice && String(choice.version) === expectedVersion && choice.digest === expectedDigest);
  const path = "chat/conversations/" + conversation + "/skill-authoring";
  const memberScope = useMemo(() => ({userId: scope.userId, organizationId: scope.organizationId}), [scope.userId, scope.organizationId]);
  const pointer = "ai-workbench:skill-generation:" + JSON.stringify([scope.userId, scope.organizationId, memberId, conversation]);
  const [requirement, setRequirement] = useState(initialRequirement);
  const [revisionRequest, setRevisionRequest] = useState("");
  const [prepared, setPrepared] = useState<{input: AIAuthoringPreview; quote?: Quote}>();
  const [prepRetry, setPrepRetry] = useState(0);
  const [original, setOriginal] = useState<PendingGeneration>();
  const [pointerReady, setPointerReady] = useState(false);
  const [job, setJob] = useState<AISkillAuthoringJob>();
  const startingSkill = selected ?? candidate;
  const matchingVersion = job?.purpose === "chat-text" ? generalAuthoringVersion : 2;
  const skill = !job || startingSkill?.version === matchingVersion ? startingSkill : choices.find(choice => choice?.version === matchingVersion);
  const general = job ? job.purpose === "chat-text" : skill?.version === generalAuthoringVersion;
  const [draft, setDraft] = useState<AITitleSkillDraft>();
  const [validation, setValidation] = useState<{draft: AITitleSkillDraft; valid: boolean; unavailable?: boolean}>();
  const [validationRetry, setValidationRetry] = useState(0);
  const [saveCheck, setSaveCheck] = useState(false);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const action = useRef<AbortController | null>(null);
  const lastConfirmation = useRef<{input: AIAuthoringPreview; quote: Quote} | undefined>(undefined);
  const list = useQuery({
    queryKey: ["chat-skill-authoring-list", scope.userId, scope.organizationId, memberId, authorizationKey, conversation],
    queryFn: ({ signal }) => requestAIWorkbench({ route: "skill-authoring-list", method: "GET", scope, path, signal }),
    retry: false, staleTime: 0,
  });

  useEffect(() => {
    let live = true;
    queueMicrotask(() => {
      if (!live) return;
      try { setOriginal(readGenerationPointer(pointer)); }
      catch { setNotice("当前浏览器无法读取原生成请求。"); }
      setPointerReady(true);
    });
    return () => { live = false; action.current?.abort(); };
  }, [pointer]);

  const pending = useMemo(() => original ?? (job?.state === "READY" ? {key: job.generationKey ?? job.key} : undefined), [original, job]);
  const recheck = pending ?? (job?.state === "UNKNOWN" ? {key: job.generationKey ?? job.key} : undefined);
  const editable = Boolean(draft && job && !job.savedSkillId && !job.savePending && !saveCheck && !pending);
  const input = useMemo<AIAuthoringPreview | undefined>(() => {
    if (!skill || !pointerReady || pending || saveCheck || job?.savePending || job?.savedSkillId) return;
    const text = (job && draft ? revisionRequest : requirement).trim();
    if (!text || new TextEncoder().encode(text).length > 8192) return;
    const next = {requirement: text, skillVersion: skill.version, skillDigest: skill.digest};
    if (!job) return next;
    if (skill.version < 2 || !draft) return;
    return {...next, root: job.key, baseRevision: job.revision ?? job.key, snapshot: draft};
  }, [skill, pointerReady, pending, saveCheck, job, draft, revisionRequest, requirement]);

  useEffect(() => {
    if (!input || busy) return;
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      void requestAIWorkbench({route: "skill-authoring-preview", method: "POST", scope: memberScope,
        path: path + "/preview", body: input, signal: controller.signal})
        .then(quote => { if (!controller.signal.aborted) setPrepared({input, quote}); })
        .catch(() => { if (!controller.signal.aborted) setPrepared({input}); });
    }, 250);
    return () => { window.clearTimeout(timer); controller.abort(); };
  }, [input, busy, memberScope, path, prepRetry]);

  useEffect(() => {
    if (!editable || !draft || busy) return;
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      void requestAIWorkbench({route: "skill-authoring-validate", method: "POST", scope: memberScope,
        path: path + "/validate", body: general ? {...draft,skillVersion:generalAuthoringVersion} : draft, signal: controller.signal})
        .then(() => { if (!controller.signal.aborted) setValidation({draft, valid: true}); })
        .catch(error => {
          if (!controller.signal.aborted) setValidation({draft, valid: false, unavailable: !(error instanceof AIWorkbenchError) || error.code !== "INVALID_REQUEST"});
        });
    }, 250);
    return () => { window.clearTimeout(timer); controller.abort(); };
  }, [draft, editable, busy, general, memberScope, path, validationRetry]);

  function clearGeneration(key: string) {
    try { if (readGenerationPointer(pointer)?.key === key) sessionStorage.removeItem(pointer); } catch { /* original server result remains readable */ }
    setOriginal(current => current?.key === key ? undefined : current);
    if (lastConfirmation.current?.quote.key === key) lastConfirmation.current = undefined;
  }

  function displayJob(result: AISkillAuthoringJob) {
    setJob(result);
    setDraft(result.draft ?? undefined);
    setRevisionRequest("");
    setValidation(undefined);
    setSaveCheck(result.savePending);
    if (result.state !== "READY" && result.state !== "QUOTED") clearGeneration(result.generationKey ?? result.key);
  }

  async function perform(run: (signal: AbortSignal) => Promise<void>, failed?: (error: unknown) => void) {
    if (action.current) return;
    const controller = new AbortController();
    action.current = controller;
    setBusy(true);
    setNotice("");
    try { await run(controller.signal); }
    catch (error) {
      if (!controller.signal.aborted) {
        setNotice(generalAuthoringError(error));
        failed?.(error);
      }
    } finally {
      if (action.current === controller) action.current = null;
      if (!controller.signal.aborted) setBusy(false);
    }
  }

  function startGeneration(confirmed: {input: AIAuthoringPreview; quote: Quote}) {
    const {quote} = confirmed;
    if (quoteExpired(quote)) {
      setPrepared(undefined);
      setPrepRetry(retry => retry + 1);
      setNotice("原确认已过期，正在重新准备。");
      return;
    }
    const next = {key: quote.key};
    try { sessionStorage.setItem(pointer, next.key); }
    catch { setNotice("浏览器无法保存恢复位置，请检查存储后重试。"); return; }
    setOriginal(next);
    lastConfirmation.current = confirmed;
    generate(next);
  }

  function generate(request: PendingGeneration) {
    const confirmed = lastConfirmation.current;
    const confirmNow = confirmed?.quote.key === request.key && !quoteExpired(confirmed.quote);
    void perform(async signal => {
      const result = confirmNow
        ? await requestAIWorkbench({route: "skill-authoring-confirm", method: "POST", scope, path: path + "/" + request.key + "/confirm", signal,
            body: {...confirmed.input, quote: confirmed.quote.quote}})
        : request.key[14] === "7"
          ? await requestAIWorkbench({route: "skill-authoring-recover-generation", method: "POST", scope,
              path: path + "/" + request.key + "/recover-generation", signal})
          : await requestAIWorkbench({route: "skill-authoring-generate", method: "POST", scope, path: path + "/" + request.key + "/generate", signal});
      if (!signal.aborted) {displayJob(result); void list.refetch();}
    }, error => {
      if (error instanceof AIWorkbenchError && error.code === "PROPOSAL_STALE") {
        if (confirmNow) {
          lastConfirmation.current = undefined;
          setNotice("请先核实原生成请求。");
          return;
        }
        clearGeneration(request.key);
        setPrepared(undefined);
        setPrepRetry(retry => retry + 1);
        void catalog.refetch();
        void list.refetch();
      }
    });
  }

  const checked = validation?.draft === draft && validation?.valid;
  function save(recovery: boolean) {
    if (!job || (!recovery && (!draft || !checked))) return;
    void perform(async signal => {
      const result = recovery
        ? await requestAIWorkbench({route: "skill-authoring-save-recover", method: "POST", scope, path: path + "/" + job.key + "/recover-save", signal})
        : await requestAIWorkbench({route: "skill-authoring-save-package", method: "POST", scope, path: path + "/" + job.key + "/save-package",
            body: {revision: job.revision ?? job.key, draft}, key: crypto.randomUUID(), signal});
      if (!signal.aborted) {
        if (recovery && !result.savePending && !result.savedSkillId && result.revision === job.revision) {
          setJob(result);
          setSaveCheck(false);
          setNotice("尚无保存记录，请核对内容后确认保存。");
        } else {
          displayJob(result);
        }
        void list.refetch();
      }
    }, error => {
      if (!(error instanceof AIWorkbenchError) || !["INVALID_REQUEST", "NOT_FOUND"].includes(error.code)) setSaveCheck(true);
      void list.refetch();
    });
  }

  const currentPrep = input && prepared?.input === input ? prepared : undefined;
  const others = list.data?.drafts.filter(item => item.purpose === "chat-text" && (item.generationKey ?? item.key) !== (job?.generationKey ?? job?.key));
  const genButton = (label: string) => <Button disabled={busy || !currentPrep?.quote} onClick={() => {
    if (currentPrep?.quote) startGeneration({input: currentPrep.input, quote: currentPrep.quote});
  }}>{label}</Button>;
  const prepFailure = input && currentPrep && !currentPrep.quote
    ? <p role="status">暂时无法准备生成。 <Button variant="outline" onClick={() => {setPrepared(undefined); setPrepRetry(value => value + 1);}}>重试</Button></p>
    : null;
  return <Card className={styles.authoringCard}>
    <header className={styles.authoringHeader}>
      <div><h2>{general ? "创建技能" : "创建商品标题技能"}</h2><p>{general ? "描述一套文字处理规则，编辑后保存到 Chat。" : "生成一份个人商品标题技能，编辑后确认保存。"}</p></div>
      <Button variant="ghost" onClick={onClose}>收起</Button>
    </header>
    {notice && <p role="status">{notice}</p>}
    {list.isError && <p role="status">暂时无法读取本会话的技能草稿。 <Button variant="outline" onClick={() => void list.refetch()}>重新读取</Button></p>}
    {others?.length ? <div className={styles.draftHistory}>{others.map(item =>
      <Button key={item.generationKey ?? item.key} variant="outline" disabled={busy || Boolean(pending)} onClick={() => displayJob(item)}>
        {item.state === "READY" || item.state === "UNKNOWN" ? "查看待核实的生成" : item.draft?.displayName || generalGenerationStatus[item.state]}
      </Button>)}</div> : null}
    {recheck && <Button variant="outline" disabled={busy} onClick={() => generate(recheck)}>核实原生成请求</Button>}
    {job && <div className={styles.jobContent}>
      <p className={styles.generationStatus} role="status">{job.superseded ? "这次修改未覆盖当前草稿。" : generalGenerationStatus[job.state]}</p>
      {job.savedSkillId ? <div className={styles.savedNotice}><p>已保存到我的技能。 <Link className="underline" href={"/workbench/agents/skills/mine?skillId=" + job.savedSkillId}>查看我的技能</Link></p>
        {job.savedSkill && onUseTextSkill && <Button type="button" onClick={() => {if (job.savedSkill) onUseTextSkill({...job.savedSkill, name:job.draft?.displayName || "个人文字技能"});}}>在 Chat 中使用</Button>}</div>
        : saveCheck || job.savePending ? <Button variant="outline" disabled={busy} onClick={() => save(true)}>核实原保存</Button>
        : editable && draft && <div className={styles.draftEditor}>
          <div className={styles.editorFields}>{generalEditorFields.map(([key, label]) => <label className={key === "displayName" || key === "name" ? styles.halfField : styles.field} key={key}>{label}
            {key === "instructions" ? <Textarea rows={7} value={draft[key]} disabled={busy} onChange={event => setDraft({...draft, [key]: event.target.value})} />
              : <Input value={draft[key]} disabled={busy} onChange={event => setDraft({...draft, [key]: event.target.value})} />}
          </label>)}</div>
          {draft.script && <div className={styles.scriptPanel}>
            <div className={styles.scriptHeader}><p>包含 Python 脚本</p><Button variant="ghost" disabled={busy} onClick={() => setDraft({...draft, script: undefined})}>移除脚本</Button></div>
            {draft.script.description && <p>{draft.script.description}</p>}
            <details>
              <summary>查看代码</summary>
              <label className={styles.field}>脚本用途<Input value={draft.script.description} disabled={busy}
                onChange={event => {if (draft.script) setDraft({...draft, script: {...draft.script, description: event.target.value}});}} /></label>
              <label className={styles.field}>Python 代码<Textarea className={styles.codeInput} rows={7} spellCheck={false} value={draft.script.source} disabled={busy}
                onChange={event => setDraft({...draft, script: {source: event.target.value, description: ""}})} /></label>
            </details>
          </div>}
          {skill && skill.version >= 2 && <div className={styles.revisionPanel}>
            <label className={styles.field}>让 Chat 修改<Textarea rows={2} placeholder={general ? "例如：缺少负责人时标记待确认" : "例如：也保留商品型号，并缩短正文规则"} value={revisionRequest} disabled={busy} onChange={event => setRevisionRequest(event.target.value)} /></label>
            {revisionRequest.trim() && <p>本次修改会使用当前草稿{draft.script ? "和其中的 Python 代码" : ""}。</p>}
            {genButton("生成修改")}
            {prepFailure}
          </div>}
          {validation && validation.draft === draft && !validation.valid && <p role="status">
            {validation.unavailable ? "暂时无法检查格式。" : "格式未通过，请检查名称、正文或脚本长度。"}
            {validation.unavailable && <Button variant="outline" onClick={() => setValidationRetry(value => value + 1)}>重新检查</Button>}
          </p>}
          <footer className={styles.saveFooter}>
            <p>{general ? "保存后仅本人可见，可在 Chat 中选用。" : "确认后保存为个人私有草稿。"}{draft.script ? "包含 Python 脚本。" : ""}</p>
            <Button disabled={busy || !checked} onClick={() => save(false)}>{general ? "保存并在 Chat 启用" : "保存到我的技能"}</Button>
          </footer>
        </div>}
    </div>}
    {!pending && !job && <div className={styles.requirementForm}>
      {skill ? <p className={styles.sourceVersion}>{skill.name} · 官方 v{skill.version}</p> : <p>当前没有可用的创作技能，可以查看原草稿。</p>}
      <label className={styles.field}>技能需求<Textarea rows={5} placeholder={general ? "例如：把交接笔记整理成待办表，缺少负责人的标记为待确认。" : "例如：保留商品品牌和型号，用 Python 合并标题中的连续空格。"} value={requirement} disabled={busy || !skill} onChange={event => setRequirement(event.target.value)} /></label>
      {genButton("生成草稿")}
      {prepFailure}
    </div>}
    {job && !pending && !saveCheck && !job.savePending && <Button variant="ghost" disabled={busy} onClick={() => {
      setJob(undefined); setPrepared(undefined); setDraft(undefined); setValidation(undefined); setRevisionRequest("");
    }}>准备另一个草稿</Button>}
  </Card>;
}
