"use client";
import { useCallback, useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
	getStoreConnection,
 refreshWalmartConnection,
	getStoreApplications,
	installStoreToken,
  beginStoreConnection,
  queryStoreConnection,
  disconnectStoreConnection,
  rememberStoreAuthorization,
  StoreConnectionError,
} from "@/lib/api/store-connection";
import type { MemberResourceScope } from "@/lib/api/member-resources";
import type { WorkbenchStore } from "@/lib/api/workbench-stores";
import { StoreServiceActions } from "./store-service-actions";
const labels = {
  connected: "已连接",
  disconnected: "未连接",
  expired: "授权已失效",
  unavailable: "连接服务未配置或暂不可用",
};
const applicationLabels={self_operated:"自运营",semi_managed:"半托管",fully_managed:"全托管"} as const;
export function StoreOfficialConnection({
  store,
  scope,
  canWrite,
  administrator,
  onChanged,
}: {
  store: WorkbenchStore;
  scope: MemberResourceScope;
  canWrite: boolean;
  administrator: boolean;
  onChanged: () => Promise<unknown>;
}) {
	const platformLabel=store.platform==="walmart"?"Walmart 美国站":store.platform==="tiktokshop"?"TikTok Shop":store.platform==="temu"?"TEMU":"SHEIN";
	const scopeKey=JSON.stringify([scope.expectedUserId,scope.expectedOrganizationId,store.id,canWrite,administrator]);
  const view = useQuery({
    queryKey: [
      "workbench",
      scope.expectedUserId,
      scope.expectedOrganizationId,
      "official-connection",
      store.id,
      store.version,
    ],
    queryFn: ({ signal }) => getStoreConnection(scope, store.id, signal),
    gcTime: 0,
    staleTime: 0,
    retry: false,
  });
	const [message, setMessage] = useState("");
	const applications=useQuery({
		queryKey:["workbench",scope.expectedUserId,scope.expectedOrganizationId,"official-applications",store.id],
		queryFn:({signal})=>getStoreApplications(scope,store.id,signal),gcTime:0,staleTime:0,retry:false,
	});
	const [selectedApp,setSelectedApp]=useState("");
	const choices=!applications.isFetching && !applications.isError ? applications.data??[]:[];
	const choiceKey=(choice:typeof choices[number])=>choice.identityPartition?`${choice.identityPartition}:${choice.appId}`:choice.appId;
	const chosen=choices.find(choice=>choiceKey(choice)===selectedApp) ?? (choices.length===1 ? choices[0]:undefined);
  const [workingFor, setWorkingFor] = useState<string|null>(null);
  const working=workingFor===scopeKey;
  const tokenInput=useRef<HTMLInputElement>(null);
  const captureTokenInput=useCallback((node:HTMLInputElement|null)=>{
    if(tokenInput.current && tokenInput.current!==node)tokenInput.current.value="";
    tokenInput.current=node;
  },[]);
  const active = useRef<{ alive: boolean; busy:boolean; controller:AbortController } | null>(null);
  useEffect(() => {
    const current = { alive: true,busy:false,controller:new AbortController() };
    active.current = current;
    return () => {
      current.alive = false;
      current.controller.abort();
    };
  }, [
    scope.expectedUserId,
    scope.expectedOrganizationId,
    store.id,
    canWrite,
    administrator,
  ]);
  const data = !view.isFetching && !view.isError ? view.data : undefined;
  async function run(action: "begin" | "query" | "disconnect" | "install" | "refresh") {
    const current = active.current;
    if (!current?.alive || current.busy) return;
    current.busy = true;
    setWorkingFor(scopeKey);
    setMessage("");
    try {
			if (action === "begin") {
				if(!chosen)return;
        const key = crypto.randomUUID();
        const result = await beginStoreConnection(
          scope,
          store.id,
          store.version,
					key,
					chosen.appId,
					chosen.identityPartition,
        );
        if (!current.alive) return;
        if(result.authorizationMethod==="manual_token") {
          await Promise.all([view.refetch(),onChanged()]);
          if(current.alive)setMessage("请粘贴该店铺在所选应用分区生成的授权 token。");
          return;
        }
        rememberStoreAuthorization({
          ...scope,
          storeId: store.id,
          attemptId: result.attemptId,
          expiresAt: result.expiresAt,
          ...((store.platform==="tiktokshop"||store.platform==="walmart")?{appId:chosen.appId}:{}),
        },store.platform==="walmart"?"walmart":store.platform==="tiktokshop"?"tiktokshop":"shein");
        window.location.assign(result.authorizationUrl);
        return;
      }
      if(action === "install" && data?.attemptId && data.authorizationMethod==="manual_token" && data.state==="awaiting_consent") {
        const accessToken=tokenInput.current?.value??"";
        if(tokenInput.current)tokenInput.current.value="";
        if(!accessToken){setMessage("请输入官方授权 token。");return;}
        await installStoreToken(scope,store.id,{attemptId:data.attemptId,accessToken},current.controller.signal);
        if(current.alive)setMessage("官方身份已核验。商品发布权限按实际授权另行判断。");
      } else if (action === "query" && data?.attemptId) {
        await queryStoreConnection(scope, store.id, data.attemptId);
      } else if(action === "refresh" && data?.walmart?.generation){
 await refreshWalmartConnection(scope,store.id,data.walmart.generation);
 } else if (action === "disconnect") {
        await disconnectStoreConnection(
          scope,
          store.id,
          store.version,
          crypto.randomUUID(),
        );
        if (!current.alive) return;
        setMessage(`本地连接已断开。${platformLabel} 端授权请在官方授权管理中自行撤销。`);
      }
      if (!current.alive) return;
      await Promise.all([view.refetch(), onChanged()]);
    } catch (error) {
      if (!current.alive) return;
      setMessage(
        error instanceof StoreConnectionError &&
          error.code === "STORE_OFFICIAL_SETUP_UNAVAILABLE"
          ? `尚未配置获批的 ${platformLabel} 开发者应用，暂不能真实连接。`
          : error instanceof StoreConnectionError &&
              error.code === "STORE_AUTHORIZATION_OUTCOME_UNKNOWN"
            ? "授权交换结果未知，不能再次交换原临时凭据。请刷新后重新发起官方授权。"
            : error instanceof StoreConnectionError && error.unknown
              ? "连接操作结果未确认，请先刷新连接状态。不会自动重复发起授权。"
              : "连接操作未完成，请刷新状态并确认当前权限。",
      );
      await view.refetch();
    } finally {
      current.busy = false;
      if (current.alive) setWorkingFor(null);
    }
  }
  return (
    <section className="mt-4 space-y-3" aria-label={`${platformLabel} 官方连接`}>
      <h2 className="font-semibold">{platformLabel} 官方连接</h2>
      <p role={view.isError ? "alert" : "status"}>
        {view.isPending || view.isFetching
          ? "正在读取官方连接…"
          : data
            ? labels[data.connectionStatus]
            : "本次未取得官方连接状态。"}
      </p>
      {data?.observedAt ? (
        <p className="text-sm text-muted-foreground">
          最近官方检查：{data.observedAt}
        </p>
      ) : null}
      <p className="text-sm text-muted-foreground">
        {store.platform==="walmart"?"前往 Walmart 美国站官方授权页，由商家主账号授权并核实 Partner ID。":store.platform==="tiktokshop" ? "前往 TikTok Shop 美国官方授权页，由店铺主账号确认。请使用美国跨境店铺；绑定时核验 Seller Center 编号。" : store.platform==="temu" ? "由店铺主账号在 TEMU 官方授权管理中生成 token，选择其原应用分区后连接。经营模式由官方核验。" : "前往 SHEIN 官方授权页，由店铺主账号确认。"}连接只证明当前授权有效，发布权限另行判断。
      </p>
			{data?.mode?<p className="text-sm">最近核验模式：{data.mode==="local"?"本土":applicationLabels[data.mode]}{data.credentialExpiresAt?`；授权到期：${data.credentialExpiresAt}`:""}</p>:null}
			{canWrite &&
			store.recordStatus !== "deleting" ? (
				<><label className="grid gap-1 text-sm">官方应用类型
					<select aria-label="官方应用类型" value={chosen?choiceKey(chosen):""} disabled={working || applications.isFetching} onChange={event=>setSelectedApp(event.target.value)} className="rounded-lg border bg-background p-2">
						<option value="">选择已配置的官方应用</option>
						{choices.map(choice=><option key={choiceKey(choice)} value={choiceKey(choice)}>{choice.platform==="walmart"?"Walmart 美国站":choice.platform==="tiktokshop"?"美国跨境":choice.platform==="temu"?`${choice.identityPartition} · ${choice.identityPartition==="CN"?"全托管 / 半托管":"本土"}`:choice.type?applicationLabels[choice.type]:""} · {choice.appId}</option>)}
					</select>
				</label>
				<p className="text-sm text-muted-foreground">{applications.isFetching ? "正在读取应用配置…": applications.isError ? "未取得应用配置，请刷新后再连接。":store.platform==="walmart"?(choices.length?"Walmart 美国站授权，商品刊登按实际获批权限检查。":"尚未配置获批的 Walmart 开发者应用。"):store.platform==="tiktokshop" ? (choices.length?"美国跨境店铺授权，商品发布按实际获批权限检查。":"尚未配置获批的 TikTok Shop 开发者应用。") : store.platform==="temu"?(choices.length?"按原官方应用分区选择，不能按店铺所在国家猜测。":"尚未配置 TEMU 官方应用。"):Object.entries(applicationLabels).filter(([type])=>!choices.some(choice=>choice.type===type)).map(([,label])=>`${label}未配置`).join("；")}</p>
				{data?.authorizationMethod==="manual_token" && data.state==="awaiting_consent" ? <div className="space-y-2"><label className="grid gap-1 text-sm">TEMU 授权 token<Input key={scopeKey} ref={captureTokenInput} type="password" autoComplete="off" spellCheck={false} maxLength={4096} disabled={working}/></label><Button disabled={working} onClick={()=>void run("install")}>验证并连接</Button></div>:null}
				<div className="flex flex-wrap gap-2">
 {store.platform==="walmart"&&data?.walmart?.generation&&data.state!=="refresh_unknown"?<Button variant="outline" disabled={working} onClick={()=>void run("refresh")}>刷新官方授权</Button>:null}
					<Button disabled={working || !data || !chosen} onClick={() => void run("begin")}>
            {store.platform==="temu"?(data?.attemptId?"重新连接":"开始连接"):(data?.attemptId ? "重新官方授权" : "前往官方授权")}
          </Button>
          {data?.state === "credential_received" ||
          data?.state === "verified" ? (
            <Button
              variant="outline"
              disabled={working}
              onClick={() => void run("query")}
            >
              核验原连接
            </Button>
          ) : null}
          {data?.attemptId && data.state !== "failed" ? (
            <Button
              variant="outline"
              disabled={working}
              onClick={() => void run("disconnect")}
            >
              断开本地连接
            </Button>
          ) : null}
          <Button
            variant="outline"
            disabled={working}
            onClick={() => void view.refetch()}
          >
            刷新连接状态
          </Button>
				</div></>
      ) : null}
      {message ? <p role="alert">{message}</p> : null}
      <StoreServiceActions
        store={store}
        scope={scope}
        canWrite={canWrite}
        administrator={administrator}
        connected={data?.connectionStatus === "connected"}
        onChanged={onChanged}
      />
    </section>
  );
}
