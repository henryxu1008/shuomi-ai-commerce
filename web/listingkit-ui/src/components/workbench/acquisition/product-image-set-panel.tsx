"use client";

import Image from "next/image";
import Link from "next/link";
import {useCallback,useEffect,useMemo,useRef,useState} from "react";
import {z} from "zod";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Select} from "@/components/ui/select";
import {Input} from "@/components/ui/input";
import {ResourceDialog} from "../resources/resource-dialog";
import {SkillPicker,ImageMethodSummary} from "../skills/skill-picker";
import type {SkillSelection} from "@/lib/contracts/skill";
import {SourceImageUploader} from "../collections/source-image-uploader";
import {configurationRequest} from "@/lib/api/agent-configuration";
import {catalogEntrySchema} from "@/lib/contracts/agent-configuration";
import {executionModeLabels} from "../agents/execution-mode-settings";
import {executionModeViewSchema} from "@/lib/contracts/execution-mode";
import {useConfigurationRead} from "../agents/agent-configuration-hooks";
import {imageTemplatesPageSchema,imageTemplateSchema,carouselTasks,detailTasks,type ImageAgentTemplate} from "@/lib/contracts/image-set-configuration";
import {genericImagePresetForSkill,matchesGenericImagePreset} from "@/lib/image-template-presets";
import {ImageSetError,imageSetRequest,type ImageSetScope} from "@/lib/api/product-image-set";
import {isAcquisitionUUID} from "@/lib/contracts/product-acquisition";
import {imageSetApprovalSchema,imageSetRunSchema,imageSetSourcesSchema,imageSetRecentSchema,imageSetInventorySchema,imageSetPreviewSchema,imageSetAcceptedSchema,imageSetRequirementsSchema,imageSetRequestSchema,type ImageSetRun,type ImageSetPrepare,type ImageSetChoice,type ImageSetSelection,type ImageSetRequirements,type ImageSetInventory,type ImageSetRoute,type ImageRouteSelection} from "@/lib/contracts/product-image-set";

import {ImagePlanRoutePicker} from "./image-plan-route-picker";
import {ImageTextCopy,imageTextLanguageNames} from "./image-text-copy";

type Props={translationItemId?:string;kind:ImageSetScope["kind"];contextId:string;target?:ImageSetPrepare["target"];effectiveVersion?:string;applyReceiptId?:string;onSaved?:()=>void;initialRunId?:string;initialSkill?:SkillSelection};
type Intent={action:ImageSetRoute;runId?:string;requestKey?:string;body:unknown};
type RetainedRequest={key:string;raw:string;intent:Intent;reason?:"IMAGE_ROUTE_CHANGED"};
type PhotoView={url:string;label:string;width?:number;height?:number};
type ChoiceView={key:string;url:string;label:string;choice:ImageSetChoice};
const stateNames:Record<string,string>={awaiting_plan_approval:"等待点数确认",executing:"生成中",evaluating:"处理结果中",repairing:"处理结果中",awaiting_final_approval:"等待人工选择",blocked:"已阻止",completed:"已保存正式素材",failed:"生成失败",cancelled:"已取消",pending:"等待执行",accepted:"已生成",rejected:"失败"};
const reasons:Record<string,string>={IMAGE_ROUTE_CHANGED:"所选图片模型版本已不可用，请重新读取并显式选择模型后准备新计划。原任务和准确技能版本保留，不会自动切换百炼。",IMAGE_TEXT_REVIEW_CHANGED:"审核译文已变化或不可用于本次图片，请重新读取并确认文案后准备计划；原请求和准确技能版本已保留。",IMAGE_METHOD_UNAVAILABLE:"所选图片技能版本已不可用，请重新选择或明确不使用技能，再准备新计划。原请求仍可核实。",OUTCOME_UNKNOWN:"本次请求结果尚未确认，请核实原请求。",IMAGE_UNAVAILABLE:"当前图片执行或依赖不可用。",IMAGE_AGENT_DISABLED:"请先由管理员启用商品图片智能体。",IMAGE_BLOCKED:"当前素材、任务依据或平台位置不满足要求，请核实后重新准备。",IMAGE_CONFIGURATION_CHANGED:"配置已变化，请重新读取并准备计划。",IMAGE_CONFLICT:"原任务或版本已变化，请读取当前结果。",IMAGE_SELECTION_CHANGED:"正式图片集合或规则已变化，请刷新后重新选择。",FORBIDDEN:"当前身份没有这项操作的权限。",IMAGE_ASSETS_NOT_READY:"图片或结算依据尚未收束，请继续核实原任务。",IMAGE_NOT_FOUND:"原请求暂未找到，请保留原编号继续核实。"};
const labels=new Map<string,string>([...carouselTasks,...detailTasks].map(task=>[task.purpose,task.label]));

export function ProductImageSetPanel(props:Props){
 const context=useWorkbenchContext(),userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 if(!userId||!organizationId||context.isSwitching||context.isLoading||context.error||context.blockingError||context.selectionRequired)return null;
 const permissions=context.permissions??[];
 if(!permissions.includes("listingkit.image_agent.read"))return null;
 return <ScopedImageSetPanel key={`${userId}:${organizationId}:${props.kind}:${props.contextId}:${permissions.join(",")}`} {...props} scope={{userId,organizationId,kind:props.kind,contextId:props.contextId}}/>;
}

