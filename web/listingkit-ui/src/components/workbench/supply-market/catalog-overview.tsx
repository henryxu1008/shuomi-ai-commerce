"use client";

import {useState} from "react";
import Image from "next/image";
import Link from "next/link";
import {ConsolePage,ConsoleState} from "@/components/workbench/console/console-page";
import {Card} from "@/components/ui/card";
import {Button} from "@/components/ui/button";
import {Select} from "@/components/ui/select";
import type {MarketScope} from "@/lib/api/supply-market";
import {CatalogConnectionForm} from "./catalog-connection-form";
import {catalogDirectory,catalogRegions,catalogWarehouseMatches,type CatalogEntry,type CatalogRegion,type WarehouseKind} from "./catalog-directory";
import styles from "./catalog-overview.module.css";

export function CatalogOverview({scope}:{scope:MarketScope}){
 const [applying,setApplying]=useState(false);
 const [method,setMethod]=useState<"api"|"plugin">("api");
 const [warehouse,setWarehouse]=useState<WarehouseKind|"">(""),[region,setRegion]=useState<CatalogRegion|"">("");
 const regions=Object.entries(catalogRegions).filter(([code])=>catalogDirectory.some(p=>p.warehouses.some(w=>catalogWarehouseMatches(w,warehouse,code as CatalogRegion))));
 const entries=catalogDirectory.filter(p=>p.warehouses.some(w=>catalogWarehouseMatches(w,warehouse,region)));
 function changeWarehouse(next:WarehouseKind|""){
  setWarehouse(next);
  if(region&&!catalogDirectory.some(p=>p.warehouses.some(w=>catalogWarehouseMatches(w,next,region))))setRegion("");
 }
 return <ConsolePage title="货盘集成" description="按对接方式、仓库类型和国家地区寻找合适的货盘。"
  breadcrumbs={[{label:"供应市场",href:"/workbench/supply"},{label:"货盘集成"}]}
  actions={<><Button onClick={()=>setApplying(true)}>申请对接货盘</Button><Button variant="ghost" asChild><Link href="/workbench/supply/catalogs/apply?view=progress" prefetch={false}>查看对接进度与记录</Link></Button></>}>
  <div className={styles.methods} role="group" aria-label="货盘对接方式">
   <Button variant={method==="api"?"default":"ghost"} aria-pressed={method==="api"} onClick={()=>setMethod("api")}>API 对接</Button>
   <Button variant={method==="plugin"?"default":"ghost"} aria-pressed={method==="plugin"} onClick={()=>setMethod("plugin")}>浏览器插件对接</Button>
  </div>
  <p className={styles.intro}>{method==="api"?"通过平台接口对接货盘。以下四家尚未接入硕米，可先访问官网了解货源。":"通过硕米浏览器插件采集货盘商品。以下四家暂未支持，实际接通后再开放采集。"}</p>
  <div className={styles.filters} role="group" aria-label="货盘筛选">
   <label>仓库类型<Select aria-label="仓库类型" value={warehouse} onChange={e=>changeWarehouse(e.target.value as WarehouseKind|"")}><option value="">全部仓库</option><option value="domestic">国内仓</option><option value="overseas">海外仓</option></Select></label>
   <label>仓库区域<Select aria-label="仓库区域" value={region} onChange={e=>setRegion(e.target.value as CatalogRegion|"")}><option value="">全部国家 / 地区</option>{regions.map(([code,label])=><option key={code} value={code}>{label}</option>)}</Select></label>
   <Button variant="ghost" onClick={()=>{setWarehouse("");setRegion("")}}>清除筛选</Button>
   <span className={styles.count} role="status">{entries.length} 家货盘</span>
  </div>
  {entries.length?<div className={styles.directory}>{entries.map(entry=><CatalogCard key={entry.id} entry={entry} method={method}/>)}</div>:<ConsoleState kind="empty" title="暂无匹配的货盘">调整仓库或区域筛选后再试。</ConsoleState>}
  <p className={styles.footnote}>国内仓指中国仓，区域按仓库所在国家或地区划分。当前展示已核实的公开覆盖信息，其他区域待补充；具体商品库存与发货条件以平台为准。资料核对：2026-11-16。</p>
  {applying?<CatalogConnectionForm scope={scope} onClose={()=>setApplying(false)}/>:null}
 </ConsolePage>;
}

function CatalogCard({entry,method}:{entry:CatalogEntry;method:"api"|"plugin"}){
 return <Card className={styles.card} role="article" aria-label={entry.name}>
  <div className={styles.logo}><Image unoptimized src={entry.logo} width={220} height={52} alt={`${entry.name} Logo`}/></div>
  <div><h2>{entry.name}</h2><p className={styles.domain}>{entry.domain}</p></div>
  <p className={styles.description}>{entry.description}</p>
  <dl className={styles.coverage}>{(["domestic","overseas"] as const).map(kind=>{
   const names=entry.warehouses.filter(w=>w.kind===kind).map(w=>catalogRegions[w.region]);
   return names.length?<div key={kind}><dt>{kind==="domestic"?"国内仓":"海外仓"}</dt><dd>{names.join("、")}</dd></div>:null;
  })}</dl>
  <div className={styles.connection}><span className={styles.status}>{method==="api"?"未接入":"插件暂未支持"}</span><p>{method==="api"?entry.apiNote:"硕米采集插件尚未支持该平台。"}</p></div>
  <div className={styles.actions}><Button variant="outline" asChild><a href={entry.website} target="_blank" rel="noopener noreferrer">访问官网</a></Button><a href={entry.coverageSource} target="_blank" rel="noopener noreferrer">仓库资料</a></div>
 </Card>;
}
