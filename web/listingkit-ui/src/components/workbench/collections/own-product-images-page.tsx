"use client";
import {useEffect,useMemo,useState} from "react";
import Link from "next/link";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {readSupplySource,type SupplyScope} from "@/lib/api/supply-chain";
import type {SupplySourceDetail} from "@/lib/contracts/supply-chain";
import {skillLaunchMatches,type SkillLaunchInput,type SkillLaunch} from "@/lib/contracts/skill-launch";
import {ConsolePage,ConsoleState} from "../console/console-page";
import {ProductImageSetPanel} from "../acquisition/product-image-set-panel";
import {SkillLaunchError,SkillLaunchSummary} from "../skills/skill-launch";

type Props={itemId:string;sourceId:string;preparationId:string;launch:SkillLaunchInput};
export function OwnProductImagesPage(props:Props){
 const context=useWorkbenchContext(),userId=context.user?.id,organizationId=context.effectiveOrganization?.id;
 const scope=useMemo(()=>userId&&organizationId?{userId,organizationId}:null,[userId,organizationId]);
 if(context.isLoading||context.isSwitching)return <ConsoleState kind="loading" title="正在确认当前企业"/>;
 if(!scope||context.error||context.blockingError||context.selectionRequired||!skillLaunchMatches(props.launch,scope,"product-image"))return <SkillLaunchError/>;
 if(!["workbench.supply.read","listingkit.image_agent.read"].every(permission=>context.permissions.includes(permission)))return <ConsoleState kind="unavailable" title="当前身份没有商品来源与图片查看权限"/>;
 return <ScopedImages key={JSON.stringify([scope,props])} {...props} launch={props.launch} scope={scope}/>;
}
function ScopedImages({scope,itemId,sourceId,preparationId,launch}:Omit<Props,"launch">&{launch:SkillLaunch;scope:SupplyScope}){
 const [source,setSource]=useState<SupplySourceDetail>(),[error,setError]=useState(false),[reload,setReload]=useState(0);
 useEffect(()=>{const controller=new AbortController();void readSupplySource(scope,sourceId,controller.signal).then(detail=>{if(detail.source.id!==sourceId||detail.source.preparationId!==preparationId||detail.source.collectionItemId!==itemId||detail.source.source.kind!=="own")throw Error("Source does not match selected product");if(!controller.signal.aborted)setSource(detail)}).catch(()=>{if(!controller.signal.aborted)setError(true)});return()=>controller.abort()},[scope,itemId,sourceId,preparationId,reload]);
 return <ConsolePage title="选择已有原图" description="使用已固定商品来源的真实原图，按原图片流程准备、确认并人工选择结果。" breadcrumbs={[{label:"官方模板",href:"/workbench/agents/skills/official"},{label:"已有商品",href:`/workbench/data/mine/own/${itemId}`},{label:"选择已有原图"}]} actions={<Button asChild variant="outline"><Link href={`/workbench/data/mine/own/${itemId}`}>返回商品资料</Link></Button>}>
  <div className="space-y-5"><SkillLaunchSummary launch={launch}/>{error?<ConsoleState kind="error" title="商品图片来源无法读取或与所选商品不符"><Button variant="outline" onClick={()=>{setError(false);setReload(value=>value+1)}}>重新读取原来源</Button></ConsoleState>:!source?<ConsoleState kind="loading" title="正在核实已固定商品来源"/>:<><div><h2 className="font-semibold">{source.product.title||"未命名商品"}</h2><p className="text-sm text-muted-foreground">已固定来源版本 {source.source.source.version}。无需重新采集或填写草稿；下方可直接选择已有原图。生成结果仍需人工选择并单独确认保存。</p></div><ProductImageSetPanel translationItemId={itemId} kind="supply" contextId={sourceId} initialSkill={launch.selection}/></>}</div>
 </ConsolePage>;
}
