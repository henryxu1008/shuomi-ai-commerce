"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { useWorkbenchStores } from "@/lib/query/use-workbench-stores";
import { GoogleRequestError, googleRequest, type GoogleResponse, type GoogleScope } from "@/lib/api/google-merchant";

export function GoogleUploadPage({sourceId,initialPlan,preparation}:{sourceId:string;initialPlan?:string;preparation?:string}) {
  const context=useWorkbenchContext();
  const scope={userId:context.user?.id??"",organizationId:context.effectiveOrganization?.id??""};
  if(!scope.userId||!scope.organizationId||context.isSwitching)return <p role="status">正在确认企业权限…</p>;
  return <ScopedGoogleUpload key={scope.userId+":"+scope.organizationId+":"+sourceId+":"+(initialPlan??"")} scope={scope} sourceId={sourceId} initialPlan={initialPlan} preparation={preparation}/>;
}

function ScopedGoogleUpload({scope,sourceId,initialPlan,preparation}:{scope:GoogleScope;sourceId:string;initialPlan?:string;preparation?:string}) {
  const productPath = "/workbench/listing/google/"+sourceId;
  const preparationQuery = preparation ? "&preparation="+preparation : "";
  const recordPath = (planId: string) => productPath+"?plan="+planId+preparationQuery;
  const storeQuery=useWorkbenchStores({platform:"googlemerchant",page:1,pageSize:100});
  const options=useQuery({queryKey:["google-upload-options",scope.organizationId,scope.userId,sourceId],
    queryFn:()=>googleRequest(scope,"options",sourceId),enabled:!initialPlan,retry:false});
  const [store,setStore]=useState("");
  const [variant,setVariant]=useState("");
  const [image,setImage]=useState("");
  const [link,setLink]=useState("");
  const [price,setPrice]=useState("");
  const [currency,setCurrency]=useState("");
  const [availability,setAvailability]=useState("IN_STOCK");
  const [condition,setCondition]=useState("NEW");
  const [color,setColor]=useState("");
  const [size,setSize]=useState("");
  const [plan,setPlan]=useState<GoogleResponse<"prepare">>();
  const [result,setResult]=useState<GoogleResponse<"result">>();
  const [busy,setBusy]=useState(false);
  const [error,setError]=useState("");
  const [submitted,setSubmitted]=useState(Boolean(initialPlan));
  const [prepareKey,setPrepareKey]=useState("");
  const [showSavedPlan,setShowSavedPlan]=useState(false);
  const confirmation = submitted ? (showSavedPlan ? result?.pendingPlan : undefined) : plan;
  async function perform(work:()=>Promise<void>,message:string){
    setBusy(true);setError("");
    try {await work();} catch (failure) {
      const code = failure instanceof GoogleRequestError ? failure.code : "";
      setError(code==="NOT_READY" ? "这项商品已有提交记录，或当前还不能上传。请先查询原记录。" :
        code==="PERMISSION_DENIED" || code.startsWith("ORGANIZATION_ACCESS_") ? "当前账号没有执行此操作的权限。" : message);
    } finally {setBusy(false);}
  }
  async function preview(){
    const key=prepareKey||crypto.randomUUID();
    setPrepareKey(key);
    await perform(async()=>setPlan(await googleRequest(scope,"prepare",sourceId,{sourceId,storeId:store,imageId:image,
      variants:[{variantId:variant,link,price,currency,availability,condition,color,size}]},key)),
    "无法生成上传预览。请核对店铺服务、商品资料和已审核的图片。");
  }
  async function confirm(){
    if(!confirmation)return;
    const planId=confirmation.id;
    function showRecord() {
      setSubmitted(true);
      window.history.replaceState(null,"",recordPath(planId));
    }
    await perform(async()=>{
      try {
        const saved=await googleRequest(scope,"confirm",sourceId,{sourceId,planId});
        setResult(saved);
        setShowSavedPlan(false);
        if (saved.confirmationBlocked!=="retry" || saved.items.some(item=>item.status!=="not_sent")) showRecord();
      } catch (failure) {
        setShowSavedPlan(false);
        showRecord();
        throw failure;
      }
    },"暂时无法确认提交结果。请先查询原提交，避免重复上传。");
    if(!initialPlan) await options.refetch();
  }
  async function refresh(){
    const planId=plan?.id||initialPlan;
    if(!planId)return;
    setShowSavedPlan(false);
    await perform(async()=>setResult(await googleRequest(scope,"result",sourceId,{sourceId,planId})),
      "暂时无法读取提交结果，请稍后再查。");
  }
  const stores=storeQuery.data?.items.filter(item=>item.recordStatus==="active"&&item.serviceStatus==="active"&&item.connectionStatus==="connected")??[];
  return <section className="mx-auto max-w-2xl space-y-5 p-6">
    <Link className="underline" href={"/workbench/listing/waiting?mode=google"+preparationQuery}>返回待上传商品</Link>
    <h1 className="text-xl font-semibold">上传到 Google Shopping</h1>
    {error&&<p role="alert">{error}</p>}
    {result?.confirmationBlocked==="retry"&&<p role="alert">暂时无法连接 Google。尚未提交的商品可以稍后继续确认。</p>}
    {(result?.confirmationBlocked==="reauthorize"||result?.lookupStatus==="reauthorize")&&
      <p role="alert">请联系有权限的成员在 <Link className="underline" href="/workbench/stores">店铺中心</Link> 重新授权 Google。</p>}
    {!initialPlan&&!plan&&<form className="space-y-3" onChange={()=>{if(prepareKey)setPrepareKey("");}} onSubmit={event=>{event.preventDefault();void preview();}}>
      {options.isPending&&<p role="status">正在读取商品资料…</p>}
      {options.isError&&<p role="alert">无法读取商品资料。请确认商品可访问且已审核商品图片。</p>}
      {options.data&&<p>{options.data.title}</p>}
      {options.data?.images.length===0&&<p role="status">上传前需要先审核至少一张商品图片。</p>}
      <label htmlFor="google-target">目标店铺</label>
      <Select id="google-target" value={store} onChange={e=>setStore(e.target.value)} required>
        <option value="">选择 Google 店铺</option>{stores.map(item=><option key={item.id} value={item.id}>{item.name}</option>)}
      </Select>
      {!storeQuery.isPending&&stores.length===0&&<Link className="underline" href="/workbench/stores">前往店铺中心连接并启用 Google 店铺</Link>}
      <label htmlFor="google-variant">商品规格</label>
      <Select id="google-variant" value={variant} onChange={e=>setVariant(e.target.value)} required>
        <option value="">选择规格</option>{options.data?.variants.map(item=><option key={item.id} value={item.id}>{item.label}</option>)}
      </Select>
      <label htmlFor="google-image">已审核的商品主图</label>
      <Select id="google-image" value={image} onChange={e=>setImage(e.target.value)} required>
        <option value="">选择图片</option>{options.data?.images.map((item,i)=><option value={item.id} key={item.id}>商品图 {i+1}</option>)}
      </Select>
      {options.data?.images.find(item=>item.id===image) && <p><a className="underline" target="_blank" rel="noreferrer" href={options.data.images.find(item=>item.id===image)?.url}>查看所选商品图</a></p>}
      <label htmlFor="google-link">买家可购买该商品的页面</label><Input id="google-link" type="url" pattern="https://.+" title="请输入以 https:// 开头的购买页面地址" value={link} onChange={e=>setLink(e.target.value)} required />
      <label htmlFor="google-price">销售价</label><Input id="google-price" inputMode="decimal" pattern="[0-9]{1,10}([.][0-9]{1,6})?" title="请输入数字价格，最多六位小数" value={price} onChange={e=>setPrice(e.target.value)} required />
      <label htmlFor="google-currency">币种</label><Input id="google-currency" placeholder="例如 USD" pattern="[A-Z]{3}" title="请输入三位币种代码" value={currency} onChange={e=>setCurrency(e.target.value.toUpperCase())} maxLength={3} required />
      <label htmlFor="google-availability">库存状态</label><Select id="google-availability" value={availability} onChange={e=>setAvailability(e.target.value)}><option value="IN_STOCK">有货</option><option value="OUT_OF_STOCK">缺货</option></Select>
      <label htmlFor="google-condition">商品状况</label><Select id="google-condition" value={condition} onChange={e=>setCondition(e.target.value)}><option value="NEW">全新</option><option value="USED">二手</option><option value="REFURBISHED">翻新</option></Select>
      {options.data && options.data.variants.length>1 && <><label htmlFor="google-color">颜色</label><Input id="google-color" value={color} onChange={e=>setColor(e.target.value)} /><label htmlFor="google-size">尺码</label><Input id="google-size" value={size} onChange={e=>setSize(e.target.value)} /></>}
      <Button type="submit" disabled={busy||options.isError||!options.data}>预览上传</Button>
    </form>}
    {confirmation&&<section className="space-y-3" aria-label="待确认商品">
      <h2 className="font-semibold">核对即将提交的商品</h2>
      {submitted&&<p>本次只提交仍待确认的商品。</p>}
      <dl>
        <dt>目标店铺</dt><dd>{stores.find(item=>item.id===confirmation.storeId)?.name}</dd>
        <dt>Merchant Center 账号</dt><dd>{confirmation.accountId}</dd>
        <dt>商品数据源</dt><dd>{confirmation.dataSourceId}</dd>
        <dt>商品语言与数据源标签</dt><dd>{confirmation.contentLanguage} / {confirmation.feedLabel}</dd>
      </dl>
      {confirmation.offers.map(item=><dl key={item.ID}>
        <dt>商品</dt><dd>{item.Title}</dd>
        {options.data?.variants.find(item=>item.id===variant)?.label&&<><dt>商品规格</dt><dd>{options.data.variants.find(item=>item.id===variant)?.label}</dd></>}
        <dt>销售价</dt><dd>{item.AmountMicros/1000000} {item.Currency}</dd>
        <dt>库存和状况</dt><dd>{item.Availability==="IN_STOCK"?"有货":"缺货"} · {{NEW:"全新",USED:"二手",REFURBISHED:"翻新"}[item.Condition]}</dd>
        {(item.Color||item.Size)&&<><dt>颜色与尺码</dt><dd>{[item.Color,item.Size].filter(Boolean).join(" / ")}</dd></>}
        <dt>商品主图</dt><dd><a className="underline" target="_blank" rel="noreferrer" href={item.ImageLink}>查看已审核主图</a></dd>
        <dt>购买页面</dt><dd className="break-all">{item.Link}</dd>
        <dt>描述</dt><dd>{item.Description}</dd>
      </dl>)}
      <Button disabled={busy} onClick={()=>void confirm()}>确认上传到 Google</Button>
      {submitted ? <Button variant="outline" disabled={busy} onClick={()=>setShowSavedPlan(false)}>返回提交记录</Button> :
        <Button variant="outline" disabled={busy} onClick={()=>{setPlan(undefined);setResult(undefined);setPrepareKey("");}}>返回修改</Button>}
    </section>}
    {!!options.data?.records.length&&<section aria-label="最近的 Google 上传记录" className="space-y-2">
      <h2 className="font-semibold">最近的 Google 上传记录</h2>
      <ul>{options.data.records.map(item=><li key={item.id}>
        <Link className="underline" href={recordPath(item.id)}>{new Date(item.createdAt).toLocaleString("zh-CN")} 的上传记录</Link>
      </li>)}</ul>
    </section>}
    {submitted&&!showSavedPlan&&<section className="space-y-3" aria-label="Google 提交结果"><h2 className="font-semibold">提交结果</h2>
      {initialPlan&&<Link className="underline" href={productPath+(preparation?"?preparation="+preparation:"")}>返回商品上传与记录</Link>}
      {busy&&<p role="status">正在读取 Google 提交结果…</p>}
      {result?.lookupStatus==="unavailable"&&<p role="alert">暂时无法查询 Google 的最新状态。下方保留已有的提交结果。</p>}
      {result?.items.map(item=><div key={item.offerId} className="rounded border p-3">
        <p>{item.status==="succeeded"?"Google 已收到商品":item.status==="failed_definitive"?"本次上传未成功":item.status==="not_sent"?"尚未提交":"正在核对提交结果"}</p>
        {item.remoteProduct&&<p className="break-all">平台商品编号：{item.remoteProduct}</p>}
        {item.observation&&<><p>商品已写入数据源。</p>
          {item.observation.UnexpectedDestination?<p role="alert">商品出现预期之外的展示渠道，请到 Merchant Center 确认。</p>:
            item.observation.FreeListings.Approved.length>0?<p>符合免费展示条件：{item.observation.FreeListings.Approved.join("、")}</p>:<p>免费展示资格尚未确认。</p>}
          {item.observation.Issues.length>0&&<p>Google 报告 {item.observation.Issues.length} 项商品资料问题，请到 Merchant Center 查看详情。</p>}
          {item.observation.FreeListings.Disapproved.length>0&&<p>未通过免费展示审核：{item.observation.FreeListings.Disapproved.join("、")}</p>}
          {item.observation.FreeListings.Pending.length>0&&<p>正在审核免费展示资格：{item.observation.FreeListings.Pending.join("、")}</p>}</>}
      </div>)}
      {result?.pendingPlan&&<Button disabled={busy} onClick={()=>setShowSavedPlan(true)}>继续核对原商品</Button>}
      <Button disabled={busy} variant="outline" onClick={()=>void refresh()}>查询提交结果</Button>
    </section>}
  </section>;
}
