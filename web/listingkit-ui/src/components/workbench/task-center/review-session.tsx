"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { applyProductTitleProposal, decideProductTitleProposal, fetchProductTitleProposal, ProductTitleReviewError } from "@/lib/api/product-title-review-client";
import { ConsoleState } from "../console/console-page";
import { TitleReviewControls, TitleReviewDetail } from "./title-review-detail";
import type { ContentValue, ProductTitleProposal } from "@/lib/api/product-title-review";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";

const queryPolicy = { gcTime: 0, staleTime: 0, retry: false, refetchOnWindowFocus: false, refetchOnReconnect: false, refetchInterval: false } as const;
const errorCode = (error: unknown) => error instanceof ProductTitleReviewError ? error.code : "DEPENDENCY_UNAVAILABLE";
const denied = (error: unknown) => error instanceof ProductTitleReviewError && (error.status === 401 || error.status === 403 || error.code.startsWith("ORGANIZATION_"));

type Intent = { kind: "decision"; request: Omit<Parameters<typeof decideProductTitleProposal>[0], "signal"> } | { kind: "apply"; request: Omit<Parameters<typeof applyProductTitleProposal>[0], "signal"> };
export function ProposalSession({ proposalId, organizationId, scope, userId, roles, permissions, onLocked, onDenied, onChanged, interactionLocked = false }: {
  proposalId: string; organizationId: string; scope: string; userId: string; roles: string[]; permissions:string[];
  onLocked: (locked: boolean) => void; onDenied: (error: unknown) => void; onChanged: () => void;
  interactionLocked?: boolean;
}) {
  const [read, setRead] = useState(0);
  const queryClient = useQueryClient();
  const [writeError, setWriteError] = useState<unknown>();
  const [pending, setPending] = useState(false);
  const [uncertain, setUncertain] = useState(false);
  const operation = useRef<Intent | undefined>(undefined);
  const writing = useRef(false);
  const mounted = useRef(false);
  const abort = useRef<AbortController | undefined>(undefined);
  const queryKey = ["workbench", scope, "title-proposal", proposalId, read];
  const query = useQuery({ ...queryPolicy, queryKey, queryFn: ({ signal }) => fetchProductTitleProposal({ organizationId, proposalId, signal }) });
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; abort.current?.abort(); operation.current = undefined; }; }, []);
  useEffect(() => { if (query.isError && denied(query.error)) onDenied(query.error); }, [query.isError, query.error, onDenied]);
  const data = query.data;
  function reread() { if (writing.current) return; if (!uncertain) setWriteError(undefined); setRead((old) => old + 1); }
  async function submit(intent: Intent) {
    if (writing.current || interactionLocked) return;
    writing.current = true; operation.current = intent; setPending(true); onLocked(true); setWriteError(undefined);
    const controller = new AbortController(); abort.current = controller;
    try {
      const next = intent.kind === "decision" ? await decideProductTitleProposal({ ...intent.request, signal: controller.signal }) : await applyProductTitleProposal({ ...intent.request, signal: controller.signal });
      if (!mounted.current) return;
      operation.current = undefined; setUncertain(false); onLocked(false);
      if (uncertain) setRead((old) => old + 1); else queryClient.setQueryData(queryKey, next);
      onChanged();
    } catch (error) {
      if (!mounted.current) return;
      if (denied(error)) {
        operation.current = undefined;
        if(error instanceof ProductTitleReviewError&&error.status===403&&["permission_denied","PERMISSION_DENIED"].includes(error.code)){
          setUncertain(false);onLocked(false);setWriteError(error);void query.refetch();return;
        }
        onDenied(error); return;
      }
      const unknown = uncertain || !(error instanceof ProductTitleReviewError) || error.outcome === "unknown";
      setUncertain(unknown); setWriteError(error); onLocked(unknown);
      if (!unknown) operation.current = undefined;
    } finally { if (mounted.current) { writing.current = false; setPending(false); } }
  }
  function decide(action: "accept" | "edit" | "reject", title?: string, content?: ContentValue[]) {
    if (!data || interactionLocked || uncertain || writing.current || query.isFetching) return;
    void submit({ kind: "decision", request: { organizationId, proposalId, idempotencyKey: crypto.randomUUID(), input: action === "edit" ? data.policy === "product-content-review-v1" ? {action,expected_revision:data.revision,content:content??[]} : { action, expected_revision: data.revision, title: title ?? "" } : { action, expected_revision: data.revision } } });
  }
  function apply() {
    if (!data || interactionLocked || data.state !== "accepted" || uncertain || writing.current || query.isFetching) return;
    void submit({ kind: "apply", request: { organizationId, proposalId, idempotencyKey: crypto.randomUUID(), input: { expected_revision: data.revision } } });
  }
  const conflict = writeError && ["stale_product_version", "operation_conflict"].includes(errorCode(writeError)) && !uncertain;
  return <>
    {uncertain ? <ConsoleState kind="error" title="结果待核实"><p>请求可能已经提交。重新读取未看到回执也不能证明没有提交。只能显式核实本次操作。</p><Button variant="outline" disabled={pending || query.isFetching} onClick={() => { if (operation.current) void submit(operation.current); }}>核实本次操作</Button></ConsoleState> : null}
    {pending ? <p role="status" className="my-4 text-sm">正在提交；停止等待不撤销服务器事实。<Button variant="outline" onClick={() => abort.current?.abort()}>停止等待</Button></p> : null}
    <Button variant="outline" className="mt-3" disabled={pending || query.isFetching} onClick={reread}>重新读取当前状态</Button>
    {query.isPending || query.isFetching ? <ConsoleState kind="loading" title="正在读取提案详情">重新校验访问权限和当前修订。</ConsoleState>
      : query.isError ? <ReviewError error={query.error} recover={reread} />
      : conflict ? <ReviewError error={writeError} recover={reread} />
      : data ? <>
        {writeError && !uncertain ? <ReviewError error={writeError} /> : null}
        {!uncertain ? <TitleReviewControls key={`${data.revision}:${read}`} proposal={data} roles={roles} permissions={permissions} userId={userId} pending={pending || interactionLocked} decide={decide} apply={apply} /> : null}
        <TitleReviewDetail proposal={data} />
        {!uncertain && !pending && !writeError ? <OwnProductApplyReturn proposal={data} organizationId={organizationId} userId={userId} /> : null}
      </> : null}
  </>;
}

