"use client";

import Link from "next/link";
import {useQuery} from "@tanstack/react-query";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {ConsolePage} from "@/components/workbench/console/console-page";
import {Card} from "@/components/ui/card";
import {readApplicationOverview} from "@/lib/api/supply-market";
import styles from "./supply-overview.module.css";

const steps=[
 {title:"发现商品资源",detail:"硕米自营、优选与外部货盘"},
 {title:"加入我的供应链",detail:"挑选商品或上传自有商品"},
 {title:"按需调用AI",detail:"可选调用Chat、智能体与工具"},
 {title:"刊登到关联店铺",detail:"生成资料并完成店铺发布"},
];

export function SupplyOverview({marketAvailable,preparationAvailable}:{marketAvailable:boolean;preparationAvailable:boolean}){
 const context=useWorkbenchContext();
 const scope=context.user&&context.effectiveOrganization?{userId:context.user.id,organizationId:context.effectiveOrganization.id}:null;
 const ready=marketAvailable&&!!scope&&!context.isLoading&&!context.isSwitching&&!context.selectionRequired&&!context.error&&!context.blockingError;
 const summary=useQuery({queryKey:["supply-market",scope?.userId,scope?.organizationId,"application-overview"],queryFn:({signal})=>readApplicationOverview(scope!,signal),enabled:ready,retry:false});
 const facts=ready&&!summary.isError?summary.data:undefined;
 const features=[
  {title:"硕米自营",tone:"teal",label:"平台自营货源",description:"平台自有工厂与自营商品资源，开放后可直接挑选并加入我的供应链。",href:"/workbench/supply/official",available:marketAvailable},
  {title:"硕米优选",tone:"blue",label:"平台优选货源",description:"平台审核的优质供应商与商品资源，开放后可筛选商品并加入我的供应链。",href:"/workbench/supply/selected",available:marketAvailable},
  {title:"货盘集成",tone:"green",label:"官方接入货盘选品",description:"浏览硕米官方已接入的第三方货盘商品，按平台、站点、类目、成本和库存筛选，并加入我的供应链。",href:"/workbench/supply/catalogs",available:marketAvailable},
  {title:"我的供应链",tone:"purple",label:"核心使用入口",description:"集中管理从硕米自营、硕米优选和第三方货盘挑选的商品，以及用户自主上传的商品，为后续AI处理与店铺刊登提供统一商品来源。",href:"/workbench/supply/mine",available:preparationAvailable},
  {title:"优选申请",tone:"pink",label:"可提交申请",description:"选择我的供应链中的自有商品，提交进入硕米优选的申请，并查看资料校验、平台审核与最终结果。",href:"/workbench/supply/applications",available:marketAvailable},
 ];
 return <ConsolePage className={styles.overview} title="让每一种货源，都能进入自己的供应链" description="连接平台自营、优选货源与第三方货盘，统一沉淀到我的供应链，再由硕米Chat、智能体和工具完成后续业务。" breadcrumbs={[{label:"供应市场"}]}>
  <Card className={styles.flow}><ol aria-label="从货源到店铺">{steps.map(step=><li key={step.title}><h2>{step.title}</h2><p>{step.detail}</p></li>)}</ol></Card>
  <div className={styles.features}>{features.map((feature,index)=><Card key={feature.title} className={`${styles.feature} ${styles[feature.tone]}`}>
   <p className={styles.kicker}><span>{String(index+1).padStart(2,"0")}</span>{feature.available?feature.label:"暂未开放"}</p>
   <h2>{feature.title}</h2><p className={styles.copy}>{feature.description}</p>
   {feature.available?<Link href={feature.href} prefetch={false} aria-label={`进入功能：${feature.title}`}>进入功能 <span aria-hidden="true">→</span></Link>:<span className={styles.unavailable}>暂未开放</span>}
  </Card>)}</div>
  <Card className={styles.usage}>
   <header><h2>我的使用情况</h2><p>只展示真实业务数据，数据未接入时统一显示 —</p></header>
   <div className={styles.summaries}>
    <Card className={`${styles.summary} ${styles.purple}`}><h3>我的供应链</h3><Metrics items={[{label:"商品总数"},{label:"待完善"},{label:"可刊登"}]}/></Card>
    <Card className={`${styles.summary} ${styles.green}`}><h3>货盘选品</h3><Metrics items={[{label:"官方接入货盘"},{label:"可选商品"}]}/></Card>
    <Card className={`${styles.summary} ${styles.pink}`}><h3>优选申请</h3><Metrics items={[{label:"待提交"},{label:"审核中",value:facts?.reviewing},{label:"已通过",value:facts?.approved}]}/></Card>
   </div>
  </Card>
 </ConsolePage>;
}

function Metrics({items}:{items:{label:string;value?:number}[]}){
 return <dl className={styles.metrics}>{items.map(item=><div key={item.label}><dt>{item.label}</dt><dd aria-label={item.label+"数量"}>{item.value??"—"}</dd></div>)}</dl>;
}
