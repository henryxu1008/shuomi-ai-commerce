"use client";
import {ShopDiagnosisFlow} from "./shop-diagnosis-flow";
import Link from "next/link";
import {ProductSelectionFlow} from "./product-selection-flow";
import {PODArtworkFlow} from "./pod-artwork-flow";
import { useEffect, useRef, useState } from "react";
import { z } from "zod";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { ConsolePage, ConsoleState } from "../console/console-page";
import {
  configurationRequest,
  ConfigurationError,
  type ConfigurationScope,
} from "@/lib/api/agent-configuration";
import {
  catalogEntrySchema,
  catalogPageSchema,
  configReceiptSchema,
  recentRunsSchema,
  templateSchema,
  templatesPageSchema,
  configVersion,
  type AgentTemplate,
  type CatalogEntry,
} from "@/lib/contracts/agent-configuration";
import { basesSchema, knowledgeRequest } from "@/lib/api/knowledge";
import styles from "./agents.module.css";
import { ExecutionModeSettings } from "./execution-mode-settings";
import { ImageAgentPage } from "./image-agent-page";
import { configurationError, useConfigurationRead as useRead } from "./agent-configuration-hooks";
import { PrivateAgentCards } from "./private-agent-page";
import { ContentSelectionFields } from "./content-selection-fields";
import { contentSelectionSchema, contentSelectionLabel, type ContentSelection } from "@/lib/contracts/content-selection";

