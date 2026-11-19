"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { modelRouteRequest, ModelRouteError, type ModelRouteScope } from "@/lib/api/model-route-pricing";
import { searchKeyResponse, searchPlatformResponse, searchPublicationResponse, searchSelectionResponse, searchSettingsResponse } from "@/lib/contracts/web-search";
import { useResourcePending } from "../resources/resource-pending";

type Props = { scope: ModelRouteScope; member: string; configure: boolean; owned: boolean; platform: boolean; canWrite: boolean };

const pendingSchema = z.strictObject({ key: z.uuid(), action: z.enum(["enterprise", "platform", "platform-key", "owned-key"]) });
type Action = z.infer<typeof pendingSchema>["action"];
const readSchemas = { enterprise: searchSettingsResponse, platform: searchPlatformResponse, "owned-key": searchKeyResponse, "platform-key": searchKeyResponse };
const writeSchemas = { ...readSchemas, enterprise: searchSelectionResponse, platform: searchPublicationResponse };

function readSetting<T>(scope: ModelRouteScope, action: Action, schema: z.ZodType<T>, signal?: AbortSignal) {
  return modelRouteRequest(scope, "/search/" + action, schema, { signal });
}

export function useSearchSettingsAvailable(scope: ModelRouteScope, configure: boolean, owned: boolean, platform: boolean) {
  return useQuery({
    queryKey: ["search-services-available", scope, configure, owned, platform],
    queryFn: async ({ signal }) => {
      const reads: Promise<unknown>[] = [];
      if (configure) reads.push(readSetting(scope, "enterprise", searchSettingsResponse, signal));
      if (owned) reads.push(readSetting(scope, "owned-key", searchKeyResponse, signal));
      if (platform) reads.push(readSetting(scope, "platform", searchPlatformResponse, signal));
      await Promise.any(reads);
      return true;
    },
    enabled: configure || owned || platform, retry: false, gcTime: 0, refetchOnWindowFocus: false,
  });
}

