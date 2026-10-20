"use client";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { requestProductAgent, ProductAgentError } from "@/lib/api/product-agent";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";
import type { ProductAgentResult } from "@/lib/contracts/product-agent";
import { agentStartRequestSchema, contentAgentStartRequestSchema,ownContentStartRequestSchema,type OwnSourceSelection } from "@/lib/contracts/product-agent";
import { contentSelectionSchema, contentSelectionLabel, type ContentSelection } from "@/lib/contracts/content-selection";
import { ContentSelectionFields } from "../agents/content-selection-fields";
import type { z } from "zod";
import {TitleAgentTemplates} from "./title-agent-templates";
import type {AgentTemplate} from "@/lib/contracts/agent-configuration";
import { TitleAgentConfirmation } from "./title-agent-confirmation";
import { executionModeViewSchema } from "@/lib/contracts/execution-mode";
import { useConfigurationRead } from "../agents/agent-configuration-hooks";
import { executionModeLabels } from "../agents/execution-mode-settings";
import { KnowledgeCitations } from "../task-center/knowledge-citations";
import {skillSelectionSchema,type SkillSelection} from "@/lib/contracts/skill";
import {SkillMethodSummary} from "../skills/skill-method-summary";
const reasons: Record<string, string> = { budget_steps: "已达到步骤上限", budget_model_calls: "已达到模型调用上限", budget_tokens: "剩余 token 预算不足", budget_cost: "剩余费用预算不足", budget_runtime: "已达到运行时限", usage_unknown: "用量尚未确认，预留额度仍保留", model_outcome_unknown: "模型结果尚未确认，请勿重新生成", unauthorized: "当前权限不可用", dependency_unavailable: "模型配置、计费策略或依赖尚未就绪", invalid_model_output: "模型输出格式无效", repair_limit: "两次修复后仍未通过校验", audit_unavailable: "调用记录保存失败", tool_error: "证据工具读取失败" };
const errors: Record<string, string> = { MANUAL_REVIEW_REQUIRED:"图片文字翻译必须使用人工确认模式，本次没有开始模型调用。",REQUEST_NOT_RETAINED:"原请求无法完整保留，本次未发送。", PRODUCT_AGENT_UNAVAILABLE: "当前企业尚未开放标题诊断，或文本模型配置尚未就绪。", OUTCOME_UNKNOWN: "请求结果尚未确认。请保留本次编号，读取当前结果，不要重新生成。", AGENT_CONFLICT: "运行状态已经变化，请读取当前结果。", CONFIGURATION_CHANGED:"配置准入已变化，请核实当前配置与原运行状态。",AGENT_DEFINITION_UNAVAILABLE:"当前执行引用的智能体版本不可用，请核实当前版本与原运行状态。",AGENT_NOT_ENABLED:"当前企业智能体已停用，请由管理员重新启用。",TEMPLATE_ARCHIVED:"所选模板已归档，请核实当前结果和模板。",IDEMPOTENCY_CONFLICT:"本次编号已绑定其他配置，请保留编号并读取当前结果。", FORBIDDEN: "当前身份没有操作权限。", ORGANIZATION_ACCESS_REVOKED: "当前企业权限已撤销。", DEPENDENCY_UNAVAILABLE: "暂时无法读取结果，请稍后用同一编号查询。" };
const knowledgeStartErrors:Record<string,string>={KNOWLEDGE_CONTEXT_TOO_LARGE:"所选知识完整内容超过输入上限，本次未启动。请缩小知识范围后重新确认。",KNOWLEDGE_NOT_READY:"所选知识没有完整可读版本，本次未启动。",KNOWLEDGE_DISABLED:"所选知识已停用，本次未启动。",KNOWLEDGE_NOT_FOUND:"所选知识当前不可用，本次未启动。",KNOWLEDGE_FORBIDDEN:"当前身份无权读取所选知识，本次未启动。"};
export type StartIntent=(z.infer<typeof agentStartRequestSchema>|z.infer<typeof contentAgentStartRequestSchema>)&{ownSource?:OwnSourceSelection};
function restoredStart(params:{get:(name:string)=>string|null},userId:string,organizationId:string,content:boolean,own=false):StartIntent|null {
    if(!isAcquisitionUUID(params.get("agent_key")??"")||params.get("agent_actor")!==userId||params.get("agent_org")!==organizationId)return null;
    const baseId=params.get("agent_knowledge_base");
    const templateId=params.get("agent_template"),revision=params.get("agent_template_version");
    let selection: unknown,method:unknown;
    const rawSkill=params.get("agent_skill");if(rawSkill){if(!content||rawSkill.length>1024)return null;try{method=skillSelectionSchema.parse(JSON.parse(rawSkill))}catch{return null}}
    if(content){const raw=params.get("agent_content");if(!raw||raw.length>2048)return null;try{selection=JSON.parse(raw)}catch{return null}}
    const parsed=(own?ownContentStartRequestSchema:content?contentAgentStartRequestSchema:agentStartRequestSchema).safeParse({expectedModeRevision:params.get("agent_mode_revision"),...(own?{ownSource:{catalogVersion:params.get("agent_own_version"),publicationId:params.get("agent_own_publication")}}:{}),...(templateId!==null||revision!==null?{templateSelection:{templateId,revision}}:{}),targetPlatform:params.get("agent_platform"),...(baseId!==null?{knowledgeSelection:{knowledgeBaseId:baseId}}:{}),...(content?{contentSelection:selection}:{}),...(method?{skillSelection:method}:{})});
    return parsed.success?parsed.data:null;
}
export type ImageTranslationEntry={language:string;original?:{key:string;start:StartIntent};persist:(key:string,start:StartIntent)=>void;reset?:()=>void;onReview:(id:string)=>void};
const intentParams=["agent_key","agent_platform","agent_knowledge_base","agent_actor","agent_org","agent_template","agent_template_version","agent_mode_revision","agent_kind","agent_content","agent_skill","agent_own_version","agent_own_publication"];
export function ProductAgentPanel({ enabled, knowledgeAvailable=false, operationId, productKey, catalogVersion,ownSource,initialSkill,imageTranslation }: {
    enabled: boolean;
    knowledgeAvailable?:boolean;
    operationId: string;
    productKey: string;
    catalogVersion: string;
    ownSource?:OwnSourceSelection;
    initialSkill?:SkillSelection;
    imageTranslation?:ImageTranslationEntry;
}) {
    const context = useWorkbenchContext();
    const params = useSearchParams();
    const [selectedKind,setKind]=useState<"title"|"content">(params.get("agent_kind")==="content"?"content":"title");
    const kind=ownSource||imageTranslation?"content":selectedKind;
    const userId = context.user?.id, organizationId = context.effectiveOrganization?.id;
    if (!enabled)
        return <Card><h2>标题诊断</h2><p>当前环境尚未开放 Product Agent。</p></Card>;
    if (!userId || !organizationId || context.isSwitching || context.isLoading || context.error || context.blockingError || context.selectionRequired)
        return null;
    return <ScopedAgentPanel key={`${userId}:${organizationId}:${operationId}:${kind}`} ownSource={ownSource} kind={kind} onKindChange={setKind} knowledgeAvailable={knowledgeAvailable} userId={userId} organizationId={organizationId} operationId={operationId} productKey={productKey} catalogVersion={catalogVersion} initialKey={imageTranslation?imageTranslation.original?.key??"":ownSource&&(params.get("agent_actor")!==userId||params.get("agent_org")!==organizationId)?"":params.get("agent_key")??""} initialSkill={initialSkill} imageTranslation={imageTranslation} initialStart={imageTranslation?imageTranslation.original?.start??null:restoredStart(params,userId,organizationId,kind==="content",!!ownSource)}/>;
}
function ScopedAgentPanel({ userId, organizationId, operationId, productKey, initialKey,initialStart,knowledgeAvailable,kind,onKindChange,ownSource,initialSkill,imageTranslation }: {
    ownSource?:OwnSourceSelection;
    kind:"title"|"content";onKindChange:(kind:"title"|"content")=>void;
    knowledgeAvailable:boolean;
    userId: string;
    organizationId: string;
    operationId: string;
    productKey: string;
    catalogVersion: string;
    initialKey: string;
    initialStart:StartIntent|null;
    initialSkill?:SkillSelection;
    imageTranslation?:ImageTranslationEntry;
}) {
    const context = useWorkbenchContext();
    const content=kind==="content",agentId=content?"product.content.agent":"product.title.agent";
    const [modeNonce,setModeNonce]=useState(0);
    const executionMode=useConfigurationRead({userId,organizationId},`execution-modes/registered_agent/${agentId}`,executionModeViewSchema,modeNonce);
    const [key, setKey] = useState(isAcquisitionUUID(initialKey) ? initialKey : "");
    // Public normalized command only; protected Knowledge payload and authority
    // stay with the server. Explicit replay never replaces an uncertain key.
    const [intent,setIntent]=useState<StartIntent|null>(initialStart);
    const authorizationKey=JSON.stringify([[...context.roles].sort(),[...context.permissions].sort()]);
    const [authority,setAuthority]=useState({authorizationKey,generation:0});
    const generation=authority.authorizationKey===authorizationKey?authority.generation:authority.generation+1;
    if(authority.authorizationKey!==authorizationKey)setAuthority({authorizationKey,generation});
    const knowledgeReadable=context.permissions.includes("workbench.knowledge.read");
    const [result, setResult] = useState<(ProductAgentResult & {knowledgeGeneration:number}) | null>(null);
    // Invalidate protected display during render, including late responses and
    // downgrade/regrant. Keep the durable operation key and all UNKNOWN intent.
    if(result&&result.knowledgeGeneration!==generation) {
        setResult({...result,knowledgeGeneration:generation,knowledge:result.knowledge?{status:"unavailable",originAgentRunId:result.knowledge.originAgentRunId,citations:[]}:undefined});
    }
    const [proposal, setProposal] = useState("");
    const [feedback, setFeedback] = useState("");
    // A history key has no complete Start intent. Let its first verified result
    // establish the platform, then retain the known run's exact binding.
    const [platform, setPlatform] = useState<string>(initialStart?.targetPlatform??(isAcquisitionUUID(initialKey)?"":content?"product":""));
    const [selection,setSelection]=useState<ContentSelection>(initialStart&&"contentSelection" in initialStart?initialStart.contentSelection:imageTranslation?{targetLanguage:imageTranslation.language,title:true,sellingPoints:true}:ownSource?{targetLanguage:"zh-CN",title:true}:{targetLanguage:"",title:true,description:true,sellingPoints:true,seoKeywords:true});
    const [busy, setBusy] = useState(false);
    const [failure, setFailure] = useState("");
    const [reconfirmable,setReconfirmable]=useState(false);
    const [confirming,setConfirming]=useState(!!imageTranslation&&!initialKey);
    const [template,setTemplate]=useState<AgentTemplate|null>(null),[templateReady,setTemplateReady]=useState(!!imageTranslation);
    const active = useRef(true), inFlight = useRef(false), abort = useRef<AbortController | null>(null);
    useEffect(() => { active.current = true; return () => { active.current = false; abort.current?.abort(); }; }, []);
    useEffect(() => context.registerOrganizationSwitchGuard(() => !inFlight.current), [context]);
    async function execute(action: "start" | "read" | "resume" | "review",knowledgeBaseId?:string,method?:SkillSelection) {
        const start:StartIntent|null|undefined=action==="start"?intent??(ownSource?ownContentStartRequestSchema:content?contentAgentStartRequestSchema:agentStartRequestSchema).safeParse({expectedModeRevision:executionMode.data?.revision,targetPlatform:platform,...(imageTranslation?{manualReviewOnly:true}:{}),...(ownSource?{ownSource}:{}),...(template?{templateSelection:{templateId:template.templateId,revision:template.version}}:{}),...(knowledgeBaseId?{knowledgeSelection:{knowledgeBaseId}}:{}),...(content?{contentSelection:selection}:{}),...(method?{skillSelection:method}:{})}).data:null;
        if (inFlight.current || !active.current || action === "start" && !start)
            return;
        const freshStart=action==="start"&&!key;
        const requestKey = key || crypto.randomUUID();
        if(freshStart&&imageTranslation&&start){try{imageTranslation.persist(requestKey,start)}catch{setFailure("REQUEST_NOT_RETAINED");return}}

        if (!key) {
            setKey(requestKey);
            if(imageTranslation){setIntent(start??null)}else{
            const url = new URL(window.location.href);
            url.searchParams.set("agent_key", requestKey);
            url.searchParams.set("agent_kind",kind);
            if(start){
                setIntent(start);url.searchParams.set("agent_platform",start.targetPlatform);
                url.searchParams.set("agent_mode_revision",start.expectedModeRevision);
                url.searchParams.set("agent_actor",userId);url.searchParams.set("agent_org",organizationId);
                url.searchParams.delete("agent_template");url.searchParams.delete("agent_template_version");
                if(start.templateSelection){url.searchParams.set("agent_template",start.templateSelection.templateId);url.searchParams.set("agent_template_version",start.templateSelection.revision)}
                url.searchParams.delete("agent_knowledge_base");
                if(start.knowledgeSelection)url.searchParams.set("agent_knowledge_base",start.knowledgeSelection.knowledgeBaseId);
                url.searchParams.delete("agent_skill");if("skillSelection" in start&&start.skillSelection)url.searchParams.set("agent_skill",JSON.stringify(start.skillSelection));
                url.searchParams.delete("agent_content");if("contentSelection" in start)url.searchParams.set("agent_content",JSON.stringify(start.contentSelection));
                if(start.ownSource){url.searchParams.set("agent_own_version",start.ownSource.catalogVersion);url.searchParams.set("agent_own_publication",start.ownSource.publicationId)}
            }
            window.history.replaceState(null, "", url);
            }
        }
        const controller = new AbortController();
        abort.current = controller;
        inFlight.current = true;
        setBusy(true);
        setFailure("");
        setReconfirmable(false);
        try {
            const next = await requestProductAgent(action, operationId, requestKey, { userId, organizationId }, controller.signal, result?.revision, feedback, start?.targetPlatform??platform,start?.knowledgeSelection?.knowledgeBaseId,start?.templateSelection,start?.expectedModeRevision,start&&"contentSelection" in start?start.contentSelection:content?selection:undefined,kind,start&&"skillSelection" in start?start.skillSelection:undefined,ownSource?{source:(start as StartIntent|null)?.ownSource}:undefined,!!imageTranslation);
            if (!active.current)
                return;
            if ("proposalId" in next) {
                setProposal(next.proposalId);
                imageTranslation?.onReview(next.proposalId);
                return;
            }
            // The immutable acquisition receipt may precede an adopted Catalog
            // head. The API verifies operation/request identity; retain a known
            // run's exact base as well as the page's product and selected platform.
            if (next.productKey !== productKey || platform && next.targetPlatform !== platform ||
                result && (next.runId !== result.runId || next.catalogVersion !== result.catalogVersion || next.publicationId !== result.publicationId))
                throw new ProductAgentError("AGENT_CONFLICT");
            setPlatform(next.targetPlatform);
            if(next.reviewId){setProposal(next.reviewId);imageTranslation?.onReview(next.reviewId)}
            setResult({...next,knowledgeGeneration:generation});
        }
        catch (error) {
            if (active.current) {
                setFailure(error instanceof ProductAgentError ? error.code : "OUTCOME_UNKNOWN");
                setReconfirmable(action==="start"&&!result&&error instanceof ProductAgentError&&["CONFIGURATION_CHANGED","TEMPLATE_ARCHIVED","AGENT_NOT_ENABLED","AGENT_DEFINITION_UNAVAILABLE","SKILL_NOT_ENABLED","SKILL_FORBIDDEN","SKILL_NOT_FOUND","SKILL_IDEMPOTENCY_CONFLICT"].includes(error.code));
                if(freshStart&&error instanceof ProductAgentError&&knowledgeStartErrors[error.code]){
                    setKey("");setIntent(null);const url=new URL(window.location.href);for(const name of intentParams)url.searchParams.delete(name);window.history.replaceState(null,"",url);
                }
                if (error instanceof ProductAgentError && (error.code === "FORBIDDEN" || error.code.startsWith("ORGANIZATION_"))) {
                    setResult(null);
                    setProposal("");
                }
            }
        }
        finally {
            inFlight.current = false;
            if (active.current)
                setBusy(false);
        }
    }
    return <Card className="space-y-3 p-5"><h2 className="text-lg font-semibold">商品内容与本地化</h2>
  {imageTranslation?<p>目标语言：{imageTranslation.language} · 自动翻译后人工审核，不应用商品</p>:ownSource?<p>自有商品标题优化 · 无需绑定店铺</p>:<label className="block">处理内容<select className="block rounded-lg border p-2" value={kind} disabled={!!key||busy||confirming} onChange={event=>onKindChange(event.target.value as "title"|"content")}><option value="title">标题优化</option><option value="content">内容完善与本地化</option></select></label>}
  <p>从当前已保存版本读取证据，生成{content?"选中内容":"标题"}建议。最多修复两次；按本次确认的权限模式处理本地采用。</p>
  {!key ? <>{!imageTranslation&&<TitleAgentTemplates key={authorizationKey} agentId={agentId} scope={{userId,organizationId}} onReady={setTemplateReady} onChoose={value=>{setTemplate(value);if(value){if(!ownSource)setPlatform(value.targetPlatform);if(value.agentId==="product.content.agent"&&!ownSource)setSelection(value.content.selection)}}}/>}<label htmlFor="agent-platform">素材查询平台</label><select id="agent-platform" className="block rounded-lg border p-2" value={platform} onChange={event => setPlatform(event.target.value)} disabled={!!ownSource||!!imageTranslation||busy||confirming}><option value="">请选择平台</option>{content&&<option value="product">标准商品（不指定平台）</option>}<option value="shein">SHEIN</option><option value="temu">Temu</option><option value="amazon">Amazon</option></select>{content&&(ownSource&&!imageTranslation?<label className="block">目标语言<Input value={selection.targetLanguage} disabled={busy||confirming} onChange={event=>setSelection({targetLanguage:event.target.value,title:true})}/><span className="text-sm">本入口仅优化标题，其他商品资料保持原样。</span></label>:imageTranslation?<p>翻译商品名称与卖点；目标语言随当前图片模板固定。</p>:<ContentSelectionFields value={selection} onChange={setSelection} disabled={busy||confirming}/>)}<p className="text-sm">{ownSource?"读取已保存的自有商品证据；请选择标准商品、仅标题模板，或不使用模板。":"用于读取已批准的素材；本次不评估平台发布规则。"}</p><Button disabled={busy || !!imageTranslation&&executionMode.data?.mode!=="default" || !platform || !templateReady || !!ownSource&&!!template&&(template.targetPlatform!=="product"||template.agentId!=="product.content.agent"||!template.content.selection.title||!!template.content.selection.description||!!template.content.selection.sellingPoints||!!template.content.selection.seoKeywords) || content&&!contentSelectionSchema.safeParse(selection).success} onClick={() => setConfirming(true)}>{content?"生成内容建议":"生成标题建议"}</Button></> : <><p className="break-all text-sm">本次编号：{key}</p><Button variant="outline" disabled={busy} onClick={() => void execute("read")}>读取当前结果</Button>{intent&&!result&&<><p className="text-sm">原请求平台：{intent.targetPlatform}；{intent.knowledgeSelection?`知识库编号：${intent.knowledgeSelection.knowledgeBaseId}`:"未选择企业知识"}。恢复将沿用同一编号和选择；服务端已有运行时只返回该运行，不会重新生成。</p>{"contentSelection" in intent&&<p>{contentSelectionLabel(intent.contentSelection)}</p>}<Button variant="outline" disabled={busy} onClick={()=>void execute("start")}>使用原请求核实启动</Button></>}</>}
  {imageTranslation&&executionMode.data&&executionMode.data.mode!=="default"&&<p role="status">图片文字翻译仅支持人工确认模式；当前模式不能开始翻译，可返回选择不加文字。</p>}
  {confirming&&!key&&(!imageTranslation||executionMode.data?.mode==="default")&&<TitleAgentConfirmation initialSkill={initialSkill} key={authorizationKey} executionMode={executionMode.data?.subject.id===agentId?executionMode.data:undefined} contentSelection={content?selection:undefined} scope={{userId,organizationId}} organization={context.effectiveOrganization?.name||organizationId} platform={platform} template={template??undefined} knowledgeAvailable={knowledgeAvailable} knowledgeReadable={knowledgeReadable} onCancel={()=>setConfirming(false)} onConfirm={(baseId,method)=>{setConfirming(false);void execute("start",baseId,method)}}/>}
  {result?<p>本次权限：{result.executionMode?`${executionModeLabels[result.executionMode]} · 模式版本 ${result.modeRevision}`:"历史记录未冻结权限模式，新采用不可用"}{result.policyAdoption?" · 已按策略采用，未逐项人工审阅":""}</p>:null}
  {busy && <p role="status">正在处理，请保持页面打开。离开页面可能使已发送的调用结果未知；返回后请用本次编号读取，勿重新生成。</p>}
  {failure && <p role="alert">{reconfirmable?`${errors[failure]??"所选技能准入已变化。"} 本次未领取执行。请明确重新确认配置，再使用新编号生成。`:key&&knowledgeStartErrors[failure]?"知识当前不可用，已保留原编号；已有运行结果尚需核实，请勿换编号重新生成。":knowledgeStartErrors[failure] ?? errors[failure] ?? "请求未完成，请核对当前身份和企业。"}</p>}
  {reconfirmable&&<Button variant="outline" disabled={busy} onClick={()=>{if(inFlight.current)return;if(imageTranslation){try{imageTranslation.reset?.()}catch{setFailure("REQUEST_NOT_RETAINED");return}}setModeNonce(v=>v+1);setKey("");setIntent(null);setTemplate(null);setTemplateReady(!!imageTranslation);setPlatform(ownSource||imageTranslation?"product":"");setFailure("");setReconfirmable(false);if(imageTranslation)return;const url=new URL(window.location.href);for(const name of intentParams)url.searchParams.delete(name);window.history.replaceState(null,"",url)}}>重新确认配置</Button>}
  {result && <div className="space-y-2"><SkillMethodSummary method={"skillVersionRef" in result?result.skillVersionRef:undefined}/>{"contentSelection" in result&&<p>{contentSelectionLabel(result.contentSelection)}</p>}{result.templateSelection&&<p>采用模板版本：{result.templateSelection.revision} · {result.templateSelection.templateId}</p>}<p>素材查询平台：{{ product:"标准商品",shein: "SHEIN", temu: "Temu", amazon: "Amazon" }[result.targetPlatform]}</p><p>状态：{result.policyAdoption?"已按策略采用，未经逐项人工审阅":result.phase === "human_review_required" ? (result.canSubmitReview ? "候选已通过校验，等待人工审核" : "候选未通过校验，不能提交人工审核") : result.phase === "interrupted" ? "需要补充说明" : result.phase === "running" ? "运行记录尚未完成，不能重复发送" : "已停止"}</p>
   {result.stopReason && <p>{reasons[result.stopReason] ?? "本次运行已停止"}</p>}
   {result.candidate.Changes?.map((change, index) => <div key={index}><p>{change.Field}：{change.Value}</p><p className="text-sm">证据：{change.EvidenceIDs?.join("、") || "未提供"}</p></div>)}
   <KnowledgeCitations knowledge={result.knowledge&&(!knowledgeReadable||result.knowledgeGeneration!==generation)?{status:"unavailable",originAgentRunId:result.knowledge.originAgentRunId,citations:[]}:result.knowledge}/>
   {result.confidence?.map(value => <p key={value.Field}>模型自报置信度（供参考）：{value.Field} {value.Known ? `${Math.round(value.Value * 100)}%` : "未知"}</p>)}
   {result.unresolved?.length ? <ul>{result.unresolved.map((item, index) => <li key={index}>{item}</li>)}</ul> : null}
   <p className="text-sm">{result.usageStatus === "observed" ? `已观测 ${result.tokens} tokens，预算估价 ${(result.estimatedCostMicros / 1000000).toFixed(4)} ${result.currency}` : "用量尚未确认，显示的预留不能当作实际账单。"}</p>
   <details><summary>查看调用记录</summary><ol>{result.steps.map(step => <li key={step.step}>{step.step}：{step.tool || "文本模型"} · {step.callId}</li>)}</ol></details>
   {result.phase === "interrupted" && <><label htmlFor="agent-feedback">补充说明</label><Input id="agent-feedback" value={feedback} onChange={e => setFeedback(e.target.value)} disabled={busy}/><Button disabled={busy || !feedback.trim()} onClick={() => void execute("resume")}>继续本次诊断</Button></>}
   {result.canSubmitReview && !proposal && <Button disabled={busy} onClick={() => void execute("review")}>{result.executionMode==="read_only"?"保存内容建议":"提交人工审核"}</Button>}
  </div>}
  {proposal && <Button asChild variant="outline"><Link href={`/workbench/ai/tasks/pending/other?proposal_id=${proposal}`}>{content?"打开内容建议":"打开标题审核"}</Link></Button>}
 </Card>;
}
