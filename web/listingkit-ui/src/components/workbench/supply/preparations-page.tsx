"use client";

import Link from "next/link";
import {SourceMaterialsPanel} from "./source-materials-panel";
import { useEffect, useMemo, useState } from "react";
import { z } from "zod";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { CollectionDialog } from "../collections/collection-page";
import { SupplyConfiguration, SupplyConfigurationStep, SupplyContinuationNotice } from "./supply-configuration";
import { listSupplyPreparations, readSupplyPreparation, listSupplySources, readSupplySource, type SupplyScope } from "@/lib/api/supply-chain";
import { preparationPageSchema, sourcePageSchema, sourceDetailSchema, type SupplyPreparation } from "@/lib/contracts/supply-chain";
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
  const [batch, setBatch] = useState<SupplyPreparation>();
  const [batchPicker, setBatchPicker] = useState(false);
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
      const page = initialPreparation && !batchPicker ? { items: [await readSupplyPreparation(scope, selected || initialPreparation, controller.signal)], total: 1 } : await listSupplyPreparations(scope, { after }, controller.signal);
      const rows = selected ? await listSupplySources(scope, selected, { after: sourceAfter }, controller.signal) : { items: [], total: 0 };
      if (alive) { setBatches(page); setSources(rows); const current = page.items.find(item => item.id === selected); if (current) setBatch(current); }
    };
    void load().catch(() => { if (alive) setError(true); }).finally(() => { if (alive) setLoading(false); });
    return () => { alive = false; controller.abort(); };
  }, [scope, initialPreparation, selected, after, sourceAfter, reload, batchPicker]);
  useEffect(() => {
    const controller = new AbortController(); let alive = true;
    if (detailID) void readSupplySource(scope, detailID, controller.signal).then(value => { if (alive) setDetail(value); }).catch(() => { if (alive) setError(true); });
    return () => { alive = false; controller.abort(); };
  }, [scope, detailID]);
  return <ConsolePage title="待上传" description="配置一次上传任务，资料缺失先补全；审核及应用后，明确继续原任务。" breadcrumbs={[{ label: "供应市场" }, { label: "我的供应链" }, { label: "待上传" }]} actions={<Button variant="outline" disabled={loading} onClick={() => {setLoading(true);setError(false);setDetail(undefined);setDetailID("");setReload(value => value + 1);}}>刷新</Button>}>
    {sourceMaterialsAvailable && pendingSource?.targetPlatform==="product" && pendingSource.itemId ? <Button variant="outline" onClick={()=>setMaterialsID(pendingSource.itemId!)}>返回来源素材并核实</Button>:null}
    {materialsID && sourceMaterialsAvailable ? <SourceMaterialsPanel scope={scope} sourceId={materialsID} onClose={()=>setMaterialsID("")}/>:null}
    <SupplyConfiguration>
      <SupplyConfigurationStep step={1} description="从我的数据选择采集批次或自有商品。">
        <Input aria-label="已选商品" value={batch ? `${batch.name || "采集批次"} · ${batch.count} 件` : ""} placeholder="请选择商品" readOnly />
        <Button variant="outline" onClick={() => { setLoading(true);setError(false);setBatchPicker(true); }}>{selected ? "更换商品批次" : "选择商品"}</Button>
        <CardDescription>固定本次所选成员，原始商品资料不变。</CardDescription>
      </SupplyConfigurationStep>
      <SupplyConfigurationStep step={2} description="先确定平台、站点及经营模式，再校验资料。">
        <Select aria-label="目标店铺" disabled><option>请选择店铺</option></Select>
        <Button variant="outline" disabled>选择目标店铺</Button>
        <CardDescription>本页为本地资料模式，店铺适配与上传尚未开放。</CardDescription>
      </SupplyConfigurationStep>
      <SupplyConfigurationStep step={3} description="适配规则必选；智能体及执行模板按需使用。">
        <Select aria-label="适配规则" disabled><option>标准适配规则（待配置店铺）</option></Select>
        <Button variant="outline" disabled>适配规则设置</Button>
        <Button variant="outline" disabled>选择智能体（可选）</Button>
        <CardDescription>需先开放店铺适配；智能体按实际能力启用。</CardDescription>
      </SupplyConfigurationStep>
      <SupplyConfigurationStep step={4} description="核对所选商品、准确版本、目标、预算与发布方式。">
        <Select aria-label="提交方式" disabled><option>提交到平台（未开放）</option></Select>
        <Button variant="outline" disabled>预览本次任务</Button>
        <CardDescription>当前可查看原始资料{completionAvailable ? "，并保存本地待补全草稿" : ""}。平台提交待相关能力开放后可用。</CardDescription>
      </SupplyConfigurationStep>
    </SupplyConfiguration>
    <SupplyContinuationNotice availability="本地资料模式不执行平台审核与上传；本地待补全草稿不代表平台已受理。" />
    {batchPicker ? <CollectionDialog title="选择商品批次" onClose={() => setBatchPicker(false)}>
      {loading ? <p>正在读取批次</p> : !batches.items.length ? <p>暂无转入批次</p> : batches.items.map(item => <Button className="w-full justify-between" variant={selected === item.id ? "default" : "outline"} key={item.id} onClick={() => { setLoading(true);setError(false);setDetail(undefined);setSelected(item.id);setBatch(item);setSourceAfter(undefined);setDetailID("");setBatchPicker(false); }}>{item.name || "采集批次"}<span>{item.count} 件</span></Button>)}
      {batches.nextCursor ? <Button variant="outline" disabled={loading} onClick={() => { setLoading(true);setError(false);setAfter(batches.nextCursor); }}>下一页批次</Button> : null}
      <Button asChild variant="outline"><Link href="/workbench/data/mine">从我的数据添加商品</Link></Button>
      <Button asChild variant="outline"><Link href="/workbench/supply/acquisition">继续采集商品</Link></Button>
    </CollectionDialog> : null}
    {error ? <ConsoleState kind="error" title="读取失败">请确认当前企业权限后重试。</ConsoleState> : selected ? loading ? <ConsoleState kind="loading" title="正在读取待上传商品" /> : <Card>
      <CardHeader className="p-4"><CardTitle><h2>原始商品资料</h2></CardTitle><CardDescription>{batch?.name || "所选批次"} · 当前显示 {sources.items.length} 件{sources.total !== undefined ? ` / 共 ${sources.total} 件` : ""}</CardDescription></CardHeader>
      <CardContent className="space-y-3 p-4 pt-0">{!sources.items.length ? <p>暂无商品</p> : sources.items.map(source => <div className="flex flex-wrap items-center gap-3 border-b py-3" key={source.id}><span className="mr-auto">{source.source.productKey} · 来源版本 {source.source.version}</span><Button variant="outline" onClick={() => {setDetail(undefined);setDetailID(source.id);}}>查看资料</Button>{sourceMaterialsAvailable && context.permissions?.includes("workbench.supply.manage")?<Button variant="outline" onClick={()=>setMaterialsID(source.id)}>来源素材</Button>:null}{completionAvailable?<Button asChild variant="outline"><Link href={`/workbench/supply/mine/missing?preparation=${source.preparationId}&source=${source.id}`}>创建 / 编辑待补全草稿</Link></Button>:null}</div>)}{sources.nextCursor ? <Button variant="outline" onClick={() => {setLoading(true);setError(false);setSourceAfter(sources.nextCursor);}}>下一页商品</Button> : null}{detail ? <section aria-label="商品资料"><h3>{detail.product.title || "未命名商品"}</h3><p>来源：{detail.source.source.productKey} · 版本 {detail.source.source.version}</p><Link href={`/workbench/data/mine?item=${detail.source.collectionItemId}`}>查看我的数据</Link></section> : null}</CardContent>
    </Card> : null}
  </ConsolePage>;
}
