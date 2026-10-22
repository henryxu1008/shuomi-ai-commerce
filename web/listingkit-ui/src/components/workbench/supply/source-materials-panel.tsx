"use client";

import { useEffect, useState } from "react";
import { z } from "zod";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { readSupplySource, readSourceProductInputs, supplyInventory, SupplyAPIError, type SupplyScope } from "@/lib/api/supply-chain";
import { sourceSelectionSchema, imageApprovalReceiptSchema, inventorySchema, type SupplySourceDetail } from "@/lib/contracts/supply-chain";
import type { SourceProductInputs } from "@/lib/contracts/source-materials";
import { CollectionDialog } from "../collections/collection-page";
import { useSupplyCommand } from "./use-supply-command";
import { SourceImageSelection, type SourceImageRole } from "./source-image-selection";
import styles from "./source-materials-panel.module.css";

type Inventory = z.infer<typeof inventorySchema>;
const selectionFor = (source: SupplySourceDetail) => sourceSelectionSchema.parse({ itemId: source.source.id, originalPublicationId: source.source.source.publicationId,
  originalSnapshotVersion: source.source.source.version, effectiveCatalogVersion: source.source.source.version, targetPlatform: "product" });
const reasons: Record<string, string> = { approved_assets_not_ready: "缺少本版本已批准素材", source_review_required: "来源资料仍需审核", product_snapshot_not_ready: "来源快照不可用" };

export function SourceMaterialsPanel({ scope, sourceId, onClose }: { scope: SupplyScope; sourceId: string; onClose: () => void }) {
  const context = useWorkbenchContext();
  if (context.isSwitching || context.isLoading || context.user?.id !== scope.userId || context.effectiveOrganization?.id !== scope.organizationId || context.error || context.blockingError || context.selectionRequired) return null;
  if (!context.permissions.includes("workbench.supply.manage")) return <CollectionDialog title="来源素材与资料检查" onClose={onClose}><p role="alert">当前身份没有来源素材权限。</p></CollectionDialog>;
  return <CollectionDialog title="来源素材与资料检查" onClose={onClose} className={`max-w-2xl rounded-2xl ${styles.panel}`}><ScopedSourceMaterials key={`${scope.userId}:${scope.organizationId}:${sourceId}`} scope={scope} sourceId={sourceId} /></CollectionDialog>;
}

