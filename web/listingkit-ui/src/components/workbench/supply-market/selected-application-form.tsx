/* eslint-disable @next/next/no-img-element */
"use client";
import {useState} from "react";
import Link from "next/link";
import {useQuery} from "@tanstack/react-query";
import {ConsolePage} from "@/components/workbench/console/console-page";
import {Button} from "@/components/ui/button";
import {Card} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import {Table,TableHeader,TableBody,TableRow,TableHead,TableCell} from "@/components/ui/table";
import {listOwnProducts,readCollectionBatch} from "@/lib/api/product-collection";
import {MarketAPIError,readMarketChoice,type MarketScope} from "@/lib/api/supply-market";
import {marketCommandSchema} from "@/lib/contracts/supply-market";
import type {CollectionItem} from "@/lib/contracts/product-collection";
import {MarketReadError,CursorButtons,marketError,useMarketCommands} from "./shared";
import {emptySupply,finalCommand,selectionKey,selectedBatchLimit,type CommonSupply,type SelectedDraft,type SelectedCommand} from "./selected-batch-intent";
import {SelectedApplicationEditor} from "./selected-application-editor";
import {useSelectedBatch} from "./use-selected-batch";
import styles from "./selected-application-form.module.css";

type Choice=Awaited<ReturnType<typeof readMarketChoice>>;
type Candidate={item:CollectionItem;choice:Choice;groupName:string};
type EligibleCandidate=Candidate&{choice:Extract<Choice,{ineligibleReason?:undefined}>};
const isEligible=(row:Candidate):row is EligibleCandidate=>row.choice.optimized&&!row.choice.ineligibleReason&&!!row.choice.product;

