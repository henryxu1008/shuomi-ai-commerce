"use client";

import Link from "next/link";
import {useEffect,useRef,useState} from "react";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Select} from "@/components/ui/select";
import {ProjectFields} from "../project-center/project-fields";
import {AIWorkbenchError,requestAIWorkbench,type AIScope} from "@/lib/api/ai-workbench";
import {ProjectError,projectRequest} from "@/lib/api/project-center";
import {projectBody,receipt,type ProjectBody} from "@/lib/contracts/project-center";
import {isAcquisitionUUID} from "@/lib/contracts/product-acquisition";
import {verifyChatStoreChoice,type ChatStoreChoice,type ChatStoreState} from "./chat-store-picker";

export type ProjectSkill = "" | "create-project" | "project-task";

export function ProjectSkillPicker({value,onChange,disabled,canCreate,canExecute}:{
  value:ProjectSkill;onChange:(value:ProjectSkill)=>void;disabled:boolean;canCreate:boolean;canExecute:boolean;
}) {
  return <Select className="w-auto max-w-full" aria-label="选择技能" value={value} disabled={disabled} onChange={event=>onChange(event.target.value as ProjectSkill)}>
    <option value="">选择技能</option>
    <option value="create-project" disabled={!canCreate}>创建项目</option>
    <option value="project-task" disabled={!canExecute}>执行项目任务</option>
  </Select>;
}

export async function readProjectWorkScope(scope:AIScope,projectId:string,signal:AbortSignal) {
  const query = new URLSearchParams({kind:"project",projectId});
  const response = await requestAIWorkbench({route:"scope-options",method:"GET",path:"chat/conversations/scopes?"+query,scope,signal});
  const option = !response.incomplete && response.options.length === 1 ? response.options[0] : undefined;
  if (!option || option.scope.kind !== "project" || option.scope.projectId !== projectId) {
    throw new AIWorkbenchError("DEPENDENCY_UNAVAILABLE");
  }
  return option;
}

type ProjectIntent = {key:string;body:ProjectBody;owner:AIScope;context:string};

export function useChatProjectCreation(scope:AIScope,context:string,enabled:boolean) {
  const key = "ai-workbench:project-create:"+JSON.stringify([scope.userId,scope.organizationId,context]);
  const [checked,setChecked] = useState("");
  const [storageError,setStorageError] = useState("");
  const [pending,setPending] = useState<ProjectIntent>();
  const [busy,setBusy] = useState(false);
  const [error,setError] = useState("");
  const [projectId,setProjectId] = useState("");
  const controller = useRef<AbortController|null>(null);

  useEffect(()=>{
    if (!enabled) return;
    let active = true;
    queueMicrotask(()=>{
      if (!active) return;
      try {
        const raw = sessionStorage.getItem(key);
        const intent = raw ? JSON.parse(raw) as ProjectIntent : undefined;
        if (intent && (!isAcquisitionUUID(intent.key) || !projectBody.safeParse(intent.body).success
          || intent.owner?.userId !== scope.userId || intent.owner?.organizationId !== scope.organizationId || intent.context !== context)) {
          throw new Error("invalid pending project");
        }
        setPending(intent);
        setStorageError("");
      } catch {
        setStorageError("原创建项目请求无法读取，请保留浏览器记录后重试。");
      }
      setChecked(key);
      setProjectId("");
      setError("");
    });
    return ()=>{active=false;controller.current?.abort();controller.current=null;};
  },[key,enabled,context,scope.userId,scope.organizationId]);

  async function submit(body:ProjectBody|undefined,choice?:ChatStoreChoice) {
    if (!enabled || checked !== key || storageError || controller.current || (!pending && !body)) return;
    const original = pending;
    const intent:ProjectIntent = original ?? {key:crypto.randomUUID(),body:body!,owner:scope,context};
    const operation = new AbortController();
    controller.current = operation;
    setBusy(true);
    setError("");
    let attempted = false;
    try {
      if (!original && intent.body.storeId) {
        if (choice?.kind !== "store" || choice.storeId !== intent.body.storeId) throw new Error("invalid store");
        await verifyChatStoreChoice(scope,choice,operation.signal);
      }
      if (operation.signal.aborted) return;
      if (!original) {
        try {
          sessionStorage.setItem(key,JSON.stringify(intent));
        } catch {
          setError("无法保存请求，请检查浏览器存储设置。项目尚未提交。");
          return;
        }
        setPending(intent);
      }
      attempted = true;
      const response = await projectRequest(scope,"",receipt,operation.signal,{path:"",method:"POST",key:intent.key,body:intent.body});
      if (operation.signal.aborted) return;
      sessionStorage.removeItem(key);
      setPending(undefined);
      setProjectId(response.id);
    } catch (cause) {
      if (operation.signal.aborted) return;
      if (!attempted) {
        setError("所选店铺暂不可用，请重新选择或取消关联。");
      } else if (!original && cause instanceof ProjectError && ["INVALID_REQUEST","FORBIDDEN","NOT_FOUND"].includes(cause.code)) {
        try {
          sessionStorage.removeItem(key);
          setPending(undefined);
          setError(cause.code === "NOT_FOUND" ? "所选店铺暂不可用，请重新选择或取消关联。" : "项目无法保存，请检查填写内容或当前权限。");
        } catch {
          setError("无法清除原请求记录，请检查浏览器存储。");
        }
      } else {
        setError("创建结果待确认，请核对原请求。");
      }
    } finally {
      if (!operation.signal.aborted) setBusy(false);
      if (controller.current === operation) controller.current = null;
    }
  }
  const ready = enabled && checked === key;
  return {pending:ready?pending:undefined,busy:enabled&&busy,checked:ready,storageError,error,projectId:ready?projectId:"",submit,reset:()=>{setError("");setProjectId("");}};
}