function ScopedImageSetPanel({scope,target,effectiveVersion,applyReceiptId,onSaved,initialRunId,initialSkill,translationItemId}:Props&{scope:ImageSetScope}){
 const context=useWorkbenchContext();
 const [modeNonce,setModeNonce]=useState(0);
 const executionMode=useConfigurationRead(scope,"execution-modes/registered_agent/product.image.agent",executionModeViewSchema,modeNonce);
 const storageKey=`product-image-set:${scope.userId}:${scope.organizationId}:${scope.kind}:${scope.contextId}`;
 const active=useRef(true),flight=useRef(false),abort=useRef<AbortController|null>(null);
 const savedCallback=useRef(onSaved),notified=useRef("");
 const restoredRun=useRef(""),rulesBinding=useRef(""),readVersion=useRef(0);
 const runTemplateBinding=useRef(""),runSourceBinding=useRef("");
 const templateEdited=useRef(false);
 const [templateNotice,setTemplateNotice]=useState("");
 useEffect(()=>{savedCallback.current=onSaved},[onSaved]);
 const [entry,setEntry]=useState<z.infer<typeof catalogEntrySchema>>(),[templates,setTemplates]=useState<ImageAgentTemplate[]>([]),[cursor,setCursor]=useState("");
 const [template,setTemplate]=useState<ImageAgentTemplate>(),[sources,setSources]=useState<z.infer<typeof imageSetSourcesSchema>>(),[run,setRun]=useState<ImageSetRun>();
 const [routeDraft,setRouteDraft]=useState<ImageRouteSelection|null|undefined>(),[routeReady,setRouteReady]=useState(true),[routeNonce,setRouteNonce]=useState(0);
 const routeRun=useRef("");
 const [textDraft,setTextDraft]=useState<{key:string;tasksKey:string;value:ImageSetPrepare["text"]|null}>();
 const [runTemplate,setRunTemplate]=useState<ImageAgentTemplate>();
 const [runSources,setRunSources]=useState<z.infer<typeof imageSetSourcesSchema>>();
 const [method,setMethod]=useState<SkillSelection|undefined>(initialSkill),[methodReady,setMethodReady]=useState(!initialSkill);
 const initialPreset=genericImagePresetForSkill(initialSkill),hasInitialSkill=!!initialSkill;
 const onMethodChange=useCallback((value:SkillSelection|undefined,ready:boolean)=>{setMethod(value);setMethodReady(ready)},[]);
 const [priorRuns,setPriorRuns]=useState<ImageSetRun[]>([]);
 const [recent,setRecent]=useState<z.infer<typeof imageSetRecentSchema>["items"]>([]),[inventory,setInventory]=useState<ImageSetInventory>();
 const [recentCursor,setRecentCursor]=useState("");
 const [requirements,setRequirements]=useState<ImageSetRequirements>(),[platform,setPlatform]=useState<"product"|"shein">("product");
 const [restoredTarget,setRestoredTarget]=useState<ImageSetPrepare["target"]>();
 const editorTarget=restoredTarget??target;
 const [shared,setShared]=useState<string[]>([]),[carousel,setCarousel]=useState<string[]>([]),[detail,setDetail]=useState<string[]>([]),[tasks,setTasks]=useState<string[]>([]);
 const [positions,setPositions]=useState<NonNullable<ImageSetPrepare["officialPlacements"]>>({});
 const [selected,setSelected]=useState<ChoiceView[]>([]),[regenerate,setRegenerate]=useState<string[]>([]);
 const [intent,setIntent]=useState<Intent|null>(null),[busy,setBusy]=useState(false),[loading,setLoading]=useState(true),[error,setError]=useState(""),[message,setMessage]=useState("");
 const savedIntentRaw=useRef<string|null>(null);
 const [retainedRequests,setRetainedRequests]=useState<RetainedRequest[]>([]);
 const routeRejectedPlan=!!run&&retainedRequests.some(record=>{const body=record.intent.body as {planRevision?:number;planDigest?:string;quoteDigest?:string};return record.reason==="IMAGE_ROUTE_CHANGED"&&record.intent.action==="confirm"&&record.intent.runId===run.runId&&body.planRevision===run.planRevision&&body.planDigest===run.planDigest&&body.quoteDigest===run.quoteDigest});
 const rejectedPreparations=retainedRequests.filter(record=>record.reason!=="IMAGE_ROUTE_CHANGED"&&record.intent.action==="prepare");
 const needsPreparationReview=rejectedPreparations.length>0;
 const textKey=JSON.stringify([sources?.source,template?.templateId,template?.version,template?.image.language,rejectedPreparations.map(record=>record.key),needsPreparationReview?tasks:null]);
 const textSelection=textDraft?.key===textKey?textDraft.value?.mode==="reviewed"&&textDraft.tasksKey!==JSON.stringify(tasks)?null:textDraft.value:needsPreparationReview?null:{mode:"none" as const};
 const rejectedRegenerations=retainedRequests.filter(record=>record.reason!=="IMAGE_ROUTE_CHANGED"&&record.intent.action==="regenerate"&&record.intent.runId===run?.runId);
 const needsRegenerationReview=rejectedRegenerations.length>0;
 const regenerationTextKey=JSON.stringify([run?.runId,run?.plan.Source,run?.template,regenerate,rejectedRegenerations.map(record=>record.key)]);
 const [regenerationTextDraft,setRegenerationTextDraft]=useState<{key:string;value:ImageSetPrepare["text"]|null}>();
 const regenerationTextSelection=regenerationTextDraft?.key===regenerationTextKey?regenerationTextDraft.value:null;
 const correctingRegeneration=needsRegenerationReview&&!!run?.regenerationAvailable&&regenerate.length>0;
 const [photo,setPhoto]=useState<PhotoView>();
 const [preview,setPreview]=useState<{command:ImageSetSelection;digest:string}|null>(null);
 const [mediaBlocked,setMediaBlocked]=useState(false);
 const blockMedia=useCallback((blocked:boolean)=>setMediaBlocked(blocked),[]);
 const {userId,organizationId,kind,contextId}=scope;
 const stableScope=useMemo(()=>({userId,organizationId,kind,contextId}),[userId,organizationId,kind,contextId]);
 const locked=busy||!!intent||mediaBlocked;
 const unresolvedRun=!run?.releaseSummary&&(!!run?.pendingCommand||!!run?.recoverableEffects?.length);
 const releasedIntent=!!intent&&imageIntentTargetsReleasedRun(intent,run);
 useEffect(()=>context.registerOrganizationSwitchGuard(()=>!flight.current),[context]);
 useEffect(()=>{active.current=true;return()=>{active.current=false;abort.current?.abort()}},[]);
 const fail=useCallback((value:unknown)=>{setError(value instanceof ImageSetError?value.code:value instanceof Error?value.message:"IMAGE_UNAVAILABLE")},[]);
 const remember=useCallback((p:ImageSetRun)=>{if(!active.current)return;if(routeRun.current!==p.runId){routeRun.current=p.runId;setRouteDraft(undefined)}setRun(p);localStorage.setItem(storageKey+":run",p.runId)},[storageKey]);
 const clearIntent=useCallback(()=>{localStorage.removeItem(storageKey+":intent");savedIntentRaw.current=null;setIntent(null)},[storageKey]);
 function retainOriginalRequest(command:Intent,reason?:"IMAGE_ROUTE_CHANGED"){
  const raw=savedIntentRaw.current;
  try{
   if(!raw||localStorage.getItem(storageKey+":intent")!==raw)throw new Error("changed");
   const key=retainedImageRequestKey(storageKey,command,reason),existing=localStorage.getItem(key);
   if(existing!==null&&existing!==raw)throw new Error("conflict");
   localStorage.setItem(key,raw);
   if(localStorage.getItem(key)!==raw)throw new Error("not retained");
   // Retain the exact verified request before releasing its pending slot.
   const retained=readRetainedImageRequests(storageKey);
   clearIntent();setRetainedRequests(retained);
  }catch{throw new Error("原请求未能完整保留，请继续核实；任务切换和新计划暂未解锁。")}
 }
 function installTemplate(t:ImageAgentTemplate,evidence?:Record<string,string>){
  setTemplate(t);setTextDraft(undefined);setPreview(null);setTasks([...t.image.carousel,...t.image.detail].filter(task=>{const definition=[...carouselTasks,...detailTasks].find(v=>v.purpose===task.purpose);return !definition||!("evidence" in definition)||!!evidence?.[definition.evidence]}).map(task=>task.id));
 }
 const readRun=useCallback(async(id:string,signal?:AbortSignal,onVerifiedRead?:(p:ImageSetRun)=>boolean)=>{
  const version=++readVersion.current,isCurrent=()=>active.current&&!signal?.aborted&&version===readVersion.current;
  const p=await imageSetRequest(stableScope,"read",imageSetRunSchema,{runId:id,signal});if(!isCurrent())return;remember(p);
  if(restoredRun.current!==p.runId){
   restoredRun.current=p.runId;setPhoto(undefined);rulesBinding.current="";setRequirements(undefined);setInventory(undefined);setPriorRuns([]);setPreview(null);setSelected([]);setRegenerate([]);setRegenerationTextDraft(undefined);
   setPlatform(p.plan.Target.Platform);setRestoredTarget(p.plan.Target.Platform==="shein"?p.plan.Target:undefined);
   setPositions(Object.fromEntries(p.slots.flatMap(slot=>slot.recipe.OfficialPlacement?[[slot.slotId,slot.recipe.OfficialPlacement]]:[])));
   setRunTemplate(undefined);runTemplateBinding.current="";
  }
  if(onVerifiedRead?.(p))return p;
  const sourceBinding=JSON.stringify([p.runId,sourceIdentity(p.plan.Source)]);
  if(runSourceBinding.current!==sourceBinding){
   runSourceBinding.current="";setRunSources(undefined);setPreview(null);setSelected([]);
   try{
    const query=new URLSearchParams({effectiveCatalogVersion:p.plan.Source.EffectiveVersion});if(p.plan.Source.ApplyReceiptID)query.set("applyReceiptId",p.plan.Source.ApplyReceiptID);
    const exact=await imageSetRequest(stableScope,"sources",imageSetSourcesSchema,{query:query.toString(),signal});if(!isCurrent())return;
    if(sourceIdentity(exact.source)!==sourceIdentity(p.plan.Source))throw new ImageSetError("IMAGE_CONFLICT",409);
    setRunSources(exact);runSourceBinding.current=sourceBinding;
   }catch(e){if(!isCurrent())return;fail(e)}
  }
  const templateBinding=JSON.stringify([p.runId,p.template]);
  if(runTemplateBinding.current!==templateBinding){
   try{
    const pinned=await configurationRequest(stableScope,`product.image.agent/templates/${p.template.templateId}/revisions/${p.template.revision}`,imageTemplateSchema,{signal});if(!isCurrent())return;
    if(pinned.templateId!==p.template.templateId||pinned.version!==p.template.revision)throw new Error("原任务模板版本不一致");
    setRunTemplate(pinned);runTemplateBinding.current=templateBinding;
   }catch(e){if(!isCurrent())return;fail(e)}
  }
  const current=await imageSetRequest(stableScope,"inventory",imageSetInventorySchema,{runId:id,signal});if(!isCurrent())return;setInventory(current);
  const binding=JSON.stringify([p.runId,p.plan.Target,p.plan.Source]);
  if(p.plan.Target.Platform==="shein"&&(rulesBinding.current!==binding||!["executing","evaluating","repairing"].includes(p.status))){
   setRequirements(undefined);setPreview(null);
   try{
    const value=await imageSetRequest(stableScope,"requirements",imageSetRequirementsSchema,{body:{target:p.plan.Target,effectiveCatalogVersion:p.plan.Source.EffectiveVersion,...p.plan.Source.ApplyReceiptID?{applyReceiptId:p.plan.Source.ApplyReceiptID}:{}},signal});if(!isCurrent())return;
    rulesBinding.current=binding;setRequirements(value);
   }catch(e){if(!isCurrent())return;rulesBinding.current="";fail(e)}
  }
  if(p.plan.Regeneration){
   const parent=await imageSetRequest(stableScope,"read",imageSetRunSchema,{runId:p.plan.Regeneration.RunID,signal});if(!isCurrent())return;
   setPriorRuns(old=>old[0]?.runId===parent.runId?[parent,...old.slice(1)]:[parent]);
  }else setPriorRuns([]);
  if(p.status==="completed"){setMessage("本次批准已保存为正式商品素材。");if(notified.current!==p.runId){notified.current=p.runId;savedCallback.current?.()}}
  return p;
 },[stableScope,remember,fail]);
 useEffect(()=>{
  const controller=new AbortController();abort.current=controller;
  void (async()=>{
   try{
    const saved=localStorage.getItem(storageKey+":intent");
    if(saved){
     const parsed=parseImageIntent(saved);
     if(parsed){savedIntentRaw.current=saved;setIntent(parsed)}
     else localStorage.removeItem(storageKey+":intent");
    }
    const agent=await configurationRequest(stableScope,"product.image.agent",catalogEntrySchema,{signal:controller.signal});if(controller.signal.aborted)return;setEntry(agent);
    if(!agent.canReadRuns)return;
    const query=new URLSearchParams();if(effectiveVersion)query.set("effectiveCatalogVersion",effectiveVersion);if(applyReceiptId)query.set("applyReceiptId",applyReceiptId);
    let source:z.infer<typeof imageSetSourcesSchema>|undefined;setSources(undefined);
    try{source=await imageSetRequest(stableScope,"sources",imageSetSourcesSchema,{query:query.toString(),signal:controller.signal});if(controller.signal.aborted)return;setSources(source);setRetainedRequests(readRetainedImageRequests(storageKey))}catch(e){if(controller.signal.aborted)return;fail(e)}
    // Recover the known run before optional template or recent-run discovery.
    const id=initialRunId??localStorage.getItem(storageKey+":run");
    if(id){try{await readRun(id,controller.signal)}catch(e){if(!controller.signal.aborted)fail(e)}}
    if(controller.signal.aborted)return;
    const defaultRef=agent.agent.defaultTemplate;
    const preset=!defaultRef&&!id&&!saved?initialPreset:undefined;
    if(agent.agent.activation!=="NOT_ENABLED"){
     // Each image template may contain 64 KiB; a single row fits the 128 KiB response cap.
     try{
      let page=await configurationRequest(stableScope,"product.image.agent/templates?pageSize=1",imageTemplatesPageSchema,{signal:controller.signal});if(controller.signal.aborted)return;
      const rows=[...page.items],seen=new Set<string>();let pages=1;
      setTemplates(rows);setCursor(page.nextCursor);
      // Discovery reads existing enterprise facts only. Never create a template
      // during navigation, replace a default, or touch a restored run/request.
      while(preset&&!rows.some(t=>matchesGenericImagePreset(t,preset))&&page.nextCursor&&pages<20&&!templateEdited.current){
       if(seen.has(page.nextCursor))break;seen.add(page.nextCursor);
       page=await configurationRequest(stableScope,`product.image.agent/templates?pageSize=1&cursor=${encodeURIComponent(page.nextCursor)}`,imageTemplatesPageSchema,{signal:controller.signal});if(controller.signal.aborted)return;
       rows.push(...page.items.filter(t=>!rows.some(v=>v.templateId===t.templateId&&v.version===t.version)));pages++;
       setTemplates([...rows]);setCursor(page.nextCursor);
      }
      if(preset&&!templateEdited.current){const match=rows.find(t=>matchesGenericImagePreset(t,preset));if(match){installTemplate(match,source?.evidence);setTemplateNotice(`已带入通用模板：${match.name} · v${match.version}。仅用于本次新计划，不改企业默认配置。`)}else setTemplateNotice(`未找到已保存的“${preset.name}”。可读取更多模板并手动选择，或请管理员从通用方案新建并保存；此入口不会自动写配置。`)}
     }catch(e){if(!controller.signal.aborted)fail(e)}
     }
    if(controller.signal.aborted)return;
    if(defaultRef){try{
     const pinned=await configurationRequest(stableScope,`product.image.agent/templates/${defaultRef.templateId}/revisions/${defaultRef.revision}`,imageTemplateSchema,{signal:controller.signal});if(controller.signal.aborted)return;
     if(pinned.templateId!==defaultRef.templateId||pinned.version!==defaultRef.revision||pinned.lifecycle!=="ACTIVE")throw new ImageSetError("IMAGE_CONFIGURATION_CHANGED",409);
     if(!templateEdited.current){installTemplate(pinned,source?.evidence);if(hasInitialSkill&&!id&&!saved)setTemplateNotice("已保留企业默认模板的准确版本、任务和数量；可以为本次计划显式改选，默认配置不会改变。")}
    }catch(e){if(!controller.signal.aborted)fail(e)}}
    if(controller.signal.aborted)return;
    const pageRuns=await imageSetRequest(stableScope,"recent",imageSetRecentSchema,{signal:controller.signal});if(controller.signal.aborted)return;setRecent(pageRuns.items);setRecentCursor(pageRuns.nextCursor);
   }catch(e){if(!controller.signal.aborted)fail(e)}finally{if(!controller.signal.aborted)setLoading(false)}
  })();return()=>controller.abort();
 },[stableScope,storageKey,effectiveVersion,applyReceiptId,initialRunId,initialPreset,hasInitialSkill,readRun,fail]);
 const running=run&&["executing","evaluating","repairing"].includes(run.status);
 useEffect(()=>{
  if(!running||locked||!run)return;
  const controller=new AbortController();const timer=setTimeout(()=>{void readRun(run.runId,controller.signal).catch(e=>{if(!controller.signal.aborted)fail(e)})},5000);
  return()=>{clearTimeout(timer);controller.abort()};
 },[run,locked,running,readRun,fail]);
 function acceptPreparation(p:ImageSetRun){remember(p);clearIntent();setInventory(undefined);setSelected([]);setPreview(null);setRegenerate([])}
 async function send(command:Intent,replay=false){
  if(flight.current||!active.current)return;
  const controller=new AbortController();abort.current=controller;flight.current=true;setBusy(true);setError("");setMessage("");
  try{
   const raw=replay&&savedIntentRaw.current?savedIntentRaw.current:JSON.stringify(command);localStorage.setItem(storageKey+":intent",raw);savedIntentRaw.current=raw;setIntent(command);
   if(command.action==="prepare"||command.action==="regenerate"){
    const p=await imageSetRequest(stableScope,command.action,imageSetRunSchema,{...command,signal:controller.signal});if(controller.signal.aborted)return;acceptPreparation(p);
   }else{
    await imageSetRequest(stableScope,command.action,imageSetAcceptedSchema,{...command,signal:controller.signal});if(controller.signal.aborted)return;clearIntent();setPreview(null);await readRun(command.runId!,controller.signal);
   }
  }catch(e){if(!controller.signal.aborted){
   fail(e);
   if(e instanceof ImageSetError&&e.status===409&&(command.action==="prepare"||command.action==="regenerate"?e.code==="IMAGE_TEXT_REVIEW_CHANGED"||e.code==="IMAGE_ROUTE_CHANGED":command.action==="confirm"&&e.code==="IMAGE_ROUTE_CHANGED")){
    try{if(e.code==="IMAGE_ROUTE_CHANGED"){setRouteDraft(command.action==="confirm"?undefined:(command.body as ImageSetPrepare).route??null);retainOriginalRequest(command,"IMAGE_ROUTE_CHANGED");setRouteReady(false);setRouteNonce(v=>v+1)}else retainOriginalRequest(command)}catch(retentionError){fail(retentionError)}
   }else if(!replay&&e instanceof ImageSetError&&e.code!=="OUTCOME_UNKNOWN"){
    clearIntent();if((command.action==="prepare"||command.action==="regenerate")&&e.code==="IMAGE_CONFIGURATION_CHANGED")setModeNonce(v=>v+1);
   }
  }}
  finally{flight.current=false;if(!controller.signal.aborted)setBusy(false)}
 }
 async function verify(){
  if(!intent||flight.current)return;flight.current=true;setBusy(true);setError("");const controller=new AbortController();abort.current=controller;
  try{
   if(intent.action==="prepare"||intent.action==="regenerate"){
    const p=await imageSetRequest(stableScope,"verify-prepare",imageSetRunSchema,{requestKey:intent.requestKey,signal:controller.signal});if(controller.signal.aborted)return;acceptPreparation(p);
   }else{
    if(intent.action==="approve"&&!releasedIntent){const body=intent.body as ImageSetSelection;const receipt=await imageSetRequest(stableScope,"approval",imageSetApprovalSchema,{runId:intent.runId,approvalId:body.actionId,signal:controller.signal});if(controller.signal.aborted)return;if(receipt.selectionDigest!==body.selectionDigest)throw new ImageSetError("IMAGE_SELECTION_CHANGED",409);clearIntent();setMessage("原批准已核实，正在读取任务状态。");await readRun(intent.runId!,controller.signal);return}
    const p=await readRun(intent.runId!,controller.signal,current=>{
     if(!imageIntentTargetsReleasedRun(intent,current)||!isRetainableImageIntent(intent))return false;
     retainOriginalRequest(intent);setMessage("核实结果：未生成，点数已释放。原请求与编号已保留，可切换任务或准备新计划。");return true;
    });if(!p||controller.signal.aborted)return;
    if(imageIntentTargetsReleasedRun(intent,p)){
     if(!isRetainableImageIntent(intent))setMessage("核实结果：未生成，点数已释放。原请求和编号保留，仅支持只读核实。");
     return;
    }
    const body=intent.body as {actionId:string;planRevision?:number;slotId?:string;attempt?:number;planDigest?:string;quoteDigest?:string};
    if(intent.action==="resume"&&p.status==="completed"){
     await imageSetRequest(stableScope,"approval",imageSetApprovalSchema,{runId:intent.runId,approvalId:body.actionId,signal:controller.signal});if(controller.signal.aborted)return;clearIntent();setMessage("原保存动作已由不可变批准回执核实。");return;
    }
    const slot=p.slots.find(value=>value.slotId===body.slotId&&value.attempt===body.attempt);
    const recovered=intent.action==="recover"&&p.planRevision===body.planRevision&&slot&&["accepted","blocked","rejected"].includes(slot.status)&&slot.closure&&["settled","no_generation"].includes(slot.closure.Kind)&&!p.recoverableEffects?.some(effect=>effect.SlotID===body.slotId&&effect.Attempt===body.attempt);
    const confirmationObserved=["awaiting_final_approval","completed","cancelled"].includes(p.status)||["blocked","failed"].includes(p.status)&&(p.regenerationAvailable||!!p.recoverableEffects?.length);
    const confirmed=intent.action==="confirm"&&p.generationAdmitted&&p.confirmationActionId===body.actionId&&p.planRevision===body.planRevision&&p.planDigest===body.planDigest&&p.quoteDigest===body.quoteDigest&&confirmationObserved;
    const restarted=intent.action==="restart"&&p.generationAdmitted&&p.planRevision===body.planRevision&&p.planDigest===body.planDigest&&p.quoteDigest===body.quoteDigest&&!["failed","planning","awaiting_plan_approval"].includes(p.status);
    const cancellationClosed=intent.action==="cancel"&&p.planRevision===body.planRevision&&["cancelled","completed","failed"].includes(p.status);
    const pendingObserved=(intent.action==="resume"||intent.action==="cancel")&&p.pendingCommand?.ActionID===body.actionId;
    if(recovered||confirmed||restarted||cancellationClosed||pendingObserved||intent.action==="resume"&&p.status==="cancelled")clearIntent();
    else setMessage(intent.action==="confirm"?"读取任务状态不能确认工作流已启动，请继续原确认；请求编号、计划和点数均沿用原值。":"原操作尚未得到明确回执，请继续核实同一编号。");
   }
  }catch(e){if(!controller.signal.aborted)fail(e)}finally{flight.current=false;if(!controller.signal.aborted)setBusy(false)}
 }
 function newPreparation(parent?:ImageSetRun){
  const content=parent?runTemplate:template,preparationSources=parent?runSources:sources;
  if(locked||unresolvedRun||!routeReady||!methodReady||!content||!preparationSources||!(parent?regenerate:tasks).length)return;
  const selectedTasks=parent?regenerate:tasks;
  const groups=parent?{carousel:content.image.carousel.filter(t=>selectedTasks.includes(t.id)).length,detail:content.image.detail.filter(t=>selectedTasks.includes(t.id)).length}:null;
  const selectedTarget=parent?parent.plan.Target.Platform==="shein"?parent.plan.Target:{Platform:"product" as const}:platform==="shein"&&editorTarget?editorTarget:{Platform:"product" as const};
  if(!compatibleImageTemplate(content,selectedTarget.Platform))return;
  if(selectedTarget.Platform==="shein"&&!currentOfficialPositions(selectedTasks.map(id=>positions[id]),requirements))return;
  const selectedSource=parent?.plan.Source??preparationSources.source;
  const originals=(group?:"carousel"|"detail")=>[...new Set(parent!.slots.filter(slot=>selectedTasks.includes(slot.slotId)&&(!group||slot.recipe.Placement.Group===group)).flatMap(slot=>slot.recipe.References.map(ref=>ref.AssetID)))];
  if(!executionMode.data){setError("尚未读取当前权限模式，请刷新后确认。");return}
  const inherited=parent?.plan.Text;
  const text:ImageSetPrepare["text"]|null=parent&&needsRegenerationReview?regenerationTextSelection:parent&&inherited?inherited.Mode==="reviewed"&&inherited.Copy&&inherited.TaskIDs?.some(id=>selectedTasks.includes(id))?{mode:"reviewed",review:inherited.Copy.Review,taskIds:inherited.TaskIDs.filter(id=>selectedTasks.includes(id))}:{mode:"none"}:textSelection;
  if(!text){setError("请先核对目标语言文案，或明确选择不加文字。");return}
  const selectedMethod=parent?parent.plan.Method?{skillId:parent.plan.Method.Skill.id,version:parent.plan.Method.Skill.version,digest:parent.plan.Method.Skill.digest}:undefined:method;
  const chosenRoute=routeDraft===undefined?run?.plan.Route:routeDraft;
  const body:ImageSetPrepare={...(chosenRoute?{route:chosenRoute}:{}),text,expectedModeRevision:executionMode.data.revision,...selectedMethod?{skill:selectedMethod}:{},template:{templateId:content.templateId,revision:content.version},target:selectedTarget,selectedTaskIds:selectedTasks,effectiveCatalogVersion:selectedSource.EffectiveVersion,...selectedSource.ApplyReceiptID?{applyReceiptId:selectedSource.ApplyReceiptID}:{},...content.image.shareOriginals?{sharedOriginalIds:parent?originals():shared}:{carouselOriginalIds:parent?originals("carousel"):groups&&!groups.carousel?[]:carousel,detailOriginalIds:parent?originals("detail"):groups&&!groups.detail?[]:detail},...selectedTarget.Platform==="shein"?{officialPlacements:Object.fromEntries(selectedTasks.flatMap(id=>{const position=positions[id];return position?[[id,position]]:[]}))}:{}};
  void send({action:parent?"regenerate":"prepare",runId:parent?.runId,requestKey:crypto.randomUUID(),body});
 }
 function useCurrentTarget(){
  if(locked||!target)return;
  setModeNonce(v=>v+1);
  ++readVersion.current;restoredRun.current="";rulesBinding.current="";runSourceBinding.current="";setRunSources(undefined);
  setRun(undefined);setPhoto(undefined);localStorage.removeItem(storageKey+":run");setRestoredTarget(undefined);setPlatform(target.Platform);
  setRequirements(undefined);setPositions({});setInventory(undefined);setPriorRuns([]);setPreview(null);setSelected([]);setRegenerate([]);setRegenerationTextDraft(undefined);setError("");setMessage("");
 }
 function choose(value:ChoiceView){if(locked)return;setPreview(null);setSelected(old=>old.some(v=>v.key===value.key)?old.filter(v=>v.key!==value.key):old.length<40?[...old,value]:old)}
 function updateChoice(index:number,edit:(choice:ImageSetChoice)=>void){if(locked)return;setPreview(null);setSelected(old=>old.map((v,i)=>{if(i!==index)return v;const choice=structuredClone(v.choice);edit(choice);return {...v,choice}}))}
 function priorChoice(parent:ImageSetRun,slot:ImageSetRun["slots"][number],candidate:ImageSetRun["slots"][number]["candidates"][number]):ChoiceView{
  return {key:`generated:${parent.runId}:${candidate.assetId}`,url:candidate.url,label:`原任务 ${imageSlotLabel(slot)}`,choice:{kind:"generated",run_id:parent.runId,plan_revision:parent.planRevision,slot_id:slot.slotId,attempt:slot.attempt,result_digest:parent.resultDigest,asset_id:candidate.assetId,presentation:{group:slot.recipe.Placement.Group,order:1},...slot.recipe.OfficialPlacement?{official_placement:toPosition(slot.recipe.OfficialPlacement)}:{}}};
 }
 async function readEarlierParent(){
  const id=priorRuns.at(-1)?.plan.Regeneration?.RunID;if(locked||!id)return;
  const controller=new AbortController();abort.current=controller;flight.current=true;setBusy(true);setError("");
  try{const parent=await imageSetRequest(stableScope,"read",imageSetRunSchema,{runId:id,signal:controller.signal});if(!controller.signal.aborted)setPriorRuns(old=>old.some(p=>p.runId===id)?old:[...old,parent])}catch(e){if(!controller.signal.aborted)fail(e)}finally{flight.current=false;if(!controller.signal.aborted)setBusy(false)}
 }
 async function previewSelection(){
  if(!run||!inventory||locked||!selected.length)return;
  if(run.plan.Target.Platform==="shein"&&!currentOfficialPositions(selected.map(item=>item.choice.official_placement?fromPosition(item.choice.official_placement):undefined),requirements,false))return;
  const orders={carousel:0,detail:0};const choices=selected.map(v=>({...v.choice,presentation:{...v.choice.presentation,order:++orders[v.choice.presentation.group]}}));
  const command:ImageSetSelection={actionId:crypto.randomUUID(),planRevision:run.planRevision,resultDigest:run.resultDigest,expectedHead:inventory.target.head,choices};
  const controller=new AbortController();abort.current=controller;flight.current=true;setBusy(true);setError("");
  try{const result=await imageSetRequest(stableScope,"preview",imageSetPreviewSchema,{runId:run.runId,body:command,signal:controller.signal});if(!controller.signal.aborted)setPreview({command,digest:result.digest})}catch(e){if(!controller.signal.aborted)fail(e)}finally{flight.current=false;if(!controller.signal.aborted)setBusy(false)}
 }
 async function loadRules(){
  if(locked||!editorTarget||!sources)return;setBusy(true);setError("");setRequirements(undefined);setPreview(null);const controller=new AbortController();abort.current=controller;
  const source=run?.plan.Target.Platform==="shein"?run.plan.Source:sources.source;
  try{
   const value=await imageSetRequest(stableScope,"requirements",imageSetRequirementsSchema,{body:{target:editorTarget,effectiveCatalogVersion:source.EffectiveVersion,...source.ApplyReceiptID?{applyReceiptId:source.ApplyReceiptID}:{}},signal:controller.signal});if(!controller.signal.aborted){setRequirements(value);setPositions({});setPreview(null)}
  }catch(e){if(!controller.signal.aborted)fail(e)}finally{if(!controller.signal.aborted)setBusy(false)}
 }
 async function loadMoreRuns(){
  if(locked||flight.current||!active.current||!recentCursor)return;
  const controller=new AbortController();abort.current=controller;flight.current=true;setBusy(true);setError("");
  try{
   const page=await imageSetRequest(stableScope,"recent",imageSetRecentSchema,{query:new URLSearchParams({cursor:recentCursor}).toString(),signal:controller.signal});
   if(!controller.signal.aborted&&active.current){
    setRecent(old=>{const merged=new Map(old.map(item=>[item.runId,item]));for(const item of page.items){if(!merged.has(item.runId))merged.set(item.runId,item)}return [...merged.values()]});
    setRecentCursor(page.nextCursor);
   }
  }catch(e){if(!controller.signal.aborted&&active.current)fail(e)}finally{flight.current=false;if(!controller.signal.aborted&&active.current)setBusy(false)}
 }
 const available=entry?.agent.activation==="ENABLED"&&entry.canUse&&entry.capabilities.some(c=>c.id==="image.generate"&&c.readiness==="AVAILABLE");
 const allTasks=template?[...template.image.carousel.map(t=>({...t,group:"carousel" as const})),...template.image.detail.map(t=>({...t,group:"detail" as const}))]:[];
 const preparePositionsReady=currentOfficialPositions(tasks.map(id=>positions[id]),requirements),regeneratePositionsReady=currentOfficialPositions(regenerate.map(id=>positions[id]),requirements);
 const positionHint="请选择当前规则允许的位置和类型，并为同一图片组设置不重复的排序。";
 const canPrepare=available&&!unresolvedRun&&routeReady&&methodReady&&!!executionMode.data&&!!template&&compatibleImageTemplate(template,platform)&&tasks.length>0&&(template.image.shareOriginals?shared.length>0:(!template.image.carousel.some(t=>tasks.includes(t.id))||carousel.length>0)&&(!template.image.detail.some(t=>tasks.includes(t.id))||detail.length>0))&&(platform==="product"||preparePositionsReady);
 const canRegenerate=available&&!unresolvedRun&&routeReady&&methodReady&&!!executionMode.data&&!!runTemplate&&compatibleImageTemplate(runTemplate,run?.plan.Target.Platform)&&!!runSources&&regenerate.length>0&&regenerate.every(id=>[...runTemplate.image.carousel,...runTemplate.image.detail].some(task=>task.id===id))&&(!needsRegenerationReview||!!regenerationTextSelection)&&(run?.plan.Target.Platform!=="shein"||regeneratePositionsReady);
 function originalChoices(group:string,values:string[],change:(next:string[])=>void){return <section className="space-y-3"><h4 className="text-sm font-medium">{group} · {values.length}/8</h4><div className="grid grid-cols-3 gap-2 sm:grid-cols-4">{sources?.originals.map((image,i)=><label key={image.id} className={`rounded-lg border p-2 ${values.includes(image.id)?"border-success-foreground bg-success":"border-border"}`}><Image src={image.displayUrl} width={100} height={100} unoptimized alt={`原始素材 ${i+1}`} className="h-20 w-full object-contain"/><span className="mt-2 flex items-center gap-1 text-xs"><input type="checkbox" aria-label={`${group} 素材 ${i+1}`} disabled={locked||!values.includes(image.id)&&values.length>=8} checked={values.includes(image.id)} onChange={e=>change(e.target.checked?[...values,image.id]:values.filter(id=>id!==image.id))}/>素材 {i+1}</span></label>)}</div></section>}
 const official=run?.plan.Target.Platform==="shein";
 const selectedPositionsReady=!official||currentOfficialPositions(selected.map(item=>item.choice.official_placement?fromPosition(item.choice.official_placement):undefined),requirements,false);
 return <Card className="space-y-5 p-6">
  <div className="flex flex-wrap items-start justify-between gap-3"><div><h2 className="text-lg font-semibold">商品图片智能体</h2><p className="mt-1 text-sm text-muted-foreground">整套主图与详情图 · 真实素材 · 逐图人工审核</p></div><Button asChild variant="outline" size="sm"><Link href="/workbench/agents/mine/product.image.agent">配置与模板</Link></Button></div>
  {error?<p role="alert" className="rounded-lg bg-warning p-3 text-sm">{reasons[error]??error}</p>:null}{message?<p role="status" className="text-sm text-success-foreground">{message}</p>:null}
  {loading?<p className="text-sm text-muted-foreground">正在读取企业配置与真实商品素材…</p>:!available?<p className="text-sm text-muted-foreground">{entry?.agent.activation==="DISABLED"?"智能体已停用，已有任务仍可核实与审核。":entry?.capabilities.find(c=>c.id==="image.generate")?.reason||"当前完整图片流程不可用。"}</p>:null}
  {intent?<div className={releasedIntent?"rounded-lg border border-border bg-muted p-4 text-sm":"rounded-lg border border-warning-foreground bg-warning p-4 text-sm"}><p>保留原请求：{intent.requestKey??(intent.action==="restart"?intent.runId:(intent.body as {actionId:string}).actionId)}</p><p className="mt-1 text-xs">{releasedIntent?"该任务未生成，点数已释放。原请求和编号保留，仅支持只读核实。":"继续操作沿用原编号与参数；暂未找到不能视为未执行。"}</p><Button className="mt-3" variant="outline" disabled={busy} onClick={()=>void verify()}>核实原请求</Button>{!releasedIntent&&intent.action==="confirm"&&run&&run.runId===intent.runId&&(!run.confirmationActionId||run.confirmationActionId===(intent.body as {actionId:string}).actionId)&&run.status!=="completed"&&run.status!=="cancelled"?<Button className="ml-2" variant="outline" disabled={busy} onClick={()=>void send(intent,true)}>继续原确认</Button>:null}{!releasedIntent&&intent.action!=="confirm"?<Button className="ml-2" variant="outline" disabled={busy} onClick={()=>void send(intent,true)}>继续原操作</Button>:null}</div>:null}
  {!loading&&sources&&retainedRequests.length?<section className="space-y-2 rounded-lg border border-border p-4 text-sm">
   <h3 className="font-semibold">已核实的原请求</h3>
   <p>{intent?"原请求与编号已保留。请先核实当前待处理请求。":"原请求与编号已保留，可切换任务或准备新计划。"}</p>
   <p className="text-muted-foreground">此处仅读取保留记录，原请求不会再次发送。</p>
   {retainedRequests.map(record=><details key={record.key} className="break-all text-muted-foreground"><summary className="cursor-pointer focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-ring">原请求编号：{imageIntentRequestId(record.intent)}</summary><div className="mt-2 space-y-2"><p>原任务：{record.intent.runId??"尚未创建"} · 操作：{({prepare:"准备图片计划",regenerate:"重新准备图片计划",recover:"核实原调用",cancel:"取消剩余工作",restart:"恢复原失败任务",confirm:"确认点数并生成",resume:"核实原保存"} as Partial<Record<ImageSetRoute,string>>)[record.intent.action]}</p><pre className="max-h-80 overflow-auto whitespace-pre-wrap break-all text-xs">{record.raw}</pre></div></details>)}
  </section>:null}
  {!loading&&sources&&context.permissions.includes("listingkit.image_agent.write")?<SkillPicker scope={stableScope} target="product-image" initialSelection={initialSkill} disabled={locked} onChange={onMethodChange}/>:null}
  {!loading&&sources&&entry?.canUse?<ImagePlanRoutePicker key={routeNonce} scope={stableScope} value={intent&&(intent.action==="prepare"||intent.action==="regenerate")?(intent.body as ImageSetPrepare).route??null:routeDraft===undefined?run?.plan.Route??null:routeDraft} disabled={locked||unresolvedRun} onChange={setRouteDraft} onValidity={setRouteReady}/>:null}
  {!loading&&sources&&!sources.originals.length?<p className="text-sm text-muted-foreground">当前商品没有原始图片，补充真实素材后才能准备生成计划。</p>:null}
  {available&&sources?<>
   {template&&!compatibleImageTemplate(template,platform)?<p className="text-sm text-warning-foreground">当前模板不适用于此素材目标，请选择匹配的模板或切换目标。</p>:null}
   {platform==="shein"&&requirements&&tasks.length>0&&!preparePositionsReady?<p className="text-sm text-warning-foreground">{positionHint}</p>:null}
   {templateNotice?<p role="status" className="rounded-lg bg-accent/40 p-4 text-sm">{templateNotice}</p>:null}
   <div className="grid gap-4 sm:grid-cols-2"><label className="space-y-2 text-sm">图片模板<Select disabled={locked} value={template?`${template.templateId}:${template.version}`:""} onChange={e=>{const t=templates.find(v=>`${v.templateId}:${v.version}`===e.target.value);if(t&&compatibleImageTemplate(t,platform)){templateEdited.current=true;setTemplateNotice("");installTemplate(t,sources.evidence)}}}><option value="">选择模板</option>{template&&!templates.some(t=>t.templateId===template.templateId&&t.version===template.version)?<option disabled={!compatibleImageTemplate(template,platform)} value={`${template.templateId}:${template.version}`}>{template.name} v{template.version}</option>:null}{templates.filter(t=>t.lifecycle==="ACTIVE").map(t=><option key={`${t.templateId}:${t.version}`} disabled={!compatibleImageTemplate(t,platform)} value={`${t.templateId}:${t.version}`}>{t.name} v{t.version}</option>)}</Select></label><label className="space-y-2 text-sm">素材目标<Select disabled={locked} value={platform} onChange={e=>{setPlatform(e.target.value as typeof platform);setRequirements(undefined);setPositions({});setPreview(null)}}><option value="product">通用商品素材</option>{editorTarget?<option value="shein">当前 SHEIN 店铺与类目</option>:null}</Select></label></div>
   {cursor?<Button size="sm" variant="outline" disabled={locked} onClick={()=>{void configurationRequest(stableScope,`product.image.agent/templates?pageSize=1&cursor=${encodeURIComponent(cursor)}`,imageTemplatesPageSchema).then(page=>{if(active.current){setTemplates(old=>[...old,...page.items.filter(t=>!old.some(v=>v.templateId===t.templateId))]);setCursor(page.nextCursor)}}).catch(fail)}}>读取更多模板</Button>:null}
   {platform==="shein"?<section className="rounded-lg bg-muted p-4 text-sm"><p>店铺 {editorTarget?.StoreID} · 站点 {editorTarget?.Site} · 类目 {editorTarget?.CategoryID}</p><Button variant="outline" size="sm" className="mt-2" disabled={locked} onClick={()=>void loadRules()}>读取当前图片规则</Button>{restoredTarget&&target&&JSON.stringify(restoredTarget)!==JSON.stringify(target)?<Button variant="outline" size="sm" className="ml-2 mt-2" disabled={locked} onClick={useCurrentTarget}>使用当前页面目标准备新计划</Button>:null}{requirements?<><p className="mt-2">规则 {requirements.version} · 当前生成尺寸 {requirements.nativeWidth} × {requirements.nativeHeight}</p>{requirements.groups.map(g=><p key={`${g.group}:${g.skc}:${g.sku}`} className="mt-1 text-xs">{g.group} / SKC {g.skc+1}{g.group==="sku"?` / SKU ${g.sku+1}`:""}：{g.types.map(t=>`类型 ${t.type}，${t.minimum}–${t.maximum} 张${t.nativeCompatible?"":"（当前生成尺寸不适用）"}`).join("；")}</p>)}</>:null}</section>:null}
   {template&&sources&&<div hidden={correctingRegeneration}><ImageTextCopy key={textKey} scope={scope} source={sources.source} language={template.image.language} templateKey={`${template.templateId}:${template.version}`} taskIds={tasks} taskLabels={Object.fromEntries([...template.image.carousel,...template.image.detail].map(t=>[t.id,labels.get(t.purpose)??t.brief??t.id]))} ownItemId={translationItemId} initialMode={needsPreparationReview?"reviewed":"none"} disabled={locked||correctingRegeneration} onChange={value=>setTextDraft({key:textKey,tasksKey:JSON.stringify(tasks),value})}/></div>}
   {template?<><div className="grid gap-4 md:grid-cols-2">{(["carousel","detail"] as const).map(group=><section key={group} className="space-y-3 rounded-xl border border-border p-4"><h3 className="font-semibold">{group==="carousel"?"主图 / 轮播图":"详情图"}</h3>{allTasks.filter(t=>t.group===group).map(task=>{
    const definition=[...carouselTasks,...detailTasks].find(v=>v.purpose===task.purpose),missing=definition&&"evidence" in definition&&!sources.evidence[definition.evidence];
    return <div key={task.id} className="space-y-2"><label className="flex items-start gap-2 text-sm"><input type="checkbox" disabled={locked||!!missing} checked={tasks.includes(task.id)} onChange={e=>{if(needsPreparationReview)setTextDraft(undefined);setTasks(old=>e.target.checked?[...old,task.id]:old.filter(id=>id!==task.id))}}/><span>{labels.get(task.purpose)??task.brief??"自定义任务"}{missing?<small className="block text-warning-foreground">缺少真实依据，请补充商品资料或取消该项</small>:null}</span></label>{platform==="shein"&&tasks.includes(task.id)?<OfficialPosition value={positions[task.id]} requirements={requirements} disabled={locked} generated onChange={value=>setPositions(old=>({...old,[task.id]:value}))}/>:null}</div>
   })}</section>)}</div><p className="text-xs text-muted-foreground">背景：{template.image.background} · 文字：{template.image.language} · {template.image.shareOriginals?"两组共用原始素材，分别生成与计费":"两组分别选择原始素材，分别生成与计费"}</p>{template.image.shareOriginals?originalChoices("共用原始素材",shared,setShared):<div className="grid gap-4 md:grid-cols-2">{originalChoices("主图素材",carousel,setCarousel)}{originalChoices("详情素材",detail,setDetail)}</div>}<Button disabled={locked||correctingRegeneration||!canPrepare||!textSelection} onClick={()=>newPreparation()}>准备整套图片计划（{tasks.length} 项）</Button></>:<p className="text-sm">请在智能体配置中创建图片模板并设为默认，或选择已有模板。</p>}
  </>:null}
  {recent.length?<label className="block space-y-2 text-sm">本商品最近任务<Select disabled={locked} value={run?.runId??""} onChange={e=>{if(e.target.value){setPreview(null);setSelected([]);void readRun(e.target.value).catch(fail)}}}><option value="">选择原任务</option>{recent.map(item=><option key={item.runId} value={item.runId}>{imageResultLabel(item.runId===run?.runId?run:item)} · {new Date(item.createdAt).toLocaleString()}</option>)}</Select></label>:null}
  {recentCursor?<Button variant="outline" disabled={locked} onClick={()=>void loadMoreRuns()}>加载更多图片任务</Button>:null}
  {run?<section className="space-y-4 border-t border-border pt-5">{run.releaseSummary?<ReleasedImageResult run={run} scope={stableScope} busy={busy} onRefresh={()=>void readRun(run.runId).catch(fail)}/>:<><div className="flex flex-wrap items-center justify-between gap-3"><div><h3 className="font-semibold">{stateNames[run.status]??run.status}</h3><p className="text-xs text-muted-foreground">任务 {run.runId} · {run.slots.filter(s=>s.status==="accepted").length}/{run.images} 张已生成 · 已确认用量 {run.settledPoints} 点</p></div><Button variant="outline" disabled={busy} onClick={()=>void readRun(run.runId).catch(fail)}>刷新原任务</Button></div>
   {run.executionMode?<p className="text-sm">本次权限：{executionModeLabels[run.executionMode]} · 模式版本 {run.modeRevision}</p>:<p className="text-sm text-warning-foreground">本次权限模式不可用，只能查看原记录。</p>}
   {run.plan.Method?<ImageMethodSummary scope={stableScope} method={run.plan.Method} refresh={run}/>:<p className="text-sm text-muted-foreground">计划使用的方法：不使用技能</p>}
   {run.plan.Text&&<section aria-label="已固定的图片文案" className="space-y-2 rounded-lg bg-success p-4 text-sm"><h4 className="font-semibold">目标语言：{imageTextLanguageNames[run.plan.Text.Language]??run.plan.Text.Language}（{run.plan.Text.Language}）</h4>{run.plan.Text.Mode==="reviewed"&&run.plan.Text.Copy?<><ul>{run.plan.Text.Copy.Lines.map((line,index)=><li key={index} className="break-words">{line}</li>)}</ul><p className="text-muted-foreground">审核 {run.plan.Text.Copy.Review.proposalId} · 修订 {run.plan.Text.Copy.Review.revision} · 适用任务：{run.plan.Text.TaskIDs?.join("、")}</p></>:<p>本次已选择不加文字。</p>}<p className="text-muted-foreground">语言和文案随原计划固定，恢复任务不会重新翻译。生成结果仍须人工检查，不合格图片不采用。</p></section>}
   {run.plan.Method?.PolicyVersion==="image-method-visual-brief-v3"?run.slots.filter(slot=>slot.recipe.Purpose==="key_benefits").map(slot=><section key={slot.slotId} aria-label="本次详情图文案" className="space-y-2 rounded-lg bg-success p-4 text-sm"><h4 className="font-semibold">本次详情图文案 · 最多两条</h4>{slot.recipe.VisibleText?.length?<ul className="space-y-1 text-muted-foreground">{slot.recipe.VisibleText.map((text,index)=><li key={index} className="break-words">{text}</li>)}</ul>:<p className="text-muted-foreground">没有合适的已确认短文案，本图不加字。</p>}<p className="text-muted-foreground">文案取自已确认的商品事实，随计划固定。商品为主、清晰留白，不增加表格、信息面板或其他文字；生成结果仍须人工检查，表格、乱码等不合格图片不采用。</p></section>):null}
   {run.modeUnavailableReason==="EXECUTION_MODE_CHANGED"?<p className="text-sm text-warning-foreground">权限配置已变化，本次运行不能获得新权限；请保留原请求，重新确认新运行。</p>:null}
   {run.executionMode==="read_only"?<p className="text-sm">只读模式不能生成图片或保存正式素材。</p>:null}
   {run.policyAdoption?<p className="text-sm">正式素材已按完全访问策略采用，未经逐项人工审阅。{run.status!=="completed"?"原工作流状态仍在核实，请沿用原操作编号。":""}</p>:null}
   {run.status==="awaiting_plan_approval"?<div className="space-y-3 rounded-xl bg-success p-4"><p>图片模型：{run.slots[0]?.recipe.Quote.Model??"原计划冻结模型"} · {run.slots[0]?.recipe.Quote.Quality??"原计划质量"} · {run.slots[0]?.recipe.Quote.Resolution??"1024x1024"}{run.plan.Route?` · 可选版本 ${run.plan.Route.RevisionID}`:" · 企业默认版本"}。修改模型需重新准备计划。</p><p>本次生成 {run.images} 张，点数上限 <strong className="text-warning-foreground">{run.points}</strong> 点。主图与详情图分别调用模型；原图及正式素材复用不调用模型。</p>{run.plan.Method?.PolicyVersion==="image-method-prompt-v1"&&<p role="status">该计划保留历史图片方法，不能新增生成。请重新选择可执行版本并准备新计划；已有结果和恢复记录仍保留。</p>}{routeRejectedPlan?<p role="status">该计划模型版本已被明确拒绝，请重新准备计划；原计划和确认请求已保留。</p>:null}<Button disabled={locked||routeRejectedPlan||!available||!run.generationAllowed||run.plan.Method?.PolicyVersion==="image-method-prompt-v1"} onClick={()=>void send({action:"confirm",runId:run.runId,body:{actionId:crypto.randomUUID(),planRevision:run.planRevision,planDigest:run.planDigest,quoteDigest:run.quoteDigest}})}>确认点数并生成</Button></div>:null}
   {run.block?<p className="text-sm text-warning-foreground">任务暂时停止：{run.block.Code}</p>:null}
   <div className="grid gap-4 md:grid-cols-2">{run.slots.map(slot=><article key={slot.slotId} className="space-y-3 rounded-xl border border-border p-4"><div className="flex justify-between gap-3"><h4 className="font-medium">{imageSlotLabel(slot)}</h4><span className="text-xs text-muted-foreground">{stateNames[slot.status]??slot.status}</span></div><div className="grid grid-cols-2 gap-3"><div><p className="mb-2 text-xs text-muted-foreground">原始素材</p>{slot.recipe.References.map((ref,index)=>{const image=run.originals.find(a=>a.ID===ref.AssetID);return image?<ImageSetPhoto key={ref.AssetID} photo={{url:image.DisplayURL,label:`原图 ${index+1}`,width:image.Width,height:image.Height}} alt="本图使用的原始素材" className="h-32 w-full object-contain" onPreview={setPhoto}/>:null})}</div><div><p className="mb-2 text-xs text-muted-foreground">生成结果</p>{slot.candidates.map(candidate=>{const choice:ImageSetChoice={kind:"generated",run_id:run.runId,plan_revision:run.planRevision,slot_id:slot.slotId,attempt:slot.attempt,result_digest:run.resultDigest,asset_id:candidate.assetId,presentation:{group:slot.recipe.Placement.Group,order:1},...slot.recipe.OfficialPlacement?{official_placement:toPosition(slot.recipe.OfficialPlacement)}:{}};const view={key:`generated:${run.runId}:${candidate.assetId}`,url:candidate.url,label:imageSlotLabel(slot),choice};return <div key={candidate.assetId}><ImageSetPhoto photo={{url:candidate.url,label:`${view.label} · 生成结果`,width:candidate.width,height:candidate.height}} alt="生成结果" className="h-32 w-full object-contain" onPreview={setPhoto}/><Button size="sm" variant="outline" className="mt-2" disabled={locked||!run.approvalAvailable} onClick={()=>choose(view)}>{selected.some(v=>v.key===view.key)?"取消采用":"选择采用"}</Button></div>})}{!slot.candidates.length?<p className="flex h-32 items-center justify-center rounded-lg bg-muted text-xs text-muted-foreground">{slot.errorCode||stateNames[slot.status]||slot.status}</p>:null}</div></div>{run.regenerationAvailable?<label className="flex items-center gap-2 text-xs"><input type="checkbox" disabled={locked} checked={regenerate.includes(slot.slotId)} onChange={e=>{setRegenerationTextDraft(undefined);setRegenerate(old=>e.target.checked?[...old,slot.slotId]:old.filter(id=>id!==slot.slotId))}}/>重新生成此项，另行确认点数</label>:null}{official&&regenerate.includes(slot.slotId)?<div className="space-y-2"><OfficialPosition value={positions[slot.slotId]} requirements={requirements} disabled={locked||!requirements} generated onChange={position=>setPositions(old=>({...old,[slot.slotId]:position}))}/>{!currentOfficialPosition(positions[slot.slotId],requirements)?<p className="text-sm text-warning-foreground">请按当前规则确认此项的图片位置与类型。</p>:null}</div>:null}</article>)}</div>
   {priorRuns.length?<section className="space-y-3 rounded-xl border border-border p-4"><h4 className="font-medium">原任务成功结果</h4><p className="text-xs text-muted-foreground">可与本次重生成结果一起明确选择，保留原结果身份，不再次生成或扣点。</p><div className="grid gap-3 sm:grid-cols-2">{priorRuns.flatMap(parent=>parent.slots.flatMap(slot=>slot.status==="accepted"?slot.candidates.map(candidate=>{const view=priorChoice(parent,slot,candidate);return <article key={view.key} className="space-y-2"><p className="text-sm">{view.label}</p><p className="text-xs text-muted-foreground">任务 {parent.runId}</p><ImageSetPhoto photo={{url:view.url,label:view.label,width:candidate.width,height:candidate.height}} alt={view.label} className="h-32 w-full object-contain" onPreview={setPhoto}/><Button size="sm" variant="outline" disabled={locked||!run.approvalAvailable||!parent.candidateSelectionAvailable} onClick={()=>choose(view)}>{selected.some(v=>v.key===view.key)?"取消采用原任务结果":"采用原任务结果"}</Button></article>}):[]))}</div>{priorRuns.at(-1)?.plan.Regeneration&&!priorRuns.some(p=>p.runId===priorRuns.at(-1)?.plan.Regeneration?.RunID)?<Button size="sm" variant="outline" disabled={locked} onClick={()=>void readEarlierParent()}>读取更早的原任务结果</Button>:null}</section>:null}
   {run.recoverableEffects?.map(effect=><div key={`${effect.SlotID}:${effect.Attempt}`} className="rounded-lg bg-warning p-3 text-sm">{effect.SlotID}：原调用待核实，点数预留保留。<Button className="ml-3" variant="outline" size="sm" disabled={locked||run.status==="failed"} onClick={()=>void send({action:"recover",runId:run.runId,body:{actionId:crypto.randomUUID(),planRevision:run.planRevision,slotId:effect.SlotID,attempt:effect.Attempt}})}>核实原调用</Button></div>)}
   {run.pendingCommand?<Button variant="outline" disabled={locked} onClick={()=>void send({action:"resume",runId:run.runId,body:{actionId:run.pendingCommand!.ActionID}})}>继续核实原保存操作</Button>:null}
   {run.status==="failed"&&run.generationAdmitted&&!run.regenerationAvailable&&!run.pendingCommand?<div className="rounded-lg bg-warning p-3 text-sm"><p>工作流未完成。恢复沿用原任务、点数预算和期限；原调用未明时先核实，超期后不派发新图片。</p><Button className="mt-2" variant="outline" disabled={locked} onClick={()=>void send({action:"restart",runId:run.runId,body:{planRevision:run.planRevision,planDigest:run.planDigest,quoteDigest:run.quoteDigest}})}>恢复原失败任务</Button></div>:null}
   {run.regenerationAvailable?<div><p className="mb-2 text-xs text-muted-foreground">{runTemplate?`重新生成沿用原任务模板 ${runTemplate.name} v${runTemplate.version} 和原始素材，另行确认点数。`:"原任务模板尚不可读取，暂不能重新生成。"}</p>{correctingRegeneration&&runTemplate&&runSources?<section aria-label="重新生成文案纠正" className="mb-4 space-y-3"><p className="text-sm text-warning-foreground">原审核译文已被明确拒绝。请按原任务模板和来源重新读取并确认文案，或明确选择不加文字；原计划、准确技能和原请求保留。取消重新生成勾选后可准备整套新计划。</p><ImageTextCopy key={regenerationTextKey} scope={scope} source={run.plan.Source} language={runTemplate.image.language} templateKey={`${runTemplate.templateId}:${runTemplate.version}`} taskIds={regenerate} taskLabels={Object.fromEntries([...runTemplate.image.carousel,...runTemplate.image.detail].map(t=>[t.id,labels.get(t.purpose)??t.brief??t.id]))} ownItemId={translationItemId} initialMode="reviewed" disabled={locked} onChange={value=>setRegenerationTextDraft({key:regenerationTextKey,value})}/></section>:null}{official&&requirements&&regenerate.length>0&&!regeneratePositionsReady?<p className="mb-2 text-sm text-warning-foreground">{positionHint}</p>:null}<Button variant="outline" disabled={locked||!canRegenerate} onClick={()=>newPreparation(run)}>准备所选 {regenerate.length} 项的新计划</Button></div>:null}
   {run.status!=="completed"&&run.status!=="cancelled"&&run.status!=="failed"&&!run.pendingCommand?<Button variant="ghost" disabled={locked} onClick={()=>void send({action:"cancel",runId:run.runId,body:{actionId:crypto.randomUUID(),planRevision:run.planRevision}})}>取消剩余工作</Button>:null}
   {run.approvalAvailable&&inventory?<section className="space-y-4"><h3 className="font-semibold">选择本次完整正式素材集合</h3><p className="text-sm text-muted-foreground">仅保存下面明确采用的图片，按组内顺序保存。未选择的结果保留在原任务中。</p><div className="flex flex-wrap gap-2">{runSources?.originals.map((image,i)=><Button key={image.id} variant="outline" size="sm" disabled={locked} onClick={()=>choose({key:`source:${image.id}`,url:image.displayUrl,label:`原图 ${i+1}`,choice:{kind:"source",source_id:image.id,presentation:{group:"carousel",order:1}}})}>选择原图 {i+1}</Button>)}</div>{([inventory.target,inventory.generic] as const).map((pool,poolIndex)=>pool?.assets.length?<div key={poolIndex} className="space-y-2"><p className="text-xs text-muted-foreground">{poolIndex?"已批准的通用素材（采用到当前平台）":"当前正式素材"}</p><div className="flex flex-wrap gap-2">{pool.assets.map((image,i)=><Button key={image.id} variant="outline" size="sm" disabled={locked} onClick={()=>choose({key:`approved:${pool.approval_action_id}:${image.id}`,url:image.url,label:`${poolIndex?"通用":"正式"}素材 ${i+1}`,choice:{kind:"approved",approval_action_id:pool.approval_action_id,asset_id:image.id,...poolIndex?{generic_head:pool.head}:{},presentation:image.presentation??{group:"carousel",order:1},...image.official_placement?{official_placement:image.official_placement}:{}}})}>采用素材 {i+1}</Button>)}</div></div>:null)}
    {runSources?.manualReplacementAvailable?<div className="space-y-2"><h4 className="text-sm font-medium">人工文件替换</h4><SourceImageUploader scope={stableScope} disabled={busy||!!intent} onBlocked={blockMedia} onImage={()=>undefined} onMedia={image=>{if(busy||intent)return;setPreview(null);const view:ChoiceView={key:`manual:${image.hash}`,url:image.url,label:"人工上传图片",choice:{kind:"manual",manual_media:{hash:image.hash,bytes:image.bytes},presentation:{group:"carousel",order:1}}};setSelected(old=>old.some(v=>v.key===view.key)?old:old.length<40?[...old,view]:old)}}/><p className="text-xs text-muted-foreground">上传后的图片仍需加入完整选择并人工批准，替换不会调用生成模型。</p></div>:null}<ol className="space-y-3">{selected.map((item,index)=><li key={item.key} className="grid gap-3 rounded-lg border border-border p-3 sm:grid-cols-[80px_1fr]"><ImageSetPhoto photo={{url:item.url,label:`拟采用：${item.label}`}} alt={`拟采用：${item.label}`} className="h-20 w-20 object-contain" onPreview={setPhoto}/><div className="space-y-2"><p className="text-sm">{index+1}. {item.label}</p><div className="flex flex-wrap gap-2"><Select aria-label={`${item.label} 素材组`} disabled={locked} value={item.choice.presentation.group} onChange={e=>updateChoice(index,c=>{c.presentation.group=e.target.value as "carousel"|"detail"})}><option value="carousel">主图 / 轮播</option><option value="detail">详情图</option></Select><Button size="sm" variant="ghost" disabled={locked||index===0} onClick={()=>{setPreview(null);setSelected(old=>{const next=[...old];[next[index-1],next[index]]=[next[index]!,next[index-1]!];return next})}}>上移</Button><Button size="sm" variant="ghost" disabled={locked||index===selected.length-1} onClick={()=>{setPreview(null);setSelected(old=>{const next=[...old];[next[index+1],next[index]]=[next[index]!,next[index+1]!];return next})}}>下移</Button><Button size="sm" variant="ghost" disabled={locked} onClick={()=>choose(item)}>移除</Button></div>{official?<OfficialPosition value={item.choice.official_placement?fromPosition(item.choice.official_placement):undefined} requirements={requirements} generated={false} disabled={locked} onChange={value=>updateChoice(index,c=>{c.official_placement=toPosition(value)})}/>:null}</div></li>)}</ol><Button variant="outline" disabled={locked||!selected.length||!selectedPositionsReady} onClick={()=>void previewSelection()}>预览完整选择</Button>{preview?<div className="rounded-xl bg-success p-4 text-sm"><p>将按以上顺序保存 {preview.command.choices.length} 张图片，更新当前正式素材集合。</p><Button className="mt-3" disabled={locked} onClick={()=>void send({action:"approve",runId:run.runId,body:{...preview.command,selectionDigest:preview.digest}})}>人工批准并保存素材</Button></div>:null}
   </section>:null}
   {run.status==="completed"&&inventory?<div className="grid grid-cols-3 gap-3">{inventory.target.assets.map((image,i)=><figure key={image.id}><ImageSetPhoto photo={{url:image.url,label:`正式素材 ${i+1}`,width:image.width,height:image.height}} alt={`正式素材 ${i+1}`} className="h-32 w-full object-contain" onPreview={setPhoto}/><figcaption className="mt-2 text-xs text-muted-foreground">{image.presentation?.group==="detail"?"详情":"主图 / 轮播"} · {image.presentation?.order??i+1}</figcaption></figure>)}</div>:null}
  </>}</section>:null}
 {photo?<ResourceDialog title="图片预览" onClose={()=>setPhoto(undefined)}><figure className="space-y-3"><Image src={photo.url} width={photo.width||1024} height={photo.height||1024} unoptimized alt={photo.label} className="max-h-[70dvh] w-full object-contain"/><figcaption className="text-sm text-muted-foreground">{photo.label}</figcaption></figure><p className="text-xs text-muted-foreground">查看大图不改变采用选择、原任务和点数。</p></ResourceDialog>:null}
 </Card>;
}

