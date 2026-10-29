'use client';
import { useEffect,useLayoutEffect,useMemo,useRef,useState } from 'react';
import type { z } from 'zod';
import { useWorkbenchContext } from '@/components/providers/workbench-context-provider';
import { parseCaptureEntry,type CaptureEntry } from './1688/capture-handoff';
import { CaptureDraftClient,captureHandoffExpiredMessage,isCaptureHandoffExpired,type CaptureDraft } from './capture-draft';
export function useCaptureDraft<T>(platform:'amazon'|'1688',schema:z.ZodType<T>) {
  const context=useWorkbenchContext(),current=useRef(context);
  const scope=useMemo(()=>({userId:context.user?.id??'',organizationId:context.effectiveOrganization?.id??'',memberId:context.effectiveMemberId??''}),[context.user?.id,context.effectiveOrganization?.id,context.effectiveMemberId]);
  const ready=!context.isLoading&&!context.isSwitching&&!context.error&&!context.blockingError&&!context.selectionRequired&&!!scope.userId&&!!scope.organizationId&&!!scope.memberId;
  const scopeKey=JSON.stringify(scope),live=useRef('');
  useLayoutEffect(()=>{current.current=context;live.current=ready?scopeKey:'';});
  const [entry,setEntry]=useState<CaptureEntry|null>(null);
  const [loaded,setLoaded]=useState<{scopeKey:string;draft?:CaptureDraft<T>;client?:CaptureDraftClient;error?:string}>();
  useEffect(()=>{let mounted=true;void Promise.resolve().then(()=>{if(mounted)setEntry(parseCaptureEntry(window.location.href,platform));});return()=>{mounted=false;};},[platform]);
  useEffect(()=>{
    if(!ready||entry?.kind!=='handoff')return;
    const abort=new AbortController(),expected=scopeKey;
    void(async()=>{
      try{
        const refreshed=await current.current.retry();
        if(!refreshed||refreshed.selectionRequired||refreshed.user.id!==scope.userId||refreshed.effectiveOrganizationId!==scope.organizationId||refreshed.effectiveMemberId!==scope.memberId||live.current!==expected||abort.signal.aborted)throw Error('IDENTITY_CONTEXT_CHANGED');
        const client=new CaptureDraftClient(entry,scope,abort.signal,()=>live.current===expected);
        const draft=await client.read(schema);
        if(!abort.signal.aborted&&live.current===expected)setLoaded({scopeKey:expected,draft,client});
      }catch(error){
        if(!abort.signal.aborted&&live.current===expected)setLoaded({scopeKey:expected,error:isCaptureHandoffExpired(error)?captureHandoffExpiredMessage:'本机资料尚未读取，请从原插件恢复资料；已保全内容不会因交接失效而删除。'});
      }
    })();
    return()=>abort.abort();
  // The live context ref avoids treating a newly allocated retry function as a
  // reason to reread a body. Actual identity/switch changes always invalidate it.
  },[ready,scopeKey,entry,platform,schema,scope]);
  const valid=ready&&loaded?.scopeKey===scopeKey;
  return {context,scope,ready,entry,draft:valid?loaded.draft:undefined,client:valid?loaded.client:undefined,error:valid?loaded.error:undefined,
    stillCurrent:()=>live.current===scopeKey&&ready};
}
