"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import type { z } from "zod";
import { observationCapabilitiesSchema, observationID, observationRequest, type ObservationKind } from "@/lib/api/store-observations";
import { catalogItemSchema, storeCenterRequest, type StoreCenterScope } from "@/lib/api/store-center";
import type { WorkbenchStore } from "@/lib/api/workbench-stores";
import styles from "./store-catalog.module.css";

type CapabilityView = { data?: z.infer<typeof observationCapabilitiesSchema>; isPending?: boolean; isFetching?: boolean; isError?: boolean };
type ReadinessStore = Pick<WorkbenchStore, "id" | "name" | "connectionStatus" | "serviceStatus">;

export function useStoreReadiness(scope: StoreCenterScope, permissions: string[], admitted: boolean, kind: ObservationKind = "products") {
  return useQuery({
    queryKey: ["store-readiness", scope.organizationId, scope.userId, kind, permissions],
    queryFn: ({ signal }) => observationRequest(`${kind}/capabilities`, scope, observationCapabilitiesSchema, signal),
    enabled: Boolean(admitted && scope.organizationId && scope.userId && permissions.includes(`workbench.store.${kind}.read`)),
    retry: false, gcTime: 0,
  });
}

export function StoreReadinessNotice({ caps, permissions, kind = "products", store, compact = false }: {
  caps: CapabilityView; permissions: string[]; kind?: ObservationKind; store?: ReadinessStore; compact?: boolean;
}) {
  const data = !caps.isError && !caps.isPending && !caps.isFetching ? caps.data : undefined;
  const canRead = permissions.includes(`workbench.store.${kind}.read`), canSync = permissions.includes(`workbench.store.${kind}.sync`);
  const canManage = permissions.includes("workbench.store.update"), canViewStore = permissions.includes("workbench.store.read");
  const label = kind === "products" ? "商品" : "订单";
  const runtime = data?.runtimeReady === true ? "已就绪" : data?.runtimeReady === false ? "暂不可用" : "待确认";
  const provider = data?.providerConfigured === true ? "已配置" : data?.providerConfigured === false ? "待配置" : "待确认";
  const connection = { connected: "已连接", disconnected: "未连接", expired: "授权已失效", unavailable: "待确认" };
  const service = { active: "有效", pending_activation: "待激活", expired: "已到期", suspended: "已暂停" };
  const next = !canRead ? `请联系企业配置管理员核对${label}读取权限；店铺可见不代表可以读取或同步。`
    : caps.isError ? "本次未取得平台读取状态，请刷新后核对。"
    : data?.providerConfigured === false ? "请由配置管理员完成官方应用和平台读取配置，再核对本店授权。"
    : store && (store.connectionStatus !== "connected" || store.serviceStatus !== "active") ? "请在本店授权信息核对官方连接和店铺服务。"
    : !canSync ? "请联系企业配置管理员核对同步权限；读取权限不包含同步权限。"
    : "平台能力与本店资格分别校验，同步以执行入口的检查为准。";
  return <section className={styles.readiness} data-compact={compact || undefined} aria-label={compact ? `${store?.name ?? "店铺"}接入条件` : `${label}接入准备`} role="status">
    {!compact ? <><h2>接入准备</h2><div className={styles.readinessFacts}>
      <span>内部运行：{runtime}</span><span>平台读取：{provider}</span>
      <span>{label}读取权限：{canRead ? "已具备" : "未开放"}</span><span>同步权限：{canSync ? "已具备" : "未开放"}</span>
    </div></> : null}
    {store ? <p>本店连接：{connection[store.connectionStatus] ?? "待确认"} · 店铺服务：{store.serviceStatus ? service[store.serviceStatus] : "尚未建立"}</p> : null}
    <div className={styles.readinessNext}><p>下一步：{!canManage&&!next.includes("请联系企业配置管理员")?"请联系企业配置管理员。":""}{next}</p>
      {canViewStore && (!store || canManage) ? <Link prefetch={false} href={store && observationID.safeParse(store.id).success ? `/workbench/stores/${store.id}#authorization` : "/workbench/stores"}>{store ? "查看本店授权信息" : "前往我的店铺查看接入条件"} →</Link> : null}
    </div>
  </section>;
}

// Resolve the selection through the current scoped Store API before offering its detail link.
export function SelectedStoreReadiness({ storeId, scope, permissions, caps, kind }: {
  storeId: string; scope: StoreCenterScope; permissions: string[]; caps: CapabilityView; kind: ObservationKind;
}) {
  const selected = useQuery({
    queryKey: ["store-readiness-selection", scope.organizationId, scope.userId, storeId, permissions],
    queryFn: ({ signal }) => storeCenterRequest(`summaries/${storeId}`, scope, catalogItemSchema, { signal }),
    enabled: observationID.safeParse(storeId).success && permissions.includes("workbench.store.read"),
    retry: false, gcTime: 0,
  });
  const store = !selected.isError && !selected.isPending && !selected.isFetching && selected.data?.id === storeId ? selected.data : undefined;
  return <StoreReadinessNotice caps={caps} permissions={permissions} kind={kind} store={store} />;
}
