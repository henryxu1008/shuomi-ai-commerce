"use client";

import {useEffect,useMemo,useRef,useState} from "react";
import Link from "next/link";
import {useRouter} from "next/navigation";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {OwnProductContents} from "./own-product-detail-page";
import {readCollectionItem,readCollectionBatch,type CollectionScope} from "@/lib/api/product-collection";
import {listSupplySources,type SupplyIntent} from "@/lib/api/supply-chain";
import {completionDraftHref} from "@/lib/contracts/completion-draft-navigation";
import {transferInputSchema,transferReceiptSchema,type SupplyPreparation} from "@/lib/contracts/supply-chain";
import type {CollectionDetail} from "@/lib/contracts/product-collection";
import {useSupplyCommand} from "../supply/use-supply-command";
import {SupplyCommandFeedback} from "../supply/command-feedback";
import {skillLaunchHref,skillLaunchMatches,type SkillLaunchInput,type SkillLaunch} from "@/lib/contracts/skill-launch";
import {SkillLaunchError,SkillLaunchSummary,useLaunchVersion} from "../skills/skill-launch";

type Basis={detail:CollectionDetail;batch:Awaited<ReturnType<typeof readCollectionBatch>>};
type Completed={preparation:SupplyPreparation;intent:SupplyIntent};
const library="/workbench/data/mine?view=own";
function selectedItem(intent:SupplyIntent,itemId:string){
 const input=transferInputSchema.safeParse(intent.command);
 return intent.route==="transfer"&&input.success&&input.data.itemIds?.length===1&&input.data.itemIds[0]===itemId;
}

export function OwnProductPreparePage({itemId,launch}:{itemId:string;launch?:SkillLaunchInput}){
 const context=useWorkbenchContext(),userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||context.error||context.blockingError||context.selectionRequired)return <ConsoleState kind="error" title="请先确认登录身份与当前企业"/>;
 if(launch&&!skillLaunchMatches(launch,scope,"product-image"))return <SkillLaunchError/>;
 if(launch&&!["listingkit.image_agent.read","listingkit.image_agent.write"].every(permission=>context.permissions.includes(permission)))return <ConsoleState kind="unavailable" title="当前身份没有图片处理权限"/>;
 if(!["workbench.collection.read","workbench.supply.read","workbench.supply.manage"].every(permission=>context.permissions.includes(permission)))return <ConsoleState kind="unavailable" title="当前身份没有转入与填写草稿权限"/>;
 return <ScopedPrepare key={JSON.stringify([scope,itemId,launch])} scope={scope} itemId={itemId} launch={launch as SkillLaunch|undefined}/>;
}

