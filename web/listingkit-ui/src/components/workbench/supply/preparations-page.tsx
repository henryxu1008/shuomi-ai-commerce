"use client";

import Link from "next/link";
import {SourceMaterialsPanel} from "./source-materials-panel";
import { useEffect, useMemo, useState } from "react";
import { z } from "zod";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { listSupplyPreparations, readSupplyPreparation, listSupplySources, readSupplySource, type SupplyScope } from "@/lib/api/supply-chain";
import { preparationPageSchema, sourcePageSchema, sourceDetailSchema } from "@/lib/contracts/supply-chain";
import { ConsolePage, ConsoleState } from "../console/console-page";

export function PreparationsPage({ initialPreparation,completionAvailable=false,sourceMaterialsAvailable=false }: { initialPreparation?: string;completionAvailable?:boolean;sourceMaterialsAvailable?:boolean }) {
  const context = useWorkbenchContext();
  const userId=context.user?.id, organizationId=context.effectiveOrganization?.id;
  const scope = useMemo(() => userId && organizationId ? {userId,organizationId} : null, [userId,organizationId]);
  if (context.isLoading || context.isSwitching) return <ConsoleState kind="loading" title="正在确认企业上下文" />;
  if (!scope || context.error || context.blockingError || context.selectionRequired) return <ConsoleState kind="error" title="企业或登录上下文不可用" />;
  return <ScopedPreparations key={`${scope.userId}:${scope.organizationId}:${initialPreparation ?? ""}`} scope={scope} initialPreparation={initialPreparation} completionAvailable={completionAvailable} sourceMaterialsAvailable={sourceMaterialsAvailable} />;
}
function ScopedPreparations({ scope, initialPreparation,completionAvailable,sourceMaterialsAvailable }: { scope: SupplyScope; initialPreparation?: string;completionAvailable:boolean;sourceMaterialsAvailable:boolean }) {
  const context = useWorkbenchContext();
  const [materialsID,setMaterialsID] = useState("");
  const pending = context.pendingSupplyIntent;
  const pendingSource = pending?.userId===scope.userId && pending.organizationId===scope.organizationId && pending.route==="approve" ? (pending.command as {selection?:{itemId?:string;targetPlatform?:string}}).selection : undefined;
  const [batches, setBatches] = useState<z.infer<typeof preparationPageSchema>>({ items: [], total: 0 });
  const [sources, setSources] = useState<z.infer<typeof sourcePageSchema>>({ items: [], total: 0 });
  const [selected, setSelected] = useState(initialPreparation ?? "");
  const [after, setAfter] = useState<string>();
  const [sourceAfter, setSourceAfter] = useState<string>();
  const [detail, setDetail] = useState<z.infer<typeof sourceDetailSchema>>();
  const [detailID, setDetailID] = useState("");
  const [reload, setReload] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  useEffect(() => {
    const controller = new AbortController(); let alive = true;

    const load = async () => {
      const page = initialPreparation ? { items: [await readSupplyPreparation(scope, initialPreparation, controller.signal)], total: 1 } : await listSupplyPreparations(scope, { after }, controller.signal);
      const rows = selected ? await listSupplySources(scope, selected, { after: sourceAfter }, controller.signal) : { items: [], total: 0 };
      if (alive) { setBatches(page); setSources(rows); }
    };
    void load().catch(() => { if (alive) setError(true); }).finally(() => { if (alive) setLoading(false); });
    return () => { alive = false; controller.abort(); };
  }, [scope, initialPreparation, selected, after, sourceAfter, reload]);
  useEffect(() => {
    const controller = new AbortController(); let alive = true;
    if (detailID) void readSupplySource(scope, detailID, controller.signal).then(value => { if (alive) setDetail(value); }).catch(() => { if (alive) setError(true); });
    return () => { alive = false; controller.abort(); };
  }, [scope, detailID]);
  return <ConsolePage title="待适配" description={completionAvailable?"查看原始资料，先保存未绑定店铺的待补全草稿；配置店铺后继续适配。":"查看已转入的商品批次和原始资料。目标平台适配与上传待配置后开放。"} breadcrumbs={[{ label: "供应市场" }, { label: "我的供应链" }, { label: "待适配" }]} actions={<Button variant="outline" disabled={loading} onClick={() => {setLoading(true);setError(false);setDetail(undefined);setDetailID("");setReload(value => value + 1);}}>刷新</Button>}>
    {sourceMaterialsAvailable && pendingSource?.targetPlatform==="product" && pendingSource.itemId ? <Button variant="outline" onClick={()=>setMaterialsID(pendingSource.itemId!)}>返回来源素材并核实</Button>:null}
    {materialsID && sourceMaterialsAvailable ? <SourceMaterialsPanel scope={scope} sourceId={materialsID} onClose={()=>setMaterialsID("")}/>:null}
    {error ? <ConsoleState kind="error" title="读取失败">请确认当前企业权限后重试。</ConsoleState> : loading ? <ConsoleState kind="loading" title="正在读取待适配商品" /> : <div className="grid gap-4 md:grid-cols-[240px_1fr]">
      <Card className="space-y-3 p-4"><h2>商品批次</h2>{!batches.items.length ? <p>暂无转入批次</p> : batches.items.map(batch => <Button className="w-full justify-between" variant={selected === batch.id ? "default" : "outline"} key={batch.id} onClick={() => { setLoading(true);setError(false);setDetail(undefined);setSelected(batch.id); setSourceAfter(undefined); setDetailID(""); }}>{batch.name || "采集批次"}<span>{batch.count}</span></Button>)}{batches.nextCursor ? <Button variant="outline" onClick={() => { setLoading(true);setError(false);setAfter(batches.nextCursor); setSelected(""); }}>下一页批次</Button> : null}<Link href="/workbench/supply/acquisition">继续采集商品</Link></Card>
      <Card className="space-y-3 p-4"><h2>原始商品资料</h2>{!selected ? <p>请选择商品批次</p> : !sources.items.length ? <p>暂无商品</p> : sources.items.map(source => <div className="flex items-center justify-between gap-3 border-b py-3" key={source.id}><span>{source.source.productKey} · 来源版本 {source.source.version}</span><Button variant="outline" onClick={() => {setDetail(undefined);setDetailID(source.id);}}>查看资料</Button>{sourceMaterialsAvailable && context.permissions?.includes("workbench.supply.manage")?<Button variant="outline" onClick={()=>setMaterialsID(source.id)}>来源素材</Button>:null}{completionAvailable?<Button asChild variant="outline"><Link href={`/workbench/supply/mine/missing?preparation=${source.preparationId}&source=${source.id}`}>创建 / 编辑待补全草稿</Link></Button>:null}</div>)}{sources.nextCursor ? <Button variant="outline" onClick={() => {setLoading(true);setError(false);setSourceAfter(sources.nextCursor);}}>下一页商品</Button> : null}{detail ? <section aria-label="商品资料"><h3>{detail.product.title || "未命名商品"}</h3><p>来源：{detail.source.source.productKey} · 版本 {detail.source.source.version}</p><Link href={`/workbench/data/mine?item=${detail.source.collectionItemId}`}>查看我的数据</Link></section> : null}</Card>
    </div>}
  </ConsolePage>;
}