export function ChatProjectCreateCard({creation,store,canCreate,disabled,onClose}:{
  creation:ReturnType<typeof useChatProjectCreation>;store:ChatStoreState;canCreate:boolean;disabled:boolean;onClose:()=>void;
}) {
  const [body,setBody] = useState<ProjectBody>({title:"",goal:"",kind:"STORE_OPERATIONS",dueDate:""});
  const [withoutStore,setWithoutStore] = useState(false);
  const [inputError,setInputError] = useState("");
  const selected = store.choice?.kind === "store" ? store.choice : undefined;
  const locked = creation.busy || Boolean(creation.pending) || disabled;
  if (creation.projectId) return <Card className="space-y-3 p-4">
    <h2>项目已创建</h2>
    <div className="flex flex-wrap gap-2">
      <Button asChild variant="outline"><Link href={"/workbench/ai/projects/"+creation.projectId}>查看项目</Link></Button>
      <Button type="button" variant="ghost" onClick={onClose}>继续聊天</Button>
    </div>
  </Card>;
  return <Card className="p-4"><form className="grid gap-3" onSubmit={event=>{
    event.preventDefault();
    if (creation.pending) {void creation.submit(undefined);return;}
    const parsed = projectBody.safeParse({...body,storeId:withoutStore ? "" : selected?.storeId ?? ""});
    if (!parsed.success) {setInputError("请检查项目名称、目标、类型和截止日期。");return;}
    setInputError("");
    void creation.submit(parsed.data,selected);
  }}>
    <h2>创建项目</h2>
    <ProjectFields body={creation.pending?.body ?? body} disabled={locked||!canCreate} onChange={(key,value)=>setBody(previous=>({...previous,[key]:value}))}/>
    <div>
      <p>关联店铺：{creation.pending ? creation.pending.body.storeId ? "按原请求核对" : "不关联店铺" : withoutStore || !selected ? "不关联店铺" : store.name ?? "已选店铺"}</p>
      {!creation.pending && (selected || !store.ready) ? <label><input type="checkbox" checked={withoutStore} disabled={locked} onChange={event=>setWithoutStore(event.target.checked)}/> 不关联店铺</label> : null}
    </div>
    {inputError || creation.storageError || creation.error ? <p role="alert">{inputError || creation.storageError || creation.error}</p> : null}
    {creation.pending && !creation.error ? <p role="status">创建结果待确认，请核对原请求。</p> : null}
    <div className="flex flex-wrap gap-2">
      <Button type="submit" disabled={!canCreate||!creation.checked||creation.busy||disabled||Boolean(creation.storageError)||!creation.pending&&!store.ready&&!withoutStore}>
        {creation.busy ? "正在保存" : creation.pending ? "核对原创建项目请求" : "创建项目"}
      </Button>
      <Button type="button" variant="outline" disabled={locked} onClick={onClose}>取消</Button>
    </div>
  </form></Card>;
}
