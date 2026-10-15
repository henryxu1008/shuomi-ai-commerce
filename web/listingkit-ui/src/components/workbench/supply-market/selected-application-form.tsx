/* eslint-disable @next/next/no-img-element */
"use client";
import {useState} from "react";
import Link from "next/link";
import {useQuery} from "@tanstack/react-query";
import {ConsolePage} from "@/components/workbench/console/console-page";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import {Textarea} from "@/components/ui/textarea";
import {Table,TableHeader,TableBody,TableRow,TableHead,TableCell} from "@/components/ui/table";
import {listOwnProducts,readCollectionBatch} from "@/lib/api/product-collection";
import {MarketAPIError,readMarketChoice,type MarketScope} from "@/lib/api/supply-market";
import type {CollectionItem} from "@/lib/contracts/product-collection";
import {MarketReadError,CursorButtons,useMarketCommands} from "./shared";
import {Qualifications} from "./qualifications";
import styles from "./selected-application-form.module.css";

type Choice=Awaited<ReturnType<typeof readMarketChoice>>;
type Candidate={item:CollectionItem;choice:Choice;groupName:string};
type EligibleCandidate=Candidate&{choice:Extract<Choice,{ineligibleReason?:undefined}>};
const isEligible=(row:Candidate):row is EligibleCandidate=>row.choice.optimized&&!row.choice.ineligibleReason&&!!row.choice.product;
const selectionKey=(choice:Choice)=>{const s=choice.selection;return [s.itemId,s.expectedRevision,s.originalPublicationId,s.originalVersion,s.effectiveVersion,s.applyId].join("/")};

async function readCandidates(scope:MarketScope,keyword:string,after:string|undefined,signal:AbortSignal){
  const page=await listOwnProducts(scope,{keyword,after,limit:20},signal);
  const own=page.items.filter(item=>item.source.kind==="own"&&!item.archivedAt);
  const products:Candidate[]=[];
  // Keep this request-local projection bounded; the existing Choice owner is
  // authoritative for Apply lineage, and submit_selected checks it again.
  for(let start=0;start<own.length;start+=4){
    const candidates=await Promise.all(own.slice(start,start+4).map(async item=>{
      const choice=await readMarketChoice(scope,item.id,signal);
      if(choice.selection.itemId!==item.id||choice.selection.expectedRevision!==item.revision||choice.selection.originalPublicationId!==item.source.publicationId||choice.selection.originalVersion!==item.source.version)throw new MarketAPIError("REVISION_CONFLICT",409);
      return {item,choice};
    }));
    for(const candidate of candidates)products.push({...candidate,groupName:""});
  }
  const groups=new Map<string,string>();
  for(const product of products){
    if(!groups.has(product.item.batchId))groups.set(product.item.batchId,(await readCollectionBatch(scope,product.item.batchId,signal)).name);
    product.groupName=groups.get(product.item.batchId)!;
  }
  return {...page,products};
}

