/* eslint-disable @next/next/no-img-element */
"use client";
import Link from "next/link";
import {useQuery} from "@tanstack/react-query";
import {ConsolePage} from "@/components/workbench/console/console-page";
import {Card} from "@/components/ui/card";
import {Button} from "@/components/ui/button";
import {readApplicationOverview,type MarketScope} from "@/lib/api/supply-market";
import {MarketReadError} from "./shared";
import styles from "./application-overview.module.css";

const metrics=[{key:"eligibleProducts",label:"可申请商品"},{key:"reviewing",label:"审核中"},{key:"supplementRequired",label:"需补充资料"},{key:"approved",label:"已通过"}] as const;

export function SupplyApplicationOverview({scope}:{scope:MarketScope}){
 const query=useQuery({queryKey:["supply-market",scope.userId,scope.organizationId,"application-overview"],queryFn:({signal})=>readApplicationOverview(scope,signal),retry:false});
 const data=query.error?undefined:query.data;
 return <ConsolePage className={styles.overview} title="优选申请" description="将已完成智能优化、具备稳定供货能力的自有商品提交至硕米优选。" breadcrumbs={[{label:"供应市场",href:"/workbench/supply"},{label:"优选申请"}]} actions={<>
  <Button asChild variant="outline"><Link href="/workbench/supply/applications/progress" prefetch={false}>申请进度</Link></Button>
  <Button asChild className={styles.secondaryButton}><Link href="/workbench/supply/applications/records" prefetch={false}>申请记录</Link></Button>
  <Button asChild className={styles.primaryButton}><Link href="/workbench/supply/applications/new" prefetch={false}>选择商品发起申请</Link></Button>
 </>}>
  <section className={styles.metrics} aria-label="申请指标" aria-busy={query.isPending}>{metrics.map(metric=><Card className={styles.metric} key={metric.key}><h2>{metric.label}</h2><p>{data?data[metric.key]:query.isPending?"读取中":"不可用"}</p></Card>)}</section>
  <Card className={styles.preview}><div className={styles.previewHeader}><h2>可申请商品</h2><p>仅显示自有商品 · 已优化</p></div>
   <table className={styles.table}><thead><tr><th scope="col">商品</th><th scope="col">所属分组</th><th scope="col">商品来源</th><th scope="col">申请状态</th></tr></thead><tbody>
    {data?.products.map(product=><tr key={product.itemId}><td><div className={styles.product}><img src={product.thumbnailUrl} alt={product.title} width={40} height={40}/><span>{product.title}</span></div></td><td data-label="所属分组">{product.groupName}</td><td data-label="商品来源">自有商品</td><td data-label="申请状态"><span className={styles.eligible}>可申请</span></td></tr>)}
   </tbody></table>
   {query.isPending?<p className={styles.state} role="status">正在核对可申请商品与申请指标</p>:query.error?<MarketReadError error={query.error} retry={()=>void query.refetch()}/>:data?.products.length===0?<p className={styles.state} role="status">暂无可申请商品</p>:null}
  </Card>
  <p className={styles.note}>供货信息和资质将在提交时核验。</p>
 </ConsolePage>;
}
