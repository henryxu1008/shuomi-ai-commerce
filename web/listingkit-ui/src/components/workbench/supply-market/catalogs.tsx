"use client";
import Link from "next/link";
import {ConsolePage} from "@/components/workbench/console/console-page";
import {Card} from "@/components/ui/card";
import {MarketBoundary,MarketLinks} from "./shared";
import {CatalogOverview} from "./catalog-overview";
type CatalogView="selection";
const catalogBreadcrumbs=[{label:"供应市场",href:"/workbench/supply"},{label:"货盘集成",href:"/workbench/supply/catalogs"}];
export function SupplyCatalogsPage({acquisitionAvailable=false,sdsAvailable=false,view}:{acquisitionAvailable?:boolean;sdsAvailable?:boolean;view?:CatalogView}){return <MarketBoundary>{scope=>view==="selection"?<CatalogSelection acquisitionAvailable={acquisitionAvailable} sdsAvailable={sdsAvailable}/>:<CatalogOverview scope={scope}/>}</MarketBoundary>}
function CatalogSelection({acquisitionAvailable,sdsAvailable}:{acquisitionAvailable:boolean;sdsAvailable:boolean}){return <ConsolePage title="货盘选品" description="查看当前实例的货盘能力，进入对应的选品与采集流程。" breadcrumbs={[...catalogBreadcrumbs,{label:"货盘选品"}]}><MarketLinks/><div className="grid gap-4 sm:grid-cols-2"><Card className="space-y-3 p-6"><h2>1688 选品与采集</h2><p>从链接或插件采集商品，查看保存结果，再选入商品刊登。网页采集沿现有资源计量规则。</p>{acquisitionAvailable?<Link href="/workbench/supply/acquisition" prefetch={false}>进入 1688 选品与采集</Link>:<p>当前实例未配置采集能力</p>}</Card><Card className="space-y-3 p-6"><h2>SDS POD 定制货盘</h2><p>选择模板与款式，使用本人已批准的图案设计，核实效果图和已保存成品后转入待上传。</p>{sdsAvailable?<Link href="/workbench/supply/catalogs/sds" prefetch={false}>查看模板与定制成品</Link>:<p>当前实例未配置 SDS 定制能力</p>}</Card></div></ConsolePage>}
