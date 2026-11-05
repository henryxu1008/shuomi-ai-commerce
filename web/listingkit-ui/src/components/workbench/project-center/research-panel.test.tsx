import {beforeEach,afterEach,it,expect,vi} from "vitest";
import {render,screen,fireEvent,waitFor,cleanup} from "@testing-library/react";
import {readFileSync} from "node:fs";
import {ResearchPanel} from "./research-panel";
import {ResearchError} from "@/lib/api/market-research";
import {researchPage} from "@/lib/contracts/market-research";
import {parseCollectionResponse,type CollectionItem} from "@/lib/contracts/product-collection";
const mock=vi.hoisted(()=>({read:vi.fn(),pickerItem:undefined as CollectionItem|undefined,context:{permissions:["workbench.collection.read","workbench.project.manage"],registerOrganizationSwitchGuard:()=>()=>{}}}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>mock.context}));
vi.mock("@/lib/api/market-research",async original=>({...await original<object>(),researchRequest:mock.read}));
vi.mock("../pod/collection-picker",()=>({CollectionPicker:({onSelect,isEligible}:{onSelect:(i:unknown)=>void;isEligible:(i:unknown)=>boolean})=>{
 const item=mock.pickerItem??{id:"22222222-2222-4222-8222-222222222222",revision:1,source:{kind:"amazon_data",publicationId:id,version:"1"}};
 return isEligible(item)?<button onClick={()=>onSelect(item)}>{mock.pickerItem?.title??"核实插件来源"}</button>:<p>不支持来源已排除</p>;
}}));
vi.mock("./research-analysis-flow",async original=>({...await original<object>(),ResearchAnalysisFlow:()=> <h3>AI分析建议</h3>}));
vi.mock("./research-comparison",()=>({ResearchComparison:()=>null}));
const id="11111111-1111-4111-8111-111111111111";
const candidate={id,revision:2,state:"TO_RESEARCH",selectedLinkId:id,evidenceDigest:"a".repeat(64),observation:{title:"真实保存商品",site:"us",asin:"B000123456",sourceUrl:"https://www.amazon.com/dp/B000123456",capturedAt:"2026-11-02T00:00:00Z",parserVersion:"amazon-v1",missing:["rating"],ref:{checksum:"original",selection:{itemId:id,revision:1,publicationId:id,version:"1"}},item:{id,revision:1,source:{kind:"amazon_data",version:"1",publicationId:id}}},evidence:[],history:[]};
beforeEach(()=>{sessionStorage.clear();mock.pickerItem=undefined;mock.context.permissions=["workbench.collection.read","workbench.project.manage"];mock.read.mockReset().mockResolvedValue({items:[candidate],incomplete:false});});
afterEach(()=>{cleanup();vi.unstubAllGlobals();});

it.each([0,1])("preserves %i saved cards when viewing the original after one confirmed SHEIN submit", async (publishedCount) => {
 const receipt={approvalId:id,state:"CONFIRMED",requestId:"original-1",credits:5,reservedCredits:0};
 const fetcher=vi.fn().mockResolvedValueOnce(Response.json({approvalId:id,state:"NOT_STARTED"}))
  .mockResolvedValueOnce(Response.json({...receipt,publishedCount}))
  .mockResolvedValueOnce(Response.json(receipt));
 vi.stubGlobal("fetch",fetcher);
 mock.context.permissions.push("workbench.data-market.use");
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false} view="research" showSheinTrial/>);
 fireEvent.click(await screen.findByRole("button",{name:"核对试取范围"}));
 expect(screen.getByText(/最多 20 张商品卡/)).toBeInTheDocument();
 fireEvent.click(screen.getByRole("button",{name:"取消"}));
 expect(screen.getByRole("button",{name:"核对试取范围"})).toHaveFocus();
 expect(fetcher).toHaveBeenCalledTimes(1);
 fireEvent.click(screen.getByRole("button",{name:"核对试取范围"}));
 fireEvent.click(screen.getByRole("button",{name:"确认提交一次"}));
 expect(await screen.findByText(`实际用量 5 credits。已保存 ${publishedCount} 张商品卡。`)).toBeInTheDocument();
 expect(fetcher).toHaveBeenCalledTimes(2);
 expect(fetcher.mock.calls[1][0]).toMatch(/\/shein-us-trial\/submit$/);
 expect(new Headers(fetcher.mock.calls[1][1].headers).get("X-Expected-SHEIN-Trial-Approval-ID")).toBe(id);
 fireEvent.click(screen.getByRole("button",{name:"查看原单"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"查看原单"})).toBeEnabled());
 expect(screen.getByText(`实际用量 5 credits。已保存 ${publishedCount} 张商品卡。`)).toBeInTheDocument();
 expect(screen.queryByRole("button",{name:"继续保存原单资料"})).not.toBeInTheDocument();
 expect(fetcher.mock.calls.map(([, init])=>init.method)).toEqual(["GET","POST","GET"]);
 expect(screen.getByRole("button",{name:"添加已有商品资料"})).toBeEnabled();
 expect(screen.getByRole("link",{name:"进入智能选品"})).toBeVisible();
});