const labels: Record<string, string> = {
  NOT_ENABLED: "尚未启用",
  ENABLED: "已启用",
  DISABLED: "已停用",
  AVAILABLE: "可用",
  NEEDS_CONFIGURATION: "需要配置",
  REQUIRES_AUTHORIZATION: "需要授权",
  UNAVAILABLE: "不可用",
  NOT_SUPPORTED: "不支持",
  REQUIRED: "必需",
  OPTIONAL: "可选",
  running: "执行中",
  interrupted: "等待继续",
  human_review_required: "待人工审核",
  stopped: "已停止",
};
const capabilityNames: Record<string, string> = {
  "text.generate": "文本生成",
  "product.source-evidence": "采集证据读取",
  "listing.materials.inspect": "草稿、店铺规则与已批准资产读取",
  "shop.operations-diagnosis.inspect":"店铺经营事实读取与诊断",
 "product.selection.inspect": "候选商品事实读取与对比",
  "knowledge.context": "企业知识引用",
  "image.generate": "图片生成",
  "platform.write": "平台写入",
};
type Intent = {
  path: string;
  payload: unknown;
  method: "POST" | "PUT";
  key: string;
  revision?: string;
  absent?: boolean;
};
export function AgentPage({
  mode,
  id,
  podAvailable = false,
}: {
  mode: "market" | "mine";
  id?: string;
  podAvailable?: boolean;
}) {
  const ctx = useWorkbenchContext();
  if (ctx.error || ctx.blockingError)
    return (
      <ConsoleState kind="error" title="企业上下文不可用">
        <Button onClick={() => void ctx.retry()}>重新确认</Button>
      </ConsoleState>
    );
  if (ctx.isLoading || ctx.isSwitching)
    return <ConsoleState kind="loading" title="正在确认当前企业" />;
  if (!ctx.user || !ctx.effectiveOrganization || ctx.selectionRequired)
    return <ConsoleState kind="unavailable" title="请先选择企业" />;
  return (
    id === "product.image.agent" ? <ImageAgentPage
      key={`${ctx.user.id}:${ctx.effectiveOrganization.id}:${[...ctx.permissions].sort().join(":")}`}
      scope={{userId:ctx.user.id,organizationId:ctx.effectiveOrganization.id}}
      organization={ctx.effectiveOrganization.name}
    /> :
    <ScopedPage
      key={`${ctx.user.id}:${ctx.effectiveOrganization.id}:${[...ctx.permissions].sort().join(":")}:${mode}:${id ?? ""}`}
      scope={{
        userId: ctx.user.id,
        organizationId: ctx.effectiveOrganization.id,
      }}
      organization={ctx.effectiveOrganization.name}
      mode={mode}
      id={id}
      podAvailable={podAvailable}
    />
  );
}
function ScopedPage({
  scope,
  organization,
  mode,
  id,
  podAvailable = false,
}: {
  scope: ConfigurationScope;
  organization: string;
  mode: "market" | "mine";
  id?: string;
  podAvailable?: boolean;
}) {
  const agentId = id ?? "product.title.agent";
  const [filter, setFilter] = useState(""),
    [after, setAfter] = useState(""),
    [nonce, setNonce] = useState(0),
    [intent, setIntent] = useState<Intent | null>(null),
    [busy, setBusy] = useState(false),
    [failure, setFailure] = useState<unknown>(null),
    [message, setMessage] = useState("");
  const [templateAfter, setTemplateAfter] = useState(""),
    [recentAfter, setRecentAfter] = useState("");
  const [disable, setDisable] = useState<CatalogEntry | null>(null),
    [editor, setEditor] = useState<AgentTemplate | "new" | null>(null),
    [chosen, setChosen] = useState<AgentTemplate | null>(null),
    [history, setHistory] = useState("");
  const active = useRef(true),
    flight = useRef(false),
    abort = useRef<AbortController | null>(null);
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
      abort.current?.abort();
    };
  }, []);
  const path = id
    ? agentId
    : `${mode}?pageSize=20${filter ? "&activation=" + filter : ""}${after ? "&cursor=" + after : ""}`;
  const list = useRead<CatalogEntry | z.infer<typeof catalogPageSchema>>(
    scope,
    path,
    id ? catalogEntrySchema : catalogPageSchema,
    nonce,
  );
  const data = list.data;
  const detail = id && data ? (data as CatalogEntry) : null;
  const entries =
    !id && data ? (data as z.infer<typeof catalogPageSchema>).items : [];
  const templates = useRead(
    scope,
    `${agentId}/templates?pageSize=100${templateAfter ? "&cursor=" + templateAfter : ""}`,
    templatesPageSchema,
    nonce,
    !!detail && detail.agent.activation !== "NOT_ENABLED" && !["product.selection.agent","shop.operations-diagnosis.agent"].includes(agentId),
  );
  const recent = useRead(
    scope,
    `${agentId}/recent-runs?pageSize=20${recentAfter ? "&cursor=" + recentAfter : ""}`,
    recentRunsSchema,
    nonce,
    !!detail?.canReadRuns && agentId!=="pod.design.agent" && !["product.selection.agent","shop.operations-diagnosis.agent"].includes(agentId),
  );
  const defaultRef = detail?.agent.defaultTemplate;
  const defaultTemplate = useRead(
    scope,
    defaultRef
      ? `${agentId}/templates/${defaultRef.templateId}/revisions/${defaultRef.revision}`
      : "",
    templateSchema,
    nonce,
    !!defaultRef,
  );
  const old = useRead(
    scope,
    chosen && history && configVersion.safeParse(history).success
      ? `${agentId}/templates/${chosen.templateId}/revisions/${history}`
      : "",
    templateSchema,
    nonce,
    !!chosen && !!history && configVersion.safeParse(history).success,
  );
  async function send(command: Intent) {
    if (flight.current || !active.current) return;
    flight.current = true;
    setBusy(true);
    setFailure(null);
    setIntent(command);
    const controller = new AbortController();
    abort.current = controller;
    const headers = new Headers({
      "Content-Type": "application/json",
      "Idempotency-Key": command.key,
    });
    if (command.absent) headers.set("If-None-Match", "*");
    else if (command.revision) headers.set("If-Match", `"${command.revision}"`);
    try {
      await configurationRequest(scope, command.path, configReceiptSchema, {
        method: command.method,
        headers,
        body: JSON.stringify(command.payload),
        signal: controller.signal,
      });
      if (!active.current) return;
      setIntent(null);
      setEditor(null);
      setDisable(null);
      setChosen(null);
      setHistory("");
      setMessage("操作已保存，正在读取当前状态。");
      setNonce((n) => n + 1);
    } catch (error) {
      if (!active.current) return;
      setFailure(error);
      setMessage("");
      if (
        !(error instanceof ConfigurationError) ||
        error.code !== "OUTCOME_UNKNOWN"
      )
        setIntent(null);
    } finally {
      flight.current = false;
      if (active.current) setBusy(false);
    }
  }
  function mutate(
    path: string,
    payload: unknown = {},
    revision?: string,
    absent = false,
    method: "POST" | "PUT" = "POST",
  ) {
    if (intent || busy) return;
    void send({
      path,
      payload,
      revision,
      absent,
      method,
      key: crypto.randomUUID(),
    });
  }
  const mutationDisabled = busy || !!intent;
  const contentOperations = entries.filter(entry => ["product.title.agent", "product.content.agent"].includes(entry.agent.agentId));
  function contentCard() {
    return <Card key="product-content" className={styles.agentCard}>
      <h2>商品内容与本地化</h2>
      <p>完善标准商品的标题、描述、卖点与关键词，按明确目标语言处理内容。</p>
      {contentOperations.map(entry => {
        const title = entry.agent.agentId === "product.title.agent";
        const operation = title ? "标题优化" : "内容完善与本地化";
        const configuration = title ? "标题优化配置与历史" : "内容本地化配置与历史";
        return <section key={entry.agent.agentId} aria-label={operation} className="space-y-2 pt-3">
          <h3>{operation}</h3>
          <div className={styles.cardActions}>
            <span className={styles.status} data-active={entry.agent.activation === "ENABLED"}>{labels[entry.agent.activation]}</span>
            <Link href={`/workbench/agents/mine/${entry.agent.agentId}`}>{configuration}</Link>
            {entry.canConfigure && entry.agent.activation !== "ENABLED" ? <Button disabled={mutationDisabled} onClick={() => mutate(`${entry.agent.agentId}/enable`, {}, entry.agent.revision || undefined, entry.agent.activation === "NOT_ENABLED")}>启用到当前企业</Button> : null}
            {mode === "mine" && entry.canUse ? <Link href={`/workbench/supply/acquisition?agent_kind=${title ? "title" : "content"}`}>进入{operation} →</Link> : null}
            {mode === "mine" && entry.canConfigure && entry.agent.activation === "ENABLED" ? <Button variant="outline" disabled={mutationDisabled} onClick={() => setDisable(entry)}>停用{operation}</Button> : null}
          </div>
          {!entry.canUse ? <p className="text-muted-foreground">当前不能执行{operation}</p> : null}
          {!entry.canUse && entry.capabilities.filter(cap => cap.support === "REQUIRED" && cap.readiness !== "AVAILABLE").map(cap => <p key={cap.id} className="text-muted-foreground"><strong>{capabilityNames[cap.id]}</strong>：<span>{cap.reason}</span></p>)}
        </section>;
      })}
    </Card>;
  }
  function card(entry: CatalogEntry) {
    return (
      <Card key={entry.agent.agentId} className={styles.agentCard}>
        <h2>{entry.name}</h2>
        <p>{entry.description}</p>
        <p className={styles.support}>{entry.agent.agentId === "pod.design.agent" ? "原创图案 · 保存后布局与效果图" : entry.agent.agentId === "product.image.agent" ? "主图与详情图 · 人工选择批准" : entry.agent.agentId === "product.content.agent" ? "标题、描述、卖点、SEO关键词 · 内容本地化" : "标题优化 · 人工审核"}</p>
        <div className={styles.cardActions}>
          <span
            className={styles.status}
            data-active={entry.agent.activation === "ENABLED"}
          >
            {labels[entry.agent.activation]}
          </span>
          {entry.canConfigure && entry.agent.activation !== "ENABLED" ? (
            <Button
              disabled={mutationDisabled}
              onClick={() =>
                mutate(
                  `${entry.agent.agentId}/enable`,
                  {},
                  entry.agent.revision || undefined,
                  entry.agent.activation === "NOT_ENABLED",
                )
              }
            >
              启用到当前企业
            </Button>
          ) : (
            <Link href={`/workbench/agents/mine/${entry.agent.agentId}`}>
              查看配置 →
            </Link>
          )}
        </div>
        {mode === "mine" && (
          <div className={styles.cardActions}>
            <Link href={`/workbench/agents/mine/${entry.agent.agentId}`}>
              能力与模板
            </Link>
            {entry.canUse ? (
              <Link href={["pod.design.agent","product.selection.agent","shop.operations-diagnosis.agent"].includes(entry.agent.agentId)?"/workbench/agents/mine/"+entry.agent.agentId:"/workbench/supply/acquisition"}>进入使用 →</Link>
            ) : (
              <span className="text-muted-foreground">当前不能执行</span>
            )}
          </div>
        )}
        {!entry.canUse &&
          entry.capabilities
            .filter(
              (cap) =>
                cap.support === "REQUIRED" && cap.readiness !== "AVAILABLE",
            )
            .map((cap) => (
              <p key={cap.id} className="text-muted-foreground">
                <strong>{capabilityNames[cap.id]}</strong>：
                <span>{cap.reason}</span>
              </p>
            ))}
      </Card>
    );
  }
  return (
    <ConsolePage
      title={
        id ? "智能体配置" : mode === "market" ? "智能体市场" : "我的智能体"
      }
      description={
        <>
          当前企业 · {organization}。
          {mode === "mine"
            ? "管理企业智能体，查看配置状态和本人最近任务。"
            : "选择适合业务场景的智能体，显式启用到当前企业。"}
        </>
      }
      breadcrumbs={[
        { label: "智能市场" },
        {
          label: id
            ? "我的智能体"
            : mode === "market"
              ? "智能体市场"
              : "我的智能体",
          href: id ? "/workbench/agents/mine" : undefined,
        },
        ...(id ? [{ label: "智能体配置" }] : []),
      ]}
      actions={
        <Link
          href={
            mode === "market"
              ? "/workbench/agents/mine"
              : "/workbench/agents/market"
          }
        >
          {mode === "market" ? "我的智能体 →" : "前往智能体市场 →"}
        </Link>
      }
    >
      {mode === "mine" && !id ? <PrivateAgentCards scope={scope}/> : null}
      {failure != null && (
        <div role="alert" className={styles.notice}>
          {configurationError(failure)}
          {intent && (
            <Button disabled={busy} onClick={() => void send(intent)}>
              重试同一次操作
            </Button>
          )}
          <Button
            variant="outline"
            disabled={busy}
            onClick={() => {
              setNonce((n) => n + 1);
              if (!intent) setFailure(null);
            }}
          >
            刷新当前状态
          </Button>
        </div>
      )}
      {message && <p role="status">{message}</p>}
      {!data && !list.error ? (
        <ConsoleState kind="loading" title="正在读取企业智能体" />
      ) : list.error ? (
        <ConsoleState kind="error" title="智能体配置不可用">
          <p>{configurationError(list.error)}</p>
          <Button onClick={() => setNonce((n) => n + 1)}>重新读取</Button>
        </ConsoleState>
      ) : id && detail ? (
        <>
          <Card className={styles.detailHead}>
            <div>
              <h2>{agentId === "product.title.agent" ? "商品内容与本地化 · 标题优化配置" : detail.name}</h2>
              <p>{detail.description}</p>
              <p>
                {labels[detail.agent.activation]} · {detail.definitionVersion}
              </p>
              {["product.title.agent", "product.content.agent"].includes(agentId) ? <nav aria-label="商品内容操作配置" className={styles.actions}>
                <Link href="/workbench/agents/mine/product.title.agent">标题优化配置与历史</Link>
                <Link href="/workbench/agents/mine/product.content.agent">内容本地化配置与历史</Link>
              </nav> : null}
            </div>
            <div className={styles.actions}>
              {detail.canUse && !["pod.design.agent","product.selection.agent","shop.operations-diagnosis.agent"].includes(agentId) && (
                <Link href="/workbench/supply/acquisition">
                  选择商品并进入使用 →
                </Link>
              )}
              {detail.canConfigure &&
                (detail.agent.activation === "ENABLED" ? (
                  <Button
                    variant="outline"
                    disabled={mutationDisabled}
                    onClick={() => setDisable(detail)}
                  >
                    停用智能体
                  </Button>
                ) : (
                  <Button
                    disabled={mutationDisabled}
                    onClick={() =>
                      mutate(
                        `${agentId}/enable`,
                        {},
                        detail.agent.revision || undefined,
                        detail.agent.activation === "NOT_ENABLED",
                      )
                    }
                  >
                    启用到当前企业
                  </Button>
                ))}
            </div>
          </Card>
          <div className={styles.detailGrid}>
            <Card className={styles.panel}>
              <h2>当前能力</h2>
              <p>配置状态不代表执行许可，执行时重新确认当前权限与预算。</p>
              {detail.capabilities.map((cap) => (
                <section key={cap.id} className={styles.capability}>
                  <div>
                    <h3>
                      {capabilityNames[cap.id]} · {labels[cap.support]}
                    </h3>
                    <p>{cap.reason}</p>
                  </div>
                  <span className={styles.capabilityStatus} data-readiness={cap.support === "NOT_SUPPORTED" ? cap.support : cap.readiness}>
                    {
                      labels[
                        cap.support === "NOT_SUPPORTED"
                          ? cap.support
                          : cap.readiness
                      ]
                    }
                  </span>
                </section>
              ))}
            </Card>
            <div className={styles.detailSidebar}>
            {detail.agent.activation !== "NOT_ENABLED" ? <ExecutionModeSettings scope={scope} subject={{kind:"registered_agent",id:detail.agent.agentId}}/> : null}
          <Card className={styles.panel}>
              <div className={styles.cardActions}>
                <h2>默认模板</h2>
                {detail.canConfigure && defaultRef && (
                  <Button
                    variant="outline"
                    disabled={mutationDisabled}
                    onClick={() =>
                      mutate(
                        `${agentId}/default-template`,
                        { templateId: null, revision: null },
                        detail.agent.revision,
                        false,
                        "PUT",
                      )
                    }
                  >
                    清除默认
                  </Button>
                )}
              </div>
              {!defaultRef ? (
                <p>{["product.selection.agent","shop.operations-diagnosis.agent"].includes(agentId)?"分析使用本次明确目标，不使用内容、刊登或图片模板。":"暂无默认模板；执行时仍可显式选择平台和是否引用知识。"}</p>
              ) : defaultTemplate.error ? (
                <p role="alert">
                  默认模板当前不可用。执行前须明确选择其他模板或不使用模板。
                </p>
              ) : defaultTemplate.data ? (
                <>
                  <h3>
                    {defaultTemplate.data.name} · v
                    {defaultTemplate.data.version}
                  </h3>
                  <p>素材平台：{defaultTemplate.data.targetPlatform}</p>
                  {defaultTemplate.data.agentId==="pod.design.agent"&&<p>创作风格：{defaultTemplate.data.artwork.style||"未指定"} · 约束：{defaultTemplate.data.artwork.constraints||"未指定"}</p>}
                  {defaultTemplate.data.agentId === "product.content.agent" && <p>{contentSelectionLabel(defaultTemplate.data.content.selection)}</p>}
                  <p>
                    {defaultTemplate.data.knowledgeAvailability ===
                    "UNAVAILABLE"
                      ? "默认知识当前不可用，执行前须重新选择或明确不使用知识。"
                      : defaultTemplate.data.defaultKnowledgeBaseId
                        ? "保存了默认企业知识；执行时仍需明确确认。"
                        : "默认不引用企业知识。"}
                  </p>
                  <p>模板更新不会自动移动此精确版本。</p>
                </>
              ) : (
                <p role="status">正在读取默认版本…</p>
              )}
            </Card>
            </div>
          </div>
            {agentId==="shop.operations-diagnosis.agent"&&<ShopDiagnosisFlow scope={scope} entry={detail} nonce={nonce}/>}
            {agentId==="product.selection.agent"&&<ProductSelectionFlow scope={scope} entry={detail} nonce={nonce}/>}
            {agentId==="pod.design.agent"&&<PODArtworkFlow scope={scope} entry={detail} templates={templates.data?.items.filter(t=>t.agentId==="pod.design.agent")??[]} nonce={nonce} podAvailable={podAvailable}/>}
          <Card className={styles.panel}>
            <div className={styles.cardActions}>
              <h2>{["product.selection.agent","shop.operations-diagnosis.agent"].includes(agentId)?"本次分析参数":"版本化模板"}</h2>
              {detail.canConfigure && !["product.selection.agent","shop.operations-diagnosis.agent"].includes(agentId) && (
                <Button
                  disabled={
                    mutationDisabled ||
                    detail.agent.activation === "NOT_ENABLED"
                  }
                  onClick={() => setEditor("new")}
                >
                  新建模板
                </Button>
              )}
            </div>
            {["product.selection.agent","shop.operations-diagnosis.agent"].includes(agentId)?<p>商品、店铺与本次分析目标在相应操作面板选择，第一版不开放命名模板。</p>:templates.error ? (
              <p role="alert">{configurationError(templates.error)}</p>
            ) : templates.data?.items.length ? (
              <div className={styles.templateList}>
                {templates.data.items.map((t) => (
                  <section key={t.templateId} className={styles.templateRow}>
                    <div>
                      <h3>
                        {t.name} · v{t.version}
                      </h3>
                      <p>
                        {t.targetPlatform} ·{" "}
                        {t.lifecycle === "ACTIVE" ? "可选用" : "已归档"}
                      </p>
                      {t.agentId === "product.content.agent" && <p>{contentSelectionLabel(t.content.selection)}</p>}
                      <p>
                        {t.knowledgeAvailability === "UNAVAILABLE"
                          ? "默认知识不可用"
                          : t.defaultKnowledgeBaseId
                            ? "含默认知识选择"
                            : "不引用企业知识"}
                      </p>
                    </div>
                    <div className={styles.actions}>
                      <Button
                        variant="outline"
                        onClick={() => {
                          setChosen(t);
                          setHistory(t.version);
                        }}
                      >
                        查看版本
                      </Button>
                      {detail.canConfigure && t.lifecycle === "ACTIVE" && (
                        <>
                          <Button
                            variant="outline"
                            disabled={mutationDisabled}
                            onClick={() => setEditor(t)}
                          >
                            编辑
                          </Button>
                          <Button
                            disabled={mutationDisabled}
                            onClick={() =>
                              mutate(
                                `${agentId}/default-template`,
                                {
                                  templateId: t.templateId,
                                  revision: t.version,
                                },
                                detail.agent.revision,
                                false,
                                "PUT",
                              )
                            }
                          >
                            设为默认 v{t.version}
                          </Button>
                          <Button
                            variant="outline"
                            disabled={mutationDisabled}
                            onClick={() =>
                              mutate(
                                `${agentId}/templates/${t.templateId}/archive`,
                                {},
                                t.revision,
                              )
                            }
                          >
                            归档
                          </Button>
                        </>
                      )}
                    </div>
                  </section>
                ))}
              </div>
            ) : (
              <p>
                {detail.agent.activation === "NOT_ENABLED"
                  ? "请先启用到当前企业。"
                  : "当前企业尚无模板。"}
              </p>
            )}
            {(templateAfter || templates.data?.nextCursor) && (
              <nav aria-label="模板分页" className={styles.actions}>
                {templateAfter && (
                  <Button
                    variant="outline"
                    onClick={() => setTemplateAfter("")}
                  >
                    回到模板首页
                  </Button>
                )}
                {templates.data?.nextCursor && (
                  <Button
                    variant="outline"
                    onClick={() => setTemplateAfter(templates.data!.nextCursor)}
                  >
                    下一页模板
                  </Button>
                )}
              </nav>
            )}
            {chosen && (
              <section className={styles.version}>
                <label>
                  查看不可变版本
                  <Input
                    value={history}
                    onChange={(e) => setHistory(e.target.value)}
                    inputMode="numeric"
                  />
                </label>
                {old.error ? (
                  <p role="alert">该版本在当前企业不可用。</p>
                ) : old.data ? (
                  <>
                    <h3>
                      {old.data.name} · v{old.data.version}
                    </h3>
                    {old.data.agentId==="pod.design.agent"&&<p>创作风格：{old.data.artwork.style||"未指定"} · 约束：{old.data.artwork.constraints||"未指定"}</p>}
                    {old.data.agentId === "product.content.agent" && <p>{contentSelectionLabel(old.data.content.selection)}</p>}
                    <p>
                      素材平台 {old.data.targetPlatform} ·{" "}
                      {old.data.knowledgeAvailability === "UNAVAILABLE"
                        ? "知识不可用"
                        : old.data.defaultKnowledgeBaseId
                          ? "默认企业知识"
                          : "不使用知识"}
                    </p>
                    {detail.canConfigure && old.data.lifecycle === "ACTIVE" && (
                      <Button
                        disabled={mutationDisabled}
                        onClick={() =>
                          mutate(
                            `${agentId}/default-template`,
                            {
                              templateId: old.data!.templateId,
                              revision: old.data!.version,
                            },
                            detail.agent.revision,
                            false,
                            "PUT",
                          )
                        }
                      >
                        设为默认 v{old.data.version}
                      </Button>
                    )}
                  </>
                ) : null}
                <Button
                  variant="outline"
                  onClick={() => {
                    setChosen(null);
                    setHistory("");
                  }}
                >
                  关闭版本
                </Button>
              </section>
            )}
          </Card>
          {!["product.selection.agent","shop.operations-diagnosis.agent"].includes(agentId)&&<Card className={styles.panel}>
            <h2>本人最近发起的任务</h2>
            <p>
              仅展示当前企业中你有商品访问权限的任务；采用方式以本次冻结权限为准。
            </p>
            {!detail.canReadRuns ? (
              <p>当前身份无权读取商品智能体任务。</p>
            ) : recent.error ? (
              <p>{configurationError(recent.error)}</p>
            ) : recent.data?.items.length ? (
              <ul className={styles.recent}>
                {recent.data.items.map((run) => (
                  <li key={run.runId}>
                    <span>
                      {run.productKey} · {labels[run.phase]}
                    </span>
                    <Link
                      href={agentId==="listing.materials.agent"?`/workbench/supply/materials/${run.operationId}?agent_key=${run.requestKey}&agent_actor=${encodeURIComponent(scope.userId)}&agent_org=${encodeURIComponent(scope.organizationId)}`:`/workbench/supply/acquisition/operation/${run.operationId}?agent_key=${run.requestKey}&agent_actor=${encodeURIComponent(scope.userId)}&agent_org=${encodeURIComponent(scope.organizationId)}&agent_platform=${run.targetPlatform}${agentId === "product.content.agent" ? "&agent_kind=content" : ""}`}
                    >
                      查看任务 →
                    </Link>
                  </li>
                ))}
              </ul>
            ) : recent.data ? (
              <p>暂无本人已领取执行的任务。</p>
            ) : (
              <p role="status">正在读取最近任务…</p>
            )}
            {(recentAfter || recent.data?.nextCursor) && (
              <nav aria-label="最近任务分页" className={styles.actions}>
                {recentAfter && (
                  <Button variant="outline" onClick={() => setRecentAfter("")}>
                    最近任务首页
                  </Button>
                )}
                {recent.data?.nextCursor && (
                  <Button
                    variant="outline"
                    onClick={() => setRecentAfter(recent.data!.nextCursor)}
                  >
                    更早的任务
                  </Button>
                )}
              </nav>
            )}
          </Card>}
        </>
      ) : (
        <>
          {mode === "mine" && (
            <div className={styles.filters}>
              {[
                ["", "全部"],
                ["ENABLED", "已启用"],
                ["DISABLED", "已停用"],
              ].map(([value, label]) => (
                <Button
                  key={value}
                  variant="outline"
                  aria-pressed={filter === value}
                  onClick={() => {
                    setFilter(value);
                    setAfter("");
                  }}
                >
                  {label}
                </Button>
              ))}
            </div>
          )}
          <div
            className={mode === "market" ? styles.marketGrid : styles.mineGrid}
          >
            <div
              className={mode === "market" ? styles.cards : styles.mineCards}
            >
              {entries.length ? (
                entries.map(entry => ["product.title.agent", "product.content.agent"].includes(entry.agent.agentId)
                  ? entry.agent.agentId === contentOperations[0]?.agent.agentId ? contentCard() : null : card(entry))
              ) : (
                <ConsoleState kind="empty" title="当前企业尚无符合条件的智能体">
                  <Link href="/workbench/agents/market">前往市场启用</Link>
                </ConsoleState>
              )}
            </div>
            {mode === "mine" && (
              <Card className={styles.panel}>
                <h2>使用流程</h2>
                <ol className={styles.steps}>
                  <li>选择企业已启用的智能体</li>
                  <li>从 1688 采集已保存商品中选择商品</li>
                  <li>确认精确模板版本、平台和企业知识选择</li>
                  <li>查看生成建议，按本次权限确认或查看策略采用回执</li>
                </ol>
                <p>启用智能体不会创建执行任务，也不会自动使用知识。</p>
              </Card>
            )}
          </div>
          {!id && data && "nextCursor" in data && data.nextCursor && (
            <Button variant="outline" onClick={() => setAfter(data.nextCursor)}>
              下一页
            </Button>
          )}
        </>
      )}
      {disable && (
        <ConfirmDisable
          busy={busy}
          onCancel={() => setDisable(null)}
          onConfirm={() =>
            mutate(`${agentId}/disable`, {}, disable.agent.revision)
          }
        />
      )}
      {editor && (
        <TemplateEditor
          scope={scope}
          agentId={agentId}
          template={editor === "new" ? undefined : editor}
          busy={mutationDisabled}
          onCancel={() => setEditor(null)}
          onSave={(payload) =>
            mutate(
              editor === "new"
                ? `${agentId}/templates`
                : `${agentId}/templates/${editor.templateId}`,
              payload,
              editor === "new" ? undefined : editor.revision,
              false,
              editor === "new" ? "POST" : "PUT",
            )
          }
        />
      )}
    </ConsolePage>
  );
}
function ConfirmDisable({
  busy,
  onConfirm,
  onCancel,
}: {
  busy: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const e = dialog.current;
    e?.showModal();
    return () => e?.close();
  }, []);
  return (
    <dialog
      ref={dialog}
      className={styles.dialog}
      aria-labelledby="disable-agent-title"
      onCancel={(e) => {
        e.preventDefault();
        onCancel();
      }}
    >
      <h2 id="disable-agent-title">停用当前企业智能体</h2>
      <p>
        停用后阻止新启动和继续执行。已经领取的有限执行可以完成；原结果、人工审核及应用不会因此删除。
      </p>
      <div className={styles.actions}>
        <Button variant="outline" disabled={busy} onClick={onCancel}>
          取消
        </Button>
        <Button disabled={busy} onClick={onConfirm}>
          确认停用
        </Button>
      </div>
    </dialog>
  );
}
function TemplateEditor({
  scope,
  agentId,
  template,
  busy,
  onSave,
  onCancel,
}: {
  scope: ConfigurationScope;
  agentId: string;
  template?: AgentTemplate;
  busy: boolean;
  onSave: (v: unknown) => void;
  onCancel: () => void;
}) {
  const content = agentId === "product.content.agent";
  const materials=agentId==="listing.materials.agent";
  const artwork=agentId==="pod.design.agent";
  const dialog = useRef<HTMLDialogElement>(null);
  const [name, setName] = useState(template?.name ?? ""),
    [platform, setPlatform] = useState(template?.targetPlatform ?? (artwork ? "sds" : content ? "product" : "shein")),
    [base, setBase] = useState(
      template?.knowledgeAvailability === "UNAVAILABLE"
        ? "unresolved"
        : (template?.defaultKnowledgeBaseId ?? ""),
    ),
    [bases, setBases] = useState<z.infer<typeof basesSchema>["items"]>([]),
    [knowledgeError, setKnowledgeError] = useState(""),
    [basePage, setBasePage] = useState(1),
    [baseTotal, setBaseTotal] = useState(0);
  const [selection, setSelection] = useState<ContentSelection>(template?.agentId === "product.content.agent" ? template.content.selection : { targetLanguage: "", title: true, description: true, sellingPoints: true, seoKeywords: true });
  const [artStyle,setArtStyle]=useState(template?.agentId==="pod.design.agent"?template.artwork.style:""),[artConstraints,setArtConstraints]=useState(template?.agentId==="pod.design.agent"?template.artwork.constraints:"");
  const validSelection = !content || contentSelectionSchema.safeParse(selection).success;
  useEffect(() => {
    const e = dialog.current;
    e?.showModal();
    const controller = new AbortController();
    if(materials||artwork) return ()=>{e?.close();controller.abort()};
    knowledgeRequest(
      scope,
      `knowledge-bases?pageSize=100&page=${basePage}`,
      basesSchema,
      { signal: controller.signal },
    )
      .then((result) => {
        if (!controller.signal.aborted) {
          setBases(result.items.filter((b) => b.state === "ACTIVE"));
          setBaseTotal(result.pagination.total);
        }
      })
      .catch(() => {
        if (!controller.signal.aborted)
          setKnowledgeError("企业知识当前不可用；仍可明确保存为不引用知识。");
      });
    return () => {
      e?.close();
      controller.abort();
    };
  }, [scope, basePage,materials,artwork]);
  return (
    <dialog
      ref={dialog}
      className={styles.dialog}
      aria-labelledby="template-editor-title"
      onCancel={(e) => {
        e.preventDefault();
        onCancel();
      }}
    >
      <h2 id="template-editor-title">
        {template ? "保存模板新版本" : artwork ? "保存原创图案模板" : materials ? "保存平台资料模板" : content ? "保存内容模板" : "保存标题模板"}
      </h2>
      <p>{materials?"模板保存目标平台；执行前仍核对当前企业权限、预算、标准商品和店铺规则。":"模板保存默认配置，执行前仍需确认当前企业权限、预算和知识版本。"}</p>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (base !== "unresolved" && validSelection)
            onSave({
              name,
              targetPlatform: platform,
              ...(!materials&&!artwork?{defaultKnowledgeBaseId:base||null}:{}),
              ...(artwork?{artwork:{schema:"pod-artwork-config-v1",style:artStyle,constraints:artConstraints}}:{}),
              ...(content ? { content: { schema: "content-config-v1", selection } } : {}),
            });
        }}
      >
        <label>
          模板名称
          <Input
            required
            maxLength={120}
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={materials ? "例如：SHEIN 美国站资料" : content ? "例如：英文商品内容" : "例如：品牌标题优化"}
          />
        </label>
        <label>
          {materials?"目标平台":"素材平台"}
          <select
            value={platform}
            onChange={(e) => setPlatform(e.target.value as typeof platform)}
          >
            {content && <option value="product">标准商品（不指定平台）</option>}
            {artwork?<option value="sds">POD 原创图案</option>:<option value="shein">SHEIN</option>}
            {!materials&&!artwork&&<option value="temu">TEMU</option>}
            {!materials&&!artwork&&<option value="amazon">Amazon</option>}
          </select>
        </label>
        {artwork&&<><label>默认创作风格<Input maxLength={512} value={artStyle} disabled={busy} onChange={e=>setArtStyle(e.target.value)}/></label><label>创作约束<textarea className="min-h-24 rounded-md border bg-background p-3" maxLength={4096} value={artConstraints} disabled={busy} onChange={e=>setArtConstraints(e.target.value)}/></label></>}
        {content && <ContentSelectionFields value={selection} onChange={setSelection} disabled={busy} />}
        {!materials&&!artwork&&<label>
          默认企业知识
          <select value={base} onChange={(e) => setBase(e.target.value)}>
            {base === "unresolved" && (
              <option value="unresolved">
                原默认知识不可用，请明确重新选择
              </option>
            )}
            <option value="">不引用企业知识</option>
            {template?.defaultKnowledgeBaseId &&
              !bases.some((b) => b.id === template.defaultKnowledgeBaseId) && (
                <option value={template.defaultKnowledgeBaseId}>
                  原默认知识选择（保存时重新确认）
                </option>
              )}
            {bases.map((b) => (
              <option key={b.id} value={b.id}>
                {b.name}
              </option>
            ))}
          </select>
        </label>}
        {!materials&&baseTotal > 100 && (
          <nav aria-label="模板知识库分页" className={styles.actions}>
            <Button
              type="button"
              variant="outline"
              disabled={basePage === 1}
              onClick={() => {
                setBase("unresolved");
                setBases([]);
                setBasePage((n) => n - 1);
              }}
            >
              上一页知识库
            </Button>
            <span>第 {basePage} 页</span>
            <Button
              type="button"
              variant="outline"
              disabled={basePage * 100 >= baseTotal}
              onClick={() => {
                setBase("unresolved");
                setBases([]);
                setBasePage((n) => n + 1);
              }}
            >
              下一页知识库
            </Button>
          </nav>
        )}
        {knowledgeError && <p role="status">{knowledgeError}</p>}
        <p className={styles.notice}>
          保存新版本不会移动企业默认版本；“设为默认”需单独确认。
        </p>
        <div className={styles.actions}>
          <Button
            type="button"
            variant="outline"
            disabled={busy}
            onClick={onCancel}
          >
            取消
          </Button>
          <Button
            type="submit"
            disabled={busy || base === "unresolved" || !name.trim() || !validSelection}
          >
            保存模板
          </Button>
        </div>
      </form>
    </dialog>
  );
}
