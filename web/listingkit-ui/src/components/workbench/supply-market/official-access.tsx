"use client";

import type {ReactNode} from "react";
import Link from "next/link";
import {useQuery} from "@tanstack/react-query";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {Button} from "@/components/ui/button";
import {ConsoleState} from "../console/console-page";
import {listMarketRecords,MarketAPIError} from "@/lib/api/supply-market";
import {MarketReadError} from "./shared";

// Same verified global read and cache as the existing platform navigation.
// Enterprise roles never confer platform publishing authority.
function usePublishingAccess(){
 const c=useWorkbenchContext(),userId=c.user?.id;
 const platform=useQuery({queryKey:["supply-market",userId,"","records",true,"selected",false,undefined],queryFn:({signal})=>listMarketRecords({userId:userId!,organizationId:""},{kind:"selected",ended:false},true,signal),enabled:!!userId&&!c.isLoading&&!c.isSwitching&&!c.error&&!c.blockingError,retry:false});
 const canApply=c.permissions?.includes("workbench.supply-market.apply");
 return {platform,canApply,canManage:canApply&&c.permissions?.includes("workbench.collection.manage")&&c.permissions?.includes("workbench.collection.read")};
}

export function OfficialMarketActions({showManage=true}:{showManage?:boolean}){
 const {platform,canManage}=usePublishingAccess();
 if(!platform.isSuccess||platform.isError)return null;
 return <>{showManage?<Button variant="outline" asChild><Link href="/workbench/supply/operator?kind=official" prefetch={false}>管理自营商品</Link></Button>:null}{canManage?<Button asChild><Link href="/workbench/supply/official/new" prefetch={false}>添加自营商品</Link></Button>:null}</>;
}

export function OfficialPublishingAccess({children}:{children:ReactNode}){
 const {platform,canManage,canApply}=usePublishingAccess();
 if(platform.isPending)return <ConsoleState kind="loading" title="正在确认自营发布权限"/>;
 if(platform.error){
  if(!(platform.error instanceof MarketAPIError)||platform.error.code!=="PERMISSION_DENIED")return <MarketReadError error={platform.error} retry={()=>void platform.refetch()}/>;
  return <ConsoleState kind="unavailable" title="当前账号不能发布自营商品">请使用有自营发布权限的专员账号。<ReturnToOfficial/></ConsoleState>;
 }
 if(!canApply)return <ConsoleState kind="unavailable" title="当前企业下没有自营供货权限">请联系企业管理员开通供货申请权限后继续。<ReturnToOfficial/></ConsoleState>;
 if(!canManage)return <ConsoleState kind="unavailable" title="当前企业下没有商品管理权限">发布自营商品还需要当前企业的自有商品查看与管理权限。<ReturnToOfficial/></ConsoleState>;
 return children;
}

function ReturnToOfficial(){return <Button variant="outline" asChild><Link href="/workbench/supply/official" prefetch={false}>返回硕米自营</Link></Button>}

