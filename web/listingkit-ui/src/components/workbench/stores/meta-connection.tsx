"use client";

import Link from "next/link";
import {useEffect,useMemo,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {z} from "zod";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Label} from "@/components/ui/label";
import {Select} from "@/components/ui/select";
import {metaRequest,type MetaScope} from "@/lib/api/meta-catalog";
import type {MetaResponse} from "@/lib/contracts/meta-catalog";

const pendingKey="shuomi-meta-simulation-consent";
const pendingSchema=z.strictObject({
  storeId:z.uuid(),attemptId:z.uuid(),userId:z.string(),organizationId:z.string(),
  state:z.string().length(43),expiresAt:z.iso.datetime({offset:true}),
});

export function MetaConnection({storeId,scope,canWrite}:{storeId:string;scope:MetaScope;canWrite:boolean}){
  const [busy,setBusy]=useState(false);
  const [error,setError]=useState("");
  const status=useQuery({queryKey:["meta-connection",scope,storeId],
    queryFn:()=>metaRequest(scope,"status",storeId),retry:false});
  async function update(action:"begin"|"disconnect"){
    if(!status.data)return;
    setBusy(true);
    setError("");
    try{
      if(action==="disconnect"){
        await metaRequest(scope,"disconnect",storeId,{version:status.data.version});
        await status.refetch();
      }else{
        const reply=await metaRequest(scope,"begin",storeId,{version:status.data.version});
        const url=new URL(reply.authorizationUrl);
        const pending=pendingSchema.parse({...scope,storeId,attemptId:reply.attemptId,expiresAt:reply.expiresAt,state:url.searchParams.get("state")});
        sessionStorage.setItem(pendingKey,JSON.stringify(pending));
        window.location.assign(url);
      }
    }catch{
      setError("无法更新授权。请刷新店铺并确认你有企业管理员权限。");
    }finally{setBusy(false);}
  }
  return <div className="space-y-3">
    <p role="status">{status.isPending?"正在读取目录授权…":status.isError?"暂时无法读取目录授权。":
      status.data.connected?"模拟目录已连接："+status.data.catalogId:"尚未连接模拟目录。"}</p>
    {status.data?.connected&&status.data.whatsAppId&&<p>已选择 WhatsApp Business 账号 {status.data.whatsAppId}</p>}
    {canWrite&&status.data&&<div className="flex flex-wrap gap-2">
      <Button disabled={busy} onClick={()=>void update("begin")}>{status.data.connected?"重新授权 Meta":"授权 Meta"}</Button>
      {status.data.connected&&<Button variant="outline" disabled={busy} onClick={()=>void update("disconnect")}>解除授权</Button>}
    </div>}
    {error&&<p role="alert">{error}</p>}
  </div>;
}

export function MetaAuthorizationCallback(){
  const context=useWorkbenchContext();
  const userId=context.user?.id??"",organizationId=context.effectiveOrganization?.id??"";
  const scope=useMemo(()=>({userId,organizationId}),[userId,organizationId]);
  if(!scope.userId||!scope.organizationId||context.isSwitching)return <p role="status">正在确认当前企业…</p>;
  return <MetaCallbackFlow key={scope.userId+":"+scope.organizationId} scope={scope}/>;
}

function MetaCallbackFlow({scope}:{scope:MetaScope}){
  const [pending,setPending]=useState<z.infer<typeof pendingSchema>>();
  const [choices,setChoices]=useState<MetaResponse<"choices">>();
  const [catalogId,setCatalogId]=useState("");
  const [whatsAppId,setWhatsAppId]=useState("");
  const [error,setError]=useState("");
  const [busy,setBusy]=useState(true);
  const [connected,setConnected]=useState(false);

  useEffect(()=>{
    let current=true;
    void Promise.resolve().then(async()=>{
      if(!current)return;
      try{
        const query=new URLSearchParams(window.location.search);
        const saved=pendingSchema.parse(JSON.parse(sessionStorage.getItem(pendingKey)??"null"));
        window.history.replaceState(null,"",window.location.pathname);
        sessionStorage.removeItem(pendingKey);
        const code=query.get("code");
        if(query.get("state")!==saved.state||!code||Date.parse(saved.expiresAt)<=Date.now()||
          scope.organizationId!==saved.organizationId||scope.userId!==saved.userId)throw new Error();
        setPending(saved);
        await metaRequest(scope,"complete",saved.storeId,{attemptId:saved.attemptId,state:saved.state,code});
        const available=await metaRequest(scope,"choices",saved.storeId,{attemptId:saved.attemptId});
        if(current)setChoices(available);
      }catch{
        window.history.replaceState(null,"",window.location.pathname);
        if(current)setError("无法完成模拟授权。请返回店铺，再次发起授权。");
      }finally{if(current)setBusy(false);}
    });
    return()=>{current=false;};
  },[scope]);

  async function bind(){
    if(!catalogId||!pending)return;
    setBusy(true);
    setError("");
    try{
      const result=await metaRequest(scope,"bind",pending.storeId,{attemptId:pending.attemptId,catalogId,whatsAppId});
      if(!result.connected)throw new Error();
      setConnected(true);
    }catch{setError("无法连接此目录。请确认账号属于同一企业，且所选 WhatsApp 账号已绑定该目录。");}
    finally{setBusy(false);}
  }
  const business=choices?.catalogs.find(c=>c.id===catalogId)?.businessId;
  return <section className="mx-auto max-w-xl space-y-4 rounded-lg border p-6">
    <h1 className="text-xl font-semibold">连接 Meta 商品目录</h1>
    {connected?<p role="status">Meta 模拟目录已连接。</p>:choices?.catalogs.length===0?
      <p role="status">未找到你可以管理的企业商品目录。</p>:choices?
      <form className="space-y-4" onSubmit={event=>{event.preventDefault();void bind();}}>
        <div><Label htmlFor="meta-catalog">商品目录</Label><Select id="meta-catalog" value={catalogId}
          onChange={event=>{setCatalogId(event.target.value);setWhatsAppId("");}}>
          <option value="">选择目录</option>{choices.catalogs.map(c=><option key={c.id} value={c.id}>{c.name}（{c.id}）</option>)}
        </Select></div>
        {business&&<div><Label htmlFor="meta-whatsapp">WhatsApp Business 账号（可选）</Label>
          <Select id="meta-whatsapp" value={whatsAppId} onChange={event=>setWhatsAppId(event.target.value)}>
            <option value="">暂不查询 WhatsApp 绑定</option>
            {choices.accounts.filter(a=>a.businessId===business).map(a=><option key={a.id} value={a.id}>{a.name}（{a.id}）</option>)}
          </Select>
        </div>}
        <Button type="submit" disabled={busy||!catalogId}>确认连接</Button>
      </form>:busy?<p role="status">正在检查可管理的商品目录…</p>:null}
    {error&&<p role="alert">{error}</p>}
    <Link className="underline" href={pending?"/workbench/stores/"+pending.storeId+"#authorization":"/workbench/stores"}>返回店铺中心</Link>
  </section>;
}