it("keeps a lost submit on the original and reconciles only after checking its status", async () => {
 const fetcher=vi.fn().mockResolvedValueOnce(Response.json({approvalId:id,state:"NOT_STARTED"}))
  .mockRejectedValueOnce(new Error("connection closed"))
  .mockResolvedValueOnce(Response.json({approvalId:id,state:"UNKNOWN",requestId:"original-1",credits:null,reservedCredits:25}))
  .mockResolvedValueOnce(Response.json({approvalId:id,state:"CONFIRMED",requestId:"original-1",credits:0,reservedCredits:0}));
 vi.stubGlobal("fetch",fetcher);
 mock.context.permissions.push("workbench.data-market.use");
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false} view="research" showSheinTrial/>);
 fireEvent.click(await screen.findByRole("button",{name:"核对试取范围"}));
 fireEvent.click(screen.getByRole("button",{name:"确认提交一次"}));
 expect(await screen.findByText(/还不能确认这次提交/)).toBeInTheDocument();
 expect(screen.queryByRole("button",{name:"核对试取范围"})).not.toBeInTheDocument();
 expect(fetcher).toHaveBeenCalledTimes(2);
 fireEvent.click(screen.getByRole("button",{name:"查看原单"}));
 fireEvent.click(await screen.findByRole("button",{name:"核对原账"}));
 expect(await screen.findByText(/实际用量 0 credits/)).toBeInTheDocument();
 expect(fetcher.mock.calls.map(([path, init])=>[path.split("/").at(-1),init.method])).toEqual([
  ["shein-us-trial","GET"],["submit","POST"],["shein-us-trial","GET"],["reconcile","POST"],
 ]);
});

it("shows actual over-limit usage with only read access to the original", async () => {
 const fetcher=vi.fn().mockResolvedValue(Response.json({approvalId:id,state:"OVER_LIMIT",requestId:"original-1",credits:26,reservedCredits:0}));
 vi.stubGlobal("fetch",fetcher);
 mock.context.permissions.push("workbench.data-market.use");
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false} view="research" showSheinTrial/>);
 expect(await screen.findByText(/实际用量 26 credits/)).toBeInTheDocument();
 expect(screen.getByRole("button",{name:"查看原单"})).toBeVisible();
 expect(screen.queryByRole("button",{name:/核对试取范围|继续保存原单资料|核对原账/})).not.toBeInTheDocument();
 expect(fetcher.mock.calls.map(([, init])=>init.method)).toEqual(["GET"]);
});