export function SearchServiceSettings({ scope, member, configure, owned, platform, canWrite }: Props) {
  const client = useQueryClient();
  const select = useQuery({
    queryKey: ["search-selection", scope, member],
    queryFn: ({ signal }) => readSetting(scope, "enterprise", searchSettingsResponse, signal),
    enabled: configure, retry: false, gcTime: 0,
  });
  const publication = useQuery({
    queryKey: ["search-platform", scope, member],
    queryFn: ({ signal }) => readSetting(scope, "platform", searchPlatformResponse, signal),
    enabled: platform, retry: false, gcTime: 0,
  });
  const [selection, setSelection] = useState("");
  const [notice, setNotice] = useState("");
  const [missing, setMissing] = useState<string>();
  const [busy, setBusy] = useState(false);
  const running = useRef<AbortController | null>(null);
  useEffect(() => () => running.current?.abort(), []);
  const pending = useResourcePending({ expectedUserId: scope.userId, expectedOrganizationId: scope.organizationId },
    ["search-settings", member], pendingSchema, { storage: "session" });
  const locked = busy || !pending.ready || Boolean(pending.command) || !member;

  async function refreshConfiguration(action: Action) {
    if (action === "platform") {
      if (configure) void select.refetch();
      return (await publication.refetch()).isSuccess;
    }
    const reads: Promise<unknown>[] = [];
    if (configure) reads.push(select.refetch({ throwOnError: true }));
    if (platform) reads.push(publication.refetch({ throwOnError: true }));
    if (action === "owned-key" || action === "platform-key") {
      reads.push(client.refetchQueries({ queryKey: ["search-key", scope, member, action], exact: true }, { throwOnError: true }));
    }
    return (await Promise.allSettled(reads)).every(result => result.status === "fulfilled");
  }

  async function abandon(command: z.infer<typeof pendingSchema>) {
    if (running.current || busy) return;
    const controller = new AbortController();
    running.current = controller;
    setBusy(true);
    try {
      const refreshed = await refreshConfiguration(command.action);
      if (controller.signal.aborted) return;
      if (!refreshed) {
        setNotice("暂时无法读取当前配置，请稍后再核对。");
        return;
      }
      pending.clear(command);
      setMissing(undefined);
      setSelection("");
      setNotice("已放弃核对，请确认当前配置后重新操作。");
    } finally {
      if (running.current === controller) running.current = null;
      if (!controller.signal.aborted) setBusy(false);
    }
  }

  async function update(action: Action, input: object, recover = false) {
    if (running.current || !member || !pending.ready) return false;
    const prior = pending.read();
    if (recover && !prior || !recover && prior) return false;
    const command = prior ?? { key: crypto.randomUUID(), action };
    const controller = new AbortController();
    running.current = controller;
    setBusy(true);
    setNotice("");
    try {
      if (!recover) pending.persist(command);
      const schema = writeSchemas[command.action];
      if (recover) {
        await modelRouteRequest(scope, "/search/" + command.action + "/receipts/" + command.key, schema as z.ZodType<unknown>, { signal: controller.signal });
      } else {
        await modelRouteRequest(scope, "/search/" + command.action, schema as z.ZodType<unknown>, {
          method: "POST",
          headers: { "Content-Type": "application/json", "Idempotency-Key": command.key },
          body: JSON.stringify({ ...input, expectedMemberId: member }), signal: controller.signal,
        });
      }
      if (controller.signal.aborted) return false;
      pending.clear(command);
      if (command.action === "enterprise") setSelection("");
      const refreshed = await refreshConfiguration(command.action);
      if (!controller.signal.aborted) setNotice(refreshed ? "已保存，当前配置已重新读取。" : "操作已确认，当前配置暂时无法读取。");
      return !controller.signal.aborted;
    } catch (error) {
      if (controller.signal.aborted) return false;
      if (!recover && error instanceof ModelRouteError && [400,403,404,409].includes(error.status)) {
        if (pending.read()?.key === command.key) pending.clear(command);
        setNotice(error.status === 409 ? "配置有变化，请重新读取后再保存。" : "保存被拒绝，请核对权限和输入。");
      } else if (recover && error instanceof ModelRouteError && error.code === "NOT_FOUND") {
        setMissing(command.key);
        setNotice("尚未找到原操作。重新保存前请先重新读取当前配置。");
      } else {
        setNotice("操作结果待核实，请查看原操作。");
      }
      return false;
    } finally {
      if (running.current === controller) running.current = null;
      if (!controller.signal.aborted) setBusy(false);
    }
  }

  const chosen = selection || select.data?.result.selection.source || "";
  return <section className="min-w-0 space-y-4" aria-labelledby="search-settings-title">
    <h2 id="search-settings-title" className="text-base font-semibold">搜索服务</h2>
    {pending.command && <Button type="button" variant="outline" disabled={busy} onClick={() => void update(pending.command!.action, {}, true)}>核对原操作</Button>}
    {pending.command && missing === pending.command.key && <Button type="button" variant="ghost" disabled={busy} onClick={() => void abandon(pending.command!)}>放弃核对</Button>}
    {notice && <p role="status">{notice}</p>}
    {pending.error && <p role="alert">原操作记录暂时无法读取。</p>}
    {configure && <div className="space-y-2">
      <h3 className="font-medium">Chat 查公开资料</h3>
      {select.data ? <>
        <p>当前选用：{select.data.result.options.find(item => item.source === select.data.result.selection.source)?.name ??
          (select.data.result.selection.source ? "原服务暂不可用，请切换" : "尚未选择")}</p>
        <label className="block">选择搜索服务<Select value={chosen} onChange={event => setSelection(event.target.value)} disabled={locked}>
          <option value="">请选择</option>
          {select.data.result.options.map(item => <option key={item.source} value={item.source}>{item.name} · {item.source === "owned" ? "企业自有" : "平台提供"} · 每日最多 {item.limit} 次</option>)}
        </Select></label>
        <Button type="button" disabled={!selection || locked || selection === select.data.result.selection.source} onClick={() => void update("enterprise", { revision: select.data!.result.selection.revision, source: selection })}>保存选择</Button>
      </> : <p role="status">{select.isPending ? "正在读取搜索服务…" : "暂时无法读取搜索服务。"}</p>}
      <Button type="button" variant="ghost" disabled={busy} onClick={() => void select.refetch()}>重新读取</Button>
    </div>}
    {owned && <SearchKeyEditor scope={scope} member={member} action="owned-key" locked={locked} update={update} />}
    {platform && publication.data && <>
      <SearchKeyEditor scope={scope} member={member} action="platform-key" locked={locked || !canWrite || !publication.data.canWrite} update={update} />
      <SearchPlatformEditor key={publication.data.result.revision} value={publication.data.result} organizations={publication.data.organizations ?? []}
        locked={locked || !canWrite || !publication.data.canWrite} update={update} />
    </>}
    {platform && publication.isError && <><p role="alert">暂时无法读取平台搜索配置。</p><Button variant="outline" onClick={() => void publication.refetch()}>重新读取</Button></>}
  </section>;
}

type Update = (action: Action, input: object) => Promise<boolean>;

