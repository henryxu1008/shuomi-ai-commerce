"use client";
import {useEffect,useMemo,useState} from "react";
import Link from "next/link";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {readSupplyPublication,type SupplyScope} from "@/lib/api/supply-chain";
import type {SupplyPublication} from "@/lib/contracts/supply-chain";
import {SheinUploadedProduct} from "./shein-uploaded-product";
import {useListingCommand} from "./use-listing-command";
import {ListingCommandFeedback} from "./command-feedback";

type Original={sourceId:string;storeId:string;recordId:string;receiptId:string};
export function StorePublicationDetail(props:Original){
 const c=useWorkbenchContext();const userId=c.user?.id,organizationId=c.effectiveOrganization?.id;const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(c.isLoading||c.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||c.selectionRequired||c.error||c.blockingError||!c.permissions.includes("workbench.supply.read"))return <ConsoleState kind="unavailable" title="当前身份不能读取原上传商品"/>;
 return <ScopedDetail key={JSON.stringify([scope,props,c.permissions])} {...props} scope={scope}/>;
}
function ScopedDetail({scope,sourceId,storeId,recordId,receiptId}:Original&{scope:SupplyScope}){
 const c=useWorkbenchContext();const [publication,setPublication]=useState<SupplyPublication>(),[error,setError]=useState(false);
 const command=useListingCommand(scope,()=>{});
 useEffect(()=>{const controller=new AbortController();let live=true;void readSupplyPublication(scope,sourceId,storeId,controller.signal).then(v=>{if(v.sourceId!==sourceId||v.storeId!==storeId||v.receiptId!==receiptId||v.recordId!==recordId||!v.product)throw new Error("original binding");if(live)setPublication(v)}).catch(()=>{if(live)setError(true)});return()=>{live=false;controller.abort()}},[scope,sourceId,storeId,recordId,receiptId]);
 return <ConsolePage title="店铺商品详情" description="从原上传资料与回执查看平台状态并管理首次上架。" breadcrumbs={[{label:"店铺商品",href:"/workbench/store-products"},{label:"原上传商品"}]}>
  <ListingCommandFeedback state={command}/><Link className="text-sm text-primary underline" href="/workbench/store-products">返回店铺商品</Link>
  {error?<p role="alert" className="mt-4 text-sm text-destructive">原上传对象不可用，或回执与当前身份、企业、店铺不匹配。</p>:publication?<SheinUploadedProduct scope={scope} sourceId={sourceId} storeId={storeId} storeName={storeId} publication={publication} command={command} management canSubmit={c.permissions.includes("workbench.supply.manage")&&c.permissions.includes("workbench.listing.submit")&&!c.pendingCollectionIntent}/>:<p role="status">正在读取原上传对象…</p>}
 </ConsolePage>;
}
