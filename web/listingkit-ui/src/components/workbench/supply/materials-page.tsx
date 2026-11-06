"use client";
import {ImageAdvicePanel} from "../acquisition/image-advice-panel";
import Link from "next/link";
import {useEffect,useMemo,useRef,useState} from "react";
import {useSearchParams} from "next/navigation";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Textarea} from "@/components/ui/textarea";
import {Select} from "@/components/ui/select";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {readSupplyRecord,supplyRules,SupplyAPIError,type SupplyScope} from "@/lib/api/supply-chain";
import {categoryLeaves} from "./draft-fields";
import {CandidateRulesPanel} from "./candidate-rules-panel";
import {prepareMaterials,startMaterials,readMaterials,resumeMaterials,adoptMaterials,readMaterialsAdoption} from "@/lib/api/listing-materials";
import type {MaterialsRun,SupplyTarget} from "@/lib/contracts/supply-chain";
import {collectionID} from "@/lib/contracts/product-collection";
import {catalogEntrySchema,materialsTemplatesPageSchema,type AgentTemplate} from "@/lib/contracts/agent-configuration";
import {executionModeViewSchema} from "@/lib/contracts/execution-mode";
import {executionModeLabels} from "../agents/execution-mode-settings";
import {useConfigurationRead,configurationError} from "../agents/agent-configuration-hooks";
import {readMaterialsIntent,storeMaterialsIntent,readMaterialsMutation,storeMaterialsMutation,type MaterialsIntent,type MaterialsMutation} from "./materials-intent";
import styles from "../task-center/title-review.module.css";

