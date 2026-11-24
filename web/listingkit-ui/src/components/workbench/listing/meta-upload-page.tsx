"use client";

import Link from "next/link";
import {useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {ConsolePage} from "@/components/workbench/console/console-page";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {Label} from "@/components/ui/label";
import {Select} from "@/components/ui/select";
import {useWorkbenchStores} from "@/lib/query/use-workbench-stores";
import {metaRequest,MetaRequestError,type MetaScope} from "@/lib/api/meta-catalog";
import type {MetaResponse} from "@/lib/contracts/meta-catalog";

type Props={sourceId:string;initialPlan?:string;preparation?:string};
const statusMessages={
  not_sent:"尚未确认上传",claimed:"正在提交",succeeded:"模拟商品目录已收到商品",
  failed_definitive:"模拟商品目录拒绝了该商品",outcome_unknown:"提交结果待确认",cancelled:"提交已取消",
};
const whatsappMessages={
  unverified:"WhatsApp 目录绑定待核验",not_selected:"未选择 WhatsApp Business 账号",
  matched:"WhatsApp 已连接此目录",different:"WhatsApp 未连接此目录",
};

export function MetaUploadPage(props:Props){
  const context=useWorkbenchContext();
  const scope={organizationId:context.effectiveOrganization?.id??"",userId:context.user?.id??""};
  if(!scope.organizationId||!scope.userId||context.isSwitching)return <p role="status">正在确认当前企业…</p>;
  const canSubmit=context.permissions.includes("workbench.supply.manage")&&context.permissions.includes("workbench.supply.submit");
  return <MetaUploadForm key={[scope.organizationId,scope.userId,props.sourceId,props.initialPlan].join(":")} {...props} scope={scope} canSubmit={canSubmit}/>;
}

function MetaUploadForm({sourceId,initialPlan,preparation,scope,canSubmit}:Props&{scope:MetaScope;canSubmit:boolean}){
  const stores=useWorkbenchStores({page:1,pageSize:100,status:"active"});
  const options=useQuery({queryKey:["meta-catalog-options",scope,sourceId],
    queryFn:()=>metaRequest(scope,"options",sourceId),retry:false});
  const [storeId,setStoreId]=useState("");
  const [variantId,setVariantId]=useState("");
  const [imageId,setImageId]=useState("");
  const [productUrl,setProductUrl]=useState("");
  const [price,setPrice]=useState("");
  const [brand,setBrand]=useState("");
  const [preview,setPreview]=useState<MetaResponse<"prepare">>();
  const [result,setResult]=useState<MetaResponse<"result">>();
  const [busy,setBusy]=useState(false);
  const [error,setError]=useState("");
  const shopOptions=stores.data?.items.filter(store=>store.platform==="metacatalog")??[];
  const planId=preview?.id??initialPlan;
  const originalBrand=options.data?.brand??"";
  const back="/workbench/listing/waiting"+(preparation?"?preparation="+preparation:"");
  const path="/workbench/listing/meta-catalog/"+sourceId;

  async function act(action:"prepare"|"confirm"|"result"){
    setBusy(true);
    setError("");
    try{
      if(action==="prepare"){
        setResult(undefined);
        setPreview(undefined);
        setPreview(await metaRequest(scope,"prepare",sourceId,{sourceId,storeId,variantId,imageId,productUrl,price,brand:originalBrand||brand},crypto.randomUUID()));
        void options.refetch();
      }else if(planId){
        const reply=await metaRequest(scope,action,sourceId,{sourceId,planId});
        setResult(reply);
      }
    }catch(error){
      if(action==="prepare"){
        setError(error instanceof MetaRequestError&&error.code==="INVALID_REQUEST"?
          "请检查商品资料和金额。售价最多保留两位小数，购买链接须使用 HTTPS。":
          "无法生成预览。请检查店铺授权和已审核主图。");
      }else if(action==="confirm"){
        setError(error instanceof MetaRequestError&&error.code==="NOT_READY"?
          "此目录商品已有提交记录，或当前授权不可用。请查询原提交。":"暂时无法确认提交结果。请查询这次提交。");
      }else{setError("暂时无法查询结果，请稍后重试。");}
    }finally{setBusy(false);}
  }

  return <ConsolePage className="space-y-5" title="上传到 Meta 商品目录"
    breadcrumbs={[{label:"商品刊登",href:back},{label:"上传到 Meta 商品目录"}]}
    description="模拟上传同一个目录商品，并分别查询目录结果和 WhatsApp 绑定。">
    {options.isError&&<p role="alert">暂时无法读取此自有商品。</p>}
    {stores.isError&&<p role="alert">暂时无法读取店铺。</p>}
    <section aria-label="上传资料" className="space-y-4 rounded-lg border p-5">
      <h2 className="text-lg font-semibold">{options.data?.title??"商品资料"}</h2>
      {canSubmit&&<form className="space-y-4" onChange={()=>{setPreview(undefined);setResult(undefined);}}
        onSubmit={event=>{event.preventDefault();void act("prepare");}}>
        <fieldset disabled={busy} className="grid gap-4 sm:grid-cols-2">
          <div><Label htmlFor="meta-store">目标店铺</Label><Select required id="meta-store" value={storeId} onChange={event=>setStoreId(event.target.value)}>
            <option value="">选择已授权的 Meta 店铺</option>{shopOptions.map(store=><option key={store.id} value={store.id}>{store.name}</option>)}
          </Select></div>
          <div><Label htmlFor="meta-variant">商品规格</Label><Select required id="meta-variant" value={variantId} onChange={event=>setVariantId(event.target.value)}>
            <option value="">选择规格</option>{options.data?.variants.map(item=><option key={item.id} value={item.id}>{item.label}</option>)}
          </Select></div>
          <div><Label htmlFor="meta-image">已审核的商品主图</Label><Select required id="meta-image" value={imageId} onChange={event=>setImageId(event.target.value)}>
            <option value="">选择主图</option>{options.data?.images.map((image,index)=><option key={image.id} value={image.id}>主图 {index+1}</option>)}
          </Select></div>
          <div><Label htmlFor="meta-brand">商品品牌</Label><Input required id="meta-brand" maxLength={100} value={originalBrand||brand}
            readOnly={Boolean(originalBrand)} onChange={event=>setBrand(event.target.value)}/></div>
          <div><Label htmlFor="meta-url">买家可购买该商品的页面</Label><Input required id="meta-url" type="url" value={productUrl} onChange={event=>setProductUrl(event.target.value)}/></div>
          <div><Label htmlFor="meta-price">销售价（USD）</Label><Input required id="meta-price" inputMode="decimal"
            pattern="[0-9]+([.][0-9]{1,2})?" value={price} onChange={event=>setPrice(event.target.value)}/></div>
        </fieldset>
        <p className="text-sm text-muted-foreground">本次模拟按新品、现货提交。</p>
        <Button type="submit" disabled={busy||!options.data||shopOptions.length===0}>预览上传</Button>
        <Link className="ml-3 underline" href="/workbench/stores">前往店铺中心</Link>
      </form>}
    </section>
    {preview&&<section aria-label="待确认商品" className="space-y-3 rounded-lg border p-5">
      <h2 className="text-lg font-semibold">核对即将提交的商品</h2>
      <p>目标目录 {preview.catalogId}</p><p>{preview.product.title}</p><p>品牌 {preview.product.brand} · 规格 {preview.variantLabel}</p>
      <p className="whitespace-pre-wrap break-words">{preview.product.description}</p>
      <p>USD {(preview.product.priceCents/100).toFixed(2)}</p>
      <p className="break-all">购买链接 {preview.product.productUrl}</p>
      <p className="break-all">主图 {preview.product.imageUrl}</p>
      <Button disabled={busy||Boolean(result&&result.status!=="not_sent")} onClick={()=>void act("confirm")}>确认上传到 Meta</Button>
    </section>}
    {planId&&<section aria-label="提交结果" className="space-y-3 rounded-lg border p-5">
      <h2 className="text-lg font-semibold">提交结果</h2>
      <Button variant="outline" disabled={busy} onClick={()=>void act("result")}>查询提交结果</Button>
      {result&&<div role="status" className="space-y-1">
        <p>{statusMessages[result.status]}</p>
        {result.remoteId&&<p>目录商品编号 {result.remoteId}</p>}
        {result.observation&&<>
          <p>{result.observation.catalog==="found"?"已在模拟目录中查到此商品":"暂未核实目录内的商品"}</p>
          <p>{whatsappMessages[result.observation.whatsApp]}</p>
          <p>Facebook 商店展示待核验</p>
        </>}
      </div>}
    </section>}
    {error&&<p role="alert">{error}</p>}
    {Boolean(options.data?.records.length)&&<section aria-label="最近的 Meta 上传记录" className="space-y-2 rounded-lg border p-5">
      <h2 className="text-lg font-semibold">最近的 Meta 上传记录</h2>
      <ul>{options.data?.records.map(record=><li key={record.id}><Link className="underline"
        href={path+"?plan="+record.id+(preparation?"&preparation="+preparation:"")}>{new Date(record.createdAt).toLocaleString("zh-CN")} 的上传记录</Link></li>)}</ul>
    </section>}
  </ConsolePage>;
}