it("removes the entry and original receipt when a fresh read denies access", async () => {
 const fetcher=vi.fn().mockResolvedValueOnce(Response.json({approvalId:id,state:"UNKNOWN",requestId:"private-receipt",credits:null,reservedCredits:25}))
  .mockResolvedValueOnce(Response.json({code:"FORBIDDEN"},{status:403}));
 vi.stubGlobal("fetch",fetcher);
 mock.context.permissions.push("workbench.data-market.use");
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false} view="research" showSheinTrial/>);
 expect(await screen.findByText(/private-receipt/)).toBeInTheDocument();
 fireEvent.click(screen.getByRole("button",{name:"查看原单"}));
 await waitFor(()=>expect(screen.queryByRole("region",{name:"SHEIN 美国站首试"})).not.toBeInTheDocument());
 expect(screen.queryByText(/private-receipt/)).not.toBeInTheDocument();
 expect(fetcher.mock.calls.map(([, init])=>init.method)).toEqual(["GET","GET"]);
});
it("places optional analysis before the independent human decision in selection review",async()=>{
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false} view="selection"/>);
 const decision=await screen.findByRole("button",{name:"开始评估"});
 expect(screen.getByRole("heading",{name:"AI分析建议"}).compareDocumentPosition(decision)&Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
});
it("shows original missing facts without dispatching a command on open",async()=>{render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false}/>);expect(await screen.findByText("真实保存商品")).toBeInTheDocument();expect(screen.getByText(/rating/)).toBeInTheDocument();expect(mock.read.mock.calls.every(c=>!c[4]?.intent)).toBe(true);});
it("links and switches the original SHEIN evidence, then masks the withdrawn card and reason", async () => {
 const fixture=JSON.parse(readFileSync("../../docs/architecture/evidence/shein-us-data-v1-p1-p2.json", "utf8"));
 function original(name:"P1"|"P2") {
  const {selection,publication,job,observationUtf8}=fixture.publications[name];
  const card=JSON.parse(observationUtf8);
  const item={id:selection.itemId,batchId:id,title:card.title,source:publication.source,revision:selection.revision,createdAt:candidate.observation.capturedAt};
  return {selection,card,item,observation:{...candidate.observation,site:"",asin:"",title:card.title,sourceUrl:card.sourceUrl,capturedAt:"",parserVersion:card.parserVersion,missing:card.missing,
   shein:{market:card.market,nativeGoodsId:card.nativeGoodsId,price:card.price},query:{...job.query,asinCount:0,limit:0,fields:[]},
   batchId:id,item,ref:{selection,source:publication.source,jobId:job.jobId,queryRef:job.queryRef,checksum:publication.rawReference.checksum,observationVersion:"1",observationEncodingVersion:"shein-us-observation-json-v1"}}};
 }
 const p1=original("P1"),p2=original("P2"),otherLink="33333333-3333-4333-8333-333333333333";
 const {item,selection,card}=p1;
 const amazonItem={...item,id,title:"Amazon 样例",source:{...item.source,kind:"amazon_data"}};
 expect(parseCollectionResponse("items",{items:[amazonItem,item],total:2})).toMatchObject({items:[{source:{kind:"amazon_data"}},{source:{kind:"shein_us_data"}}]});
 expect(parseCollectionResponse("batches",{items:[{id,name:"Amazon 数据",kind:"amazon_data",revision:1,count:1,createdAt:item.createdAt,supplyTransferSupported:true},{id:selection.itemId,name:"SHEIN 数据",kind:"shein_us_data",revision:1,count:1,createdAt:item.createdAt,supplyTransferSupported:false}],total:2})).not.toBeNull();
 const reason="仅绑定 P1 的人工理由";
 const history={state:"APPROVED_FOR_TEST",reason,evidenceDigest:candidate.evidenceDigest,revision:1,at:item.createdAt};
 const evidence=[{id,observation:p1.observation},{id:otherLink,observation:p2.observation}];
 const first={...candidate,state:"APPROVED_FOR_TEST",observation:p1.observation,reason,history:[history],evidence};
 const switched={...first,state:"TO_RESEARCH",reason:undefined,revision:3,selectedLinkId:otherLink,observation:p2.observation};
 const capturedAt="2026-11-02T01:02:03,123Z";
 const timed=researchPage.parse({items:[{...first,observation:{...p1.observation,capturedAt}}],incomplete:false});
 expect(timed.items[0].observation?.capturedAt).toBe(capturedAt);
 let response=researchPage.parse({items:[first],incomplete:false});
 mock.pickerItem=item;
 mock.read.mockImplementation(async(_scope,_project,_schema,_signal,options)=>{
  if(options?.intent?.body.action==="select") response=researchPage.parse({items:[switched],incomplete:false});
  return response;
 });
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false}/>);
 expect(await screen.findByText("SHEIN 美国站商品卡片")).toBeInTheDocument();
 expect(screen.getByText(/USD 38\.88/)).toBeInTheDocument();
 expect(screen.getByRole("link",{name:/查看来源页面/})).toHaveAttribute("href",card.sourceUrl);
 expect(screen.getByRole("checkbox")).toBeDisabled();
 expect(screen.queryByRole("button",{name:"开始评估"})).not.toBeInTheDocument();
 fireEvent.click(screen.getByRole("button",{name:"添加已有商品资料"}));
 fireEvent.click(screen.getByRole("button",{name:card.title}));
 await waitFor(()=>expect(mock.read.mock.calls.find(call=>call[4]?.intent)?.[4].intent.body).toEqual({action:"link",selections:[selection]}));
 fireEvent.click(await screen.findByText(/已收集的资料记录/));
 fireEvent.click(screen.getByRole("button",{name:"使用这份资料"}));
 expect(await screen.findByText(/USD 36\.50/)).toBeInTheDocument();
 expect(mock.read.mock.calls.map(call=>call[4]?.intent?.body)).toContainEqual({action:"select",candidateId:id,expectedRevision:2,linkId:otherLink});
 fireEvent.click(screen.getByText(/过往选择记录/));
 expect(screen.getByText(reason)).toBeVisible();
 response=researchPage.parse({items:[{...switched,evidence:[{id},evidence[1]],history:[{...history,reason:undefined,evidenceDigest:undefined}]}],incomplete:false});
 fireEvent.click(screen.getByRole("button",{name:"刷新商品资料"}));
 await screen.findByText(/USD 36\.50/);
 expect(screen.getByText("这份资料暂时无法查看")).toBeInTheDocument();
 expect(screen.queryByText(/USD 38\.88/)).not.toBeInTheDocument();
 expect(screen.queryByText(reason)).not.toBeInTheDocument();
});
it("focuses the requested function while keeping selected candidates and avoiding commands between steps",async()=>{
 const scope={userId:"u",organizationId:"o"};
 const view=render(<ResearchPanel scope={scope} projectId={id} archived={false} view="selection" workflowStep="decisions"/>);
 await screen.findByRole("button",{name:"开始评估"});
 await waitFor(()=>expect(document.getElementById("research-decisions")).toHaveFocus());
 fireEvent.click(screen.getByRole("checkbox"));expect(screen.getByRole("checkbox")).toBeChecked();
 view.rerender(<ResearchPanel scope={scope} projectId={id} archived={false} view="selection" workflowStep="transfer"/>);
 await waitFor(()=>expect(document.getElementById("research-transfer")).toHaveFocus());
 expect(screen.getByRole("checkbox")).toBeChecked();expect(mock.read.mock.calls.every(c=>!c[4]?.intent)).toBe(true);
});
it("keeps the original unknown write and recovers it by readonly key lookup",async()=>{mock.read.mockImplementation(async(_s,_p,_z,_a,options)=>{if(options?.intent)throw new ResearchError("OUTCOME_UNKNOWN");if(options?.lookup)return {key:options.lookup,candidateIds:[id],replayed:true};return {items:[candidate],incomplete:false};});render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false}/>);fireEvent.click(await screen.findByRole("button",{name:"开始评估"}));await screen.findByRole("button",{name:"查询操作结果"});const original=mock.read.mock.calls.find(c=>c[4]?.intent)?.[4].intent;fireEvent.click(screen.getByRole("button",{name:"查询操作结果"}));await waitFor(()=>expect(mock.read.mock.calls.some(c=>c[4]?.lookup===original.key)).toBe(true));expect(mock.read.mock.calls.filter(c=>c[4]?.intent)).toHaveLength(1);});
it("masks unavailable evidence and keeps archived projects readonly",async()=>{mock.read.mockResolvedValue({items:[{...candidate,observation:undefined,evidenceDigest:undefined}],incomplete:false});render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived/>);expect(await screen.findByText("这份资料暂时无法查看")).toBeInTheDocument();expect(screen.queryByText("真实保存商品")).not.toBeInTheDocument();expect(screen.queryByRole("button",{name:"开始评估"})).not.toBeInTheDocument();});

