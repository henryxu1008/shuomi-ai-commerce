"use client";

import Link from "next/link";
import { useState, type ReactNode } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { z } from "zod";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { cockpitRequest, cockpitErrorText, type CockpitScope } from "@/lib/api/operations-cockpit";
import { cockpitCapabilities, cockpitMatrix, cockpitRules, moneyText, type CockpitCapabilities } from "@/lib/contracts/operations-cockpit";
import { readCockpitMetrics } from "@/lib/api/cockpit-metrics";
import { listingMetrics, sheinPublishMetrics, taskMetrics, storeSubmissionMetrics, type TaskCounts } from "@/lib/contracts/cockpit-metrics";
import { CockpitTaskDistribution } from "./cockpit-task-distribution";
import styles from "./cockpit-overview.module.css";

type Matrix = z.infer<typeof cockpitMatrix>;
type Rules = z.infer<typeof cockpitRules>;
type Rule = Rules["rules"][number];

const statuses = {
  loss: "亏损",
  break_even: "收支平衡",
  recorded_complete: "录入完整",
  data_incomplete: "数据不完整",
};
const metrics = [
  ["总 GMV", "GMV 数据待接入", "info"],
  ["活跃店铺", "所选范围内已启用的授权店铺", "success"],
  ["待处理商品", "商品统计待接入", "warning"],
  ["待审核商品", "审核统计待接入", "accent"],
  ["上传成功率", "上传结果统计待接入", "info"],
  ["异常任务", "异常任务统计待接入", "danger"],
] as const;
const stages = [
  ["商品采集", "来源与原始版本", "success"],
  ["AI 处理", "内容与图片", "success"],
  ["人工审核", "明确应用结果", "accent"],
  ["平台上传", "提交与结果确认", "warning"],
  ["经营观察", "店铺经营反馈", "success"],
] as const;
const periods = [[1, "昨日"], [7, "近7天"], [30, "近30天"]] as const;

function dayOffset(day: string, days: number) {
  const date = new Date(day + "T00:00:00Z");
  date.setUTCDate(date.getUTCDate() + days);
  return date.toISOString().slice(0, 10);
}

function Panel({ title, detail, link, children }: {
  title: string; detail: string; link?: { href: string; label: string }; children: ReactNode;
}) {
  return <Card className={styles.panel}>
    <header className={styles.panelHeader}>
      <h2>{title}</h2>
      {link ? <Link href={link.href} prefetch={false} className={styles.textLink}>{link.label} →</Link> : <small>{detail}</small>}
    </header>
    {link && <p className={styles.panelNote}>{detail}</p>}
    {children}
  </Card>;
}

export function CockpitOverview({ scope, permissions, enabled }: { scope?: CockpitScope; permissions?: readonly string[]; enabled: boolean }) {
  if (!enabled || !scope) return <OverviewPanels message={enabled ? "没有运营驾驶舱查看权限" : "经营数据暂未接入"} />;
  return <ReadOverview scope={scope} permissions={permissions} />;
}

function ReadOverview({ scope, permissions }: { scope: CockpitScope; permissions?: readonly string[] }) {
  const query = useQuery({
    queryKey: ["operations-cockpit", scope.userId, scope.organizationId, "home-capabilities", permissions],
    queryFn: ({ signal }) => cockpitRequest(scope, "capabilities", cockpitCapabilities, { signal }),
    retry: false, gcTime: 0,
  });
  if (query.isPending) return <OverviewPanels message="正在读取店铺与当前权限…" />;
  if (query.error || !query.data) return <OverviewPanels message={cockpitErrorText(query.error)}
    filters={<Button variant="outline" onClick={() => void query.refetch()} disabled={query.isFetching}>重新读取</Button>} />;
  return <ConnectedOverview key={JSON.stringify([scope, query.data.access])}
    scope={scope} capabilities={query.data} refreshing={query.isFetching} permissions={permissions} />;
}