function ScopedSourceMaterials({ scope, sourceId }: { scope: SupplyScope; sourceId: string }) {
  const [source, setSource] = useState<SupplySourceDetail>();
  const [inventory, setInventory] = useState<Inventory>();
  const [diagnostic, setDiagnostic] = useState<SourceProductInputs>();
  const [selected, setSelected] = useState<Record<string, SourceImageRole>>({});
  const [editing, setEditing] = useState(false);
  const [receipt, setReceipt] = useState<string>();
  const [failure, setFailure] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [reload, setReload] = useState(0);
  const command = useSupplyCommand(scope, (result, intent) => {
    if (intent.route !== "approve" || !source) return;
    const input = (intent.command as { selection: unknown }).selection;
    if (JSON.stringify(sourceSelectionSchema.parse(input)) !== JSON.stringify(selectionFor(source))) return;
    setReceipt(imageApprovalReceiptSchema.parse(result).action_id);
    setEditing(false); setLoading(true); setReload(value => value + 1);
  });
  const pending = command.pending;
  const pendingCommand = pending?.route === "approve" ? pending.command as { selection: { itemId: string; targetPlatform: string }; images: { id: string; role: SourceImageRole }[] } : undefined;
  const ownPending = !!pending && pending.userId === scope.userId && pending.organizationId === scope.organizationId && pendingCommand?.selection.itemId === sourceId && pendingCommand.selection.targetPlatform === "product";
  const visibleSelection = ownPending ? Object.fromEntries(pendingCommand!.images.map(image => [image.id, image.role])) : selected;
  const denied = command.error === "PERMISSION_DENIED" || command.error === "IDENTITY_CONTEXT_CHANGED" || command.error === "ORGANIZATION_CONTEXT_CHANGED";
  useEffect(() => {
    const controller = new AbortController(); let alive = true;
    const load = async () => {
      const original = await readSupplySource(scope, sourceId, controller.signal);
      if (!alive || controller.signal.aborted) return;
      const [assets, inputs] = await Promise.all([supplyInventory(scope, selectionFor(original), controller.signal), readSourceProductInputs(scope, original, controller.signal)]);
      const exact = original.source.source;
      if (assets.scope.tenant_id !== scope.organizationId || assets.scope.product_key !== exact.productKey || assets.scope.target_platform !== "product" || assets.scope.source_snapshot_version !== exact.version) throw new SupplyAPIError("DEPENDENCY_UNAVAILABLE", 502);
      if (!alive) return;
      setSource(original); setInventory(assets); setDiagnostic(inputs); setFailure(undefined);
      const choices: Record<string, SourceImageRole> = {};
      for (const image of assets.assets) if (image.source_asset_id && original.images.some(v => v.id === image.source_asset_id)) choices[image.source_asset_id] = image.role;
      setSelected(choices); setEditing(assets.assets.length === 0);
    };
    void load().catch(error => { if (alive) { setSource(undefined); setInventory(undefined); setDiagnostic(undefined); setFailure(error instanceof SupplyAPIError ? error.code : "DEPENDENCY_UNAVAILABLE"); } }).finally(() => { if (alive) setLoading(false); });
    return () => { alive = false; controller.abort(); };
  }, [scope, sourceId, reload]);
  const disabled = loading || !!failure || denied || command.busy || !!pending || !command.ready;
  const inputs = diagnostic?.inputs;
  return <div className="space-y-5">
    {pending ? <Card className="space-y-3 bg-muted p-5" role="status"><h3 className="text-lg font-semibold">结果待核实</h3><p className="text-sm">{ownPending ? "确认请求未得到确定回执。保留此次选择，暂不能发起新的确认。" : "另有原操作待核实，请回到原身份、企业和来源处理。"}</p>{ownPending ? <Button disabled={command.busy || !!failure || denied || loading} onClick={command.verify}>核实原确认</Button> : null}</Card> : null}
    {command.error ? <p className="text-sm text-destructive" role="alert">{command.error === "BROWSER_STORAGE_UNAVAILABLE" ? "无法保存原确认，请检查浏览器存储后重试。" : command.error === "PERMISSION_DENIED" ? "当前权限不足，原确认仍需核实。" : command.error === "REVISION_CONFLICT" ? "确认内容与原操作冲突，请核对原选择。" : "确认尚未完成，请保留原操作并核实。"}</p> : null}
    {loading ? <p role="status">正在读取来源与本版本素材…</p> : failure || denied ? <Card className="space-y-3 p-5" role="alert"><h3 className="font-semibold">当前不可操作</h3><p>{(failure === "PERMISSION_DENIED" || denied) ? "当前身份没有来源素材权限，请关闭后确认当前身份与企业。" : "来源素材尚未配置或暂时无法读取，不能据此判断是否已批准或资料就绪。"}</p><Button variant="outline" onClick={() => {setLoading(true);setReload(value => value + 1);}}>重新读取</Button></Card> : source && inventory ? <>
      <div className="space-y-2"><h3 className="text-lg font-semibold">{source.product.title || "未命名商品"}</h3><p className="break-all text-sm text-muted-foreground">来源：{source.product.sources?.map(s => s.platform || s.type).filter(Boolean).join("、") || source.source.source.productKey} · 原始版本 {source.source.source.version}</p><p className="text-sm font-semibold">来源资料只读 · 本次采用的图片由你确认</p></div>
      {!editing && !ownPending ? <section className="space-y-3"><h3 className="text-lg font-semibold text-success-foreground">本版本已批准 {inventory.assets.length} 张图片</h3>{receipt ? <p className="break-all text-xs text-muted-foreground">确认记录：{receipt} · 作用版本 {source.source.source.version}</p> : null}<Button variant="outline" disabled={disabled} onClick={() => setEditing(true)}>重新选择图片</Button></section> : <section className="space-y-4"><h3 className="text-lg font-semibold">选择图片及用途</h3><SourceImageSelection images={source.images} selected={visibleSelection} disabled={disabled} onChange={setSelected} />{!source.images.length ? <p>来源没有可确认的图片。</p> : null}<p className="text-sm text-muted-foreground">选择图片不会自动成为正式素材；本次确认将替换本版本采用的完整集合。</p><Button className="min-h-11 w-full rounded-2xl sm:w-70" disabled={disabled || !Object.keys(selected).length || Object.keys(selected).length > 40} onClick={() => command.execute("approve", { selection: selectionFor(source), images: Object.entries(selected).map(([id, role]) => ({ id, role })), approved: [] })}>确认采用所选图片</Button></section>}
      {inputs && !ownPending ? <Card className="space-y-3 bg-muted p-5"><h3 className="text-lg font-semibold">{inputs.status === "ready" ? "本版本资料就绪" : inventory.assets.length ? "图片已确认，资料仍待处理" : "资料暂不就绪"}</h3>{inputs.reasons.map(reason => <p key={reason} className="text-sm">{reasons[reason] || reason}</p>)}{inputs.diagnostics.warnings.map((warning, i) => <p key={i} className="text-sm">{warning.message} {warning.field ? `（${warning.field}）` : ""}</p>)}{inputs.diagnostics.review_reasons.map((reason, i) => <p key={`review:${i}`} className="text-sm">{reason}</p>)}<p className="text-xs text-muted-foreground">仅检查商品资料输入；平台规则、店铺与上传条件尚未评估。图片批准不会清除来源警告。</p><Button variant="outline" disabled={command.busy} onClick={() => {setLoading(true);setReload(value => value + 1);}}>重新读取资料诊断</Button></Card> : null}
    </> : null}
  </div>;
}