function SearchKeyEditor({ scope, member, action, locked, update }: { scope: ModelRouteScope; member: string; action: "owned-key" | "platform-key"; locked: boolean; update: Update }) {
  const status = useQuery({
    queryKey: ["search-key", scope, member, action],
    queryFn: ({ signal }) => readSetting(scope, action, searchKeyResponse, signal),
    retry: false, gcTime: 0,
  });
  const [apiKey, setKey] = useState("");
  const value = !status.isError ? status.data?.result : undefined;
  async function save(enabled: boolean) {
    if (!value || locked || (enabled && !apiKey.trim())) return;
    const key = enabled ? apiKey.trim() : undefined;
    setKey("");
    await update(action, { revision: value.revision, apiKey: key, enabled });
  }
  return <div className="space-y-2">
    <h3 className="font-medium">{action === "platform-key" ? "平台 Brave Key" : "企业自有 Brave Key"}</h3>
    <p>{value ? value.enabled ? "Key 已保存，调用权限未验证" : value.keyPresent ? "已停用" : "尚未保存 Key" : "当前无法读取 Key 状态"}</p>
    {value?.updatedAt && <p className="text-sm text-muted-foreground">更新时间 {new Intl.DateTimeFormat("zh-CN", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value.updatedAt))}</p>}
    <label className="block">Brave API Key<Input type="password" value={apiKey} autoComplete="off" onChange={event => setKey(event.target.value)} disabled={locked || !value} maxLength={4096} /></label>
    <div className="flex flex-wrap gap-2">
      <Button type="button" disabled={locked || !value || !apiKey.trim()} onClick={() => void save(true)}>{value?.keyPresent ? "更换 Key" : "保存 Key"}</Button>
      {value?.enabled && <Button type="button" variant="outline" disabled={locked} onClick={() => void save(false)}>停用</Button>}
      <Button type="button" variant="ghost" onClick={() => void status.refetch()}>重新读取 Key 状态</Button>
    </div>
  </div>;
}

type Publication = z.infer<typeof searchPlatformResponse>["result"];

function SearchPlatformEditor({ value, organizations, locked, update }: { value: Publication; organizations: string[]; locked: boolean; update: Update }) {
  const [draft, setDraft] = useState(() => ({
    ...value, name: value.name || "平台 Brave 搜索", country: value.country || "US", language: value.language || "en",
    organizations: value.organizations ?? [],
  }));
  const include = (org: string, on: boolean) => setDraft({ ...draft, organizations: on ? [...draft.organizations, org] : draft.organizations.filter(item => item !== org) });
  function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!locked) void update("platform", draft);
  }
  return <form className="space-y-2" onSubmit={save}>
    <h3 className="font-medium">配置平台开放</h3>
    <p>厂商费用由平台承担，企业使用搜索不扣 AI 点数。</p>
    <fieldset disabled={locked} className="grid gap-2 sm:grid-cols-2">
      <label>显示名称<Input value={draft.name} onChange={event => setDraft({ ...draft, name: event.target.value })} maxLength={128} required /></label>
      <label>地区<Select value={draft.country} onChange={event => setDraft({ ...draft, country: event.target.value })}><option value="US">美国</option><option value="ALL">不限</option></Select></label>
      <label>语言<Select value={draft.language} onChange={event => setDraft({ ...draft, language: event.target.value })}><option value="en">英文</option><option value="zh-hans">简体中文</option></Select></label>
      <label>每次成本上限（百万分之一美元）<Input type="number" min={1} max={1000000} required value={draft.maxUsdMicros || ""} onChange={event => setDraft({ ...draft, maxUsdMicros: Number(event.target.value) })} /></label>
      <label>平台每日搜索次数上限<Input type="number" min={1} max={10000} required value={draft.totalLimit || ""} onChange={event => setDraft({ ...draft, totalLimit: Number(event.target.value) })} /></label>
      <label>每个企业每日搜索次数上限<Input type="number" min={1} max={draft.totalLimit || 10000} required value={draft.tenantLimit || ""} onChange={event => setDraft({ ...draft, tenantLimit: Number(event.target.value) })} /></label>
      <div className="space-y-1 sm:col-span-2"><p>开放给指定企业</p>{organizations.map(org => <label key={org} className="block break-all"><input type="checkbox" checked={draft.organizations.includes(org)} onChange={event => include(org, event.target.checked)} /> {org}</label>)}</div>
      <label className="sm:col-span-2"><input type="checkbox" checked={draft.published} disabled={!draft.organizations.length && !draft.published} onChange={event => setDraft({ ...draft, published: event.target.checked })} /> 开放服务</label>
    </fieldset>
    <Button type="submit" disabled={locked}>{draft.published ? "保存并开放" : "保存草稿"}</Button>
  </form>;
}