function imageSlotLabel(slot:ImageSetRun["slots"][number]){
 return labels.get(slot.recipe.Purpose)??`${slot.recipe.Placement.Group==="detail"?"详情图":"主图 / 轮播"} ${slot.recipe.Placement.Order}`;
}

function ImageSetPhoto({photo,alt,className,onPreview}:{photo:PhotoView;alt:string;className:string;onPreview:(photo:PhotoView)=>void}){
 return <div className="space-y-2"><Image src={photo.url} width={photo.width||1024} height={photo.height||1024} unoptimized alt={alt} className={className}/><Button type="button" size="sm" variant="outline" aria-label={`查看${photo.label}大图`} onClick={()=>onPreview(photo)}>查看大图</Button></div>;
}

function imageResultLabel(run:Pick<ImageSetRun,"status"|"releaseSummary">){return run.releaseSummary?"未生成，点数已释放":stateNames[run.status]??run.status}

function imageIntentTargetsReleasedRun(intent:Intent,run:ImageSetRun|undefined){return !!run?.releaseSummary&&intent.runId===run.runId&&intent.action!=="regenerate"&&intent.action!=="prepare"}

function parseImageIntent(raw:string):Intent|undefined{
 let decoded:unknown;try{decoded=JSON.parse(raw)}catch{return}
 const uuid=z.string().refine(isAcquisitionUUID);
 const parsed=z.object({action:z.enum(["prepare","regenerate","confirm","approve","cancel","recover","resume","restart"]),runId:uuid.optional(),requestKey:uuid.optional(),body:z.unknown()}).strict().refine(command=>(command.action==="prepare"||!!command.runId)&&(!(command.action==="prepare"||command.action==="regenerate")||!!command.requestKey)).safeParse(decoded);
 if(parsed.success&&imageSetRequestSchema(parsed.data.action).safeParse(parsed.data.body).success)return parsed.data;
}
function isRetainableImageIntent(intent:Intent){return ["prepare","regenerate","recover","cancel","restart","confirm","resume"].includes(intent.action)}
function imageIntentRequestId(intent:Intent){return intent.requestKey??(intent.action==="restart"?intent.runId:(intent.body as {actionId:string}).actionId)}
function retainedImageRequestKey(storageKey:string,intent:Intent,reason?:"IMAGE_ROUTE_CHANGED"){return `${storageKey}:${reason?"verified-route-intent":"verified-intent"}:${intent.runId}:${intent.action}:${imageIntentRequestId(intent)}`}
function readRetainedImageRequests(storageKey:string):RetainedRequest[]{
 return Object.keys(localStorage).filter(key=>(key.startsWith(storageKey+":verified-intent:")||key.startsWith(storageKey+":verified-route-intent:"))).flatMap(key=>{
  const raw=localStorage.getItem(key);if(!raw)return [];
  const intent=parseImageIntent(raw);
  const reason=key.startsWith(storageKey+":verified-route-intent:")?"IMAGE_ROUTE_CHANGED" as const:undefined;
  return intent&&isRetainableImageIntent(intent)&&retainedImageRequestKey(storageKey,intent,reason)===key?[{key,raw,intent,...reason?{reason}:{}}]:[];
 });
}