it("clears a definite unsupported-origin rejection and lets research continue",async()=>{
 mock.read.mockImplementation(async(_s,_p,_z,_a,options)=>{if(options?.intent)throw new ResearchError("SOURCE_NOT_SUPPORTED");return {items:[candidate],incomplete:false};});
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false}/>);
 await screen.findByText("真实保存商品");fireEvent.click(screen.getByRole("button",{name:"添加已有商品资料"}));fireEvent.click(screen.getByRole("button",{name:"核实插件来源"}));
 await screen.findByText(/这份资料暂不能用于调研/);
 await waitFor(()=>expect(screen.queryByRole("button",{name:"查询操作结果"})).not.toBeInTheDocument());
 expect(screen.getByRole("button",{name:"开始评估"})).toBeEnabled();expect(screen.getByText("不支持来源已排除")).toBeInTheDocument();
});

it("carries exact selected observations into the other center and rejects stale handoff",async()=>{
 const other="33333333-3333-4333-8333-333333333333",second={...candidate,id:other,selectedLinkId:other,observation:{...candidate.observation,title:"第二项观察",ref:{...candidate.observation.ref,selection:{...candidate.observation.ref.selection,itemId:other}}}};
 mock.read.mockResolvedValue({items:[candidate,second],incomplete:false});
 const view=render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false} view="research"/>);
 await screen.findByText("真实保存商品");for(const checkbox of screen.getAllByRole("checkbox"))fireEvent.click(checkbox);
 const href=screen.getByRole("link",{name:"进入智能选品"}).getAttribute("href")!,sample=new URL(href,"https://app.example").searchParams.get("sample")!;
 expect(JSON.parse(sample).map((c:{candidateId:string})=>c.candidateId)).toEqual([id,other]);expect(JSON.parse(sample)[0].evidenceDigest).toBe(candidate.evidenceDigest);
 view.unmount();mock.read.mockResolvedValue({items:[{...candidate,revision:3},second],incomplete:false});
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false} view="selection" sample={sample}/>);
 await screen.findByText("所选商品资料已更新或暂时无法查看，请重新选择。");expect(screen.getAllByRole("checkbox").every(v=>!(v as HTMLInputElement).checked)).toBe(true);
});