export function SelectedApplicationForm({scope}:{scope:MarketScope}){
  const [search,setSearch]=useState(""),[keyword,setKeyword]=useState(""),[after,setAfter]=useState<string>();
  const [selected,setSelected]=useState<EligibleCandidate>(),[step,setStep]=useState<"select"|"supply">("select");
  const [supply,setSupply]=useState({stock:0,capacity:"",minimumQuantity:1,leadDays:0,province:"",city:"",priceNote:"",afterSaleNote:""});
  const [fileIds,setFileIDs]=useState<string[]>([]),[uploadLocked,setUploadLocked]=useState(false),[disclosure,setDisclosure]=useState(false);
  const commands=useMarketCommands(scope);
  const query=useQuery({queryKey:["supply-market",scope.userId,scope.organizationId,"application-products",keyword,after],queryFn:({signal})=>readCandidates(scope,keyword,after,signal),retry:false});
  const data=query.error?undefined:query.data;
  const current=!query.isFetching&&selected?data?.products.filter(isEligible).find(row=>selectionKey(row.choice)===selectionKey(selected.choice)):undefined;
  const interactionLocked=commands.locked||!!commands.saved;
  const locked=interactionLocked||uploadLocked;
  const blocked=locked||!current||!disclosure||!fileIds.length;
  function changePage(cursor?:string){setAfter(cursor);setSelected(undefined)}
  const selecting=step==="select";
  return <ConsolePage className={styles.page} title={selecting?"选择申请商品":"填写供货信息"} description={selecting?"仅已优化且符合基本申请条件的自有商品可申请。每次申请选择一件商品。":"补充本次申请的库存、交付和合作信息，提交后由专员联系评估。"} breadcrumbs={[{label:"供应市场",href:"/workbench/supply"},{label:"优选申请",href:"/workbench/supply/applications"},{label:selecting?"选择商品":"填写供货信息"}]}>
    {commands.notice}
    {commands.saved?.recordId?<Card className={styles.summary} role="status">申请已保存。<Link href={"/workbench/supply/applications/"+commands.saved.recordId} prefetch={false}>查看申请进度</Link></Card>:null}
    {selecting?<>
      <Card className={styles.filters}>
        <div><span>商品来源</span><strong>自有商品</strong></div><div><span>申请条件</span><strong>已完成智能优化</strong></div>
        <form onSubmit={e=>{e.preventDefault();setKeyword(search.trim());setAfter(undefined);setSelected(undefined)}}><Input aria-label="商品关键词" placeholder="搜索自有商品" maxLength={80} value={search} disabled={locked} onChange={e=>setSearch(e.target.value)}/><Button type="submit" variant="outline" disabled={locked}>搜索</Button></form>
        <p>{query.isFetching?"正在核对申请资格":data?`本页 ${data.products.filter(isEligible).length} 件可申请`:"申请资格暂不可用"}</p>
      </Card>
      <Card className={styles.list}>
        <div className={styles.listActions}><p><strong>已选择 {current?1:0} 件</strong><span>选择商品后进入供货信息填写</span></p><Button className={styles.next} disabled={locked||!current} onClick={()=>setStep("supply")}>下一步：填写供货信息</Button></div>
        <Table className={styles.table} aria-label="自有商品申请资格"><TableHeader><TableRow><TableHead>商品（单选）</TableHead><TableHead>来源</TableHead><TableHead>所属分组</TableHead><TableHead>申请状态</TableHead></TableRow></TableHeader><TableBody>
          {!query.isFetching&&data?.products.map(row=>{
            const title=row.choice.product?.title||row.item.title||"自有商品",image=row.choice.product?.images[0],eligible=isEligible(row);
            return <TableRow key={row.item.id}><TableCell><label className={styles.product}><input type="radio" name="application-product" aria-label={"选择 "+title} checked={current?.item.id===row.item.id} disabled={locked||!eligible} onChange={()=>{if(isEligible(row)){setSelected(row);setDisclosure(false)}}}/>{image?<img src={image} alt={title} width={64} height={64}/>:<span className={styles.noImage}>暂无图片</span>}<span><strong>{title}</strong><small>{row.choice.product?.variants?.[0]?.sku||"自有商品"}</small></span></label></TableCell><TableCell><span className={styles.badge}>自有商品</span></TableCell><TableCell>{row.groupName}</TableCell><TableCell><strong>{eligible?"可申请":"不可申请"}</strong><small>{eligible?"已完成智能优化":row.choice.ineligibleReason==="incomplete_product"?"商品资料不符合申请要求":"未完成智能优化"}</small></TableCell></TableRow>;
          })}
        </TableBody></Table>
        {query.isFetching?<p className={styles.state} role="status">正在读取自有商品并核对优化结果</p>:query.error?<MarketReadError error={query.error} retry={()=>void query.refetch()}/>:data?.products.length===0?<div className={styles.state} role="status"><h2>{data.total===0&&!keyword?"暂无自有商品":"本页暂无可申请商品"}</h2><p>自有商品须完成智能优化，才能申请硕米优选。采集、市场选入及 SDS 商品不支持优选申请。</p><Button asChild variant="outline"><Link href={data.total===0&&!keyword?"/workbench/data/mine/own/new":"/workbench/data/mine?view=own"} prefetch={false}>{data.total===0&&!keyword?"添加自有商品":"查看自有商品"}</Link></Button></div>:null}
        {data?<div className={styles.pagination}><p>按自有商品分页，不可申请商品需先完善资料并完成智能优化。</p><CursorButtons after={after} next={data.nextCursor} total={data.total} onAfter={changePage}/></div>:null}
      </Card>
    </>:<>
      <Card className={styles.summary}><div><strong>已选择 1 件自有商品</strong><p>{selected?.choice.product.title}</p></div><Button type="button" variant="ghost" disabled={locked} onClick={()=>setStep("select")}>返回修改商品</Button></Card>
      {!current?<div className={styles.state} role="status"><p>{query.isFetching?"正在重新核对商品资格":"商品或申请资格已变化，请返回选择商品并重新确认。"}</p>{query.error?<MarketReadError error={query.error} retry={()=>void query.refetch()}/>:null}</div>:null}
      <form onSubmit={e=>{e.preventDefault();if(blocked||!current)return;void commands.execute({action:"submit_selected",selection:current.choice.selection,supply,fileIds,disclosure:true})}}>
        <fieldset disabled={interactionLocked||!current} className={styles.fields}>
          <Card className={styles.formCard}><h2>供货与交付</h2><p>填写本次申请商品的基础供货信息</p><div className={styles.inputGrid}>
            <label>可供库存（件）<Input type="number" required min={0} max={1e9} value={supply.stock} onChange={e=>setSupply({...supply,stock:Number(e.target.value)})}/></label>
            <label>最小起订量（件）<Input type="number" required min={1} max={1e9} value={supply.minimumQuantity} onChange={e=>setSupply({...supply,minimumQuantity:Number(e.target.value)})}/></label>
            <label>发货时效（天）<Input type="number" required min={0} max={3650} value={supply.leadDays} onChange={e=>setSupply({...supply,leadDays:Number(e.target.value)})}/></label>
            <label>生产能力<Input placeholder="无现货时请填写生产能力" required={supply.stock===0} maxLength={4000} value={supply.capacity} onChange={e=>setSupply({...supply,capacity:e.target.value})}/></label>
            <label>发货省份<Input required maxLength={100} value={supply.province} onChange={e=>setSupply({...supply,province:e.target.value})}/></label>
            <label>发货城市<Input required maxLength={100} value={supply.city} onChange={e=>setSupply({...supply,city:e.target.value})}/></label>
          </div><label>供货价说明<Textarea placeholder="例如：含税 / 不含运费，可由专员进一步确认" maxLength={4000} value={supply.priceNote} onChange={e=>setSupply({...supply,priceNote:e.target.value})}/></label></Card>
          <Card className={styles.formCard}><h2>合作与资质</h2><p>只保留平台评估所需的必要信息</p><label>售后政策<Textarea placeholder="简要说明退换货及质量保障" maxLength={4000} value={supply.afterSaleNote} onChange={e=>setSupply({...supply,afterSaleNote:e.target.value})}/></label>
            <div className={styles.upload}><Qualifications scope={scope} ids={fileIds} onIDs={setFileIDs} onLocked={setUploadLocked}/></div>
            <div className={styles.service}><h3>服务说明</h3><p>如涉及资料整理、技术对接或上架服务，专员将另行确认内容与费用。</p></div>
          </Card>
        </fieldset>
        <label className={styles.disclosure}><input type="checkbox" disabled={locked||!current} checked={disclosure} onChange={e=>setDisclosure(e.target.checked)}/>我确认这是本人/企业自有商品，并允许平台评估；明确发布后，列明的商品字段及供货声明可供其他获授权企业查看。</label>
        <Card className={styles.footer}><p>提交后不可直接修改；需补充资料时可在申请记录中处理。</p><div><Button type="button" variant="outline" disabled={locked} onClick={()=>setStep("select")}>返回选择商品</Button><Button type="submit" className={styles.submit} disabled={blocked}>提交申请</Button></div></Card>
      </form>
    </>}
  </ConsolePage>;
}