function ReleasedImageResult({run,scope,busy,onRefresh}:{run:ImageSetRun;scope:ImageSetScope;busy:boolean;onRefresh:()=>void}){
 return <>
  <div className="space-y-3 rounded-lg bg-accent p-4 text-sm">
   <h3 className="text-xl font-semibold text-primary">未生成，点数已释放</h3>
   <p>本次未生成图片，原预留 {run.releaseSummary!.releasedPoints} 点已释放。</p>
   <p className="break-all text-xs text-muted-foreground">任务 {run.runId} · 已确认用量 {run.settledPoints} 点</p>
   <p className="text-muted-foreground">原商品与素材保持不变，本次没有重新发起生成。</p>
   <Button size="sm" disabled={busy} onClick={onRefresh}>刷新原任务</Button>
  </div>
  <details className="rounded-lg border border-border p-4 text-sm">
   <summary className="cursor-pointer font-medium focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-ring">原始失败记录与核实依据</summary>
   <div className="mt-3 space-y-2 text-muted-foreground">
    <p>原始状态：{stateNames[run.status]??run.status}</p>
    {run.block?<p className="break-all">原始阻止代码：{run.block.Code}</p>:null}
    {run.slots.map(slot=><div key={slot.slotId} className="space-y-1 break-all"><p>原始图片项：{slot.slotId} · 计划版本 {run.planRevision} · 尝试 {slot.attempt}</p><p>原始错误代码：{slot.errorCode||"无"}</p></div>)}
    {run.recoverableEffects?.map(effect=><p key={`${effect.SlotID}:${effect.Attempt}`} className="break-all">原始核实记录：{effect.SlotID} · 尝试 {effect.Attempt} · {effect.Code}</p>)}
    <p>核实结果：未生成图片，原预留点数已释放。</p>
    <p>原始失败记录与审计证据均保留，原请求未被重试。</p>
    {run.plan.Method?<ImageMethodSummary scope={scope} method={run.plan.Method} refresh={run}/>:<p>计划使用的方法：不使用技能</p>}
   </div>
  </details>
 </>;
}