it("hands only the selected approved observations to the original per-batch confirmation",async()=>{
 const secondId="33333333-3333-4333-8333-333333333333",pendingId="44444444-4444-4444-8444-444444444444";
 const batchA="55555555-5555-4555-8555-555555555555",batchB="66666666-6666-4666-8666-666666666666";
 const first={...candidate,state:"APPROVED_FOR_TEST",observation:{...candidate.observation,batchId:batchA}};
 const second={...candidate,id:secondId,state:"APPROVED_FOR_TEST",observation:{...candidate.observation,title:"另一批已批准样本",batchId:batchB,ref:{...candidate.observation.ref,selection:{...candidate.observation.ref.selection,itemId:secondId}}}};
 const pending={...candidate,id:pendingId,observation:{...candidate.observation,title:"尚未决定的样本",batchId:batchA,ref:{...candidate.observation.ref,selection:{...candidate.observation.ref.selection,itemId:pendingId}}}};
 mock.read.mockResolvedValue({items:[first,second,pending],incomplete:false});
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false} view="selection" workflowStep="transfer"/>);
 await screen.findByRole("heading",{name:"尚未决定的样本"});for(const checkbox of screen.getAllByRole("checkbox"))fireEvent.click(checkbox);
 const links=screen.getAllByRole("link",{name:/批次 \d+ · 核对 1 件商品/});expect(links).toHaveLength(2);
 const transfers=links.map(link=>new URL(link.getAttribute("href")!,"https://app.example"));
 expect(transfers.map(url=>url.pathname)).toEqual(["/workbench/data/mine","/workbench/data/mine"]);
 expect(transfers.map(url=>url.searchParams.get("batchId"))).toEqual([batchA,batchB]);
 expect(transfers.map(url=>JSON.parse(url.searchParams.get("transfer")!))).toEqual([[first.observation.ref.selection],[second.observation.ref.selection]]);
 expect(mock.read.mock.calls.every(call=>!call[4]?.intent)).toBe(true);
});

it("rechecks the exact Amazon observation for a cost preview and keeps the original decision independent", async () => {
 const selected={...candidate,state:"PENDING_CONFIRMATION"};
 mock.read.mockResolvedValue({items:[selected],incomplete:false});
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false} view="selection"/>);
 fireEvent.click(await screen.findByRole("button",{name:"填写经营假设"}));
 fireEvent.click(screen.getByRole("button",{name:"查看参考试算"}));
 expect(await screen.findByText("资料仍有缺项")).toBeVisible();
 expect(mock.read.mock.calls.filter(call=>!call[4]?.intent)).toHaveLength(2);
 expect(mock.read.mock.calls.filter(call=>call[4]?.intent)).toHaveLength(0);
 fireEvent.change(screen.getByRole("textbox",{name:"选择理由"}),{target:{value:"继续核对货源条件"}});
 fireEvent.click(screen.getByRole("button",{name:"批准测试"}));
 await waitFor(()=>expect(mock.read.mock.calls.find(call=>call[4]?.intent)?.[4].intent.body).toEqual({
   action:"decide",candidateId:id,expectedRevision:2,evidenceDigest:"a".repeat(64),state:"APPROVED_FOR_TEST",reason:"继续核对货源条件",
 }));
});

