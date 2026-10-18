"use client";
import Link from "next/link";
import {Button} from "@/components/ui/button";
import {configurationError} from "../agents/agent-configuration-hooks";
import {nativeContextLabel,nativePlatformLabel,type NativeRunRecord} from "./native-run-records";
import type {NativeRuns} from "./use-native-runs";

export function NativeRunDetail({record,locked}:{record:NativeRunRecord;locked:boolean}){
 return <section className="space-y-4 p-4"><h3 className="font-semibold">{record.title}</h3><p>{record.label}</p>
  <dl className="space-y-2 break-words"><dt>业务上下文</dt><dd>{nativeContextLabel(record.contextKind)} · {record.contextId}</dd>
   {record.productKey?<><dt>商品</dt><dd>{record.productKey}</dd></>:null}<dt>目标范围</dt><dd>{nativePlatformLabel(record.targetPlatform)}</dd>
   <dt>发起时间</dt><dd>{new Date(record.at).toLocaleString()}</dd><dt>运行编号</dt><dd>{record.runId}</dd></dl>
  <p>这是原智能体的运行记录；建议是否采用、素材是否保存或商品是否发布，以原业务结果为准。</p>
  {!record.direct?<p>该上下文暂无直达入口。请在原智能体的任务历史中，按运行编号选择记录。</p>:null}
  <Button asChild variant="outline"><Link href={record.href} prefetch={false} aria-disabled={locked||undefined} tabIndex={locked?-1:undefined} onClick={event=>{if(locked)event.preventDefault();}}>打开原业务工作区</Link></Button>
 </section>;
}
export function NativeRunReadStatus({runs,locked}:{runs:NativeRuns;locked:boolean}){
 if(!runs.enabled)return null;
 return <section className="space-y-3 p-4" aria-label="智能体运行来源">
  {runs.catalog.isPending?<p role="status">正在读取可用任务来源…</p>:runs.catalog.isError?<p role="alert">任务来源读取失败：{configurationError(runs.catalog.error)} <Button variant="outline" disabled={locked||runs.catalog.isFetching} onClick={()=>void runs.catalog.refetch()}>重试任务来源</Button></p>:null}
  {runs.catalog.isSuccess&&!runs.catalog.isFetching&&!runs.sources.length?<p>当前没有可读取的智能体运行</p>:null}
  {runs.sources.map(source=><div key={source.entry.agent.agentId}>
   {source.loading?<p role="status">正在读取{source.entry.name}记录…</p>:source.error?<p role="alert">{source.entry.name}：{configurationError(source.error)} <Button variant="outline" disabled={locked} onClick={()=>void source.retry()}>重试{source.entry.name}</Button></p>:source.repeated?<p role="alert">{source.entry.name}分页未能继续，请刷新来源后重试。</p>:source.next?<Button variant="outline" disabled={locked} onClick={source.more}>加载更多{source.entry.name}</Button>:null}
  </div>)}
  {runs.catalog.hasNextPage?<Button variant="outline" disabled={locked||runs.catalog.isFetching} onClick={()=>void runs.catalog.fetchNextPage()}>加载更多任务来源</Button>:null}
 </section>;
}
