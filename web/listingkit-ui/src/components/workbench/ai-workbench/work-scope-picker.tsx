"use client";

import { useInfiniteQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { requestAIWorkbench, type AIScope } from "@/lib/api/ai-workbench";
import type { WorkScopeOption, WorkScopeSelection } from "@/lib/contracts/ai-workbench";
import styles from "./work-scope-picker.module.css";

export function scopeLabel(value: Omit<WorkScopeOption,"confirmation"> | undefined): string {
  if (!value) return "未记录工作范围";
  if (value.scope.kind === "general") return "通用业务";
  if (value.scope.kind === "store") return value.storeName ?? "店铺";
  return [value.projectName ?? "项目", value.storeName].filter(Boolean).join(" · ");
}

export function scopeSelection(value: WorkScopeOption): WorkScopeSelection {
  if (value.scope.kind === "project" && value.scope.projectId) return {kind:"project",projectId:value.scope.projectId};
  if (value.scope.kind === "store" && value.scope.storeId) return {kind:"store",storeId:value.scope.storeId};
  return {kind:"general"};
}

export function WorkScopePicker({scope, authorizationKey, disabled, selected, onSelect, initialKind = "general", fixedKind=false}: {
  scope:AIScope; authorizationKey:string; disabled?:boolean; selected?:WorkScopeOption;
  onSelect:(option:WorkScopeOption | undefined)=>void; fixedKind?:boolean; initialKind?:"general"|"project"|"store";
}) {
  const [kind,setKind] = useState(initialKind);
  const options = useInfiniteQuery({
    queryKey:["ai-work-scope",scope.userId,scope.organizationId,authorizationKey,kind,fixedKind],
    initialPageParam:"",
    queryFn:({signal,pageParam}) => requestAIWorkbench({
      route:fixedKind ? "task-scope-options" : "scope-options",method:"GET",scope,signal,path:(fixedKind ? "tasks/scopes?" : "chat/conversations/scopes?") + new URLSearchParams({kind,...(pageParam?{after:pageParam}:{})}),
    }),
    getNextPageParam:page=>page.next || undefined,
    retry:false, gcTime:0,
  });
  const rows = options.data?.pages.flatMap(page=>page.options) ?? [];
  return <div className={`${styles.picker} ${fixedKind ? styles.fixedPicker : ""}`}>
    {!fixedKind ? <label>范围类型 <select value={kind} disabled={disabled} onChange={event=>{
      setKind(event.target.value as typeof kind); onSelect(undefined);
    }}><option value="general">通用业务</option><option value="project">项目</option><option value="store">店铺</option></select></label> : null}
    <label>{fixedKind ? kind === "project" ? "项目" : kind === "store" ? "店铺" : "工作范围" : "工作范围"} <select value={selected?.confirmation ?? ""} disabled={disabled || options.isPending || options.isError}
      onChange={event=>onSelect(rows.find(row=>row.confirmation===event.target.value))}>
      <option value="">{options.isPending ? "正在读取…" : "请选择"}</option>
      {rows.map(row=><option key={row.confirmation} value={row.confirmation}>{scopeLabel(row)}</option>)}
    </select></label>
    {options.isError ? <><p role="status">当前范围不可用。</p><Button type="button" variant="outline" disabled={disabled} onClick={()=>void options.refetch()}>重新读取</Button></> : null}
    {options.data?.pages.some(page=>page.incomplete) ? <p role="status">部分范围不可用。</p> : null}
    {options.hasNextPage ? <Button type="button" variant="outline" disabled={disabled || options.isFetching} onClick={()=>void options.fetchNextPage()}>加载更多范围</Button> : null}
  </div>;
}