it.each(["account","organization","project","permission","archived"])("drops open preview and draft after %s changes", async change => {
 const scope={userId:"u",organizationId:"o"};
 const result=render(<ResearchPanel scope={scope} projectId={id} archived={false} view="selection"/>);
 fireEvent.click(await screen.findByRole("button",{name:"填写经营假设"}));
 fireEvent.change(screen.getByLabelText("假设售价（USD/件）"),{target:{value:"30"}});
 fireEvent.click(screen.getByRole("button",{name:"查看参考试算"}));
 expect(await screen.findByText("资料仍有缺项")).toBeVisible();
 if(change==="permission")mock.context.permissions=["workbench.collection.read"];
 const changedScope=change==="account"?{...scope,userId:"another-user"}:change==="organization"?{...scope,organizationId:"another-org"}:scope;
 const changedProject=change==="project"?"22222222-2222-4222-8222-222222222222":id;
 result.rerender(<ResearchPanel scope={changedScope} projectId={changedProject} archived={change==="archived"} view="selection"/>);
 await waitFor(()=>expect(screen.queryByText("资料仍有缺项")).not.toBeInTheDocument());
 expect(screen.queryByRole("form",{name:"经营假设和参考试算"})).not.toBeInTheDocument();
 if(change==="permission"||change==="archived")expect(screen.queryByRole("button",{name:"填写经营假设"})).not.toBeInTheDocument();
});

it.each(["new version","withdrawn","SHEIN"])("does not reuse preview evidence when the fresh page returns %s", async change => {
 const next=change==="new version"
  ? {...candidate,observation:{...candidate.observation,ref:{...candidate.observation.ref,selection:{...candidate.observation.ref.selection,version:"2"}}}}
  : change==="withdrawn"?{...candidate,observation:undefined,evidenceDigest:undefined}
  : {...candidate,observation:{...candidate.observation,query:{mode:"category",categoryRef:"1727"},shein:{market:"US",nativeGoodsId:"456",price:null}}};
 mock.read.mockResolvedValueOnce({items:[candidate],incomplete:false}).mockResolvedValue({items:[next],incomplete:false});
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false} view="selection"/>);
 fireEvent.click(await screen.findByRole("button",{name:"填写经营假设"}));
 fireEvent.click(screen.getByRole("button",{name:"查看参考试算"}));
 expect(await screen.findByText(/原商品资料已变化或暂不可读/)).toBeVisible();
 expect(screen.queryByText("资料仍有缺项")).not.toBeInTheDocument();
 // After the original page refresh, only a currently readable Amazon US version can reopen.
 fireEvent.click(screen.getByRole("button",{name:"刷新商品资料"}));
 await waitFor(()=>expect(screen.queryByRole("form",{name:"经营假设和参考试算"})).not.toBeInTheDocument());
 if(change!=="new version")expect(screen.queryByRole("button",{name:"填写经营假设"})).not.toBeInTheDocument();
 expect(mock.read.mock.calls.every(call=>!call[4]?.intent)).toBe(true);
});


it("keeps one candidate's assumptions out of another candidate", async () => {
 const otherId="22222222-2222-4222-8222-222222222222";
 const other={...candidate,id:otherId,selectedLinkId:otherId,evidenceDigest:"b".repeat(64),observation:{
  ...candidate.observation,title:"另一件保存商品",
 }};
 mock.read.mockResolvedValue({items:[candidate,other],incomplete:false});
 render(<ResearchPanel scope={{userId:"u",organizationId:"o"}} projectId={id} archived={false} view="selection"/>);
 const entries=await screen.findAllByRole("button",{name:"填写经营假设"});
 expect(entries).toHaveLength(2);
 fireEvent.click(entries[0]);
 fireEvent.change(screen.getByLabelText("假设售价（USD/件）"),{target:{value:"30"}});
 fireEvent.click(screen.getByRole("button",{name:"填写经营假设"}));
 expect(screen.getAllByRole("form",{name:"经营假设和参考试算"})).toHaveLength(1);
 expect(screen.getByLabelText("假设售价（USD/件）")).toHaveValue("");
});