function sourceIdentity(source:ImageSetRun["plan"]["Source"]){return JSON.stringify([source.ContextKind,source.ProductID,source.OperationID,source.OriginalPublicationID,source.OriginalVersion,source.EffectiveVersion,source.ApplyReceiptID??""])}
function compatibleImageTemplate(template:ImageAgentTemplate|undefined,platform:string|undefined){return !!template&&(template.targetPlatform==="product"||template.targetPlatform===platform)}
type OfficialPosition=NonNullable<ImageSetPrepare["officialPlacements"]>[string];
function toPosition(p:OfficialPosition):NonNullable<ImageSetChoice["official_placement"]>{return {group:p.Group,skc:p.SKC,sku:p.SKU,type:p.Type,sort:p.Sort,site:p.Site}}
function fromPosition(p:NonNullable<ImageSetChoice["official_placement"]>):OfficialPosition{return {Group:p.group,SKC:p.skc,SKU:p.sku,Type:p.type,Sort:p.sort,Site:p.site}}
function currentOfficialPosition(value:OfficialPosition|undefined,requirements:ImageSetRequirements|undefined,generated=true){
 if(!value||!requirements||value.Site!==requirements.site||!Number.isInteger(value.Sort)||value.Sort<1||value.Sort>100)return false;
 if((value.Type===1||value.Group==="sku")&&value.Sort!==1)return false;
 return requirements.groups.some(group=>group.group===value.Group&&group.skc===value.SKC&&group.sku===value.SKU&&group.types.some(type=>type.type===value.Type&&type.maximum>0&&(!generated||type.nativeCompatible)));
}
function currentOfficialPositions(values:(OfficialPosition|undefined)[],requirements:ImageSetRequirements|undefined,generated=true){
 if(!requirements)return false;
 const seen=new Set<string>(),counts=new Map<string,number>();
 return values.length>0&&values.every(value=>{
  if(!currentOfficialPosition(value,requirements,generated)||!value)return false;
  const key=`${value.Group}:${value.SKC}:${value.SKU}:${value.Sort}`;
  if(seen.has(key))return false;
  const typeKey=`${value.Group}:${value.SKC}:${value.SKU}:${value.Type}`,count=(counts.get(typeKey)??0)+1;
  const rule=requirements.groups.find(group=>group.group===value.Group&&group.skc===value.SKC&&group.sku===value.SKU)?.types.find(type=>type.type===value.Type);
  if(!rule||count>rule.maximum)return false;
  counts.set(typeKey,count);seen.add(key);return true;
 });
}
function OfficialPosition({value,requirements,disabled,generated,onChange}:{value?:OfficialPosition;requirements?:ImageSetRequirements;disabled:boolean;generated:boolean;onChange:(value:OfficialPosition)=>void}){
 const groups=requirements?.groups??[];const group=groups.find(g=>g.group===value?.Group&&g.skc===value?.SKC&&g.sku===value?.SKU);
 return <div className="grid grid-cols-3 gap-2"><Select aria-label="官方图片位置" disabled={disabled||!requirements} value={value?`${value.Group}:${value.SKC}:${value.SKU}`:""} onChange={e=>{const [name,skc,sku]=e.target.value.split(":");const selected=groups.find(g=>g.group===name&&g.skc===Number(skc)&&g.sku===Number(sku));const type=selected?.types.find(t=>t.maximum>0&&(!generated||t.nativeCompatible));if(selected&&type)onChange({Group:selected.group,SKC:selected.skc,SKU:selected.sku,Type:type.type,Sort:1,Site:requirements!.site})}}><option value="">选择平台位置</option>{groups.filter(g=>g.types.some(t=>t.maximum>0&&(!generated||t.nativeCompatible))).map(g=><option key={`${g.group}:${g.skc}:${g.sku}`} value={`${g.group}:${g.skc}:${g.sku}`}>{g.group} / SKC {g.skc+1}{g.group==="sku"?` / SKU ${g.sku+1}`:""}</option>)}</Select><Select aria-label="官方图片类型" disabled={disabled||!group} value={value?.Type??""} onChange={e=>{if(value)onChange({...value,Type:Number(e.target.value)})}}><option value="">类型</option>{group?.types.filter(t=>t.maximum>0&&(!generated||t.nativeCompatible)).map(t=><option key={t.type} value={t.type}>类型 {t.type}（最多 {t.maximum} 张）</option>)}</Select><Input aria-label="官方图片排序" type="number" min={1} max={100} disabled={disabled||!value} value={value?.Sort??""} onChange={e=>{if(value)onChange({...value,Sort:Number(e.target.value)})}}/></div>
}