function OwnProductApplyReturn({ proposal, organizationId, userId }: { proposal: ProductTitleProposal; organizationId: string; userId: string }) {
  // The authorized Review response already carries the immutable own binding.
  // Never accept a caller-provided return URL or infer own identity from a key.
  const own = proposal.policy === "product-content-review-v1" && proposal.state === "applied" && proposal.apply_receipt &&
    proposal.execution.scope.OrganizationID === organizationId && proposal.execution.scope.ActorID === userId &&
    proposal.execution.binding.ContextKind === "own_product" && proposal.execution.binding.TargetPlatform === "product" &&
    isAcquisitionUUID(proposal.execution.binding.ContextID) ? proposal.execution.binding.ContextID : undefined;
  const client = useQueryClient();
  const refreshEligibility = useCallback(() => {
    void client.invalidateQueries({ queryKey: ["supply-market", userId, organizationId] });
  }, [client, userId, organizationId]);
  useEffect(() => { if (own) refreshEligibility(); }, [own, proposal.apply_receipt?.publication_id, refreshEligibility]);
  if (!own) return null;
  return <section aria-label="自有商品应用结果" className="mt-4 space-y-3">
    <p>自有商品内容已应用；返回优选申请后会重新核对当前资格。</p>
    <div className="flex flex-wrap gap-3">
      <Button asChild><Link href="/workbench/supply/applications/new" prefetch={false} onClick={refreshEligibility}>返回优选申请并刷新资格</Link></Button>
      <Button asChild variant="outline"><Link href={`/workbench/data/mine/own/${own}`} prefetch={false}>查看自有商品</Link></Button>
    </div>
  </section>;
}

export function ReviewError({ error, recover }: { error: unknown; recover?: () => void }) {
  const context = useWorkbenchContext();
  const alive = useRef(false);
  const [recovering, setRecovering] = useState(false);
  useEffect(() => { alive.current = true; return () => { alive.current = false; }; }, []);
  const messages: Record<string, string> = {
    PERMISSION_DENIED: "当前身份没有标题审核权限", permission_denied: "当前身份没有标题审核权限",
    AUTHENTICATION_REQUIRED: "登录状态已失效", ORGANIZATION_ACCESS_REVOKED: "当前企业访问已撤销", ORGANIZATION_ACCESS_DENIED: "当前企业访问被拒绝",
    ORGANIZATION_SUSPENDED: "当前企业已暂停访问", ORGANIZATION_CONTEXT_CHANGED: "企业上下文已变化", ORGANIZATION_SELECTION_REQUIRED: "请重新选择企业",
    stale_product_version: "资料已变化，请重新读取", operation_conflict: "资料已变化，请重新读取",
    INVALID_REQUEST: "请求内容不合法，请检查标题与提案标识", invalid_request: "请求内容不合法，请检查标题与提案标识", input_too_large: "标题或请求内容过长", INPUT_TOO_LARGE: "标题或请求内容过长",
    not_found: "提案不存在或当前身份不可读取", INVALID_UPSTREAM_RESPONSE: "提案响应不合法", DEADLINE_EXCEEDED: "读取提案超时", deadline_exceeded: "读取提案超时",
    DEPENDENCY_UNAVAILABLE: "标题提案服务暂不可用", unavailable: "标题提案服务暂不可用",
  };
  async function restore() {
    setRecovering(true); const next = await context.retry();
    if (!alive.current) return; setRecovering(false);
    const org = next?.organizations.find((item) => item.id === next.effectiveOrganizationId);
    if (next && !next.selectionRequired && next.user.id === context.user?.id && org?.id === context.effectiveOrganization?.id && JSON.stringify(org?.roles) === JSON.stringify(context.roles)) recover?.();
  }
  return <ConsoleState kind="error" title={messages[errorCode(error)] ?? "标题提案请求失败"}><p>未取得可用的当前结果。请重新授权读取；不会自动重发写请求。</p>
    {recover && denied(error) ? <Button variant="outline" disabled={recovering} onClick={restore}>重新加载企业上下文</Button> : null}
  </ConsoleState>;
}

export { queryPolicy as reviewQueryPolicy, denied as reviewAccessDenied };
