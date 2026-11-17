"use client";

import {useEffect,useRef,useState} from "react";
import {Button} from "@/components/ui/button";
import {ModelRouteError,modelRouteRequest,type ModelRouteScope} from "@/lib/api/model-route-pricing";
import {matchesModelQuery,modelQueryResult,type ModelQueryInput,type ModelQueryResult} from "@/lib/contracts/channel-credentials";

const statusText:Record<ModelQueryResult["result"]["status"],string>={
 found:"Key 通过查询验证 · 已查到指定模型",
 "invalid-key":"OpenAI 拒绝了该 Key，请核对或更新后重试。",
 "authentication-rejected":"OpenAI 未接受本次查询，请核对 Key 和项目设置。",
 "read-forbidden":"无权查询模型资料，请核对 OpenAI 项目和 Key 的读取权限。",
 "region-restricted":"当前地区无法查询 OpenAI 模型资料。",
 rejected:"OpenAI 拒绝查询。",
 "not-found":"未查到指定模型，请核对型号和项目权限。",
 "rate-limited":"请求过多，请稍后重试。",
 timeout:"查询超时，本次结果无法确认。",
 unconfirmed:"本次查询无法确认。",
 unavailable:"查询暂不可用，请稍后重试。",
};

export function ChannelModelQuery({scope,connectionId,input,disabled}:{
 scope:ModelRouteScope;connectionId:string;input:ModelQueryInput;disabled:boolean;
}){
 const [pending,setPending]=useState(false);
 const [failure,setFailure]=useState("");
 const [result,setResult]=useState<ModelQueryResult["result"]|null>(null);
 const [waiting,setWaiting]=useState(false);
 const active=useRef<AbortController|null>(null);
 useEffect(()=>()=>active.current?.abort(),[]);
 const retryAfter=result?.retryAfter;
 useEffect(()=>{
  if(!retryAfter)return;
  const until=Date.parse(retryAfter);
  let timer:number;
  function schedule(){
   const remaining=until-Date.now();
   if(remaining<=0){setWaiting(false);return}
   timer=window.setTimeout(schedule,Math.min(remaining,2_147_483_647));
  }
  schedule();
  return()=>window.clearTimeout(timer);
 },[retryAfter]);

 async function check(){
  if(disabled||active.current||waiting)return;
  const request=new AbortController();
  active.current=request;
  setPending(true);
  setFailure("");
  setResult(null);
  try{
   const reply=await modelRouteRequest(scope,"/connections/"+encodeURIComponent(connectionId)+"/model-query",modelQueryResult,{
    method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(input),signal:request.signal,
   });
   if(request.signal.aborted)return;
   if(!matchesModelQuery(reply,scope,connectionId,input))throw new ModelRouteError("UNAVAILABLE");
   setWaiting(!!reply.result.retryAfter&&Date.parse(reply.result.retryAfter)>Date.now());
   setResult(reply.result);
  }catch(error){
   if(request.signal.aborted)return;
   const code=error instanceof ModelRouteError?error.code:"UNAVAILABLE";
   setFailure(code==="CONFLICT"?"配置已变更，请重新读取渠道状态后检查。":
    code==="FORBIDDEN"||code==="PERMISSION_DENIED"?"当前无权检查，请重新确认企业权限。":
    "查询暂不可用，请稍后重试。");
  }finally{
   active.current=null;
   if(!request.signal.aborted)setPending(false);
  }
 }

 return <div>
  <Button type="button" variant="outline" onClick={()=>void check()} disabled={disabled||pending||waiting} aria-busy={pending}>
   {pending?"正在检查…":"检查模型查询权限"}
  </Button>
  <p>本次仅查询模型资料，不消耗硕米 AI 点数。</p>
  <div role="status">
   {failure&&<p>{failure}</p>}
   {result&&<>
    <p>{statusText[result.status]}</p>
    <p>模型调用权限未验证</p>
    <p>{result.model} · {new Date(result.checkedAt).toLocaleString("zh-CN")}</p>
    {waiting&&retryAfter&&<p>可在 {new Date(retryAfter).toLocaleString("zh-CN")} 后重试。</p>}
   </>}
  </div>
 </div>;
}
