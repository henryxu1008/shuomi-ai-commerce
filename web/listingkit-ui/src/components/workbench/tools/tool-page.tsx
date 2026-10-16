"use client";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type FormEvent } from "react";
import {
  useInfiniteQuery,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { z } from "zod";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { useResourcePending } from "../resources/resource-pending";
import {
  toolPendingSchema,
  freezeToolIntent,
  restoreToolIntent,
  forgetToolIntent,
  type ToolIntent as Intent,
} from "./pending";
import { ConsolePage, ConsoleState } from "../console/console-page";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  downloadTool,
  downloadToolReference,
  toolRequest,
  ToolMarketError,
  type ToolScope,
} from "@/lib/api/tool-market";
import {
  marketSchema,
  requestsSchema,
  detailSchema,
  toolReceipt,
  demandInput,
  progressInput,
  maxToolFileBytes,
  type Tool,
  type ToolRequest,
} from "@/lib/contracts/tool-market";
import styles from "./tools.module.css";

type Mode = "official" | "mine" | "custom" | "submit" | "progress" | "admin";
const stageNames: Record<string, string> = {
  SUBMITTED: "待评估",
  EVALUATING: "需求评估",
  PLAN_CONFIRMED: "方案确认",
  DEVELOPING: "开发联调",
  DELIVERED: "专员已记录交付",
  CLOSED: "已关闭",
};
const stageHints: Record<string, string> = {
  SUBMITTED: "需求已提交，等待硕米专员联系并评估。",
  EVALUATING: "专员正在核对数据来源、接口权限和技术可行性。",
  PLAN_CONFIRMED: "方案已确认，具体范围、报价与交付安排以线下确认为准。",
  DEVELOPING: "专员正在按确认范围安排开发与联调。",
  DELIVERED: "专员已记录交付，请按实际交付内容核实使用方式。",
  CLOSED: "需求已关闭，关闭原因可在下方进度记录查看。",
};
const progressStageNames: Record<string, string> = {
  SUBMITTED: "提交需求",
  EVALUATING: "技术可行性评估",
  PLAN_CONFIRMED: "方案与报价确认",
  DEVELOPING: "开发联调与验收",
  DELIVERED: "专员记录交付",
};
const kindNames: Record<string, string> = {
  DATA: "数据采集",
  CONNECTION: "系统连接",
  AUTOMATION: "自动化流程",
  OUTPUT: "输出与权限",
};
const toolTags: Record<string, string> = {
  "product-acquisition": "商品采集 · 我的数据",
  "fingerprint-browser": "环境隔离 · 多账号",
  "image-processing": "图片处理 · 批量优化",
  "data-import": "表格导入 · 字段映射",
  automation: "流程自动化 · 定时执行",
  "bulk-listing": "批量刊登 · 多店铺",
  "order-processing": "订单汇总 · 异常处理",
  "stock-sync": "库存同步 · 异常提醒",
};
function errorText(e: unknown) {
  const code = e instanceof ToolMarketError ? e.code : "";
  return (
    (
      {
        FORBIDDEN: "当前身份没有操作权限。",
        ORGANIZATION_CONTEXT_CHANGED: "企业已变化，请恢复原企业后确认原操作。",
        IDENTITY_CONTEXT_CHANGED: "登录身份已变化，请恢复原身份后确认原操作。",
        REVISION_MISMATCH: "状态已被更新，请刷新后再次确认。",
        NOT_FOUND: "此需求不属于当前企业，或已不可用。",
        INVALID_REQUEST: "请检查填写内容。",
        IDEMPOTENCY_CONFLICT:
          "原请求编号与内容不一致，请保留原操作并联系平台处理。",
        OUTCOME_UNKNOWN: "结果尚未确认，请重试原操作读取真实回执。",
      } as Record<string, string>
    )[code] ??
    (e instanceof Error && !code ? e.message : "当前能力暂不可用，请稍后重试。")
  );
}
function useCommands(scope: ToolScope, admin: boolean) {
  const context = useWorkbenchContext(),
    live = useRef(context),
    client = useQueryClient(),
    running = useRef(false),
    mounted = useRef(true),
    activeRequest = useRef<AbortController | null>(null),
    [busy, setBusy] = useState(false),
    [message, setMessage] = useState("");
  useEffect(() => {
    live.current = context;
  }, [context]);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      activeRequest.current?.abort();
    };
  }, []);
  const pending = useResourcePending(
    {
      expectedUserId: scope.userId,
      expectedOrganizationId: scope.organizationId,
    },
    ["tool-market", admin ? "platform" : "enterprise"],
    toolPendingSchema,
    { maxLength: 300000 },
  );
  const originalIntent = restoreToolIntent(pending.storageKey, pending.command);
  useEffect(
    () =>
      context.registerOrganizationSwitchGuard(
        () => !running.current && !busy && !pending.error,
      ),
    [context, busy, pending.error],
  );
  async function execute(i: Intent) {
    if (running.current) throw new Error("当前操作尚未确认");
    let saved: z.infer<typeof toolPendingSchema> | undefined;
    let timer: ReturnType<typeof setTimeout> | undefined;
    try {
      if (live.current.user?.id !== scope.userId)
        throw new ToolMarketError("IDENTITY_CONTEXT_CHANGED");
      if (
        live.current.error ||
        live.current.blockingError ||
        live.current.isLoading ||
        live.current.isSwitching ||
        (!admin &&
          live.current.effectiveOrganization?.id !== scope.organizationId)
      )
        throw new ToolMarketError("ORGANIZATION_CONTEXT_CHANGED");
      if (!pending.ready) throw new Error("当前操作尚未确认");
      const command = i;
      if (command.path.startsWith("admin/") !== admin)
        throw new ToolMarketError("FORBIDDEN");
      running.current = true;
      setBusy(true);
      setMessage("");
      saved = await freezeToolIntent(pending.storageKey, command);
      if (
        !mounted.current ||
        live.current.user?.id !== scope.userId ||
        (!admin &&
          live.current.effectiveOrganization?.id !== scope.organizationId)
      )
        throw new ToolMarketError("ORGANIZATION_CONTEXT_CHANGED");
      pending.persist(saved);
      const controller = new AbortController();
      activeRequest.current = controller;
      timer = setTimeout(() => controller.abort(), 15000);
      const headers = new Headers({
        "Content-Type": "application/json",
        "Idempotency-Key": command.key,
      });
      if (command.absent) headers.set("If-None-Match", "*");
      else headers.set("If-Match", '"' + command.revision + '"');
      const receipt = await toolRequest(scope, command.path, toolReceipt, {
        method: command.path.startsWith("activations/") ? "PUT" : "POST",
        headers,
        body: command.body,
        signal: controller.signal,
      });
      if (receipt.commandId !== command.key)
        throw new ToolMarketError("OUTCOME_UNKNOWN");
      if (!mounted.current) throw new ToolMarketError("OUTCOME_UNKNOWN");
      pending.clear(saved);
      forgetToolIntent(pending.storageKey, command.key);
      setMessage("已保存，正在读取真实状态。");
      await client.invalidateQueries({
        queryKey: ["tool-market", scope.userId, scope.organizationId],
      });
      return receipt;
    } catch (e) {
      if (mounted.current) setMessage(errorText(e));
      if (
        e instanceof ToolMarketError &&
        ((e.status === 400 && e.code === "INVALID_REQUEST") ||
          (e.status === 404 && e.code === "NOT_FOUND") ||
          (e.status === 412 && e.code === "REVISION_MISMATCH"))
      ) {
        try {
          if (saved) {
            pending.clear(saved);
            forgetToolIntent(pending.storageKey, i.key);
          }
        } catch {}
      }
      throw e;
    } finally {
      running.current = false;
      clearTimeout(timer);
      activeRequest.current = null;
      if (mounted.current) setBusy(false);
    }
  }
  const locked = !pending.ready || !!pending.command || busy || pending.error;
  const notice =
    message || pending.command || pending.error ? (
      <Card role="status" className={styles.notice}>
        <p>
          {pending.error
            ? "无法读取待确认操作，已暂停新提交。请保留浏览器数据并联系平台处理。"
            : pending.command && !originalIntent
              ? "原操作尚未确认，附件载荷无法恢复。请在定制进度中查看并联系平台核实，暂不能提交新需求。"
              : message || "原操作尚未确认，请重试原操作。"}
        </p>
        {originalIntent && !busy && !pending.error ? (
          <Button
            variant="outline"
            onClick={() => void execute(originalIntent).catch(() => undefined)}
          >
            重试原操作
          </Button>
        ) : null}
      </Card>
    ) : null;
  return { locked, notice, execute, message };
}
export function ToolPage({
  mode,
  requestId,
}: {
  mode: Mode;
  requestId?: string;
}) {
  const c = useWorkbenchContext();
  if (c.error || c.blockingError)
    return (
      <ConsoleState kind="error" title="身份上下文不可用">
        <Button onClick={() => void c.retry()}>重新确认</Button>
      </ConsoleState>
    );
  if (c.isLoading || c.isSwitching)
    return <ConsoleState kind="loading" title="正在确认当前身份" />;
  if (
    !c.user ||
    (mode !== "admin" && (!c.effectiveOrganization || c.selectionRequired))
  )
    return (
      <ConsoleState
        kind="unavailable"
        title={mode === "admin" ? "请先登录专员身份" : "请先选择企业"}
      />
    );
  const scope = {
    userId: c.user.id,
    organizationId: mode === "admin" ? "" : c.effectiveOrganization!.id,
  };
  return (
    <ScopedPage
      key={JSON.stringify([scope, mode, requestId, c.permissions])}
      mode={mode}
      scope={scope}
      requestId={requestId}
    />
  );
}
function ScopedPage({
  mode,
  scope,
  requestId,
}: {
  mode: Mode;
  scope: ToolScope;
  requestId?: string;
}) {
  const router = useRouter();
  const admin = mode === "admin",
    commands = useCommands(scope, admin),
    [filter, setFilter] = useState("全部"),
    [cursor, setCursor] = useState(""),
    [selected, setSelected] = useState(() =>
      z.uuid().safeParse(requestId).success ? requestId! : "",
    );
  const market = useQuery({
    queryKey: [
      "tool-market",
      scope.userId,
      scope.organizationId,
      mode === "mine" ? "mine" : "market",
    ],
    queryFn: ({ signal }) =>
      toolRequest(scope, mode === "mine" ? "mine" : "market", marketSchema, {
        signal,
      }),
    enabled: !admin,
  });
  const requests = useQuery({
    queryKey: [
      "tool-market",
      scope.userId,
      scope.organizationId,
      "requests",
      cursor,
    ],
    queryFn: ({ signal }) =>
      toolRequest(
        scope,
        (admin ? "admin/" : "") +
          "requests" +
          (cursor ? "?cursor=" + cursor : ""),
        requestsSchema,
        { signal },
      ),
    enabled: admin || mode === "progress",
  });
  const title =
    mode === "official"
      ? "官方工具"
      : mode === "mine"
        ? "我的工具"
        : mode === "submit"
          ? "提交工具定制需求"
          : mode === "progress"
            ? "我的工具定制需求"
            : admin
              ? "工具定制 · 专员处理"
              : "工具定制";
  const actions =
    mode === "official" ? (
      <Button asChild variant="outline">
        <Link href="/workbench/tools/mine">我的工具</Link>
      </Button>
    ) : mode === "mine" ? (
      <Button asChild variant="outline">
        <Link href="/workbench/tools/official">前往官方工具</Link>
      </Button>
    ) : admin ? (
      <Link href="/workbench/tools/custom">返回工具定制</Link>
    ) : mode === "submit" ? (
      <Button asChild variant="outline">
        <Link href="/workbench/tools/custom/progress">查看定制进度</Link>
      </Button>
    ) : (
      <>
        {mode === "custom" ? (
          <Button asChild variant="outline">
            <Link href="/workbench/tools/custom/progress">查看定制进度</Link>
          </Button>
        ) : null}
        <Button asChild>
          <Link href="/workbench/tools/custom/new">
            {mode === "progress" ? "提交新需求" : "提交定制需求"}
          </Link>
        </Button>
      </>
    );
  function activate(t: Tool, enabled: boolean) {
    return commands.execute({
      path: "activations/" + t.id,
      key: crypto.randomUUID(),
      body: JSON.stringify({ enabled }),
      ...(t.activation
        ? { revision: t.activation.revision }
        : { absent: true }),
    });
  }
  return (
    <ConsolePage
      className={`${styles.page} ${mode === "submit" || mode === "progress" ? styles.customWorkspace : ""}`}
      title={title}
      description={
        mode === "official"
          ? "浏览平台官方维护的电商工具，了解适用范围、权限和使用方式。"
          : mode === "mine"
            ? "管理当前企业已启用工具，选择本地插件或在线服务快速使用。"
            : mode === "submit"
              ? "描述需要定制的工具及运行规则，由专员评估技术方案和报价。"
              : mode === "progress"
                ? "查看工具定制申请的状态、技术评估及后续交付安排。"
                : admin
                  ? "平台授权独立核实。报价付款线下办理，进度由专员据实记录。"
                  : "针对特殊业务流程，定制专属采集、连接或自动化工具。"
      }
      breadcrumbs={[
        { label: "工具市场", href: "/workbench/tools" },
        ...(mode === "submit" || mode === "progress"
          ? [
              { label: "工具定制", href: "/workbench/tools/custom" },
              { label: mode === "submit" ? "提交定制需求" : "定制进度" },
            ]
          : [{ label: title }]),
      ]}
      actions={actions}
    >
      {commands.notice}
      {!admin && market.isPending ? (
        <ConsoleState kind="loading" title="正在读取企业工具" />
      ) : !admin && market.error ? (
        <ConsoleState kind="error" title={errorText(market.error)}>
          <Button onClick={() => void market.refetch()}>重新读取</Button>
        </ConsoleState>
      ) : null}
      {mode === "official" && market.data ? (
        <>
          <div className={styles.catalogHeading}>
            <h2>官方工具列表</h2>
            <p>按工具分类查找适合的执行能力。</p>
          </div>
          <div className={styles.filters} role="group" aria-label="工具分类">
            {["全部", "采集", "图片", "环境", "数据", "流程", "其他"].map(
              (f) => (
                <Button
                  key={f}
                  variant="outline"
                  aria-pressed={f === filter}
                  onClick={() => setFilter(f)}
                >
                  {f === filter ? (
                    <span className={styles.dot} aria-hidden="true" />
                  ) : null}
                  {f}
                </Button>
              ),
            )}
          </div>
          <div className={styles.cards}>
            {market.data.tools
              .filter((t) => filter === "全部" || t.category === filter)
              .map((t) => (
                <Card
                  key={t.id}
                  className={styles.toolCard}
                  data-state={t.status}
                >
                  <div className={styles.cardHeading}>
                    <h2>{t.name}</h2>
                  </div>
                  <p>{t.description}</p>
                  <small>{toolTags[t.id] ?? t.category}</small>
                  <div className={styles.cardActions}>
                    {t.status === "AVAILABLE" ? (
                      <Button
                        disabled={
                          commands.locked ||
                          !market.data.canManage ||
                          !!t.activation?.enabled
                        }
                        onClick={() =>
                          void activate(t, true).catch(() => undefined)
                        }
                      >
                        {t.activation?.enabled ? "已启用" : "立即启用"}
                      </Button>
                    ) : null}
                    <span
                      className={styles.badge}
                      data-ready={t.status === "AVAILABLE"}
                    >
                      {t.status === "AVAILABLE"
                        ? "已开放"
                        : t.status === "DEVELOPING"
                          ? "开发中"
                          : "暂不可用"}
                    </span>
                  </div>
                </Card>
              ))}
          </div>
          {!market.data.canManage ? (
            <p className={styles.muted}>请由当前企业管理员启用工具。</p>
          ) : null}
        </>
      ) : null}
      {mode === "mine" && market.data ? (
        <>
          <p className={styles.enabledCount}>
            已启用工具 {market.data.tools.length}
          </p>
          <div className={styles.mineGrid}>
            <div>
              {!market.data.tools.length ? (
                <ConsoleState kind="empty" title="当前企业尚未启用工具">
                  <Link href="/workbench/tools/official">前往官方工具</Link>
                </ConsoleState>
              ) : (
                market.data.tools.map((t) => (
                  <Card key={t.id} className={styles.mineCard}>
                    <div className={styles.cardHeading}>
                      <h2>{t.name}</h2>
                      {market.data.canManage ? (
                        <Button
                          variant="outline"
                          disabled={commands.locked}
                          onClick={() =>
                            void activate(t, false).catch(() => undefined)
                          }
                        >
                          从清单停用
                        </Button>
                      ) : null}
                    </div>
                    <p>
                      采集1688商品的标题、主图、规格与价格等信息；结果统一保存到「数据服务
                      / 我的数据」。
                    </p>
                    {t.status !== "AVAILABLE" ? (
                      <p role="status">{t.reason}</p>
                    ) : null}
                    <h3 className={styles.channelTitle}>选择采集方式</h3>
                    <div className={styles.channels}>
                      <div>
                        <div className={styles.channelHeading}>
                          <h3>本地插件采集</h3>
                          <span className={styles.free}>免费</span>
                        </div>
                        <p>
                          安装在电脑本地运行，通过 Edge 或 Chrome
                          插件采集当前1688商品。
                        </p>
                        <strong className={styles.freeCopy}>
                          免费 · 不消耗数据额度
                        </strong>
                        {t.localCapture ? (
                          <>
                            <DownloadButton
                              scope={scope}
                              enabled={t.download}
                            />
                          </>
                        ) : (
                          <p role="status">本地插件采集尚未开放</p>
                        )}
                      </div>
                      <div>
                        <div className={styles.channelHeading}>
                          <h3>在线采集</h3>
                          <span className={styles.paid}>按数据收费</span>
                        </div>
                        <p>
                          由平台执行1688采集；支持范围、计价与额度在采集页面确认。
                        </p>
                        <strong className={styles.paidCopy}>
                          按实际成功采集数据条数扣除额度
                        </strong>
                        {t.onlineCapture ? (
                          <Link
                            className={styles.primaryLink}
                            href="/workbench/supply/acquisition"
                          >
                            在线采集
                          </Link>
                        ) : (
                          <Button disabled>当前安装未开放在线采集</Button>
                        )}
                      </div>
                    </div>
                    <Link
                      className={styles.resultLink}
                      href="/workbench/data/mine"
                    >
                      采集结果统一保存至「数据服务 / 我的数据」 →
                    </Link>
                    {t.localCapture ? (
                      <details className={styles.installHelp}>
                        <summary>插件安装与使用说明</summary>
                        <ol>
                          <li>下载压缩包并解压到固定文件夹。</li>
                          <li>
                            打开 Edge 或 Chrome
                            扩展管理，启用开发者模式，加载解压缩的扩展。
                          </li>
                          <li>
                            打开1688商品详情页，点击插件采集，
                            <Link href="/capture/1688">打开插件接收页</Link>
                            确认保存。
                          </li>
                        </ol>
                      </details>
                    ) : null}
                    <p className={styles.muted}>
                      停用只移出企业清单；已安装插件和已开始的采集按原权限继续处理。
                    </p>
                  </Card>
                ))
              )}
            </div>
            <Card className={styles.side}>
              <h2>采集方式说明</h2>
              <p>
                根据运行环境选择采集方式，两种方式的结果都会进入「我的数据」。
              </p>
              <div className={styles.explanation}>
                <h3>本地插件采集</h3>
                <p>
                  适合：电脑本地运行，通过浏览器插件采集商品
                  <br />
                  费用：免费，不消耗数据额度
                </p>
              </div>
              <div className={styles.explanation}>
                <h3>在线采集</h3>
                <p>
                  适合：由平台执行商品采集
                  <br />
                  费用：按实际成功采集数据条数扣除额度
                </p>
              </div>
              <p className={styles.tip}>
                本地插件免费；在线采集按实际数据量计费
              </p>
            </Card>
          </div>
        </>
      ) : null}
      {mode === "custom" ? (
        <>
          <Card className={styles.intro}>
            <div>
              <h2>把业务需求变成可调用的工具</h2>
              <p>
                定制数据采集、系统连接与自动化工具，适配官方工具未覆盖的业务需求。
              </p>
            </div>
            <div className={styles.serviceNote}>
              <h3>有偿定制服务</h3>
              <p>提交需求后由专员联系</p>
              <p>评估范围 · 确认方案 · 提供报价</p>
            </div>
          </Card>
          <h2 className={styles.sectionTitle}>可定制内容</h2>
          <div className={styles.cards}>
            {Object.entries(kindNames).map(([k, n]) => (
              <Card key={k} className={styles.customCard}>
                <h2>{n}</h2>
                <p>
                  {
                    (
                      {
                        DATA: "按规则获取网页、接口或文件数据",
                        CONNECTION: "对接 ERP、OMS 及第三方平台",
                        AUTOMATION: "实现同步、判断和批量处理",
                        OUTPUT: "定义交付格式、账号及授权要求",
                      } as Record<string, string>
                    )[k]
                  }
                </p>
              </Card>
            ))}
          </div>
          <Card className={styles.process}>
            <h2>定制流程</h2>
            <p>工具定制属于有偿服务，评估与方案确认后线下报价付款。</p>
            <div className={styles.steps}>
              {["提交需求", "技术评估", "方案报价", "开发验收", "授权启用"].map(
                (n, i) => (
                  <Card key={n}>
                    <span>0{i + 1}</span>
                    <h3>{n}</h3>
                    <p>
                      {
                        [
                          "说明目标和业务场景",
                          "确认接口与运行条件",
                          "明确范围、周期和费用",
                          "完成联调测试与验收",
                          "实际工具交付及开放另行核实",
                        ][i]
                      }
                    </p>
                  </Card>
                ),
              )}
            </div>
          </Card>
          <Link
            href="/workbench/tools/custom/review"
            className={styles.specialist}
          >
            硕米专员处理入口 →
          </Link>
        </>
      ) : null}
      {mode === "submit" ? (
        <DemandForm
          scope={scope}
          locked={commands.locked || !market.data?.canCustomize}
          submit={async (body) => {
            const r = await commands.execute({
              path: "requests",
              key: crypto.randomUUID(),
              body: JSON.stringify(body),
              absent: true,
            });
            router.push("/workbench/tools/custom/progress?request=" + r!.id);
          }}
        />
      ) : null}
      {admin || mode === "progress" ? (
        <>
          <Card
            className={`${styles.requestList} ${!admin ? styles.requestSelector : ""}`}
          >
            <details
              key={selected || "list"}
              open={admin || !selected || requests.error ? true : undefined}
            >
              <summary>
                <h2>{admin ? "所有企业的定制需求" : "定制申请"}</h2>
                {selected && !admin ? <span>切换定制需求</span> : null}
              </summary>
              {requests.isPending ? (
                <p role="status">正在读取需求与权限</p>
              ) : requests.error ? (
                <p role="alert">
                  {errorText(requests.error)}{" "}
                  <Button onClick={() => void requests.refetch()}>
                    重新读取
                  </Button>
                </p>
              ) : (
                <>
                  {!requests.data?.items.length ? (
                    <p>暂无定制需求。</p>
                  ) : (
                    requests.data.items.map((r) => (
                      <button
                        className={styles.requestRow}
                        key={r.id}
                        onClick={() => setSelected(r.id)}
                      >
                        <span>
                          {r.title}
                          <small>
                            {r.kind === "OUTPUT" ? "其他" : kindNames[r.kind]}
                            {admin ? " · 企业 " + r.organizationId : ""}
                          </small>
                        </span>
                        <span>{stageNames[r.stage]} →</span>
                      </button>
                    ))
                  )}
                  <div className={styles.cardActions}>
                    <Button
                      variant="outline"
                      disabled={!cursor}
                      onClick={() => setCursor("")}
                    >
                      返回第一页
                    </Button>
                    <Button
                      variant="outline"
                      disabled={!requests.data?.nextCursor}
                      onClick={() => setCursor(requests.data!.nextCursor)}
                    >
                      下一页
                    </Button>
                  </div>
                </>
              )}
            </details>
          </Card>
          {selected ? (
            <RequestDetail
              key={selected}
              scope={scope}
              id={selected}
              admin={admin}
              commands={commands}
            />
          ) : null}
        </>
      ) : null}
    </ConsolePage>
  );
}
function DownloadButton({
  scope,
  enabled,
}: {
  scope: ToolScope;
  enabled: boolean;
}) {
  const [busy, setBusy] = useState(false),
    [message, setMessage] = useState("");
  return (
    <>
      <Button
        disabled={!enabled || busy}
        onClick={() => {
          setBusy(true);
          setMessage("");
          void downloadTool(scope)
            .catch((e) => setMessage(errorText(e)))
            .finally(() => setBusy(false));
        }}
      >
        {busy ? "正在下载" : enabled ? "下载插件" : "当前安装未配置插件包"}
      </Button>
      {message ? <p role="alert">{message}</p> : null}
    </>
  );
}
function DemandForm({
  locked,
  submit,
  scope,
}: {
  locked: boolean;
  submit: (v: z.infer<typeof demandInput>) => Promise<void>;
  scope: ToolScope;
}) {
  const context = useWorkbenchContext(),
    active = useRef(true),
    preparing = useRef(false);
  const [preparingFiles, setPreparingFiles] = useState(false),
    [scenario, setScenario] = useState(""),
    [contactName, setContactName] = useState(""),
    [contactMethod, setContactMethod] = useState(""),
    [consent, setConsent] = useState(false),
    [files, setFiles] = useState<File[]>([]);
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  useEffect(
    () => context.registerOrganizationSwitchGuard(() => !preparing.current),
    [context],
  );
  const [kind, setKind] = useState("DATA"),
    [title, setTitle] = useState(""),
    [description, setDescription] = useState(""),
    [error, setError] = useState("");
  const send = async (e: FormEvent) => {
    e.preventDefault();
    if (locked || preparing.current) return;
    const fields = {
      kind,
      title,
      description,
      scenario,
      contactName,
      contactMethod,
      consent,
    };
    const result = demandInput.safeParse(fields);
    if (!result.success) {
      setError("请填写必填信息，并明确同意平台专员联系评估。");
      return;
    }
    setError("");
    preparing.current = true;
    setPreparingFiles(true);
    try {
      const uploaded = await Promise.all(
        files.map(
          (file) =>
            new Promise<{ name: string; data: string }>((resolve, reject) => {
              const reader = new FileReader();
              reader.onerror = () =>
                reject(new Error("无法读取参考资料，请重新选择。"));
              reader.onload = () =>
                resolve({
                  name: file.name,
                  data: String(reader.result).split(",")[1] ?? "",
                });
              reader.readAsDataURL(file);
            }),
        ),
      );
      if (
        !active.current ||
        context.user?.id !== scope.userId ||
        context.effectiveOrganization?.id !== scope.organizationId
      )
        return;
      await submit(
        demandInput.parse({
          ...result.data,
          ...(uploaded.length ? { files: uploaded } : {}),
        }),
      );
    } catch (e) {
      if (active.current) setError(errorText(e));
    } finally {
      preparing.current = false;
      if (active.current) setPreparingFiles(false);
    }
  };
  return (
    <div className={styles.formGrid}>
      <Card className={styles.form}>
        <h2>告诉我们你需要怎样的工具</h2>
        <p>描述工具目标与运行场景，具体技术方案由专员沟通确认。</p>
        <form onSubmit={(e) => void send(e)}>
          <fieldset disabled={locked || preparingFiles}>
            <h3>基本信息</h3>
            <div className={styles.fieldRow}>
              <label>
                工具需求名称 *
                <Input
                  required
                  maxLength={120}
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="例如：商品数据采集与 ERP 同步工具"
                />
              </label>
              <label>
                使用场景 *
                <Input
                  required
                  maxLength={240}
                  value={scenario}
                  onChange={(e) => setScenario(e.target.value)}
                  placeholder="请说明工具使用场景"
                />
              </label>
            </div>
            <div className={styles.kindField}>
              <span>定制工具类型 *</span>
              <div
                role="group"
                aria-label="定制工具类型"
                className={styles.kindOptions}
              >
                {Object.entries(kindNames).map(([k, v]) => (
                  <Button
                    key={k}
                    variant="outline"
                    aria-pressed={kind === k}
                    onClick={() => setKind(k)}
                  >
                    {k === "OUTPUT" ? "其他" : v}
                  </Button>
                ))}
              </div>
            </div>
            <h3>需求描述</h3>
            <label>
              工具功能与执行规则 *
              <textarea
                required
                maxLength={4000}
                rows={3}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="请描述数据来源或目标系统、使用场景、执行步骤、运行频率和预期输出。"
              />
            </label>
            <div className={styles.uploadRow}>
              <label className={styles.uploadLabel}>
                ＋ 上传参考资料
                <input
                  aria-label="上传参考资料"
                  type="file"
                  multiple
                  accept=".pdf,.png,.jpg,.jpeg,.txt,.csv,application/pdf,image/png,image/jpeg,text/plain"
                  onChange={(e) => {
                    const chosen = Array.from(e.currentTarget.files ?? []);
                    e.currentTarget.value = "";
                    if (
                      chosen.length > 3 ||
                      chosen.some(
                        (f) =>
                          f.size === 0 ||
                          f.size > maxToolFileBytes ||
                          [...f.name].length > 200 ||
                          /[\\/\u0000-\u001f\u007f-\u009f]/.test(f.name),
                      )
                    ) {
                      setError(
                        "最多选择3个文件，每个不超过2MiB，文件名不能含路径或控制字符。",
                      );
                      return;
                    }
                    setFiles(chosen);
                    setError("");
                  }}
                />
              </label>
              <p>支持 PDF、PNG、JPEG、TXT/CSV，最多3个，每个2MiB；非必填。</p>
            </div>
            {files.length ? (
              <ul className={styles.fileList}>
                {files.map((f, i) => (
                  <li key={i}>
                    <span>{f.name}</span>
                    <Button
                      variant="ghost"
                      size="sm"
                      aria-label={"移除 " + f.name}
                      onClick={() => setFiles(files.filter((_, n) => n !== i))}
                    >
                      移除
                    </Button>
                  </li>
                ))}
              </ul>
            ) : null}
            <h3>联系人与沟通方式</h3>
            <div className={styles.fieldRow}>
              <label>
                联系人 *
                <Input
                  required
                  maxLength={80}
                  value={contactName}
                  onChange={(e) => setContactName(e.target.value)}
                  placeholder="请输入联系人姓名"
                />
              </label>
              <label>
                手机或微信 *
                <Input
                  required
                  maxLength={160}
                  value={contactMethod}
                  onChange={(e) => setContactMethod(e.target.value)}
                  placeholder="请输入方便联系的方式"
                />
              </label>
            </div>
            <label className={styles.consent}>
              <input
                type="checkbox"
                required
                checked={consent}
                onChange={(e) => setConsent(e.target.checked)}
              />
              我同意平台专员联系我，以便确认需求、方案和报价。
            </label>
            <div className={styles.cardActions}>
              <Button asChild variant="outline">
                <Link href="/workbench/tools/custom">取消</Link>
              </Button>
              <Button type="submit">
                {preparingFiles ? "正在保存需求" : "提交定制需求"}
              </Button>
            </div>
          </fieldset>
          {error ? <p role="alert">{error}</p> : null}
        </form>
      </Card>
      <aside className={styles.formAside} aria-label="定制需求填写指引">
        <Card>
          <span className={styles.free}>有偿定制服务</span>
          <h2>服务与报价说明</h2>
          <p>
            提交需求无需付费。
            <br />
            评估完成后再确认方案与报价，付款线下办理。
          </p>
          <p className={styles.tip}>方案、报价确认后才进入开发</p>
        </Card>
        <Card>
          <h2>提交后会怎样</h2>
          <ol className={styles.followup}>
            {[
              ["专员联系", "与你确认目标、使用场景和现有资料"],
              ["需求评估", "判断实现范围、数据条件与交付周期"],
              ["方案与报价", "确认服务内容与费用后开始定制"],
            ].map(([title, description], i) => (
              <li key={title}>
                <span>0{i + 1}</span>
                <div>
                  <h3>{title}</h3>
                  <p>{description}</p>
                </div>
              </li>
            ))}
          </ol>
        </Card>
        <Card>
          <h2>填写建议</h2>
          <p>
            请说明数据来源、目标系统及运行规则。接口和授权资料可在评估阶段补充。
          </p>
          <p className={styles.freeCopy}>请勿提交密码、token或账号凭据。</p>
        </Card>
      </aside>
    </div>
  );
}
function ReferenceDownload({
  scope,
  admin,
  id,
  file,
}: {
  scope: ToolScope;
  admin: boolean;
  id: string;
  file: ToolRequest["attachments"][number];
}) {
  const [busy, setBusy] = useState(false),
    [message, setMessage] = useState("");
  const active = useRef(true);
  const download = useRef<AbortController | null>(null);
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
      download.current?.abort();
    };
  }, []);
  return (
    <div>
      <Button
        variant="outline"
        disabled={busy}
        onClick={() => {
          setBusy(true);
          setMessage("");
          download.current = new AbortController();
          void downloadToolReference(
            scope,
            admin,
            id,
            file,
            download.current.signal,
          )
            .catch(() => {
              if (active.current) setMessage("参考资料暂时无法下载，请重试。");
            })
            .finally(() => {
              if (active.current) setBusy(false);
            });
        }}
      >
        {busy ? "正在下载" : "下载 " + file.name}
      </Button>
      {message ? <p role="alert">{message}</p> : null}
    </div>
  );
}
function nextStages(r: ToolRequest) {
  const order = [
    "SUBMITTED",
    "EVALUATING",
    "PLAN_CONFIRMED",
    "DEVELOPING",
    "DELIVERED",
  ];
  const next = order[order.indexOf(r.stage) + 1];
  return [r.stage, ...(next ? [next] : []), "CLOSED"];
}
function RequestDetail({
  scope,
  id,
  admin,
  commands,
}: {
  scope: ToolScope;
  id: string;
  admin: boolean;
  commands: ReturnType<typeof useCommands>;
}) {
  const detail = useInfiniteQuery({
    queryKey: ["tool-market", scope.userId, scope.organizationId, "detail", id],
    initialPageParam: "",
    queryFn: ({ signal, pageParam }) =>
      toolRequest(
        scope,
        (admin ? "admin/" : "") +
          "requests/" +
          id +
          (pageParam ? "?eventsBefore=" + pageParam : ""),
        detailSchema,
        { signal },
      ),
    getNextPageParam: (page) => page.nextEventsBefore || undefined,
  });
  const [stage, setStage] = useState(""),
    [note, setNote] = useState("");
  if (detail.isPending)
    return <ConsoleState kind="loading" title="正在读取原需求与进度" />;
  const historyUnavailable =
    detail.isFetchNextPageError &&
    detail.error instanceof ToolMarketError &&
    detail.error.status >= 500;
  if ((detail.error && !historyUnavailable) || !detail.data)
    return <ConsoleState kind="error" title={errorText(detail.error)} />;
  const r = detail.data.pages[0].request;
  const events = detail.data.pages
    .slice()
    .reverse()
    .flatMap((page) => page.events);
  return (
    <section className={styles.detail}>
      <Card className={styles.requestSummary}>
        <div className={styles.summaryHeading}>
          <div>
            <span className={styles.recordLabel}>
              {admin ? "客户企业的定制需求" : "当前企业的定制需求"}
            </span>
            <h2>{r.title}</h2>
          </div>
          <span className={styles.summaryStage}>{stageNames[r.stage]}</span>
        </div>
        <p className={styles.description}>{r.description}</p>
        <p>工具类型：{r.kind === "OUTPUT" ? "其他" : kindNames[r.kind]}</p>
        <details className={styles.requestExtra} open={admin}>
          <summary>查看提交信息与参考资料</summary>
          <dl className={styles.requestFacts}>
            <dt>使用场景</dt>
            <dd>{r.scenario}</dd>
            <dt>联系人</dt>
            <dd>
              {r.contactName} · {r.contactMethod}
            </dd>
            <dt>联系授权</dt>
            <dd>提交时已明确同意平台专员联系评估</dd>
          </dl>
          {r.attachments.length ? (
            <div className={styles.fileList}>
              {r.attachments.map((file) => (
                <ReferenceDownload
                  key={file.id}
                  scope={scope}
                  admin={admin}
                  id={id}
                  file={file}
                />
              ))}
            </div>
          ) : null}
        </details>
      </Card>
      <div className={styles.progressGrid}>
        <Card className={styles.progressPanel}>
          <div className={styles.cardHeading}>
            <h2>办理进度</h2>
            <span className={styles.freeCopy}>{stageNames[r.stage]}</span>
          </div>
          <ol className={styles.stageList}>
            {[
              "SUBMITTED",
              "EVALUATING",
              "PLAN_CONFIRMED",
              "DEVELOPING",
              "DELIVERED",
            ].map((s, i, stages) => {
              const current = stages.indexOf(r.stage);
              return (
                <li
                  key={s}
                  data-current={s === r.stage}
                  aria-current={s === r.stage ? "step" : undefined}
                >
                  <span className={styles.stageNumber}>0{i + 1}</span>
                  <div>
                    <h3>{progressStageNames[s]}</h3>
                    <p>
                      {
                        [
                          "说明工具目标与业务场景",
                          "确认数据来源、接口权限和运行条件",
                          "线下确认范围、交付周期和费用",
                          "按确认范围开发与联调",
                          "实际工具交付及开放另行核实",
                        ][i]
                      }
                    </p>
                  </div>
                  <span className={styles.badge}>
                    {r.stage === "CLOSED"
                      ? "已关闭"
                      : current > i
                        ? "已完成"
                        : current === i
                          ? s === "DELIVERED"
                            ? "已记录"
                            : "当前阶段"
                          : "待开始"}
                  </span>
                </li>
              );
            })}
          </ol>
        </Card>
        <Card className={styles.progressAside}>
          <h2>现在需要做什么</h2>
          <div className={styles.currentStage}>
            <h3>当前阶段：{stageNames[r.stage]}</h3>
            <p>{stageHints[r.stage]}</p>
          </div>
          <h3>你可以准备</h3>
          <p>整理业务流程、接口说明和授权要求，方便后续评估与联调。</p>
          <hr />
          <h3>交付说明</h3>
          <p>进度记录不代表付款或工具已安装。实际工具交付及开放另行核实。</p>
        </Card>
      </div>
      <Card className={styles.history}>
        <h3>专员记录的进度</h3>
        <ol>
          {events.map((e) => (
            <li key={e.revision}>
              <span>
                {stageNames[e.stage]} ·{" "}
                {new Date(e.occurredAt).toLocaleString("zh-CN")}
              </span>
              <p>{e.note}</p>
            </li>
          ))}
        </ol>
        {detail.hasNextPage ? (
          <Button
            variant="outline"
            disabled={detail.isFetching}
            onClick={() => void detail.fetchNextPage()}
          >
            {detail.isFetchingNextPage ? "正在读取更早进度" : "加载更早进度"}
          </Button>
        ) : null}
        {detail.isFetchNextPageError ? (
          <p role="alert">更早进度暂时无法读取，请重试；已读取进度仍可查看。</p>
        ) : null}
        {admin && !["DELIVERED", "CLOSED"].includes(r.stage) ? (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              const value = progressInput.safeParse({
                stage: stage || r.stage,
                note,
              });
              if (!value.success) return;
              void commands
                .execute({
                  path: "admin/requests/" + id + "/progress",
                  key: crypto.randomUUID(),
                  body: JSON.stringify(value.data),
                  revision: r.revision,
                })
                .then(() => {
                  setStage("");
                  setNote("");
                })
                .catch(() => undefined);
            }}
          >
            <fieldset disabled={commands.locked}>
              <label>
                处理阶段
                <select
                  value={stage || r.stage}
                  onChange={(e) => setStage(e.target.value)}
                >
                  {nextStages(r).map((s) => (
                    <option key={s} value={s}>
                      {s === r.stage ? "追加本阶段进度" : stageNames[s]}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                真实处理进度 / 关闭原因
                <textarea
                  required
                  maxLength={2000}
                  rows={4}
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                />
              </label>
              <Button type="submit" disabled={!note.trim()}>
                记录进度
              </Button>
            </fieldset>
          </form>
        ) : null}
      </Card>
    </section>
  );
}
