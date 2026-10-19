"use client";
import {useEffect,useState} from "react";
import Image from "next/image";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {readSupplyRecord,type SupplyScope} from "@/lib/api/supply-chain";
import type {SupplyTarget,SupplyPublication} from "@/lib/contracts/supply-chain";
import {DescriptionView} from "../description-view";
import {SheinPublicationPanel} from "./shein-publication-panel";
import type {SupplyCommandState} from "./use-supply-command";

export function SheinUploadedProduct({scope,sourceId,storeId,storeName,publication,command,canSubmit}:{scope:SupplyScope;sourceId:string;storeId:string;storeName:string;publication:SupplyPublication;command:SupplyCommandState;canSubmit:boolean}){
 const [record,setRecord]=useState<SupplyTarget>(),[error,setError]=useState(false),[status,setStatus]=useState(false);
 useEffect(()=>{let live=true;const c=new AbortController();if(publication.recordId)void readSupplyRecord(scope,publication.recordId,c.signal).then(v=>{if(v.source.id!==sourceId||v.merchant.store_id!==storeId)throw new Error("receipt mismatch");if(live)setRecord(v)}).catch(()=>{if(live)setError(true)});return()=>{live=false;c.abort()};},[scope,sourceId,storeId,publication.recordId]);
 if(error)return <p role="alert">原上传商品资料读取失败，请关闭后重新打开。</p>;
 if(!record)return <p role="status">正在读取原上传商品资料…</p>;
 const product=record.result.product,images=product.image_info?.image_info_list??product.skc_list.flatMap(s=>s.image_info.image_info_list);
 return <div className="space-y-4"><div className="flex flex-wrap items-center justify-between gap-3"><div><h3 className="text-lg font-semibold">{product.multi_language_name_list[0]?.name||"原上传商品"}</h3><p className="text-sm text-muted-foreground">{storeName} · SHEIN · 原上传资料版本 {record.revision}</p></div><Button variant={status?"outline":"default"} onClick={()=>setStatus(v=>!v)}>{status?"返回商品资料":"平台状态"}</Button></div>
  <p className="rounded-lg bg-muted p-4 text-sm">此处只读展示原上传回执对应资料。后续本地编辑不改变该上传对象。</p>
  {status?<SheinPublicationPanel scope={scope} sourceId={sourceId} storeId={storeId} receiptId={publication.receiptId!} storeName={storeName} command={command} canSubmit={canSubmit}/>:<>
   <Card className="space-y-3 p-4"><h4 className="font-semibold">商品信息</h4><p className="break-all text-sm">SPU {publication.product?.spu_name} · 原审核版本 {publication.product?.version||"缺失，无法按原版本核实"}</p><p className="text-sm">类目 {product.category_id} · 商品类型 {product.product_type_id}</p><DescriptionView value={product.multi_language_desc_list?.[0]?.name}/><div className="flex flex-wrap gap-3">{images.map((image,i)=><Image key={i} src={"/api/image-proxy?url="+encodeURIComponent(image.image_url)} unoptimized width={112} height={112} alt={"原上传商品图片 "+(i+1)} className="rounded-md object-contain"/>)}</div><dl className="grid gap-3 text-sm md:grid-cols-2">{product.product_attribute_list.map((a,i)=><div key={i}><dt className="text-muted-foreground">属性 {a.attribute_id}</dt><dd>{a.attribute_extra_value||a.custom_attribute_value||a.attribute_value_id||"未填写"}</dd></div>)}</dl></Card>
   <Card className="space-y-3 p-4"><h4 className="font-semibold">原提交的规格、价格与库存</h4><p className="text-xs text-muted-foreground">以下为上传时提交内容，当前可售库存请在平台状态查询。</p>{product.skc_list.map((skc,i)=><div key={i} className="space-y-2 rounded-lg border border-border p-3"><p className="break-all text-sm">款式 {skc.supplier_code||i+1}</p>{skc.sku_list.map(sku=><div key={sku.supplier_sku} className="space-y-1 text-sm"><p className="break-all">{sku.supplier_sku}</p>{sku.price_info_list?.map((p,j)=><p key={j}>原售价 {p.base_price} {p.currency} · {p.sub_site}</p>)}{sku.cost_info?<p>原供货价 {sku.cost_info.cost_price} {sku.cost_info.currency}</p>:null}{sku.stock_info_list.map((s,j)=><p key={j}>原提交库存 {s.inventory_num} · {s.supplier_warehouse_name||s.supplier_warehouse_id||"未指定仓库"}</p>)}</div>)}</div>)}</Card>
  </>}
 </div>;
}
