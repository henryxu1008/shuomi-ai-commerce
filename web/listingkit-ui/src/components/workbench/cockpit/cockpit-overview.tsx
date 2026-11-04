"use client";
import Image from "next/image";
import Link from "next/link";
import { useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { cockpitRequest, cockpitErrorText, type CockpitScope } from "@/lib/api/operations-cockpit";
import { cockpitCapabilities, cockpitMatrix, cockpitRules, moneyText, ratioText, type CockpitCapabilities } from "@/lib/contracts/operations-cockpit";
import styles from "./cockpit-overview.module.css";

const status: Record<string, string> = { loss: "亏损", break_even: "收支平衡", recorded_complete: "录入完整", data_incomplete: "数据不完整" };
function dayOffset(day: string, days: number) { const date = new Date(day + "T00:00:00Z"); date.setUTCDate(date.getUTCDate() + days); return date.toISOString().slice(0, 10); }
function Panel({ title, subtitle, link, children }: { title: string; subtitle: string; link?: string; children: ReactNode }) {
  return <Card className={styles.panel}><header><div><h2>{title}</h2><p>{subtitle}</p></div>{link ? <Link href={link} prefetch={false}>查看全部 →</Link> : null}</header>{children}</Card>;
}
export function CockpitOverview({ scope, permissions, enabled }: { scope?: CockpitScope; permissions?: readonly string[]; enabled: boolean }) {
  if (!enabled || !scope) return <OverviewPanels message={enabled ? "没有运营驾驶舱查看权限" : "经营数据暂未接入"} />;
  return <ReadOverview scope={scope} permissions={permissions} />;
}
function ReadOverview({ scope, permissions }: { scope: CockpitScope; permissions?: readonly string[] }) {
  const query = useQuery({
    queryKey: ["operations-cockpit", scope?.userId, scope?.organizationId, "home-capabilities", permissions],
    queryFn: ({ signal }) => cockpitRequest(scope!, "capabilities", cockpitCapabilities, { signal }),
    retry: false, gcTime: 0,
  });
  if (query.isPending || query.isFetching) return <OverviewPanels message="正在读取店铺与当前权限…" />;
  if (query.error || !query.data) return <OverviewPanels message={cockpitErrorText(query.error)} />;
  return <ConnectedOverview key={JSON.stringify([scope, query.data.access])} scope={scope} capabilities={query.data} />;
}
function ConnectedOverview({ scope, capabilities }: { scope: CockpitScope; capabilities: CockpitCapabilities }) {
  const [platform, setPlatform] = useState(""), [region, setRegion] = useState(""), [store, setStore] = useState(""), [days, setDays] = useState(7);
  const selected = capabilities.stores.filter(s => (!platform || s.platform === platform) && (!region || s.region === region) && (!store || s.id === store));
  const narrowed = !!(platform || region || store);
  const params = new URLSearchParams({ startDate: dayOffset(capabilities.today, -days), endDate: dayOffset(capabilities.today, -1), page: "1", sort: "profit" });
  if (narrowed) selected.forEach(s => params.append("storeId", s.id));
  const query = params.toString(), valid = query.length < 16000 && (!narrowed || selected.length > 0);
  const access = capabilities.access;
  const matrix = useQuery({ queryKey: ["operations-cockpit", scope.userId, scope.organizationId, "home-matrix", query, access.storesRead], queryFn: ({ signal }) => cockpitRequest(scope, "stores?" + query, cockpitMatrix, { signal }), enabled: access.storesRead && valid, retry: false, gcTime: 0 });
  const alerts = useQuery({ queryKey: ["operations-cockpit", scope.userId, scope.organizationId, "home-alerts", query, access.alertsRead], queryFn: ({ signal }) => cockpitRequest(scope, "alerts?" + query, cockpitRules, { signal }), enabled: access.alertsRead && valid, retry: false, gcTime: 0 });
  const advice = useQuery({ queryKey: ["operations-cockpit", scope.userId, scope.organizationId, "home-advice", query, access.adviceRead], queryFn: ({ signal }) => cockpitRequest(scope, "advice?" + query, cockpitRules, { signal }), enabled: access.adviceRead && valid, retry: false, gcTime: 0 });
  const data = !matrix.isFetching && !matrix.error && valid ? matrix.data : undefined;
  const readState = (allowed: boolean, fetching: boolean, error: unknown) => !allowed ? "当前身份无查看权限" : !valid ? query.length >= 16000 ? "范围较大，请按店铺筛选" : "所选范围暂无店铺" : fetching ? "正在读取经营事实…" : error ? cockpitErrorText(error) : null;
  const matrixState = readState(access.storesRead, matrix.isFetching, matrix.error), alertState = readState(access.alertsRead, alerts.isFetching, alerts.error), adviceState = readState(access.adviceRead, advice.isFetching, advice.error);
  const filters = <div className={styles.filters} aria-label="经营数据范围">
    <label><span className="sr-only">平台</span><select aria-label="平台" value={platform} onChange={e => { setPlatform(e.target.value); setRegion(""); setStore(""); }}><option value="">全部平台</option>{[...new Set(capabilities.stores.map(s => s.platform))].map(value => <option key={value} value={value}>{value.toUpperCase()}</option>)}</select></label>
    <label><span className="sr-only">站点</span><select aria-label="站点" value={region} onChange={e => { setRegion(e.target.value); setStore(""); }}><option value="">全部站点</option>{[...new Set(capabilities.stores.filter(s => !platform || s.platform === platform).map(s => s.region))].map(value => <option key={value} value={value}>{value}</option>)}</select></label>
    <select aria-label="店铺类型" disabled title="当前经营合同尚未提供店铺类型"><option>店铺类型未接入</option></select>
    <label><span className="sr-only">店铺</span><select aria-label="店铺" value={store} onChange={e => setStore(e.target.value)}><option value="">全部店铺</option>{capabilities.stores.filter(s => (!platform || s.platform === platform) && (!region || s.region === region)).map(s => <option key={s.id} value={s.id}>{s.name}</option>)}</select></label>
    <select aria-label="时间范围" value={days} onChange={e => setDays(Number(e.target.value))}><option value={7}>近7个完整日</option><option value={30}>近30个完整日</option></select>
  </div>;
  return <OverviewPanels filters={filters} storeCount={access.storesRead ? capabilities.stores.length : undefined} quickLinks={[
    ...(access.storesRead ? [{ href: "/workbench/overview/stores", label: "店铺经营", description: "查看可访问店铺" }] : []),
    ...(access.alertsRead ? [{ href: "/workbench/overview/alerts", label: "经营预警", description: "处理真实规则风险" }] : []),
    ...(access.adviceRead ? [{ href: "/workbench/overview/advice", label: "经营建议", description: "查看人工处理建议" }] : []),
  ]} profit={data?.summaryComplete ? moneyText(data.summary.netProfit) : undefined} profitNote={data ? data.summaryComplete ? "人工收入与成本 · 完整覆盖所选店铺" : "待完整数据，未按部分录入推定完整利润" : matrixState ?? undefined}>
    <Panel title="经营趋势" subtitle={`近${days}日 GMV 与利润变化`}><div className={styles.chartEmpty}><p>日趋势暂未接入</p><small>当前经营事实按录入区间保存，不能拆分推定逐日金额。</small></div>{access.storesRead && valid ? <Link className={styles.detailLink} href={`/workbench/overview/stores?${query}`} prefetch={false}>查看真实经营明细 →</Link> : null}</Panel>
    <Panel title="经营预警" subtitle="基于已保存的人工经营事实" link={access.alertsRead ? "/workbench/overview/alerts" : undefined}>
      {!alertState && alerts.data?.goalUnavailable && <p className={styles.empty}>企业当前目标或关联店铺不可访问，目标评判暂不可用。</p>}
      {alertState ? <p className={styles.empty}>{alertState}</p> : alerts.data?.rules.length ? <div className={styles.items}>{alerts.data.rules.slice(0, 3).map(rule => <Link key={rule.id} href={rule.actionPath} prefetch={false} className={styles.item}><Image src={`/figma/primary-homepages/cockpit-${rule.level === "critical" ? "imgEllipse4" : "imgEllipse5"}.svg`} width={9} height={9} alt="" /><div><strong>{rule.title}</strong><p>{rule.reason}</p></div><span>{rule.level === "critical" ? "高" : rule.level === "attention" ? "需关注" : "数据"}</span></Link>)}</div> : !alerts.data?.goalUnavailable ? <p className={styles.empty}>当前范围暂无规则预警</p> : null}
    </Panel>
    <Panel title="店铺矩阵" subtitle={data ? `所选期间店铺经营对比 · 共${data.total}家` : "所选期间店铺经营对比"} link={access.storesRead ? "/workbench/overview/stores" : undefined}>
      {matrixState ? <p className={styles.empty}>{matrixState}</p> : <div className={styles.tableWrap}><table><thead><tr><th>店铺</th><th>GMV</th><th>净利润</th><th>较上期</th><th>经营状态</th></tr></thead><tbody>{data?.rows.slice(0, 3).map(row => <tr key={row.store.id}><td><strong>{row.store.name}</strong><small>{row.store.platform.toUpperCase()} · {row.store.region}</small></td><td title="GMV来源未接入">—</td><td>{row.complete ? moneyText(row.totals.netProfit) : "待完整数据"}</td><td>{ratioText(row.growth)}</td><td>{status[row.state]}</td></tr>)}</tbody></table>{data && !data.rows.length ? <p className={styles.empty}>当前范围暂无店铺</p> : null}</div>}
    </Panel>
    <Panel title="AI执行与待办决策" subtitle="AI执行汇总暂未接入 · 当前展示人工处理建议" link={access.adviceRead ? "/workbench/overview/advice" : undefined}>
      {!adviceState && advice.data?.goalUnavailable && <p className={styles.empty}>企业当前目标或关联店铺不可访问，目标评判暂不可用。</p>}
      {adviceState ? <p className={styles.empty}>{adviceState}</p> : advice.data?.rules.length ? <div className={styles.items}>{advice.data.rules.slice(0, 3).map(rule => <Link key={rule.id} href={rule.actionPath} prefetch={false} className={styles.item}><div><strong>{rule.title}</strong><p>{rule.advice}</p></div><span>{rule.actionLabel} →</span></Link>)}</div> : !advice.data?.goalUnavailable ? <p className={styles.empty}>当前范围暂无人工处理建议</p> : null}
    </Panel>
  </OverviewPanels>;
}
function OverviewPanels({ message, filters, profit, profitNote, storeCount, quickLinks, children }: {
  message?: string;
  filters?: ReactNode;
  profit?: string;
  profitNote?: string;
  storeCount?: number;
  quickLinks?: readonly { href: string; label: string; description: string }[];
  children?: ReactNode;
}) {
  const metrics = [
    { label: "总 GMV", value: "—", note: "平台 GMV 汇总尚未接入", tone: "info" },
    { label: "可访问店铺", value: storeCount === undefined ? "—" : String(storeCount), note: storeCount === undefined ? "当前权限范围尚未读取" : "当前企业可访问的店铺", tone: "success" },
    { label: "待处理商品", value: "—", note: "商品处理统计尚未接入", tone: "warning" },
    { label: "待审核商品", value: "—", note: "审核数据统计尚未接入", tone: "purple" },
    { label: "净利润", value: profit ?? "—", note: profitNote ?? message ?? "仅完整经营事实可计算", tone: "info" },
    { label: "异常任务", value: "—", note: "跨业务任务汇总尚未接入", tone: "danger" },
  ] as const;
  const stages = ["商品采集", "AI 处理", "人工审核", "平台上传", "经营观察"] as const;

  return <section className={styles.page} aria-labelledby="cockpit-home">
    <header className={styles.header}>
      <div className={styles.heading}>
        <p className={styles.eyebrow}>运营驾驶舱 / 经营总览</p>
        <h1 id="cockpit-home">运营驾驶舱</h1>
        <p className={styles.subtitle}>统一查看店铺经营、风险与人工决策；仅展示有依据的业务数据。</p>
      </div>
      {filters ?? <p className={styles.availability} role="status">{message}</p>}
    </header>

    <div className={styles.metrics} aria-label="运营指标概览">
      {metrics.map(metric => <Card className={styles.metric} key={metric.label} data-tone={metric.tone}>
        <h2>{metric.label}</h2>
        <strong>{metric.value}</strong>
        <small>{metric.note}</small>
        <span className={styles.metricIndicator} aria-hidden="true" />
      </Card>)}
    </div>

    <section className={styles.flow} aria-label="跨境电商业务链路">
      <div className={styles.flowHeader}>
        <h2>跨境电商业务链路</h2>
        <p>业务说明 · 非实时进度</p>
      </div>
      <ol className={styles.flowSteps}>
        {stages.map((stage, index) => <li key={stage}>
          <span aria-hidden="true">{String(index + 1).padStart(2, "0")}</span>
          <strong>{stage}</strong>
        </li>)}
      </ol>
    </section>

    <div className={styles.panels}>
      {children ?? ["经营趋势", "经营预警", "店铺矩阵", "AI执行与待办决策"].map(title => <Panel key={title} title={title} subtitle="尚未取得经营数据">
        <p className={styles.empty}>{message}</p>
        {title === "店铺矩阵" ? <Button asChild variant="outline" size="sm"><Link href="/workbench/stores" prefetch={false}>查看我的店铺</Link></Button> : null}
      </Panel>)}
    </div>

    {quickLinks?.length ? <nav className={styles.quickActions} aria-label="驾驶舱快捷入口">
      <div className={styles.quickHeader}><h2>快捷开始</h2><p>仅展示当前权限允许查看的业务入口</p></div>
      <div className={styles.quickGrid}>
        {quickLinks.map(link => <Link key={link.href} href={link.href} prefetch={false} className={styles.quickLink}>
          <span className={styles.quickIcon} aria-hidden="true">↗</span>
          <span><strong>{link.label}</strong><small>{link.description}</small></span>
        </Link>)}
      </div>
    </nav> : null}
  </section>;
}