function ConnectedOverview({ scope, capabilities, refreshing, permissions }: {
  scope: CockpitScope; capabilities: CockpitCapabilities; refreshing: boolean; permissions?: readonly string[];
}) {
  const client = useQueryClient();
  const [platform, setPlatform] = useState("");
  const [region, setRegion] = useState("");
  const [store, setStore] = useState("");
  const [days, setDays] = useState(7);
  const platformStores = capabilities.stores.filter(s => !platform || s.platform === platform);
  const regionStores = platformStores.filter(s => !region || s.region === region);
  const selected = regionStores.filter(s => !store || s.id === store);
  const narrowed = Boolean(platform || region || store);
  const period = { startDate: dayOffset(capabilities.today, -days), endDate: dayOffset(capabilities.today, -1) };
  const params = new URLSearchParams({ ...period, page: "1", sort: "profit" });
  if (narrowed) selected.forEach(s => params.append("storeId", s.id));
  const query = params.toString();
  const tooLarge = query.length >= 16000;
  const valid = !tooLarge && (!narrowed || selected.length > 0);
  const statsParams = new URLSearchParams(period);
  for (const id of selected.map(item => item.id).sort()) statsParams.append("storeId", id);
  const statsQuery = statsParams.toString();
  const statsScopeMessage = statsQuery.length >= 15999 || selected.length > 500 ? "范围较大，请按店铺筛选"
    : selected.length === 0 ? "所选范围暂无店铺" : undefined;
  const listingAllowed = permissions?.includes("workbench.supply.read") ?? false;
  const tasksAllowed = permissions?.includes("workbench.task.read") ?? false;
  const singleUSStore = Boolean(store) && selected.length === 1 && selected[0].region === "US";
  const selectedPlatform = singleUSStore ? selected[0].platform : undefined;
  const submissionOwner = selectedPlatform === "walmart" ? "walmart-feed" : selectedPlatform === "tiktokshop" ? "tiktok-publish" : undefined;
  const submissionAllowed = Boolean(submissionOwner) && listingAllowed;
  const showSheinPublication = selectedPlatform === "shein";
  const sheinPublicationAllowed = showSheinPublication && listingAllowed;
  const access = capabilities.access;
  const key = ["operations-cockpit", scope.userId, scope.organizationId];
  const matrix = useQuery({ queryKey: [...key, "home-matrix", query, access.storesRead], queryFn: ({ signal }) => cockpitRequest(scope, "stores?" + query, cockpitMatrix, { signal }), enabled: access.storesRead && valid, retry: false, gcTime: 0 });
  const alerts = useQuery({ queryKey: [...key, "home-alerts", query, access.alertsRead], queryFn: ({ signal }) => cockpitRequest(scope, "alerts?" + query, cockpitRules, { signal }), enabled: access.alertsRead && valid, retry: false, gcTime: 0 });
  const advice = useQuery({ queryKey: [...key, "home-advice", query, access.adviceRead], queryFn: ({ signal }) => cockpitRequest(scope, "advice?" + query, cockpitRules, { signal }), enabled: access.adviceRead && valid, retry: false, gcTime: 0 });

  const stats = useQuery({
    queryKey: [...key, "home-metrics", statsQuery, listingAllowed, tasksAllowed, submissionOwner, submissionAllowed, sheinPublicationAllowed],
    enabled: !refreshing && !statsScopeMessage && (listingAllowed || tasksAllowed),
    retry: false,
    gcTime: 0,
    queryFn: async ({ signal }) => {
      const [listing, tasks, submission, sheinPublication] = await Promise.allSettled([
        listingAllowed ? readCockpitMetrics(scope, "supply-preparations", statsQuery, listingMetrics, signal) : undefined,
        tasksAllowed ? readCockpitMetrics(scope, "tasks", statsQuery, taskMetrics, signal) : undefined,
        listingAllowed && submissionOwner ? readCockpitMetrics(scope, submissionOwner, statsQuery, storeSubmissionMetrics, signal) : undefined,
        sheinPublicationAllowed ? readCockpitMetrics(scope, "shein-publish", statsQuery, sheinPublishMetrics, signal) : undefined,
      ] as const);
      return { listing, tasks, submission, sheinPublication };
    },
  });
  const listingRead = stats.data?.listing.status === "fulfilled" ? stats.data.listing.value : undefined;
  const taskRead = stats.data?.tasks.status === "fulfilled" ? stats.data.tasks.value : undefined;
  function metricsMessage(allowed: boolean, read?: { message?: string; result?: { status: string } }) {
    if (!allowed) return "当前身份无查看权限";
    if (statsScopeMessage) return statsScopeMessage;
    if (refreshing || stats.isFetching || stats.isPending) return "正在读取所选范围统计…";
    if (read?.result?.status === "complete") return undefined;
    if (read?.result?.status === "limited") return "记录较多，请缩小日期或店铺范围";
    return read?.message ?? "统计暂不可用";
  }
  const listingMessage = metricsMessage(listingAllowed, listingRead);
  const taskMessage = metricsMessage(tasksAllowed, taskRead);
  const submissionRead = stats.data?.submission.status === "fulfilled" ? stats.data.submission.value : undefined;
  const submissionMessage = metricsMessage(submissionAllowed, submissionRead);
  const sheinRead = stats.data?.sheinPublication.status === "fulfilled" ? stats.data.sheinPublication.value : undefined;
  const sheinMessage = metricsMessage(sheinPublicationAllowed, sheinRead);
  const sheinCounts = !sheinMessage && sheinRead?.result?.status === "complete" ? sheinRead.result.counts : undefined;
  const submissionCounts = !submissionMessage && submissionRead?.result?.status === "complete" ? submissionRead.result.counts : undefined;

  function readState(allowed: boolean, fetching: boolean, error: unknown) {
    if (refreshing) return "正在刷新经营事实…";
    if (!allowed) return "当前身份无查看权限";
    if (!valid) return tooLarge ? "范围较大，请按店铺筛选" : "所选范围暂无店铺";
    if (fetching) return "正在读取经营事实…";
    return error ? cockpitErrorText(error) : undefined;
  }
  const matrixState = readState(access.storesRead, matrix.isFetching, matrix.error);
  const alertState = readState(access.alertsRead, alerts.isFetching, alerts.error);
  const adviceState = readState(access.adviceRead, advice.isFetching, advice.error);
  const matrixHref = !refreshing && access.storesRead && valid ? "/workbench/overview/stores?" + query : undefined;
  const ruleParams = new URLSearchParams(period);
  if (narrowed && selected.length === 1) ruleParams.set("storeId", selected[0].id);
  const broaderRules = narrowed && selected.length > 1;
  const ruleLinkLabel = broaderRules ? "查看期间全部" : "查看全部";
  const alertLink = !refreshing && access.alertsRead && valid ? { href: "/workbench/overview/alerts?" + ruleParams, label: ruleLinkLabel } : undefined;
  const adviceLink = !refreshing && access.adviceRead && valid ? { href: "/workbench/overview/advice?" + ruleParams, label: ruleLinkLabel } : undefined;

  function actionFor(rule: Rule) {
    if (refreshing) return undefined;
    return (rule.actionPath === "/workbench/overview/goals" ? access.goalsRead : access.storesRead)
      ? rule.actionPath : undefined;
  }

  const filters = <fieldset className={styles.filters} disabled={refreshing}>
    <legend className="sr-only">经营数据范围</legend>
    <div className={styles.periods} role="group" aria-label="完整日周期">
      {periods.map(([length, label]) => <Button key={length} type="button" variant={days === length ? "default" : "ghost"}
        size="sm" aria-pressed={days === length} onClick={() => setDays(length)}>{label}</Button>)}
    </div>
    <Select aria-label="平台" value={platform} onChange={e => { setPlatform(e.target.value); setRegion(""); setStore(""); }}>
      <option value="">全部平台</option>
      {[...new Set(capabilities.stores.map(s => s.platform))].map(value => <option key={value} value={value}>{value.toUpperCase()}</option>)}
    </Select>
    <Select aria-label="站点" value={region} onChange={e => { setRegion(e.target.value); setStore(""); }}>
      <option value="">全部站点</option>
      {[...new Set(platformStores.map(s => s.region))].map(value => <option key={value} value={value}>{value}</option>)}
    </Select>
    <Select aria-label="店铺" value={store} onChange={e => setStore(e.target.value)}>
      <option value="">全部店铺</option>
      {regionStores.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
    </Select>
    <Button variant="outline" size="sm" onClick={() => void client.invalidateQueries({ queryKey: key })}>
      {refreshing ? "刷新中…" : "刷新"}
    </Button>
  </fieldset>;
  return <OverviewPanels
    filters={filters}
    message={refreshing ? "正在刷新经营事实…" : undefined}
    activeStores={refreshing ? undefined : selected.filter(s => s.status === "active").length}
    listingCounts={!listingMessage && listingRead?.result?.status === "complete" ? listingRead.result.counts : undefined}
    taskCounts={!taskMessage && taskRead?.result?.status === "complete" ? taskRead.result.counts : undefined}
    listingMessage={listingMessage}
    taskMessage={taskMessage}
    storeSubmission={showSheinPublication
      ? <SheinPublicationCounts counts={sheinCounts} message={sheinMessage} />
      : submissionOwner ? <StoreSubmissionRate owner={submissionOwner} counts={submissionCounts} message={submissionMessage} /> : undefined}
    platforms={refreshing ? undefined : [...new Set(selected.map(s => s.platform))].sort()}
    matrix={matrixState ? undefined : matrix.data}
    matrixState={matrixState}
    matrixHref={matrixHref}
    rowHref={id => "/workbench/overview/stores?" + new URLSearchParams({ ...period, storeId: id })}
    alerts={<RuleList data={alertState ? undefined : alerts.data} state={alertState} mode="alerts" actionFor={actionFor} />}
    advice={<RuleList data={adviceState ? undefined : advice.data} state={adviceState} mode="advice" actionFor={actionFor} />}
    alertLink={alertLink}
    adviceLink={adviceLink}
  />;
}

function RuleList({ data, state, mode, actionFor }: {
  data?: Rules; state?: string; mode: "alerts" | "advice"; actionFor: (rule: Rule) => string | undefined;
}) {
  if (state) return <p className={styles.empty}>{state}</p>;
  return <>
    {data?.goalUnavailable && <p className={styles.empty}>企业当前目标或关联店铺不可访问，目标评判暂不可用。</p>}
    {data?.rules.length ? <ul className={styles.list}>{data.rules.slice(0, 3).map(rule => {
      const href = actionFor(rule);
      const tone = rule.level === "critical" ? "danger" : rule.level === "attention" ? "warning" : "info";
      return <li className={styles.item} key={rule.id}>
        <span className={styles.mark} data-tone={tone} aria-hidden="true">●</span>
        <div><strong>{rule.title}</strong><p>{mode === "alerts" ? rule.reason : rule.advice}</p></div>
        {href && <Button asChild variant="outline" size="sm"><Link href={href} prefetch={false}>{rule.actionLabel}</Link></Button>}
      </li>;
    })}</ul> : data && !data.goalUnavailable && <p className={styles.empty}>
      {mode === "alerts" ? "当前范围暂无规则预警" : "当前范围暂无人工处理建议"}
    </p>}
  </>;
}

function SheinPublicationCounts({ counts, message }: {
  counts?: { confirmed: number; unknown: number };
  message?: string;
}) {
  return <Panel title="SHEIN US 商品发布" detail="按正式发布的原尝试统计">
    {counts ? <dl className={styles.sheinPublicationCounts} aria-live="polite">
      <div><dt>已确认提交</dt><dd>{counts.confirmed}</dd></div>
      <div><dt>结果待核实</dt><dd>{counts.unknown}</dd></div>
    </dl> : <p className={styles.empty} role="status">{message}</p>}
  </Panel>;
}

function StoreSubmissionRate({ owner, counts, message }: {
  owner: "walmart-feed" | "tiktok-publish";
  counts?: { accepted: number; rejected: number; unknown: number };
  message?: string;
}) {
  const denominator = counts ? counts.accepted + counts.rejected : 0;
  const rate = counts && denominator > 0
    ? (100 * counts.accepted / denominator).toFixed(1) + "%"
    : "暂无已确认结果";
  const labels = owner === "walmart-feed"
    ? { title: "Walmart Feed 提交受理率", detail: "按原 Feed 提交尝试统计", accepted: "已受理", rejected: "明确拒绝" }
    : { title: "TikTok Shop US 提交审核率", detail: "按全球商品发布到美国站的原尝试统计", accepted: "已提交审核", rejected: "明确失败" };
  return <Panel title={labels.title} detail={labels.detail}>
    {counts ? <div className={styles.profit} aria-live="polite">
      <strong>{rate}</strong>
      <small>{labels.accepted} {counts.accepted} · {labels.rejected} {counts.rejected} · 结果未知 {counts.unknown}</small>
    </div> : <p className={styles.empty} role="status">{message}</p>}
  </Panel>;
}
function OverviewPanels({ filters, message, activeStores, listingCounts, taskCounts, listingMessage, taskMessage, storeSubmission, platforms, matrix, matrixState, matrixHref, rowHref, alerts, advice, alertLink, adviceLink }: {
  filters?: ReactNode;
  message?: string;
  activeStores?: number;
  listingCounts?: { pending: number; review: number };
  taskCounts?: TaskCounts;
  listingMessage?: string;
  taskMessage?: string;
  storeSubmission?: ReactNode;
  platforms?: readonly string[];
  matrix?: Matrix;
  matrixState?: string;
  matrixHref?: string;
  rowHref?: (id: string) => string;
  alerts?: ReactNode;
  advice?: ReactNode;
  alertLink?: { href: string; label: string };
  adviceLink?: { href: string; label: string };
}) {
  const values = [undefined, activeStores, listingCounts?.pending, listingCounts?.review, undefined, taskCounts?.ERROR];
  const notes = [undefined, undefined, listingMessage ?? (listingCounts && "期间新增刊登资料 · 仍待补全或上传"),
    listingMessage ?? (listingCounts && "期间新增刊登资料 · 仍待人工审核"), undefined,
    taskMessage ?? (taskCounts && "期间创建的本人店铺任务 · 当前异常")];
  const profit = matrix?.summaryComplete && matrix.total > 0 ? moneyText(matrix.summary.netProfit) : undefined;
  const profitNote = matrix ? profit ? "人工收入与成本 · 完整覆盖所选店铺"
    : "待完整数据，未按部分录入推定完整利润" : matrixState ?? message ?? "待完整数据";
  const shortcuts = [
    ["商品采集", "供应市场", "/workbench/supply", "info"],
    ["AI 工作台", "执行与审核", "/workbench/ai", "accent"],
    ["经营预警", "查看判断依据", alertLink?.href, "warning"],
    ["经营建议", "核查与人工处理", adviceLink?.href, "success"],
    ["店铺中心", "店铺与商品", "/workbench/store-center", "info"],
    ["套餐与权益", "资源使用", "/workbench/plans", "accent"],
  ] as const;
  return <section className={styles.page} aria-labelledby="cockpit-home">
    <div className={styles.content}>
      <header className={styles.header}>
        <div>
          <p className={styles.eyebrow}>运营驾驶舱 / 经营总览</p>
          <h1 id="cockpit-home">运营驾驶舱</h1>
          <p className={styles.panelNote}>采集、AI 处理、上架与店铺经营一屏掌握</p>
        </div>
        {filters}
      </header>
      {message && <p className={styles.notice} role="status">{message}</p>}

      <div className={styles.metrics}>
        {metrics.map(([title, note, tone], i) => <Card className={styles.metric} key={title}>
          <div className={styles.metricTop}><h2>{title}</h2><span className={styles.mark} data-tone={tone} aria-hidden="true">◉</span></div>
          <strong>{values[i] ?? "—"}</strong>
          <small>{notes[i] ?? note}</small>
        </Card>)}
      </div>

      {storeSubmission}
      <div className={styles.split}>
        <Panel title="经营与刊登趋势" detail="所选完整日">
          <div className={styles.trend}>
            <div className={styles.profit}><span>净利润</span><strong>{profit ?? "—"}</strong><small>{profitNote}</small></div>
            <p className={styles.chartNote}><strong>日趋势暂未接入</strong><span>经营事实按录入区间保存，不能拆分推定逐日金额。</span></p>
          </div>
          {matrixHref && <Link className={styles.textLink} href={matrixHref} prefetch={false}>查看真实经营明细 →</Link>}
        </Panel>
        <Panel title="执行状态分布" detail="期间创建的本人店铺任务 · 当前状态">
          <CockpitTaskDistribution counts={taskCounts} message={taskMessage ?? message} />
        </Panel>
      </div>

      <Panel title="跨境电商业务链路" detail="从采集到经营反馈">
        <ol className={styles.stages}>{stages.map(([label, note, tone], i) => <li className={styles.item} key={label}>
          <span className={styles.mark} data-tone={tone} aria-hidden="true">{String(i + 1).padStart(2, "0")}</span>
          <div><strong>{label}</strong><p>{note}</p></div>
        </li>)}</ol>
      </Panel>

      <div className={styles.actions}>
        <Panel title="经营风险与提醒" detail="基于已保存的人工经营事实" link={alertLink}>
          {alerts ?? <p className={styles.empty}>{message}</p>}
        </Panel>
        <Panel title="今日待办" detail="当前经营建议 · 由人工核查" link={adviceLink}>
          {advice ?? <p className={styles.empty}>{message}</p>}
        </Panel>
        <div className={styles.quickPanel}>
          <Panel title="快捷开始" detail="常用工作入口">
            <ul className={styles.shortcuts}>{shortcuts.map(([title, note, href, tone]) => {
              if (!href) return null;
              return <li key={title}>
                <Link href={href} className={styles.shortcut} prefetch={false}>
                  <span className={styles.mark} data-tone={tone} aria-hidden="true">↗</span>
                  <span><strong>{title}</strong><small>{note}</small></span>
                </Link>
              </li>;
            })}</ul>
          </Panel>
        </div>
      </div>

      <div className={styles.split}>
        <Panel title="店铺表现概览" detail={matrix ? "所选期间经营数据 · 共" + matrix.total + "家" : "人工经营数据"}
          link={matrixHref ? { href: matrixHref, label: "查看全部店铺" } : undefined}>
          {matrix?.rows.length ? <div className={styles.tableWrap} tabIndex={0} role="region" aria-label="店铺表现表">
            <table>
              <thead><tr><th scope="col">店铺</th><th scope="col">平台 / 站点</th><th scope="col">净利润</th><th scope="col">数据状态</th><th scope="col">详情</th></tr></thead>
              <tbody>{matrix.rows.slice(0, 3).map(row => <tr key={row.store.id}>
                <th scope="row">{row.store.name}</th>
                <td>{row.store.platform.toUpperCase()} · {row.store.region}</td>
                <td>{row.complete ? moneyText(row.totals.netProfit) : "待完整数据"}</td>
                <td><span className={styles.storeStatus} data-state={row.state}>{statuses[row.state]}</span></td>
                <td>{rowHref && <Link className={styles.textLink} href={rowHref(row.store.id)} prefetch={false} aria-label={"查看 " + row.store.name}>查看</Link>}</td>
              </tr>)}</tbody>
            </table>
          </div> : <p className={styles.empty}>{matrixState ?? message ?? "当前范围暂无店铺"}</p>}
        </Panel>
        <Panel title="平台与来源" detail="当前授权店铺的平台">
          {platforms?.length ? <ul className={styles.list}>{platforms.map(platform =>
            <li className={styles.item} key={platform}><span className={styles.dot} data-tone="success" aria-hidden="true" />
              <strong>{platform.toUpperCase()}</strong><small className={styles.sourceNote}>店铺目录</small>
            </li>)}</ul> : <p className={styles.empty}>{message ?? "当前范围暂无平台来源"}</p>}
        </Panel>
      </div>
      <p className={styles.footer}>经营事实以已保存的人工录入为准，店铺范围以当前授权为准。</p>
    </div>
  </section>;
}
