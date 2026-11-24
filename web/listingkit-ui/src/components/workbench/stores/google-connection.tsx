"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { z } from "zod";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { googleRequest, type GoogleResponse, type GoogleScope } from "@/lib/api/google-merchant";

const pendingKey = "merchant-google-pending";
const pendingSchema = z.strictObject({
  userId: z.string(), organizationId: z.string(), storeId: z.uuid(), attemptId: z.uuid(),
  state: z.string().length(43), expiresAt: z.string(),
});

export function GoogleConnection({ storeId, version, scope, canWrite, onChanged }: {
  storeId: string; version: number; scope: GoogleScope; canWrite: boolean; onChanged: () => Promise<void>;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const view = useQuery({
    queryKey:["google-connection",scope.organizationId,scope.userId,storeId,version],
    queryFn:() => googleRequest(scope,"status",storeId), retry:false,
  });
  async function begin() {
    setBusy(true);
    setError("");
    try {
      const reply = await googleRequest(scope,"begin",storeId,{version});
      const url = new URL(reply.authorizationUrl);
      const pending = pendingSchema.parse({...scope,storeId,attemptId:reply.attemptId,state:url.searchParams.get("state"),expiresAt:reply.expiresAt});
      sessionStorage.setItem(pendingKey,JSON.stringify(pending));
      window.location.assign(url);
    } catch {
      setError("无法开始 Google 授权。请刷新店铺并确认你有企业管理员权限。");
      setBusy(false);
    }
  }
  async function disconnect() {
    setBusy(true);
    setError("");
    try {
      await googleRequest(scope,"disconnect",storeId,{version});
      await onChanged();
      await view.refetch();
    } catch {
      setError("无法确认是否解除授权。请刷新店铺状态。");
    } finally {
      setBusy(false);
    }
  }
  return <div className="space-y-3">
    {view.isPending ? <p role="status">正在读取 Google 授权…</p> : view.isError ? <p role="alert">暂时无法读取 Google 授权。</p> :
      <p role="status">{view.data.status==="connected" ? "已连接 Google Merchant Center "+view.data.accountId :
        view.data.status==="expired" ? "Google 授权已过期，请重新连接。" : "尚未连接 Google Merchant Center。"}</p>}
    {canWrite && !view.isError && <Button disabled={busy || view.isPending} onClick={() => void begin()}>{view.data?.status==="connected"?"重新授权":"连接 Google"}</Button>}
    {canWrite && view.data?.status==="connected" && <Button variant="outline" disabled={busy} onClick={() => void disconnect()}>解除授权</Button>}
    {error && <p role="alert">{error}</p>}
  </div>;
}

export function GoogleAuthorizationCallback() {
  const context = useWorkbenchContext();
  const userId = context.user?.id ?? "", organizationId = context.effectiveOrganization?.id ?? "";
  return <GoogleCallbackFlow key={userId+":"+organizationId} scope={{userId,organizationId}} />;
}

function GoogleCallbackFlow({scope}:{scope:GoogleScope}) {
  const [callback,setCallback]=useState<{pending:z.infer<typeof pendingSchema>;code:string}>();
  const [accounts,setAccounts]=useState<GoogleResponse<"accounts">>();
  const [sources,setSources]=useState<GoogleResponse<"sources">>();
  const [account,setAccount]=useState("");
  const [source,setSource]=useState("");
  const [language,setLanguage]=useState("");
  const [label,setLabel]=useState("");
  const [dedicated,setDedicated]=useState(false);
  const [connected,setConnected]=useState(false);
  const [busy,setBusy]=useState(false);
  const [error,setError]=useState("");
  useEffect(()=>{
    if(!scope.organizationId||!scope.userId)return;
    let current = true;
    void Promise.resolve().then(()=>{
      if(!current)return;
      const query=new URLSearchParams(window.location.search);
      window.history.replaceState(null,"",window.location.pathname);
      try {
        const pending=pendingSchema.parse(JSON.parse(sessionStorage.getItem(pendingKey)??"null"));
        const code=query.get("code");
        if(pending.userId!==scope.userId||pending.organizationId!==scope.organizationId||
          Date.parse(pending.expiresAt)<=Date.now()||query.getAll("state").length!==1||query.get("state")!==pending.state||
          query.getAll("code").length!==1||!code)throw Error("invalid callback");
        setCallback({pending,code});
      } catch {
        setError("这次 Google 授权已失效，请返回店铺重新连接。");
      }
    });
    return ()=>{current=false;};
  },[scope.organizationId,scope.userId]);
  async function perform(work:()=>Promise<void>,failure:string) {
    setBusy(true);setError("");
    try {await work();} catch {setError(failure);}
    finally {setBusy(false);}
  }
  async function loadAccounts(next?:string) {
    if(!callback)return;
    const {pending,code}=callback;
    await perform(async()=>{
      if(!accounts){
        await googleRequest(scope,"complete",pending.storeId,{attemptId:pending.attemptId,state:pending.state,code});
        sessionStorage.removeItem(pendingKey);
        setAccounts({Accounts:[],NextPageToken:""});
      }
      const result=await googleRequest(scope,"accounts",pending.storeId,{attemptId:pending.attemptId,page:next});
      setAccounts(prior=>next&&prior?{...result,Accounts:[...prior.Accounts,...result.Accounts]}:result);
    },"无法确认授权或读取 Google 账号，请核对当前企业后重试。");
  }
  async function loadSources(next?:string) {
    if(!callback||!account)return;
    const {pending}=callback;
    await perform(async()=>{
      const result=await googleRequest(scope,"sources",pending.storeId,{attemptId:pending.attemptId,accountId:account,page:next});
      setSources(prior=>next&&prior?{...result,Sources:[...prior.Sources,...result.Sources]}:result);
    },"无法读取商品数据源，请检查该账号的访问权限。");
  }
  async function bind() {
    if(!callback||!source||!dedicated)return;
    const {pending}=callback;
    await perform(async()=>{
      await googleRequest(scope,"bind",pending.storeId,{attemptId:pending.attemptId,accountId:account,dataSourceId:source,
        contentLanguage:language,feedLabel:label,dedicatedSourceConfirmed:true});
      setConnected(true);
    },"无法连接。请确认你有此账号的权限，并选择专供硕米使用且仅启用免费商品展示的 API 数据源。");
  }
  return <section className="mx-auto max-w-xl space-y-4 p-6">
    <h1 className="text-xl font-semibold">连接 Google Merchant Center</h1>
    {error && <p role="alert">{error}</p>}
    {connected && callback ? <p role="status">Google 店铺已连接。 <Link className="underline" href={"/workbench/stores/"+callback.pending.storeId+"#authorization"}>返回店铺</Link></p> :
      callback && <>
        {!accounts || !accounts.Accounts.length && !account ? <Button disabled={busy} onClick={()=>void loadAccounts()}>读取 Google 账号</Button> : null}
        {!!accounts?.Accounts.length && <div className="space-y-2"><label htmlFor="google-account">Merchant Center 账号</label>
          <Select id="google-account" value={account} disabled={busy} onChange={event=>{setAccount(event.target.value);setSources(undefined);setSource("");setDedicated(false);}}>
            <option value="">选择账号</option>{accounts.Accounts.map(row=><option value={row.ID} key={row.ID}>{row.DisplayName||row.ID}</option>)}
          </Select><Button onClick={()=>void loadSources()} disabled={busy||!account}>查看商品数据源</Button>
          {accounts.NextPageToken&&<Button variant="outline" disabled={busy} onClick={()=>void loadAccounts(accounts.NextPageToken)}>更多账号</Button>}
        </div>}
        {sources && <form className="space-y-3" onSubmit={event=>{event.preventDefault();void bind();}}>
          <label htmlFor="google-source">商品数据源</label>
          <Select id="google-source" value={source} onChange={event=>{
            const row=sources.Sources.find(item=>item.ID===event.target.value);setSource(event.target.value);
            setLanguage(row?.ContentLanguage||"");setLabel(row?.FeedLabel||"");setDedicated(false);
          }}>
            <option value="">选择数据源</option>{sources.Sources.map(row=><option value={row.ID} key={row.ID}>{row.DisplayName||row.ID}</option>)}
          </Select>
          {sources.NextPageToken&&<Button type="button" variant="outline" disabled={busy} onClick={()=>void loadSources(sources.NextPageToken)}>更多数据源</Button>}
          {source&&<><label htmlFor="google-language">商品语言代码</label><Input id="google-language" value={language} onChange={e=>setLanguage(e.target.value)} required />
            <label htmlFor="google-feed">数据源标签</label><Input id="google-feed" value={label} onChange={e=>setLabel(e.target.value)} required />
            <label className="flex gap-2"><input type="checkbox" checked={dedicated} onChange={event=>setDedicated(event.target.checked)}/>此数据源专供硕米上传商品。</label>
            <Button type="submit" disabled={busy||!dedicated}>确认连接</Button></>}
        </form>}
      </>}
    {!connected && <Link className="block underline" href={callback?"/workbench/stores/"+callback.pending.storeId:"/workbench/stores"}>返回店铺</Link>}
  </section>;
}
