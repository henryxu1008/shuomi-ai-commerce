"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { getCommercialOverview } from "@/lib/api/commercial";
import { listWorkbenchStores } from "@/lib/api/workbench-stores";
import { ConsoleState } from "../console/console-page";
import { EntitlementsOverview } from "../commercial/commercial-views";
import { EnterpriseResources } from "../commercial/enterprise-resources";
import { AccountShell } from "../account/account-shell";
import { MemberPointLimits } from "./member-point-limits";
import { MemberResources } from "./member-resources";
import { EnterpriseStorage } from "./enterprise-storage";
import styles from "./resources.module.css";

export function ResourcesPage() {
  const context = useWorkbenchContext();
  const [sequence, setSequence] = useState(0);
  const org = context.effectiveOrganization;
  const scope = JSON.stringify([context.user?.id, org?.id, context.roles]);
  const readScope = JSON.stringify([context.user?.id, org?.id, context.roles, context.permissions]);
  const valid = context.user && org && !context.isLoading && !context.isSwitching && !context.selectionRequired && !context.error && !context.blockingError;
  return <AccountShell pathname="/workbench/account/organization/resources" title="资源与额度" description="查看企业余额、存储容量和成员额度。"
    actions={valid ? <Button variant="outline" onClick={() => setSequence(value => value + 1)}>刷新资源</Button> : undefined}>
    {context.isLoading || context.isSwitching ? <ConsoleState kind="loading" title="正在确认当前企业">确认后将显示该企业的资源。</ConsoleState>
      : !valid || !context.user ? <ConsoleState kind="error" title="暂时无法查看企业资源">请确认已登录并选择了企业。<Button variant="outline" onClick={() => void context.retry()}>重试</Button></ConsoleState>
      : <ScopedResources key={scope} scope={readScope} userId={context.user.id} organizationId={org.id} sequence={sequence} storageAvailable={context.storageCapacityAvailable===true} canPurchase={context.permissions.includes("workbench.commercial.purchase")} canManage={context.permissions.includes("workbench.organization_member.manage")} />}
  </AccountShell>;
}

function ScopedResources({ scope, userId, organizationId, sequence, canManage, canPurchase, storageAvailable }: { scope: string; userId: string; organizationId: string; sequence: number; canManage: boolean; canPurchase: boolean; storageAvailable: boolean }) {
  return <div className={styles.stack}>
    <section className={styles.section} aria-label="企业资源余额">
      <EnterpriseResources userId={userId} organizationId={organizationId} scope={scope} sequence={sequence} memberResourcesAnchor="member-allowances" showObservation={false} />
      {storageAvailable && <EnterpriseStorage userId={userId} organizationId={organizationId} scope={scope} sequence={sequence} canPurchase={canPurchase} />}
    </section>
    <section className={styles.section} aria-labelledby="member-allowances">
      <h2 id="member-allowances" tabIndex={-1} className={styles.sectionTitle}>成员资源与额度</h2>
      <MemberResources userId={userId} organizationId={organizationId} sequence={sequence} canManage={canManage} />
      <MemberPointLimits userId={userId} organizationId={organizationId} sequence={sequence} canManage={canManage} />
    </section>
    <section className={styles.section} aria-labelledby="store-and-source">
      <h2 id="store-and-source" className={styles.sectionTitle}>店铺服务与源账号</h2>
      <div className={styles.grid}>
        <StoreResources scope={scope} userId={userId} organizationId={organizationId} sequence={sequence} />
        <Card role="region" aria-label="源账号资源" className={styles.entryPanel}>
          <div><h3>源账号</h3><p>可登记和管理商品来源账号。采集公开商品无需登记源账号。</p></div>
          <Button asChild variant="outline"><Link href="/workbench/account/organization/resources/source-accounts" prefetch={false}>管理源账号</Link></Button>
        </Card>
      </div>
      <ResourceRequest key={sequence} scope={scope} organizationId={organizationId} sequence={sequence} />
    </section>
  </div>;
}

function StoreResources({ scope, userId, organizationId, sequence }: { scope: string; userId: string; organizationId: string; sequence: number }) {
  const stores = useQuery({ queryKey: ["workbench", userId, organizationId, "account-store-count", scope, sequence], queryFn: ({ signal }) => listWorkbenchStores({ page: 1, pageSize: 1 }, organizationId, signal), gcTime: 0, staleTime: 0, retry: false });
  return <Card role="region" aria-label="店铺资源" className={styles.entryPanel}>
    <div>
      <h3>店铺资源</h3>
      {stores.isPending || stores.isFetching ? <p role="status">正在读取店铺记录数…</p> :
        stores.isError || !stores.data ? <p role="alert">本次未取得店铺记录数，请确认当前企业后重试。</p> :
          <dl><dt>店铺记录数</dt><dd>{stores.data.pagination.total} 家</dd></dl>}
      <p>仅统计你有权查看的店铺。服务期限与平台连接状态可在「我的店铺」查看。</p>
    </div>
    <Button asChild variant="outline"><Link href="/workbench/stores" prefetch={false}>管理店铺</Link></Button>
  </Card>;
}

function ResourceRequest({ scope, organizationId, sequence }: { scope: string; organizationId: string; sequence: number }) {
  const response = useQuery({ queryKey: ["workbench", organizationId, "account-resources", scope, sequence], queryFn: ({ signal }) => getCommercialOverview(organizationId, signal), gcTime: 0, staleTime: 0, retry: false, refetchOnWindowFocus: true, refetchOnReconnect: true });
  if (response.isPending || response.isFetching) return <p role="status">正在确认当前企业权益权限；店铺服务与权益正在读取。</p>;
  if (response.isError || !response.data || response.data.organization_id !== organizationId) {
    const code = response.error && "code" in response.error ? String(response.error.code) : "INVALID_UPSTREAM_RESPONSE";
    const labels: Record<string, string> = { PERMISSION_DENIED: "无权益查看权限", AUTHENTICATION_REQUIRED: "登录已失效", ORGANIZATION_ACCESS_REVOKED: "企业访问已撤销", ORGANIZATION_ACCESS_DENIED: "企业访问被拒绝", DEPENDENCY_UNAVAILABLE: "权益服务暂不可用", DEADLINE_EXCEEDED: "读取超时" };
    return <p role="alert">{labels[code] ?? "权益读取失败"}：本次未取得店铺服务与权益；企业资源余额以上方读取结果为准。请刷新资源重新读取。</p>;
  }
  return <EntitlementsOverview data={response.data} showManagementLinks={false} />;
}
