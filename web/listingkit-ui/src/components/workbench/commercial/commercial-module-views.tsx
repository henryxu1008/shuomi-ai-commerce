"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useState, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import type { CommercialOverview } from "@/lib/api/commercial";
import { readAcquisitionProduct, type AcquisitionContext } from "@/lib/api/product-acquisition";
import {
  readResourceBusinessObject,
  type ResourceBusinessObject,
  type ResourceEventPage,
  type ResourceEventFilters,
} from "@/lib/api/resource-events";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";
import { AccountDialog } from "../account/account-dialog";
import { resourceDefinitions } from "./resource-purchase-panel";
import styles from "./commercial.module.css";
import { PlansOverview } from "./plans-overview";

export function CommercialOverviewView({ data }: { data: CommercialOverview }) {
  return <PlansOverview data={data} />;
}

const reasons: Record<string, string> = {
  purchase: "购买到账",
  allocated: "分配",
  reclaimed: "收回",
  reserved: "预留",
  consumed: "使用",
  released: "释放预留",
  grant_commercial_purchase: "购买到账",
  grant_local_ai_trial: "AI 试用赠送",
  allocate_member_resource: "分配给成员",
  reclaim_member_resource: "收回成员额度",
  reserve: "预留",
  commit: "使用",
  release: "释放预留",
  model_point_reserve: "模型调用预留",
  model_point_committed: "模型调用扣除",
  model_point_released: "释放模型调用预留",
};

const businessSources: Record<string, string> = {
  store_service_v1: "店铺服务",
  product_acquisition_v1: "1688 商品采集",
  amazon_data_v1: "Amazon 数据采集",
  image_generation_v1: "图片生成",
  model_invocation_v1: "AI 模型调用",
  commercial_order_item: "资源购买",
  member_resource_position: "成员额度管理",
  local_ai_trial_v1: "AI 试用赠送",
  onboarding_welcome_store_period: "新企业赠送",
};

const balances = [
  { label: "可用", delta: "available_delta", after: "available_after" },
  { label: "已分配", delta: "allocated_delta", after: "allocated_after" },
  { label: "预留", delta: "reserved_delta", after: "reserved_after" },
  { label: "已使用", delta: "consumed_delta", after: "consumed_after" },
] as const;

type UsageEvent = ResourceEventPage["items"][number];

function quantity(value: string, signed = true) {
  const negative = value.startsWith("-");
  const [whole, fraction] = (negative ? value.slice(1) : value).split(".");
  const integer = BigInt(whole);
  if (integer === BigInt(0) && (!fraction || BigInt(fraction) === BigInt(0))) return "0";
  const grouped = integer.toLocaleString("zh-CN");
  return `${negative ? "-" : signed ? "+" : ""}${grouped}${fraction ? `.${fraction}` : ""}`;
}

function resourceFor(event: UsageEvent) {
  return resourceDefinitions.find(({ type }) => type === event.resource_type);
}

function reasonFor(event: UsageEvent) {
  return Object.hasOwn(reasons, event.reason) ? reasons[event.reason] : "其他变动";
}

function sourceFor(event: UsageEvent) {
  return Object.hasOwn(businessSources, event.source_type) ? businessSources[event.source_type] : "未识别来源";
}

function UsageEventTime({ timestamp, showFractionalSeconds = true }: {
  timestamp: string;
  showFractionalSeconds?: boolean;
}) {
  const text = timestamp.replace("T", " ").replace("Z", "");
  return <time dateTime={timestamp}>{showFractionalSeconds ? text : text.slice(0, 19)}</time>;
}

