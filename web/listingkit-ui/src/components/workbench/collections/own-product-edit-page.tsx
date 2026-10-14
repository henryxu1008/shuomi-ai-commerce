"use client";

import {useEffect,useMemo,useState} from "react";
import Link from "next/link";
import {useRouter} from "next/navigation";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {readCollectionItem,CollectionAPIError,type CollectionScope} from "@/lib/api/product-collection";
import type {CollectionDetail} from "@/lib/contracts/product-collection";
import {Button} from "@/components/ui/button";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {OwnProductForm} from "./own-product-entry-page";
import {hydrateOwnProduct,type OwnProductFormState} from "./own-product-editor-model";
import {collectionFailureText,useCollectionCommand} from "./use-collection-command";
import {CollectionCommandFeedback} from "./collection-command-feedback";

export function OwnProductEditPage({itemId}:{itemId:string}){
 const context=useWorkbenchContext(),userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||context.selectionRequired||context.error||context.blockingError)return <ConsoleState kind="error" title="请先确认登录身份与当前企业"/>;
 return <ScopedEditor key={`${scope.userId}:${scope.organizationId}:${itemId}`} scope={scope} itemId={itemId}/>;
}
function ScopedEditor({scope,itemId}:{scope:CollectionScope;itemId:string}){
 const [edit,setEdit]=useState<{detail:CollectionDetail;initial:OwnProductFormState}|null>(null),[error,setError]=useState(""),[reload,setReload]=useState(0);
 useEffect(()=>{const controller=new AbortController();let active=true;
  void readCollectionItem(scope,itemId,controller.signal).then(detail=>{if(!active)return;if(detail.item.id!==itemId||detail.item.source.kind!=="own")throw new CollectionAPIError("NOT_FOUND",404);setEdit({detail,initial:hydrateOwnProduct(detail)});setError("")}).catch(failure=>{if(active)setError(failure instanceof CollectionAPIError?collectionFailureText[failure.code]??"暂时无法读取商品资料":failure instanceof Error?failure.message:"暂时无法读取商品资料")});
  return()=>{active=false;controller.abort()};
 },[scope,itemId,reload]);
 if(edit&&!error)return <OwnProductForm scope={scope} edit={edit}/>;
 return <ConsolePage title="编辑自有商品" description="读取已保存的准确商品资料后再编辑。" breadcrumbs={[{label:"我的数据",href:"/workbench/data/mine?view=own"},{label:"编辑自有商品"}]} actions={<Button asChild variant="outline"><Link href={`/workbench/data/mine/own/${itemId}`}>返回商品详情</Link></Button>}>
  {error?<ConsoleState kind="error" title={error}><Button variant="outline" onClick={()=>setReload(value=>value+1)}>重试读取</Button></ConsoleState>:<ConsoleState kind="loading" title="正在读取商品资料"/>}
  <EditorRecovery scope={scope}/>
 </ConsolePage>;
}
function EditorRecovery({scope}:{scope:CollectionScope}){
 const router=useRouter(),context=useWorkbenchContext();
 const commands=useCollectionCommand(scope,(receipt,intent)=>{if(intent.command.action==="revise_product"&&receipt.itemId===intent.command.itemId)router.push(`/workbench/data/mine/own/${intent.command.itemId}`)},!!context.pendingSupplyIntent||!context.supplyIntentReady);
 return <CollectionCommandFeedback state={commands}/>;
}