function ScopedPrepare({scope,itemId,launch}:{scope:CollectionScope;itemId:string;launch?:SkillLaunch}){
 const router=useRouter();
 const version=useLaunchVersion(launch,scope);
 const [basis,setBasis]=useState<Basis>(),[loading,setLoading]=useState(true),[loadError,setLoadError]=useState(false),[reload,setReload]=useState(0);
 const [submitted,setSubmitted]=useState(false),[completed,setCompleted]=useState<Completed>(),[opening,setOpening]=useState(false),[openError,setOpenError]=useState(false);
 const alive=useRef(true),lookup=useRef<AbortController|null>(null);
 useEffect(()=>{alive.current=true;return()=>{alive.current=false;lookup.current?.abort()}},[]);

 async function openCompleted(value:Completed){
  lookup.current?.abort();const controller=new AbortController();lookup.current=controller;setOpening(true);setOpenError(false);
  try{
   const page=await listSupplySources(scope,value.preparation.id,{limit:2},controller.signal),source=page.items[0];
   if(page.total!==1||page.items.length!==1||page.nextCursor||!source||source.preparationId!==value.preparation.id||source.collectionItemId!==itemId||source.source.kind!=="own")throw new Error("Fixed source differs from selected item");
   const input=transferInputSchema.parse(value.intent.command);
   if(basis?.batch.id===input.batchId&&basis.batch.revision===input.expectedRevision&&(source.collectionRevision!==basis.detail.item.revision||JSON.stringify(source.source)!==JSON.stringify(basis.detail.item.source)))throw new Error("Fixed source differs from confirmed version");
   if(alive.current&&!controller.signal.aborted)router.push(launch?skillLaunchHref(`/workbench/data/mine/own/${itemId}/images?${new URLSearchParams({source:source.id,preparation:source.preparationId})}`,launch):completionDraftHref(source.id,source.preparationId,source.collectionItemId));
  }catch{if(alive.current&&!controller.signal.aborted)setOpenError(true)}
  finally{if(alive.current&&!controller.signal.aborted)setOpening(false)}
 }
 const command=useSupplyCommand(scope,(result,intent)=>{
  if(!selectedItem(intent,itemId))return;
  const receipt=transferReceiptSchema.safeParse(result),input=transferInputSchema.parse(intent.command);
  if(!receipt.success||receipt.data.preparation.count!==1||receipt.data.preparation.sourceBatchId!==input.batchId||receipt.data.preparation.sourceRevision!==input.expectedRevision){setOpenError(true);return}
  const value={preparation:receipt.data.preparation,intent};setCompleted(value);setSubmitted(true);void openCompleted(value);
 });
 const otherPending=!!command.pending&&!selectedItem(command.pending,itemId);
 useEffect(()=>{
  const controller=new AbortController();let live=true;
  void(async()=>{
   const first=await readCollectionItem(scope,itemId,controller.signal);
   if(first.item.source.kind!=="own")throw new Error("Own product required");
   const batch=await readCollectionBatch(scope,first.item.batchId,controller.signal);
   // The aggregate version precedes the displayed snapshot. The existing
   // transfer CAS rejects every edit/move/archive after this batch read.
   const detail=await readCollectionItem(scope,itemId,controller.signal);
   if(detail.item.source.kind!=="own"||detail.item.batchId!==batch.id||!batch.supplyTransferSupported)throw new Error("Collection changed");
   if(live){setBasis({detail,batch});setLoadError(false)}
  })().catch(()=>{if(live)setLoadError(true)}).finally(()=>{if(live)setLoading(false)});
  return()=>{live=false;controller.abort()};
 },[scope,itemId,reload]);
 function refresh(){setBasis(undefined);setLoading(true);setLoadError(false);setSubmitted(false);setReload(value=>value+1)}
 const blocked=loading||loadError||!basis||submitted||!!completed||!!command.pending||command.busy||opening||!command.ready||!version.ready;
 const waiting=completed?"/workbench/supply/mine/waiting?"+new URLSearchParams({preparation:completed.preparation.id,mode:"local"}):"";
 return <ConsolePage title={launch?"确认商品图片来源":"转入并填写待补全草稿"} description={launch?"确认单件固定来源后，直接选择已有原图。此步骤不生成图片。":"确认这一件商品及其原始版本，转入后填写独立候选资料。"} breadcrumbs={[{label:"我的数据",href:library},{label:"自有商品资料",href:"/workbench/data/mine/own/"+itemId},{label:"转入待补全"}]} actions={<Button asChild variant="outline"><Link href={"/workbench/data/mine/own/"+itemId}>返回商品资料</Link></Button>}>
  {launch?<SkillLaunchSummary launch={launch} version={version.version}/>:null}
  {version.error?<p role="alert" className="my-4 text-sm text-destructive">{version.error}</p>:launch&&!version.ready?<p role="status">正在核实准确技能版本…</p>:null}
  {otherPending?<ConsoleState kind="unavailable" title="请先完成已有供应链操作">当前原操作属于另一件商品或业务，请从原入口核实。</ConsoleState>:<>
   <SupplyCommandFeedback state={{...command,error:submitted||command.pending?command.error:null}}/>
   <Card className="mb-6 space-y-3 border-border p-5"><h2 className="font-semibold">确认单件来源</h2><p className="text-sm text-muted-foreground">只转入当前商品。原始版本固定保存，之后编辑自有商品不会改变这份来源。{launch?"转入后直接选择原图，无需填写标题或描述草稿。":"转入后还需填写并保存候选标题和描述。"}</p>
    {basis?<><p className="font-medium">{basis.detail.product.title}</p><p className="text-sm">确认来源版本 <strong>{basis.detail.item.source.version}</strong> · 1 件商品 · 未绑定店铺</p></>:null}
    {completed?<div className="space-y-3"><p role="status">已转入 1 件商品，待补全草稿尚未保存。</p>{opening?<p role="status">正在读取已固定来源…</p>:null}{openError?<p role="alert">已转入来源暂时无法读取，请重读原结果。</p>:null}<Button disabled={opening} onClick={()=>void openCompleted(completed)}>重新读取已转入来源</Button><Button asChild variant="outline"><Link href={waiting}>查看已转入商品</Link></Button></div>:<div className="flex flex-wrap gap-3"><Button disabled={blocked} onClick={()=>{if(!basis)return;setSubmitted(true);void command.execute("transfer",{batchId:basis.batch.id,expectedRevision:basis.batch.revision,itemIds:[itemId]})}}>{launch?"确认转入并选择原图":"确认转入并填写草稿"}</Button><Button variant="outline" disabled={command.busy||!!command.pending||loading} onClick={refresh}>重新读取商品</Button></div>}
   </Card>
   {loading?<ConsoleState kind="loading" title="正在读取确认资料"/>:loadError?<ConsoleState kind="error" title="商品或所属批次已改变，请重新读取">核实原操作仍使用原命令，不会以当前资料重建已有来源。</ConsoleState>:basis?<OwnProductContents key={basis.detail.item.source.publicationId+":"+basis.detail.item.source.version} detail={basis.detail}/>:null}
  </>}
 </ConsolePage>;
}