function UsageAcquisitionObject({ scope, operationId }: { scope: AcquisitionContext; operationId: string }) {
  const product = useQuery({
    queryKey: ["workbench", scope.organizationId, "usage-acquisition-product", scope.userId, operationId],
    queryFn: ({ signal }) => readAcquisitionProduct(operationId, scope, signal),
    gcTime: 0,
    staleTime: 0,
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
  return (
    <div>
      <dt>关联商品</dt>
      <dd>
        {product.isPending || product.isFetching ? (
          <span role="status">正在查找关联商品…</span>
        ) : product.isSuccess ? (
          <Link
            className={styles.usageObjectLink}
            href={`/workbench/supply/acquisition/operation/${operationId}`}
            prefetch={false}
          >
            {product.data.title?.trim() || "查看采集商品"}
          </Link>
        ) : "未能查看关联商品"}
      </dd>
    </div>
  );
}

function usageObjectKind(event: UsageEvent): ResourceBusinessObject["kind"] | undefined {
  if (!isAcquisitionUUID(event.event_id)) return;
  if (event.resource_type === "store_renewal_period" && event.source_type === "store_service_v1") return "store";
  if (event.resource_type === "data_row" && event.source_type === "amazon_data_v1") return "amazon_product";
  if (event.resource_type === "ai_point" && event.source_type === "model_invocation_v1") return "task";
}

const usageObjectLabels = { store: "关联店铺", amazon_product: "关联商品", task: "关联任务" };

function UsageBusinessObject({ scope, eventId, kind }: {
  scope: AcquisitionContext;
  eventId: string;
  kind: ResourceBusinessObject["kind"];
}) {
  const details = useQuery({
    queryKey: ["workbench", scope.organizationId, "usage-business-object", scope.userId, eventId],
    queryFn: ({ signal }) => readResourceBusinessObject(scope.userId, scope.organizationId, eventId, signal),
    gcTime: 0,
    staleTime: 0,
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
  const object = details.isSuccess && details.data.object?.kind === kind ? details.data.object : null;
  if (details.isPending || details.isFetching) {
    return <div><dt>{usageObjectLabels[kind]}</dt><dd role="status">正在查找{usageObjectLabels[kind]}…</dd></div>;
  }
  if (!object) return null;
  const href = object.kind === "store" ? `/workbench/stores/${object.id}`
    : object.kind === "task" ? `/workbench/ai/tasks/${object.id}` : null;
  return (
    <div>
      <dt>{usageObjectLabels[kind]}</dt>
      <dd>{href
        ? <Link href={href} className={styles.usageObjectLink} prefetch={false}>{object.name}</Link>
        : object.name}</dd>
    </div>
  );
}

function UsageEventDetails({ event, scope, onClose }: { event: UsageEvent; scope: AcquisitionContext; onClose: () => void }) {
  const resource = resourceFor(event);
  const objectKind = usageObjectKind(event);
  const acquisition = event.source_type === "product_acquisition_v1" &&
    event.resource_type === "data_row" && isAcquisitionUUID(event.source_identity);
  return (
    <AccountDialog title="用量记录详情" onClose={onClose}>
      <div className={styles.usageDetails}>
        <dl className={styles.orderFacts}>
          <div><dt>时间（UTC）</dt><dd><UsageEventTime timestamp={event.occurred_at} /></dd></div>
          <div><dt>资源</dt><dd>{resource?.name}</dd></div>
          <div><dt>业务来源</dt><dd>{sourceFor(event)}</dd></div>
          <div><dt>变动说明</dt><dd>{reasonFor(event)}</dd></div>
          <div><dt>关联编号</dt><dd>{event.source_identity}</dd></div>
          {objectKind ? <UsageBusinessObject scope={scope} eventId={event.event_id} kind={objectKind} /> : null}
          {acquisition ? <UsageAcquisitionObject scope={scope} operationId={event.source_identity} /> : null}
        </dl>
        <table className={styles.usageDetailTable}>
          <caption className="sr-only">数量与变动后余额</caption>
          <thead><tr><th scope="col">项目</th><th scope="col">本次变动</th><th scope="col">变动后</th></tr></thead>
          <tbody>
            {balances.map(({ label, delta, after }) => (
              <tr key={delta}>
                <th scope="row">{label}</th>
                <td>{quantity(event[delta])} {resource?.unit}</td>
                <td>{quantity(event[after], false)} {resource?.unit}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </AccountDialog>
  );
}

function UsageEventsTable({ items, scope }: { items: UsageEvent[]; scope: AcquisitionContext }) {
  const [selected, setSelected] = useState<UsageEvent>();
  return (
    <>
      <div className={styles.usageTableWrap}>
        <table className={styles.usageTable}>
          <caption className="sr-only">企业用量记录</caption>
          <thead>
            <tr>
              <th scope="col" className={styles.usageRecord}>业务记录</th>
              <th scope="col" className={styles.usageDeduction}>实际扣除</th>
              <th scope="col" className={styles.usageTimeColumn}>时间（UTC）</th>
              <th scope="col">操作</th>
            </tr>
          </thead>
          <tbody>
            {items.map((event) => {
              const resource = resourceFor(event);
              const deduction = quantity(event.consumed_delta, false);
              const hasDeduction = deduction !== "0" && !deduction.startsWith("-");
              return (
                <tr key={event.event_id}>
                  <td>
                    <strong>{reasonFor(event)}</strong>
                    <span className={styles.usageMeta}>{sourceFor(event)} · {resource?.name}</span>
                    <span className={styles.usageCompact}>
                      <UsageEventTime timestamp={event.occurred_at} showFractionalSeconds={false} /> UTC
                    </span>
                  </td>
                  <td className={styles.usageDeduction}>
                    {hasDeduction ? <strong>{deduction} {resource?.unit}</strong> : <span className={styles.usageNoDeduction}>—</span>}
                  </td>
                  <td className={styles.usageTimeColumn}>
                    <UsageEventTime timestamp={event.occurred_at} showFractionalSeconds={false} />
                  </td>
                  <td><Button size="sm" variant="outline" onClick={() => setSelected(event)}>详情</Button></td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      {selected ? <UsageEventDetails event={selected} scope={scope} onClose={() => setSelected(undefined)} /> : null}
    </>
  );
}

function boundary(value: string, next = false) {
  return new Date(`${value}T00:00:00Z`).getTime() + (next ? 86400000 : 0);
}

export function UsageDetailsView({
  scope,
  page,
  notice,
  filters = {},
  pageNumber = 1,
  onFilter,
  onPrevious,
  onNext,
}: {
  scope: AcquisitionContext;
  page?: ResourceEventPage;
  notice?: ReactNode;
  filters?: ResourceEventFilters;
  pageNumber?: number;
  onFilter: (filters: ResourceEventFilters) => void;
  onPrevious?: () => void;
  onNext?: () => void;
}) {
  const [from, setFrom] = useState(filters.from?.slice(0, 10) ?? "");
  const [until, setUntil] = useState(
    filters.until ? new Date(Date.parse(filters.until) - 86400000).toISOString().slice(0, 10) : "",
  );
  const [invalid, setInvalid] = useState(false);
  const categories = [{ name: "全部记录", type: undefined }, ...resourceDefinitions];
  return (
    <Card role="region" aria-label="用量记录" className={`${styles.panel} ${styles.usagePanel}`}>
      <h2>用量记录</h2>
      <p className={styles.observation}>打开详情可核对完整变动、余额和关联对象。</p>
      <div role="group" aria-label="资源类型" className={styles.usageCategories}>
        {categories.map(({ type, name }) => (
          <Button
            key={type ?? "all"}
            size="sm"
            variant={filters.resourceType === type ? "secondary" : "ghost"}
            aria-pressed={filters.resourceType === type}
            onClick={() => onFilter({ ...filters, resourceType: type })}
          >
            {name}
          </Button>
        ))}
      </div>
      <form
        className={styles.filterBar}
        aria-label="日期筛选"
        onSubmit={(event) => {
          event.preventDefault();
          if (from && until && boundary(from) >= boundary(until, true)) {
            setInvalid(true);
            return;
          }
          setInvalid(false);
          onFilter({
            resourceType: filters.resourceType,
            from: from ? new Date(boundary(from)).toISOString() : undefined,
            until: until ? new Date(boundary(until, true)).toISOString() : undefined,
          });
        }}
      >
        <label>
          开始日期（UTC）
          <input type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
        </label>
        <label>
          结束日期（UTC）
          <input type="date" value={until} onChange={(event) => setUntil(event.target.value)} />
        </label>
        <div className={styles.actions}>
          <Button type="submit">筛选</Button>
          <Button type="button" variant="outline" onClick={() => {
            setFrom("");
            setUntil("");
            setInvalid(false);
            onFilter({});
          }}>
            重置
          </Button>
        </div>
      </form>
      {invalid ? <p role="alert" className={styles.usageEmpty}>结束日期不能早于开始日期。</p> : null}
      {notice}
      {page ? (
        <>
          {page.items.length ? <UsageEventsTable items={page.items} scope={scope} /> : (
            <p className={styles.usageEmpty}>暂无符合条件的用量记录。</p>
          )}
          <div className={styles.pagination}>
            <span aria-live="polite">第 {pageNumber} 页 · {page.items.length} 条记录</span>
            <div className={styles.actions}>
              <Button size="sm" variant="outline" disabled={!onPrevious} onClick={onPrevious}>上一页</Button>
              <Button size="sm" variant="outline" disabled={!onNext} onClick={onNext}>下一页</Button>
            </div>
          </div>
        </>
      ) : null}
    </Card>
  );
}
