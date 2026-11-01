"use client";

import Link from "next/link";
import { cloneElement, isValidElement, useEffect, useId, useRef, useState, type ReactElement, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ConsolePage, ConsoleState, ConsoleToolbar } from "../console/console-page";
import { useResourcePending } from "../resources/resource-pending";
import { KernelError, kernelMessage, kernelRequest } from "@/lib/api/kernel-releases";
import { kernelArtifact, kernelIntent, kernelPage, kernelQuery, kernelResult, matchesKernelArtifact, type KernelIntent, type KernelQuery } from "@/lib/contracts/kernel-releases";
import styles from "./kernel-releases.module.css";

export const kernelPagePath = "/workbench/account/organization/kernel-releases";
export function KernelReleasesEntry() {
  const context = useWorkbenchContext();
  if (!context.permissions.includes("listingkit.platform_admin")) return null;
  return <Card className={styles.panel}><h2>浏览器内核</h2><p>查询兼容发行记录，登记固定内核制品。</p><Button asChild variant="outline"><Link href={kernelPagePath} prefetch={false}>内核发行管理</Link></Button></Card>;
}
export function KernelReleasesPage({ expectedUserId, create = false }: { expectedUserId: string; create?: boolean }) {
  const context = useWorkbenchContext();
  const unavailable = context.error || context.blockingError || context.user?.id !== expectedUserId || !context.permissions.includes("listingkit.platform_admin");
  return <ConsolePage title={create ? "登记内核" : "内核发行管理"} description={create ? "登记固定制品，并读回核对本次记录" : "查询指定平台与客户端可使用的发行记录"} breadcrumbs={[{ label: "我的账户", href: "/workbench/account" }, { label: "企业空间", href: "/workbench/account/organization" }, { label: "内核发行管理", ...(create ? { href: kernelPagePath } : {}) }, ...(create ? [{ label: "登记内核" }] : [])]}>
    {context.isLoading || context.isSwitching ? <ConsoleState kind="loading" title="正在确认维护身份" /> : unavailable ? <ConsoleState kind="error" title="无平台维护权限或登录已失效" /> : create ? <KernelRegistration key={expectedUserId} userId={expectedUserId} /> : <KernelReleaseList key={expectedUserId} userId={expectedUserId} />}
  </ConsolePage>;
}
const initialQuery: KernelQuery = { os: "windows", arch: "amd64", clientVersion: "1.8.0" };
function queryString(query: KernelQuery) { return new URLSearchParams(query).toString(); }
export function KernelReleaseList({ userId }: { userId: string }) {
  const [editing, setEditing] = useState(initialQuery), [query, setQuery] = useState(initialQuery), [cursor, setCursor] = useState(""), [error, setError] = useState("");
  const path = "?" + queryString(query) + "&limit=100" + (cursor ? "&cursor=" + cursor : "");
  const result = useQuery({ queryKey: ["kernel-admin", userId, query, cursor], queryFn: ({ signal }) => kernelRequest(userId, path, kernelPage, { signal }), retry: false, gcTime: 0, refetchOnWindowFocus: false });
  return <>
    <ConsoleToolbar><form className={styles.filters} onSubmit={event => { event.preventDefault(); const parsed = kernelQuery.safeParse(editing); if (!parsed.success) { setError("请填写完整的客户端 SemVer，例如 1.8.0。"); return; } setError(""); setQuery(parsed.data); setCursor(""); if (queryString(parsed.data) === queryString(query) && !cursor) void result.refetch(); }}>
      <PlatformFields os={editing.os} arch={editing.arch} setOS={os => setEditing(v => ({ ...v, os }))} setArch={arch => setEditing(v => ({ ...v, arch }))} />
      <Field label="客户端版本"><input value={editing.clientVersion} onChange={e => setEditing(v => ({ ...v, clientVersion: e.target.value }))} /></Field>
      <Button type="submit" disabled={result.isFetching}>查询</Button><Button asChild><Link href={kernelPagePath + "/new"} prefetch={false}>登记内核</Link></Button>
    </form></ConsoleToolbar>
    {error ? <p role="alert">{error}</p> : null}
    {result.isPending ? <ConsoleState kind="loading" title="正在读取发行目录" /> : result.isError ? <ConsoleState kind="error" title="目录读取失败">{kernelMessage(result.error)}</ConsoleState> : <Card className={styles.panel}>
      <p>范围：{query.os} / {query.arch} · 客户端 {query.clientVersion}</p>
      {result.data.releases.length ? <div className={styles.table}><table><caption className="sr-only">兼容内核发行记录</caption><thead><tr><th>内核版本</th><th>发行编号</th><th>客户端范围</th><th>推荐</th></tr></thead><tbody>{result.data.releases.map(row => <tr key={row.releaseId}><td>{row.version}</td><td><code>{row.releaseId}</code></td><td>[{row.minClientVersion}, {row.maxClientVersionExclusive || "不限"})</td><td>{row.recommended ? "是" : "否"}</td></tr>)}</tbody></table></div> : <p>本页没有兼容记录。</p>}
      <div className={styles.actions}>{cursor ? <Button variant="outline" onClick={() => setCursor("")}>返回首页</Button> : null}{result.data.nextCursor ? <Button disabled={result.isFetching} onClick={() => setCursor(result.data.nextCursor)}>下一页</Button> : null}</div>
    </Card>}
  </>;
}
const blank = { key: "", version: "", os: "windows" as KernelQuery["os"], arch: "amd64" as KernelQuery["arch"], minClientVersion: "1.8.0", maxClientVersionExclusive: "", archiveFormat: "zip", size: "", sha256: "", downloadUrl: "", clientVersion: "1.8.0" };
export function KernelRegistration({ userId }: { userId: string }) {
  const [fields, setFields] = useState(blank), [errors, setErrors] = useState<Record<string, string>>({}), [failure, setFailure] = useState<unknown>(), [busy, setBusy] = useState(false), [receipt, setReceipt] = useState<KernelIntent | null>(null);
  const pending = useResourcePending({ expectedUserId: userId, expectedOrganizationId: "" }, ["kernel-publication"], kernelIntent, { storage: "session", maxLength: 16384 });
  const flight = useRef(false), active = useRef(true), controller = useRef<AbortController | null>(null);
  useEffect(() => { active.current = true; return () => { active.current = false; controller.current?.abort(); }; }, []);
  const locked = busy || !!pending.command || !pending.ready || pending.error;
  function change(field: keyof typeof blank, value: string) { setFields(old => ({ ...old, [field]: value })); setErrors({}); setFailure(undefined); }
  async function run(operation: (signal: AbortSignal) => Promise<void>) {
    if (flight.current) return; flight.current = true; setBusy(true); setFailure(undefined); const c = new AbortController(); controller.current = c;
    try { await operation(c.signal); } catch (error) { if (active.current) setFailure(error); } finally { flight.current = false; if (active.current) setBusy(false); }
  }
  async function verify(intent: KernelIntent, signal: AbortSignal) {
    const query = queryString({ os: intent.payload.os, arch: intent.payload.arch, clientVersion: intent.clientVersion });
    const artifact = await kernelRequest(userId, "/" + intent.key + "/artifact?" + query, kernelArtifact, { signal });
    if (!matchesKernelArtifact(intent.key, intent.payload, artifact)) throw new KernelError("READBACK_MISMATCH");
    if (active.current) { pending.clear(intent); setReceipt(intent); }
  }
  async function submit() {
    if (locked || flight.current) return;
    const { key, clientVersion, size, ...payload } = fields;
    const parsed = kernelIntent.safeParse({ key, clientVersion, payload: { ...payload, size: /^[1-9][0-9]*$/.test(size) ? Number(size) : 0 } });
    if (!parsed.success) { setErrors(Object.fromEntries(parsed.error.issues.map(issue => [issue.path.at(-1)!, issue.message]))); return; }
    const intent = parsed.data;
    await run(async signal => {
      pending.persist(intent);
      try {
        const result = await kernelRequest(userId, "/" + intent.key, kernelResult, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(intent.payload), signal });
        if (result.releaseId !== intent.key || result.state !== "published") throw new KernelError("OUTCOME_UNKNOWN");
      } catch (error) {
        if (error instanceof KernelError && (error.code === "NOT_SENT" || error.status >= 400 && error.status < 500 && error.code !== "OUTCOME_UNKNOWN")) pending.clear(intent);
        throw error;
      }
      if (active.current) await verify(intent, signal);
    });
  }
  if (receipt) return <Card className={styles.panel} role="status"><h2>已登记并读回确认</h2><IntentSummary intent={receipt} /><p>推荐：否</p><div className={styles.actions}><Button asChild><Link href={kernelPagePath} prefetch={false}>返回发行目录</Link></Button><Button variant="outline" onClick={() => { setReceipt(null); setFields(blank); }}>登记下一条</Button></div></Card>;
  return <>
    {pending.command || pending.error ? <Card className={styles.panel}><h2>登记结果待核实</h2>{pending.command ? <IntentSummary intent={pending.command} /> : <p>无法读取保留的请求，暂不可登记新记录。</p>}<p>保留原编号和参数。未确认前不能再次登记或生成新编号。</p><Button disabled={busy || !pending.command} onClick={() => { const intent = pending.command; if (intent) void run(signal => verify(intent, signal)); }}>查询本次登记</Button></Card> : null}
    <Card className={styles.panel}><form onSubmit={event => { event.preventDefault(); void submit(); }}>
      <fieldset disabled={locked} className={styles.form}><legend className="sr-only">内核登记参数</legend>
        <div className={styles.actions}><Field label="发行编号" error={errors.key}><input value={fields.key} onChange={e => change("key", e.target.value)} /></Field><Button type="button" variant="outline" onClick={() => change("key", crypto.randomUUID())}>生成编号</Button></div>
        <div className={styles.grid}><Field label="内核版本" error={errors.version}><input value={fields.version} onChange={e => change("version", e.target.value)} /></Field><PlatformFields os={fields.os} arch={fields.arch} setOS={os => change("os", os)} setArch={arch => change("arch", arch)} />
          <Field label="最低客户端版本" error={errors.minClientVersion}><input value={fields.minClientVersion} onChange={e => change("minClientVersion", e.target.value)} /></Field><Field label="结束客户端版本（不含）" hint="留空表示不限" error={errors.maxClientVersionExclusive}><input value={fields.maxClientVersionExclusive} onChange={e => change("maxClientVersionExclusive", e.target.value)} /></Field><Field label="读回客户端版本" hint="须在上述兼容范围内" error={errors.clientVersion}><input value={fields.clientVersion} onChange={e => change("clientVersion", e.target.value)} /></Field>
          <Field label="归档格式"><select value={fields.archiveFormat} onChange={e => change("archiveFormat", e.target.value)}>{["zip", "tar.gz", "tar.xz"].map(format => <option key={format}>{format}</option>)}</select></Field><Field label="制品大小（字节）" hint="1 至 2147483648 的整数" error={errors.size}><input inputMode="numeric" value={fields.size} onChange={e => change("size", e.target.value)} /></Field>
        </div>
        <Field label="SHA256" hint="64 位小写十六进制摘要" error={errors.sha256}><input value={fields.sha256} onChange={e => change("sha256", e.target.value)} /></Field>
        <Field label="固定 HTTPS 下载地址" hint="不含用户名、密码、查询参数或片段；来源须在服务端允许范围内" error={errors.downloadUrl}><input type="text" value={fields.downloadUrl} onChange={e => change("downloadUrl", e.target.value)} /></Field>
        {Object.keys(errors).length ? <p role="alert">请检查标出的字段。大小、摘要和版本范围须符合登记要求。</p> : null}
        <p>登记后固定元数据不可修改，初始不设为推荐版本。</p><div className={styles.actions}><Button type="submit">{busy ? "正在登记和核实…" : "登记并核实"}</Button><Button asChild variant="outline"><Link href={kernelPagePath} prefetch={false}>返回目录</Link></Button></div>
      </fieldset>
    </form></Card>
    {failure ? <p role="alert">{kernelMessage(failure)}</p> : null}
  </>;
}
function IntentSummary({ intent }: { intent: KernelIntent }) { return <dl className={styles.summary}><dt>发行编号</dt><dd>{intent.key}</dd><dt>内核版本</dt><dd>{intent.payload.version}</dd><dt>平台</dt><dd>{intent.payload.os} / {intent.payload.arch}</dd><dt>兼容客户端</dt><dd>[{intent.payload.minClientVersion}, {intent.payload.maxClientVersionExclusive || "不限"})</dd><dt>制品大小</dt><dd>{intent.payload.size} 字节</dd><dt>SHA256</dt><dd>{intent.payload.sha256}</dd><dt>下载地址</dt><dd>{intent.payload.downloadUrl}</dd></dl>; }
function Field({ label, hint, error, children }: { label: string; hint?: string; error?: string; children: ReactNode }) {
  const id = useId(), help = [hint ? id + "-hint" : "", error ? id + "-error" : ""].filter(Boolean).join(" ");
  const control = isValidElement(children) ? cloneElement(children as ReactElement<{ id?: string; "aria-invalid"?: boolean; "aria-describedby"?: string }>, { id, "aria-invalid": !!error, "aria-describedby": help || undefined }) : children;
  return <div className={styles.field}><label htmlFor={id}>{label}</label>{control}{hint ? <small id={id + "-hint"}>{hint}</small> : null}{error ? <small id={id + "-error"} role="alert">{error}</small> : null}</div>;
}
function PlatformFields({ os, arch, setOS, setArch }: { os: KernelQuery["os"]; arch: KernelQuery["arch"]; setOS: (os: KernelQuery["os"]) => void; setArch: (arch: KernelQuery["arch"]) => void }) { return <><Field label="操作系统"><select value={os} onChange={e => setOS(e.target.value as KernelQuery["os"])}>{["windows", "linux", "darwin"].map(value => <option key={value}>{value}</option>)}</select></Field><Field label="架构"><select value={arch} onChange={e => setArch(e.target.value as KernelQuery["arch"])}>{["amd64", "arm64"].map(value => <option key={value}>{value}</option>)}</select></Field></>; }