const failureText:Record<string,string>={OUTCOME_UNKNOWN:"结果尚未确认，请保留本次编号并核对原请求。",AGENT_CONFLICT:"原运行或草稿已变化，请核对原编号。",HUMAN_CONFIRMATION_REQUIRED:"当前资料需要本人核对后明确保存。",PERMISSION_DENIED:"当前身份或权限模式不允许此操作。",DEPENDENCY_UNAVAILABLE:"模型配置、店铺规则或依赖尚未就绪。",AGENT_NOT_ENABLED:"当前企业尚未启用平台资料智能体。",TEMPLATE_ARCHIVED:"原模板已归档，新执行或保存被拒绝。",CONFIGURATION_CHANGED:"配置已变化，原请求编号继续保留。",NOT_FOUND:"原结果尚未找到；这不表示可以更换编号重试。",INPUT_TOO_LARGE:"完整资料超过输入上限，本次无法继续。"};
const phaseText={running:"执行中，先核对原编号",interrupted:"等待继续",human_review_required:"建议已生成，待采用策略核对",stopped:"执行已停止"};
export function MaterialsPage({recordId}:{recordId:string}){
 const context=useWorkbenchContext();
 if(context.error||context.blockingError)return <ConsoleState kind="error" title="企业上下文不可用"/>;
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!context.user||!context.effectiveOrganization||context.selectionRequired)return <ConsoleState kind="unavailable" title="请先选择企业"/>;
 return <MaterialsSession key={`${context.user.id}:${context.effectiveOrganization.id}:${[...context.permissions].sort().join(":")}:${recordId}`} recordId={recordId} userId={context.user.id} organizationId={context.effectiveOrganization.id} organization={context.effectiveOrganization.name}/>;
}
function MaterialsSession({recordId,userId,organizationId,organization}:{recordId:string;userId:string;organizationId:string;organization:string}){
 const context=useWorkbenchContext(),query=useSearchParams();
 const scope=useMemo<SupplyScope>(()=>({userId,organizationId}),[userId,organizationId]);
 const [base,setBase]=useState<SupplyTarget>(),[loadError,setLoadError]=useState("");
 const [intent,setIntent]=useState<MaterialsIntent|null>(()=>readMaterialsIntent(query.get("materials_intent")));
 const [purpose,setPurpose]=useState<"materials-edit"|"category-rules">(()=>intent?.prepare.candidateCategoryId||query.get("purpose")==="category-rules"?"category-rules":"materials-edit");
 const [candidateCategoryId,setCandidateCategoryId]=useState(intent?.prepare.candidateCategoryId??"");
 const [categoryResult,setCategoryResult]=useState<{base:SupplyTarget;categories:Array<{id:number;type:number;name:string}>;error:string}>();
 const [mutation,setMutation]=useState<MaterialsMutation|null>(()=>readMaterialsMutation(query.get("materials_mutation")));
 const historyKey=collectionID.safeParse(query.get("agent_key")).data??"";
 const historyActor=query.get("agent_actor"),historyOrg=query.get("agent_org");
 const key=intent?.key??historyKey;
 const brokenIntent=!!query.get("materials_intent")&&!intent;
 const brokenMutation=!!query.get("materials_mutation")&&!mutation;
 const foreign=!!intent&&(intent.userId!==userId||intent.organizationId!==organizationId||intent.recordId!==recordId)||!!mutation&&(mutation.userId!==userId||mutation.organizationId!==organizationId||mutation.recordId!==recordId||mutation.key!==key)||!!historyKey&&(historyActor!==userId||historyOrg!==organizationId);
 const [result,setResult]=useState<MaterialsRun>(),[receipt,setReceipt]=useState<MaterialsRun["adoption"]>(),[error,setError]=useState("");
 const [busy,setBusy]=useState(!!key&&!foreign),[confirm,setConfirm]=useState(false),[template,setTemplate]=useState<AgentTemplate>(),[feedback,setFeedback]=useState(""),[human,setHuman]=useState(false);
 const active=useRef(true),flight=useRef(false),abort=useRef<AbortController|null>(null),unknown=useRef(!!mutation);
 const config=useConfigurationRead(scope,"listing.materials.agent",catalogEntrySchema);
 const templates=useConfigurationRead(scope,"listing.materials.agent/templates?lifecycle=ACTIVE&pageSize=100",materialsTemplatesPageSchema,0,!!config.data?.canUse);
 const mode=useConfigurationRead(scope,"execution-modes/registered_agent/listing.materials.agent",executionModeViewSchema);
 const categories=categoryResult?.base===base?categoryResult?.categories??[]:[];
 const categoryError=categoryResult?.base===base?categoryResult?.error??"":"";
 const categoryLoading=purpose==="category-rules"&&!!base&&!key&&!foreign&&categoryResult?.base!==base;
 const selectedCategory=categories.find(v=>String(v.id)===candidateCategoryId);
 const rulesPurpose=result?result.purpose==="category-rules":!!intent?.prepare.candidateCategoryId||purpose==="category-rules";
 const mayGenerate=!!base&&!key&&!foreign&&!brokenIntent&&!brokenMutation&&!!config.data?.canUse&&config.data.agent.activation==="ENABLED"&&config.data.capabilities.filter(c=>c.support==="REQUIRED").every(c=>c.readiness==="AVAILABLE")&&!!mode.data&&(purpose!=="category-rules"||!!selectedCategory);
 useEffect(()=>{active.current=true;const c=new AbortController();if(!foreign)void readSupplyRecord(scope,recordId,c.signal).then(v=>{if(active.current)setBase(v)}).catch(e=>{if(active.current)setLoadError(e instanceof SupplyAPIError?e.code:"DEPENDENCY_UNAVAILABLE")});return()=>{active.current=false;c.abort();abort.current?.abort()}},[recordId,scope,foreign]);
 useEffect(()=>context.registerOrganizationSwitchGuard(()=>!flight.current&&!unknown.current&&!mutation),[context,mutation]);
 useEffect(()=>{
  if(purpose!=="category-rules"||!base||key||foreign)return;
  const c=new AbortController();let live=true;
  const input={sourceId:base.input.sourceId,storeId:base.input.storeId,expectedRevision:base.revision,effectiveVersion:base.effectiveVersion,...(base.applyReceiptId?{applyReceiptId:base.applyReceiptId}:{}),draft:{product:{category_id:0,product_type_id:0,source_system:"OpenAPI",suit_flag:"0",is_spu_pic:false,multi_language_name_list:[],product_attribute_list:[],skc_list:[]},images:[]}};
  void supplyRules(scope,input,c.signal).then(v=>{if(live)setCategoryResult({base,categories:categoryLeaves(v.rules.categories),error:""})}).catch(e=>{if(live)setCategoryResult({base,categories:[],error:e instanceof SupplyAPIError?failureText[e.code]??"类目目录当前不可读。":"类目目录当前不可读。"})});
  return()=>{live=false;c.abort()};
 },[purpose,base,key,foreign,scope]);
 function retain(next:MaterialsIntent){storeMaterialsIntent(next);setIntent(next);return next}
 function retainMutation(next:MaterialsMutation|null){storeMaterialsMutation(next);setMutation(next);return next}
 function observeRun(next:MaterialsRun){
  setResult(next);setReceipt(next.adoption);unknown.current=next.phase==="running"||next.usageStatus==="unknown_reserved";
  if(next.adoption||mutation?.action==="resume"&&!unknown.current&&BigInt(next.revision)>BigInt(mutation.input.revision))retainMutation(null);
 }
 async function execute(action:"start"|"read"|"resume"|"adopt"|"adoption"){
  if(rulesPurpose&&(action==="adopt"||action==="adoption"))return;
  if(flight.current||foreign||!active.current||(brokenIntent||brokenMutation)&&action!=="read"&&action!=="adoption")return;
  flight.current=true;setBusy(true);setError("");setConfirm(false);
  const c=new AbortController();abort.current=c;
  try{
   if(action==="start"){
    let original=intent;
    if(!original){if(!mayGenerate||!mode.data)throw new SupplyAPIError("DEPENDENCY_UNAVAILABLE",503);const requestKey=crypto.randomUUID();original=retain({recordId,userId,organizationId,key:requestKey,prepare:{requestKey,expectedModeRevision:mode.data.revision,goalSummary:purpose==="category-rules"?"查询并解释用户选择类目的基础规则":"使用当前标准商品与店铺规则完善本地平台资料",...(purpose==="category-rules"?{candidateCategoryId}:{}),...(template?{template:{templateId:template.templateId,revision:template.version}}:{})}})}
    if(!original.prepared){const prepared=await prepareMaterials(scope,recordId,original.prepare,c.signal);original=retain({...original,prepared})}
    const next=await startMaterials(scope,recordId,original.key,original.prepared!,c.signal);if(active.current){setResult(next);setReceipt(next.adoption);unknown.current=next.phase==="running"||next.usageStatus==="unknown_reserved"}
   }else if(action==="read"){
    const next=await readMaterials(scope,recordId,key,c.signal);if(active.current)observeRun(next);
   }else if(action==="resume"){
    if(!result||result.phase!=="interrupted")throw new SupplyAPIError("AGENT_CONFLICT",409);
    if(mutation&&mutation.action!=="resume")throw new SupplyAPIError("AGENT_CONFLICT",409);
    const original=mutation?.action==="resume"?mutation:retainMutation({recordId,userId,organizationId,key,action:"resume",input:{configurationSnapshotRef:result.configurationSnapshotRef,revision:result.revision,feedback:feedback.trim()}})!;
    if(original.action!=="resume")throw new SupplyAPIError("AGENT_CONFLICT",409);
    const next=await resumeMaterials(scope,recordId,key,original.input,c.signal);if(active.current){setResult(next);setReceipt(next.adoption);unknown.current=next.phase==="running"||next.usageStatus==="unknown_reserved";if(!unknown.current)retainMutation(null)}
   }else{
    let saved:NonNullable<MaterialsRun["adoption"]>;
    if(action==="adoption")saved=await readMaterialsAdoption(scope,recordId,key,c.signal);
    else{if(mutation&&mutation.action!=="adopt")throw new SupplyAPIError("AGENT_CONFLICT",409);if(!mutation&&(!result?.canRequestAdoption||!result.candidateHash||!human))throw new SupplyAPIError("HUMAN_CONFIRMATION_REQUIRED",409);const original=mutation?.action==="adopt"?mutation:retainMutation({recordId,userId,organizationId,key,action:"adopt",input:{configurationSnapshotRef:result!.configurationSnapshotRef,candidateHash:result!.candidateHash!,humanConfirmed:human}})!;if(original.action!=="adopt")throw new SupplyAPIError("AGENT_CONFLICT",409);saved=await adoptMaterials(scope,recordId,key,original.input,c.signal)}
    if(base&&(saved.record.targetId!==base.targetId||saved.record.input.sourceId!==base.input.sourceId||saved.record.input.storeId!==base.input.storeId||saved.record.effectiveVersion!==base.effectiveVersion))throw new SupplyAPIError("OUTCOME_UNKNOWN",502);
    if(active.current){setReceipt(saved);unknown.current=false;retainMutation(null)}
   }
  }catch(e){if(active.current){const code=e instanceof SupplyAPIError?e.code:"OUTCOME_UNKNOWN";setError(code);if(code==="OUTCOME_UNKNOWN")unknown.current=true}}
  finally{flight.current=false;if(active.current)setBusy(false)}
 }
 useEffect(()=>{
  if(!key||foreign)return;
  const c=new AbortController();flight.current=true;
  void readMaterials(scope,recordId,key,c.signal).then(next=>{if(!c.signal.aborted&&active.current)observeRun(next)}).catch(e=>{if(!c.signal.aborted&&active.current)setError(e instanceof SupplyAPIError?e.code:"DEPENDENCY_UNAVAILABLE")}).finally(()=>{if(!c.signal.aborted&&active.current){flight.current=false;setBusy(false)}});
  return()=>c.abort();/* restore by reading the original immutable key once per bound session */
 // eslint-disable-next-line react-hooks/exhaustive-deps
 },[]);
 if(foreign)return <ConsoleState kind="unavailable" title="原请求属于另一身份或企业">请返回原身份和企业核对，本次编号不会更换。</ConsoleState>;
 if(brokenIntent&&!key)return <ConsoleState kind="unavailable" title="原确认资料不可读，请保留原编号核对。"/>;
 return <ConsolePage title="平台上架资料" description="使用已保存的标准商品和真实店铺规则完善本地草稿。商品内容由内容智能体负责，平台发布另行授权。" breadcrumbs={[{label:"商品刊登",href:"/workbench/supply/mine"},{label:"平台上架资料"}]}>
  <Card className="space-y-4 p-6"><h2 className="text-lg font-semibold">本次资料依据</h2>{base?<><p>当前企业 · {organization} · SHEIN 美国站</p><p>店铺 {base.input.storeId} · 本地草稿 v{base.revision} · 标准商品 v{base.effectiveVersion}</p><p className="break-all text-sm text-muted-foreground">记录 {recordId}</p></>:<p role="status">{loadError?failureText[loadError]??"草稿当前不可读。":"正在读取原草稿…"}</p>}
   {key?<><p className="break-all">本次编号 {key}</p><div className="flex flex-wrap gap-3"><Button variant="outline" disabled={busy} onClick={()=>void execute("read")}>核对原运行</Button>{!rulesPurpose&&<Button variant="outline" disabled={busy} onClick={()=>void execute("adoption")}>核对原保存结果</Button>}{intent&&!result&&!brokenIntent&&!brokenMutation&&!mutation&&<Button variant="outline" disabled={busy} onClick={()=>void execute("start")}>重试原确认与请求</Button>}</div></>:<><p>{mode.data?`${executionModeLabels[mode.data.mode]} · 权限版本 ${mode.data.revision}`:"正在读取当前权限模式…"}</p><label className="block space-y-2">本次用途<Select value={purpose} disabled={busy} onChange={e=>setPurpose(e.target.value as "materials-edit"|"category-rules")}><option value="materials-edit">准备平台资料</option><option value="category-rules">查询候选类目规则</option></Select></label>{purpose==="category-rules"&&<section className="space-y-2"><label className="block space-y-2">候选类目<Select value={candidateCategoryId} disabled={busy||categoryLoading} onChange={e=>setCandidateCategoryId(e.target.value)}><option value="">先选择一个候选类目</option>{categories.map(v=><option key={v.id} value={String(v.id)}>{v.name||String(v.id)}</option>)}</Select></label><p className="text-sm text-muted-foreground">只查询基础规则，不改变已保存资料；依赖具体属性取值的关联条件未评价。</p>{categoryLoading&&<p role="status">正在读取类目目录…</p>}{categoryError&&<p role="alert">{categoryError}</p>}</section>}<label className="block space-y-2">本次模板<Select disabled={busy} value={template?.templateId??""} onChange={e=>setTemplate(templates.data?.items.find(t=>t.templateId===e.target.value))}><option value="">本次不使用模板</option>{templates.data?.items.map(t=><option key={t.templateId} value={t.templateId}>{t.name} · v{t.version}</option>)}</Select></label><Button disabled={!mayGenerate||busy} onClick={()=>setConfirm(true)}>{purpose==="category-rules"?"查询候选类目规则":"准备平台资料"}</Button></>}
   {!!config.error&&<p role="status">{configurationError(config.error)}</p>}{config.data?.capabilities.filter(c=>c.support==="REQUIRED"&&c.readiness!=="AVAILABLE").map(c=><p role="status" key={c.id}>{c.reason}</p>)}
   {brokenMutation&&<p role="alert">原写入请求资料不可读，请先核对原运行或保存结果。</p>}{error&&<p role="alert">{failureText[error]??"操作暂不可用，请用原编号核对。"}</p>}{busy&&<p role="status">正在处理原请求，请保留本次编号…</p>}
  </Card>
  {result&&<Card className="space-y-4 p-6"><h2 className="text-lg font-semibold">{rulesPurpose&&result.phase==="human_review_required"?(!result.stopReason&&result.ruleExplanation?"规则解释已生成，待核对":"规则解释未完成，请核对停止原因"):phaseText[result.phase]}</h2><p>{executionModeLabels[result.executionMode]} · 冻结权限版本 {result.modeRevision} · {result.templateSelection?`模板版本 ${result.templateSelection.revision}`:"未使用模板"}</p>{result.stopReason&&<p>停止原因：{result.stopReason}</p>}
   {rulesPurpose?<CandidateRulesPanel result={result}/>:<>{result.proposal&&result.after&&<MaterialDifferences result={result} saved={!!receipt}/>}<ImageAdvicePanel advice={result.imageAdvice}/></>}
   {!!result.unresolved.length&&<section><h3 className="font-semibold">需要补齐或核对</h3><ul className="list-inside list-disc">{result.unresolved.map((v,i)=><li key={i}>{v}</li>)}</ul></section>}
   {!rulesPurpose&&!receipt&&result.canRequestAdoption&&<><label className="flex items-start gap-2"><input type="checkbox" checked={human} disabled={busy||!!mutation||brokenIntent||brokenMutation} onChange={e=>setHuman(e.target.checked)}/>我已逐项核对建议与缺项，确认保存为本地草稿。</label><Button disabled={busy||brokenIntent||brokenMutation||mutation?.action==="resume"||!human&&!mutation} onClick={()=>void execute("adopt")}>{mutation?.action==="adopt"?"重试原保存请求":"确认保存本地草稿"}</Button></>}
   {result.executionMode==="read_only"&&<p>本次为只读模式，建议可查看，不能保存到平台草稿。</p>}
   {result.phase==="interrupted"&&result.canResume===true&&<><label className="block">补充核对意见<Textarea value={mutation?.action==="resume"?mutation.input.feedback:feedback} maxLength={512} onChange={e=>setFeedback(e.target.value)} disabled={busy||!!mutation||brokenMutation}/></label><Button disabled={busy||brokenIntent||brokenMutation||mutation?.action==="adopt"||!mutation&&!feedback.trim()} onClick={()=>void execute("resume")}>{mutation?.action==="resume"?"重试原继续请求":"按原编号继续"}</Button></>}
   <p className="text-sm text-muted-foreground">已记录 {result.tokens} tokens · 费用 {result.estimatedCostMicros/1000000} {result.currency} · {result.usageStatus==="unknown_reserved"?"用量待核实，原预留仍保留":"用量已观察"}</p>
  </Card>}
  {receipt&&<Card className="space-y-4 p-6"><h2 className="text-lg font-semibold">本地草稿已保存 · v{receipt.record.revision}</h2><p className="break-all">原 Listing Target 记录 {receipt.record.id}</p><p>{receipt.record.result.ready_for_upload?"当前本地资料校验完整；真实发布仍需另外授权。":"已保存待补全草稿，仍有实际缺项。"}</p><ul className="list-inside list-disc">{receipt.record.result.issues.map((v,i)=><li key={i}>{v.message}</li>)}</ul><Button asChild variant="outline"><Link href={`/workbench/supply/materials/${receipt.record.id}`}>使用已保存版本继续完善</Link></Button></Card>}
  {confirm&&base&&mode.data&&<MaterialsConfirmation organization={organization} base={base} mode={executionModeLabels[mode.data.mode]} template={template} candidate={purpose==="category-rules"?{id:candidateCategoryId,name:selectedCategory?.name??candidateCategoryId}:undefined} busy={busy} onCancel={()=>setConfirm(false)} onConfirm={()=>void execute("start")}/>}
 </ConsolePage>;
}
function MaterialDifferences({result,saved}:{result:MaterialsRun;saved:boolean}){
 if(!result.before||!result.after)return null;
 const before=result.before.draft,after=result.after.draft;
 const rows:[string,unknown,unknown][]=[];
 if(result.proposal?.categoryId!==undefined)rows.push(["类目",before.product.category_id,after.product.category_id]);
 if(result.proposal?.brandCode!==undefined)rows.push(["品牌代码",before.product.brand_code??"未填写",after.product.brand_code]);
 if(result.proposal?.attributes!==undefined)rows.push(["商品属性",before.product.product_attribute_list,after.product.product_attribute_list]);
 if(result.proposal?.images!==undefined)rows.push(["已批准图片位置",before.images,after.images]);
 for(const item of result.proposal?.skcs??[]){
  const old=before.product.skc_list[item.index],next=after.product.skc_list[item.index];
  if(item.saleAttribute!==undefined)rows.push([`SKC ${item.index+1} 销售属性`,old?.sale_attribute,next?.sale_attribute]);
  for(const sku of item.skus??[]){const a=old?.sku_list[sku.index],b=next?.sku_list[sku.index],label=a?.supplier_sku??`SKU ${sku.index+1}`;
   if(sku.saleAttributes!==undefined)rows.push([`${label} 销售属性`,a?.sale_attribute_list,b?.sale_attribute_list]);
   if(sku.attributes!==undefined)rows.push([`${label} SKU 属性`,a?.sku_scope_attribute_list,b?.sku_scope_attribute_list]);
  }
 }
 return <section className="overflow-x-auto"><h3 className="mb-3 font-semibold">{saved?"已采用的建议差异":"建议差异 · 尚未保存"}</h3><table className="w-full text-left text-sm"><thead><tr><th>资料</th><th>原值</th><th>建议值</th></tr></thead><tbody>{rows.map(([label,old,next])=><tr key={label} className="border-t border-border"><th className="p-3">{label}</th><td className="max-w-md break-all p-3">{typeof old==="string"?old:JSON.stringify(old)}</td><td className="max-w-md break-all p-3">{typeof next==="string"?next:JSON.stringify(next)}</td></tr>)}</tbody></table></section>;
}
function MaterialsConfirmation({organization,base,mode,template,candidate,busy,onCancel,onConfirm}:{organization:string;base:SupplyTarget;mode:string;template?:AgentTemplate;candidate?:{id:string;name:string};busy:boolean;onCancel:()=>void;onConfirm:()=>void}){
 const dialog=useRef<HTMLDialogElement>(null);
 useEffect(()=>{const current=dialog.current,previous=document.activeElement;current?.showModal();return()=>{current?.close();if(previous instanceof HTMLElement&&previous.isConnected)previous.focus()}},[]);
 return <dialog ref={dialog} className={styles.confirmation} aria-labelledby="materials-confirmation-title" onCancel={e=>{e.preventDefault();onCancel()}}><h2 id="materials-confirmation-title">{candidate?"确认查询候选类目规则":"确认准备平台资料"}</h2><p>当前企业 · {organization}</p><section className="mt-4 rounded-lg border border-border bg-secondary p-4"><p>原草稿 v{base.revision} · 标准商品 v{base.effectiveVersion} · SHEIN 美国站</p><p className="break-all">店铺 {base.input.storeId}</p>{candidate&&<p>候选类目 {candidate.name} · {candidate.id}</p>}<p>{template?`${template.name} · v${template.version}`:"本次不使用模板"}</p><p>{mode}</p></section><p className="mt-4">{candidate?"只查询并解释该类目的基础规则，不改变当前资料；具体属性取值的关联条件未评价。任何权限模式下，本次都不保存或发布。":"只生成平台资料建议。合格本地资料可按当前策略采用；有缺项或不确定的资料仍需人工核对。发布、改价和下架另行授权。"}</p><p>发送前重新检查企业权限、完整输入预算和成员额度；当前未报价。模型配置缺失时不执行。</p><div className={styles.actions}><Button variant="outline" disabled={busy} onClick={onCancel}>取消</Button><Button disabled={busy} onClick={onConfirm}>{candidate?"确认查询并解释":"确认生成建议"}</Button></div></dialog>;
}