async function readCandidates(scope:MarketScope,keyword:string,after:string|undefined,signal:AbortSignal){
  const page=await listOwnProducts(scope,{keyword,after,limit:20},signal);
  const own=page.items.filter(item=>item.source.kind==="own"&&!item.archivedAt),products:Candidate[]=[];
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
  const [drafts,setDrafts]=useState<SelectedDraft[]>([]),[step,setStep]=useState<"select"|"supply"|"review">("select");
  const [common,setCommon]=useState<CommonSupply>({province:"",city:"",afterSaleNote:""}),[commonFiles,setCommonFiles]=useState<string[]>([]);
  const [uploadLocked,setUploadLocked]=useState(false),[checking,setChecking]=useState(false),[disclosure,setDisclosure]=useState(false),[issues,setIssues]=useState<string[]>([]);
  const [reviewed,setReviewed]=useState<{title:string;command:SelectedCommand}[]>([]);
  const single=useMarketCommands(scope),batch=useSelectedBatch(scope);
  const query=useQuery({queryKey:["supply-market",scope.userId,scope.organizationId,"application-products",keyword,after],queryFn:({signal})=>readCandidates(scope,keyword,after,signal),retry:false});
  const data=query.error?undefined:query.data;
  const validSelection=!query.isFetching&&!query.error&&drafts.length>0&&drafts.every(d=>{const row=data?.products.find(row=>row.item.id===d.selection.itemId);return !row||isEligible(row)&&selectionKey(row.choice.selection)===selectionKey(d.selection)});
  const interactionLocked=single.locked||!!single.saved||batch.locked||checking,locked=interactionLocked||uploadLocked;
  const resultRows=batch.results.length?batch.results:batch.journal?.items.map(item=>({title:item.title,key:item.intent.key,state:"waiting" as const}));
  const showingResults=!!batch.journal||!!resultRows?.length;
  function changed(){setDisclosure(false);setReviewed([]);setIssues([])}
  function updateDraft(id:string,patch:Partial<SelectedDraft>){changed();setDrafts(previous=>previous.map(d=>d.selection.itemId===id?{...d,...patch}:d))}
  function toggle(row:EligibleCandidate){changed();setDrafts(previous=>previous.some(d=>d.selection.itemId===row.item.id)?previous.filter(d=>d.selection.itemId!==row.item.id):previous.length>=selectedBatchLimit?previous:[...previous,{title:row.choice.product.title,selection:row.choice.selection,supply:emptySupply(),inheritLocation:true,inheritAfterSale:true,useCommonFiles:true,fileIds:[]}])}
  async function review(){
    if(locked||!validSelection)return;setChecking(true);setIssues([]);setDisclosure(false);const errors:string[]=[],commands:{title:string;command:SelectedCommand}[]=[];
    try{for(const draft of drafts){
      const choice=await readMarketChoice(scope,draft.selection.itemId);
      if(!choice.optimized||choice.ineligibleReason||!choice.product||selectionKey(choice.selection)!==selectionKey(draft.selection)){errors.push(draft.title+"：商品或申请资格已变化，请返回选择商品并重新确认。");continue}
      const command=marketCommandSchema.safeParse(finalCommand(draft,common,commonFiles));
      if(!command.success||command.data.action!=="submit_selected"){errors.push(draft.title+"：请检查库存/产能、起订量、交期、发货地及合并后的 1–6 件资质附件。");continue}
      commands.push({title:draft.title,command:command.data});
    }
    if(!errors.length){setReviewed(commands);setStep("review")}
    }catch(e){errors.push(marketError(e))}finally{setIssues(errors);setChecking(false)}
  }
  const title=showingResults?"本批申请结果":step==="select"?"选择申请商品":step==="supply"?"填写供货信息":"确认本批申请";
  return <ConsolePage className={styles.page} title={title} description="选择多件已优化自有商品，复用公共信息、逐件核对供货差异；每件申请独立审核，批准后仍需明确发布。" breadcrumbs={[{label:"供应市场",href:"/workbench/supply"},{label:"优选申请",href:"/workbench/supply/applications"},{label:title}]}>
    {!batch.journal?single.notice:null}
    {single.saved?<Card className={styles.summary} role="status"><p>原操作已确认。</p><Link href={single.saved.recordId?"/workbench/supply/applications/"+single.saved.recordId:"/workbench/data/mine"} prefetch={false}>查看原操作结果</Link></Card>:null}
    {batch.error?<Card className="p-4" role="alert">{marketError(batch.error)}</Card>:null}
    {batch.pendingError?<Card className="p-4" role="alert">无法读取原申请，已暂停新提交。请保留浏览器数据。</Card>:null}
    {showingResults?<>
      <Card className={styles.summary}><p>每件申请独立审核。待确认时先核实；明确继续会使用原请求键和原资料，已确认成功项不会重复发送。</p></Card>
      <Card className={styles.list}><Table aria-label="逐商品申请结果"><TableHeader><TableRow><TableHead>商品</TableHead><TableHead>结果</TableHead><TableHead>说明</TableHead></TableRow></TableHeader><TableBody>{resultRows?.map(row=><TableRow key={row.key}><TableCell>{row.title}</TableCell><TableCell>{{submitted:"已提交",failed:"明确失败",unknown:"结果待确认",waiting:"尚未处理 / 待核实",processing:"处理中"}[row.state]}</TableCell><TableCell>{"receipt" in row&&row.receipt?.recordId?<Link href={"/workbench/supply/applications/"+row.receipt.recordId} prefetch={false}>查看独立申请</Link>:"error" in row&&row.error?marketError(row.error):""}</TableCell></TableRow>)}</TableBody></Table></Card>
      <div className={styles.resultActions}>{batch.journal?<><Button disabled={batch.busy} onClick={()=>void batch.verify()}>核实本批结果</Button><Button variant="outline" disabled={batch.busy} onClick={()=>void batch.resume()}>使用原请求继续</Button></>:batch.results.some(row=>row.state==="failed")?<Button onClick={()=>{setDrafts(batch.results.filter(row=>row.state==="failed").flatMap(row=>{const command=row.intent.command;return command.action==="submit_selected"?[{title:row.title,selection:command.selection,supply:{...command.supply},fileIds:[...command.fileIds],inheritLocation:false,inheritAfterSale:false,useCommonFiles:false}]:[]}));batch.reset();changed();setStep("supply")}}>修改明确失败商品</Button>:null}<Button asChild variant="outline"><Link href="/workbench/supply/applications" prefetch={false}>申请列表</Link></Button></div>
    </>:step==="select"?<>
      <Card className={styles.filters}><div><span>商品来源</span><strong>自有商品</strong></div><div><span>申请条件</span><strong>已完成智能优化</strong></div>
        <form onSubmit={e=>{e.preventDefault();setKeyword(search.trim());setAfter(undefined);changed()}}><Input aria-label="商品关键词" maxLength={80} placeholder="搜索自有商品" disabled={locked} value={search} onChange={e=>setSearch(e.target.value)}/><Button type="submit" variant="outline" disabled={locked}>搜索</Button></form><p>{query.isFetching?"正在核对申请资格":data?`本页 ${data.products.filter(isEligible).length} 件可申请`:"申请资格暂不可用"}</p>
      </Card>
      <Card className={styles.list}><div className={styles.listActions}><p><strong>已选择 {drafts.length} 件</strong><span>跨搜索和分页保留，最多 {selectedBatchLimit} 件</span></p><Button className={styles.next} disabled={locked||!validSelection} onClick={()=>setStep("supply")}>下一步：填写供货信息</Button></div>
        <Table className={styles.table} aria-label="自有商品申请资格"><TableHeader><TableRow><TableHead>商品（多选）</TableHead><TableHead>来源</TableHead><TableHead>所属分组</TableHead><TableHead>申请状态</TableHead></TableRow></TableHeader><TableBody>{!query.isFetching&&data?.products.map(row=>{
          const title=row.choice.product?.title||row.item.title||"自有商品",image=row.choice.product?.images[0],eligible=isEligible(row),selected=drafts.find(d=>d.selection.itemId===row.item.id);
          return <TableRow key={row.item.id}><TableCell><label className={styles.product}><input type="checkbox" aria-label={"选择 "+title} checked={!!selected} disabled={locked||!eligible||!selected&&drafts.length>=selectedBatchLimit} onChange={()=>{if(isEligible(row))toggle(row)}}/>{image?<img src={image} alt={title} width={64} height={64}/>:<span className={styles.noImage}>暂无图片</span>}<span><strong>{title}</strong><small>{row.choice.product?.variants?.[0]?.sku||"自有商品"}</small></span></label></TableCell><TableCell><span className={styles.badge}>自有商品</span></TableCell><TableCell>{row.groupName}</TableCell><TableCell><strong>{eligible?"可申请":"不可申请"}</strong><small>{eligible?"已完成智能优化":row.choice.ineligibleReason==="incomplete_product"?"商品资料不符合申请要求":"未完成智能优化"}</small></TableCell></TableRow>;
        })}</TableBody></Table>
        {query.isFetching?<p className={styles.state} role="status">正在读取自有商品并核对优化结果</p>:query.error?<MarketReadError error={query.error} retry={()=>void query.refetch()}/>:data?.products.length===0?<div className={styles.state} role="status"><h2>{data.total===0&&!keyword?"暂无自有商品":"本页暂无可申请商品"}</h2><p>自有商品须完成智能优化，才能申请硕米优选。采集、市场选入及 SDS 商品不支持优选申请。</p><Button asChild variant="outline"><Link href={data.total===0&&!keyword?"/workbench/data/mine/own/new":"/workbench/data/mine?view=own"} prefetch={false}>{data.total===0&&!keyword?"添加自有商品":"查看自有商品"}</Link></Button></div>:null}
        {data?<div className={styles.pagination}><p>资格在汇总前逐件重新核对。</p><CursorButtons after={after} next={data.nextCursor} total={data.total} onAfter={cursor=>{setAfter(cursor);changed()}}/></div>:null}
      </Card>
      {drafts.length?<Card className={styles.selectedItems} aria-label="已选商品">{drafts.map(d=><div key={d.selection.itemId}><span>{d.title}</span><Button type="button" variant="ghost" aria-label={"移除 "+d.title} disabled={locked} onClick={()=>{setDrafts(previous=>previous.filter(row=>row.selection.itemId!==d.selection.itemId));changed()}}>移除</Button></div>)}</Card>:null}
      {!validSelection&&drafts.length>0&&!query.isFetching?<p role="status">商品或申请资格已变化，请返回选择商品并重新确认。</p>:null}
    </>:<>
      <Card className={styles.summary}><strong>已选择 {drafts.length} 件自有商品</strong><Button variant="ghost" disabled={locked} onClick={()=>{changed();setStep("select")}}>返回修改商品</Button></Card>
      {!validSelection?<div className={styles.state} role="status"><p>{query.isFetching?"正在重新核对商品资格":"商品或申请资格已变化，请返回选择商品并重新确认。"}</p>{query.error?<MarketReadError error={query.error} retry={()=>void query.refetch()}/>:null}</div>:null}
      {issues.length?<Card className="p-4" role="alert">{issues.map((issue,index)=><p key={index}>{issue}</p>)}</Card>:null}
      {step==="supply"?<SelectedApplicationEditor scope={scope} drafts={drafts} onDraft={updateDraft} common={common} onCommon={value=>{changed();setCommon(value)}} commonFiles={commonFiles} onCommonFiles={ids=>{changed();setCommonFiles(ids)}} locked={interactionLocked||!validSelection} uploadLocked={uploadLocked} onUploadLocked={setUploadLocked} onReview={()=>void review()} onBack={()=>{changed();setStep("select")}}/>:<>
        <Card className={styles.list}><Table aria-label="本批最终供货信息"><TableHeader><TableRow><TableHead>商品</TableHead><TableHead>供货差异</TableHead><TableHead>交付与资质</TableHead></TableRow></TableHeader><TableBody>{reviewed.map(({title,command},index)=><TableRow key={command.selection.itemId}><TableCell>{title}<small>逐件独立审核</small></TableCell><TableCell>库存 {command.supply.stock} 件；产能 {command.supply.capacity||"未填写"}；起订 {command.supply.minimumQuantity} 件；交期 {command.supply.leadDays} 天<br/>价格：{command.supply.priceNote||"待专员确认"}</TableCell><TableCell>{command.supply.province} {command.supply.city}（{drafts[index]?.inheritLocation?"继承公共":"单独填写"}）<br/>售后：{command.supply.afterSaleNote||"未填写"}（{drafts[index]?.inheritAfterSale?"继承公共":"单独填写"}）<br/>资质 {command.fileIds.length} 件（{drafts[index]?.useCommonFiles?"引用通用及商品专属":"仅商品专属"}）</TableCell></TableRow>)}</TableBody></Table></Card>
        <label className={styles.disclosure}><input type="checkbox" checked={disclosure} disabled={locked||!validSelection} onChange={e=>setDisclosure(e.target.checked)}/>我确认以上 {reviewed.length} 件均为本人/企业自有商品，逐件供货及资质资料准确，并允许平台独立评估；仅明确发布后列明字段才可供其他获授权企业查看。</label>
        <Card className={styles.footer}><p>逐件提交；明确失败独立显示，未知结果会暂停后续。提交后不可直接修改。</p><div><Button variant="outline" disabled={locked} onClick={()=>{changed();setStep("supply")}}>返回修改供货信息</Button><Button disabled={locked||!validSelection||!disclosure||!reviewed.length} onClick={()=>void batch.start(reviewed)}>提交 {reviewed.length} 件申请</Button></div></Card>
      </>}
    </>}
  </ConsolePage>;
}
