import {act,cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {SheinImportPage} from "./shein-import-page";
import Route from "@/app/workbench/listing/[stage]/[sourceId]/shein/page";
import {sheinImportFixture,importIds as id} from "@/test/shein-import-fixture";
import {SupplyAPIError,type SupplyIntent} from "@/lib/api/supply-chain";
const f=vi.hoisted(()=>({context:{user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.supply.read","workbench.supply.manage","workbench.store.read"],supplyIntentReady:true,pendingSupplyIntent:null as SupplyIntent|null,setPendingSupplyIntent:vi.fn<(v:SupplyIntent|null)=>boolean>(),registerOrganizationSwitchGuard:()=>()=>{}},source:vi.fn(),head:vi.fn(),revision:vi.fn(),stores:vi.fn(),target:vi.fn(),record:vi.fn(),replace:vi.fn(),rules:vi.fn(),preview:vi.fn(),save:vi.fn(),verify:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>f.context}));
vi.mock("next/server",()=>({connection:async()=>{}}));vi.mock("next/navigation",()=>({useRouter:()=>({replace:f.replace}),notFound:()=>{throw new Error("NOT_FOUND")}}));
vi.mock("@/lib/api/workbench-stores",()=>({listWorkbenchStores:f.stores}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),readSupplySource:f.source,readSupplyCompletionDraft:f.head,readCompletionRevision:f.revision,readSupplyTarget:f.target,readSupplyRecord:f.record,supplyRules:f.rules,previewSheinImport:f.preview,supplyCommand:f.save,readSupplyCommand:f.verify}));
const props={sourceId:id.source,preparationId:id.batch,itemId:id.batch,revision:3,recordId:id.base};
beforeEach(()=>{vi.clearAllMocks();const data=sheinImportFixture();f.context.effectiveOrganization.id="org";f.context.permissions=["workbench.supply.read","workbench.supply.manage","workbench.store.read"];f.context.pendingSupplyIntent=null;f.context.setPendingSupplyIntent.mockImplementation(v=>{f.context.pendingSupplyIntent=v;return true});f.source.mockResolvedValue(data.detail);f.head.mockResolvedValue(data.candidate);f.revision.mockResolvedValue(data.candidate);f.stores.mockResolvedValue({items:[{id:id.store,name:"SHEIN 店铺",platform:"shein"}],pagination:{total:1}});f.target.mockResolvedValue(data.base);f.record.mockResolvedValue(data.base);f.rules.mockResolvedValue(data.rules);f.preview.mockResolvedValue(data.preview);f.save.mockResolvedValue(data.saved);f.verify.mockResolvedValue(data.saved)});
afterEach(()=>{cleanup();vi.unstubAllEnvs()});
const preview=()=>screen.getByRole("button",{name:"预览带入结果"}),confirm=()=>screen.getByRole("button",{name:"确认保存为新平台草稿版本"});
it("keeps ordinary single-product intake out of the historical split flow",async()=>{
 render(<SheinImportPage {...props}/>);await screen.findByRole("button",{name:"映射颜色与销售规格"});expect(screen.queryByRole("button",{name:"按颜色拆分"})).not.toBeInTheDocument();
});
it("uses an independent strict route with all return context",async()=>{
 vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED","true");vi.stubEnv("LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED","true");vi.stubEnv("LISTINGKIT_SUPPLY_CHAIN_ENABLED","true");
 const route=(query:Record<string,string|string[]|undefined>)=>Route({params:Promise.resolve({stage:"missing",sourceId:id.source}),searchParams:Promise.resolve(query)});
 expect((await route({preparation:id.batch,item:id.batch,revision:"3",target:id.base})).props).toEqual(props);
 for(const query of [{revision:"0"},{target:[id.base,id.base]},{returnTo:"https://bad.test"}])await expect(route(query)).rejects.toThrow("NOT_FOUND");
 render(<SheinImportPage {...props}/>);await screen.findByText(/候选 R3/);expect(screen.queryByRole("dialog")).not.toBeInTheDocument();expect(screen.getByRole("link",{name:"返回草稿"})).toHaveAttribute("href",`/workbench/listing/missing/${id.source}?preparation=${id.batch}&item=${id.batch}`);expect(screen.getByRole("link",{name:"返回商品"})).toHaveAttribute("href",`/workbench/data/mine/own/${id.batch}`);expect(confirm()).toBeDisabled();
});
it("requires preview, invalidates it on selection change, and saves the exact preview input",async()=>{
 const candidate=sheinImportFixture().candidate;
 f.revision.mockResolvedValue({...candidate,content:{...candidate.content,internalNote:"WORKBENCH_ONLY_SHEIN_NOTE"}});
 render(<SheinImportPage {...props}/>);await screen.findByText(/候选 R3/);await waitFor(()=>expect(preview()).toBeEnabled());fireEvent.click(preview());await waitFor(()=>expect(confirm()).toBeEnabled());
 fireEvent.click(screen.getByRole("checkbox",{name:"带入描述"}));expect(confirm()).toBeDisabled();fireEvent.change(screen.getByRole("combobox",{name:"来源 SKU 1 的目标"}),{target:{value:"0:0"}});fireEvent.click(screen.getByRole("checkbox",{name:"带入库存 SKU 1"}));fireEvent.change(screen.getByRole("combobox",{name:"库存仓库 SKU 1"}),{target:{value:"a"}});
 fireEvent.click(preview());await waitFor(()=>expect(f.preview).toHaveBeenLastCalledWith(expect.anything(),expect.objectContaining({completionDraft:{recordId:id.candidate,revision:3},selection:expect.objectContaining({description:true,skus:[{sourceIndex:0,skcIndex:0,skuIndex:0,price:false,stock:true,warehouseId:"a"}]})}),expect.any(AbortSignal)));
 await waitFor(()=>expect(confirm()).toBeEnabled());fireEvent.click(confirm());await waitFor(()=>expect(f.save).toHaveBeenCalledWith(expect.objectContaining({route:"save-target",command:sheinImportFixture().preview.input}),expect.any(AbortSignal)));expect(await screen.findByText(/本次已保存 Target R2/)).toBeVisible();
 const platformInput=JSON.stringify([f.preview.mock.lastCall?.[1],f.save.mock.lastCall?.[0].command]);
 expect(platformInput).not.toMatch(/internalNote|WORKBENCH_ONLY_SHEIN_NOTE/);
});
it("locks a conflict until reread and retains the original command during UNKNOWN recovery",async()=>{
 f.save.mockRejectedValueOnce(new SupplyAPIError("REVISION_CONFLICT",409));render(<SheinImportPage {...props}/>);await waitFor(()=>expect(preview()).toBeEnabled());fireEvent.click(preview());await waitFor(()=>expect(confirm()).toBeEnabled());fireEvent.click(confirm());await screen.findByText(/目标版本或规则已变化/);expect(preview()).toBeDisabled();fireEvent.click(screen.getByRole("button",{name:"重新读取目标版本"}));await waitFor(()=>expect(preview()).toBeEnabled());fireEvent.click(preview());await waitFor(()=>expect(confirm()).toBeEnabled());f.save.mockRejectedValueOnce(new SupplyAPIError("OUTCOME_UNKNOWN",503));fireEvent.click(confirm());await screen.findByText("结果待核实");const original=f.context.pendingSupplyIntent;expect(confirm()).toBeDisabled();fireEvent.click(screen.getByRole("button",{name:"核实原操作"}));await waitFor(()=>expect(f.verify).toHaveBeenCalledWith(expect.objectContaining({key:original!.key,command:original!.command}),expect.any(AbortSignal)));await screen.findByText(/本次已保存 Target R2/);
});
it("rejects wrong context and ignores late reads across organization changes",async()=>{
 const wrong=render(<SheinImportPage {...props} itemId={id.store}/>);await screen.findByText("带入资料读取失败");expect(screen.queryByRole("button",{name:"预览带入结果"})).not.toBeInTheDocument();wrong.unmount();
 let resolve!:(v:unknown)=>void;f.source.mockReturnValueOnce(new Promise(r=>{resolve=r}));const view=render(<SheinImportPage {...props}/>);f.context.effectiveOrganization.id="different";f.source.mockReturnValue(new Promise(()=>{}));view.rerender(<SheinImportPage {...props}/>);await act(async()=>resolve(sheinImportFixture().detail));expect(screen.queryByText(/候选 R3/)).not.toBeInTheDocument();
});
it("uses an explicit attribute option and disables incompatible currency",async()=>{
 const data=sheinImportFixture();data.candidate.content.structured.variants[0].price.currency="CNY";f.revision.mockResolvedValue(data.candidate);render(<SheinImportPage {...props}/>);await waitFor(()=>expect(preview()).toBeEnabled());
 fireEvent.change(screen.getByRole("combobox",{name:"候选属性 1 的目标"}),{target:{value:"9"}});expect(preview()).toBeDisabled();fireEvent.change(screen.getByRole("combobox",{name:"候选属性 1 的平台值"}),{target:{value:"11"}});
 fireEvent.change(screen.getByRole("combobox",{name:"来源 SKU 1 的目标"}),{target:{value:"0:0"}});expect(screen.getByRole("checkbox",{name:"带入价格 SKU 1"})).toBeDisabled();expect(preview()).toBeDisabled();fireEvent.click(screen.getByRole("checkbox",{name:"带入库存 SKU 1"}));fireEvent.change(screen.getByRole("combobox",{name:"库存仓库 SKU 1"}),{target:{value:"a"}});fireEvent.click(preview());await waitFor(()=>expect(f.preview).toHaveBeenCalledWith(expect.anything(),expect.objectContaining({selection:expect.objectContaining({attributes:[{sourceIndex:0,attributeId:9,optionId:11}]})}),expect.any(AbortSignal)));
});
it("keeps readonly users and a store without saved Target out of confirmation",async()=>{
 f.context.permissions=["workbench.supply.read","workbench.store.read"];const view=render(<SheinImportPage {...props}/>);await screen.findByText(/当前身份没有供应链管理权限/);expect(preview()).toBeDisabled();expect(confirm()).toBeDisabled();expect(f.rules).not.toHaveBeenCalled();view.unmount();
 f.context.permissions.push("workbench.supply.manage");f.target.mockRejectedValue(new SupplyAPIError("NOT_FOUND",404));render(<SheinImportPage {...props} recordId={undefined}/>);await screen.findByText(/候选 R3/);fireEvent.change(screen.getByRole("combobox",{name:"SHEIN 店铺"}),{target:{value:id.store}});await screen.findByText(/此店铺尚无已保存的平台草稿/);expect(confirm()).toBeDisabled();expect(f.save).not.toHaveBeenCalled();
 expect(screen.getByRole("link",{name:"前往平台草稿"})).toHaveAttribute("href",`/workbench/listing/waiting?preparation=${id.batch}&store=${id.store}`);
});
it("clears the selected store and its preview when paging the store list",async()=>{
 f.stores.mockResolvedValueOnce({items:[{id:id.store,name:"第一页店铺",platform:"shein"}],pagination:{total:101}}).mockResolvedValueOnce({items:[{id:id.saved,name:"第二页店铺",platform:"shein"}],pagination:{total:101}});
 render(<SheinImportPage {...props} recordId={undefined}/>);await screen.findByText(/候选 R3/);fireEvent.change(screen.getByRole("combobox",{name:"SHEIN 店铺"}),{target:{value:id.store}});await waitFor(()=>expect(preview()).toBeEnabled());fireEvent.click(preview());await waitFor(()=>expect(confirm()).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"下一页店铺"}));await screen.findByRole("option",{name:"第二页店铺"});expect(screen.getByRole("combobox",{name:"SHEIN 店铺"})).toHaveValue("");expect(screen.queryByRole("button",{name:"确认保存为新平台草稿版本"})).not.toBeInTheDocument();expect(screen.queryByText("本次覆盖预览")).not.toBeInTheDocument();expect(f.save).not.toHaveBeenCalled();
});

it.each([
 {baseReady:false,savedReady:true,before:"missing",after:"waiting"},
 {baseReady:true,savedReady:false,before:"waiting",after:"missing"},
])("returns to the visible platform stage before and after saving $after",async({baseReady,savedReady,before,after})=>{
 const data=sheinImportFixture();
 data.base.result.ready_for_upload=baseReady;
 data.saved.record.result={...data.saved.record.result,ready_for_upload:savedReady};
 f.record.mockResolvedValue(data.base);
 f.save.mockResolvedValue(data.saved);
 render(<SheinImportPage {...props}/>);
 const originalHref=new URL((await screen.findByRole("link",{name:"返回平台草稿"})).getAttribute("href")!,"https://test.invalid");
 expect(originalHref.pathname).toBe(`/workbench/listing/${before}`);
 expect(Object.fromEntries(originalHref.searchParams)).toEqual({preparation:id.batch,store:id.store,...before==="missing"?{mode:"platform"}:{}});
 await waitFor(()=>expect(preview()).toBeEnabled());
 fireEvent.click(preview());
 await waitFor(()=>expect(confirm()).toBeEnabled());
 fireEvent.click(confirm());
 await screen.findByText(/本次已保存 Target R2/);
 const savedHref=new URL(screen.getByRole("link",{name:"返回平台草稿"}).getAttribute("href")!,"https://test.invalid");
 expect(savedHref.pathname).toBe(`/workbench/listing/${after}`);
 expect(Object.fromEntries(savedHref.searchParams)).toEqual({preparation:id.batch,store:id.store,...after==="missing"?{mode:"platform"}:{}});
 expect(f.save).toHaveBeenCalledTimes(1);
});

it("keeps another store's original recovery receipt out of the current store navigation",async()=>{
 const data=sheinImportFixture();
 f.save.mockRejectedValueOnce(new SupplyAPIError("OUTCOME_UNKNOWN",503));
 const original=render(<SheinImportPage {...props}/>);
 await waitFor(()=>expect(preview()).toBeEnabled());
 fireEvent.click(preview());
 await waitFor(()=>expect(confirm()).toBeEnabled());
 fireEvent.click(confirm());
 await screen.findByText("结果待核实");
 const pending=f.context.pendingSupplyIntent;
 original.unmount();

 const secondStore=id.candidate;
 const secondBase={...data.base,id:id.saved,input:{...data.base.input,storeId:secondStore},merchant:{...data.base.merchant,store_id:secondStore}};
 f.record.mockResolvedValue(secondBase);
 f.rules.mockResolvedValue({...data.rules,merchant:secondBase.merchant});
 f.verify.mockResolvedValue({...data.saved,record:{...data.saved.record,result:{...data.saved.record.result,ready_for_upload:true}}});
 render(<SheinImportPage {...props} recordId={secondBase.id}/>);
 await screen.findByText("目标平台草稿 R1");
 fireEvent.click(screen.getByRole("button",{name:"核实原操作"}));
 await waitFor(()=>expect(f.verify).toHaveBeenCalledWith(expect.objectContaining({key:pending!.key,command:pending!.command}),expect.any(AbortSignal)));
 await waitFor(()=>expect(screen.queryByText("结果待核实")).not.toBeInTheDocument());
 expect(screen.queryByText(/本次已保存 Target R2/)).not.toBeInTheDocument();
 const target=new URL(screen.getByRole("link",{name:"返回平台草稿"}).getAttribute("href")!,"https://test.invalid");
 expect(target.pathname).toBe("/workbench/listing/missing");
 expect(Object.fromEntries(target.searchParams)).toEqual({preparation:id.batch,store:secondStore,mode:"platform"});
 expect(f.save).toHaveBeenCalledTimes(1);
});

it("keeps a restored split command verifiable and retryable before a store or family is reopened",async()=>{
 const original={userId:"actor",organizationId:"org",key:id.saved,route:"create-color-split",command:{request:{sourceId:id.source,baseRecordId:id.base}}} as unknown as SupplyIntent;
 f.context.pendingSupplyIntent=original;
 f.verify.mockRejectedValueOnce(new SupplyAPIError("NOT_FOUND",404));
 f.save.mockRejectedValueOnce(new SupplyAPIError("OUTCOME_UNKNOWN",503));
 render(<SheinImportPage {...props} recordId={undefined}/>);
 await screen.findByText(/候选 R3/);
 fireEvent.click(screen.getByRole("button",{name:"核实原操作"}));
 await waitFor(()=>expect(f.verify).toHaveBeenCalledWith(original,expect.any(AbortSignal)));
 fireEvent.click(await screen.findByRole("button",{name:"重试原请求"}));
 await waitFor(()=>expect(f.save).toHaveBeenCalledWith(original,expect.any(AbortSignal)));
 expect(f.context.pendingSupplyIntent?.key).toBe(original.key);
 expect(screen.getByRole("combobox",{name:"SHEIN 店铺"})).toHaveValue("");
});

it("reopens the exact recovered split family without creating another command",async()=>{
 const data=sheinImportFixture();
 const original={userId:"actor",organizationId:"org",key:id.saved,route:"create-color-split",command:{request:{sourceId:id.source,baseRecordId:id.base}}} as unknown as SupplyIntent;
 f.context.pendingSupplyIntent=original;
 const familyId=id.saved;
 const records=[id.saved,id.candidate].map((recordId,n)=>({...data.base,id:recordId,targetId:recordId,input:{...data.base.input,targetId:recordId,colorPartition:{familyId,partitionId:recordId,colorAttributeId:12,colorValueId:34+n,baseRecordId:id.base,completionDraft:{recordId:id.candidate,revision:3},members:[{sourceIndex:n,supplierSku:"target-sku",saleAttributes:[]}],localOnly:true}}}));
 f.verify.mockResolvedValueOnce({familyId,records});
 render(<SheinImportPage {...props} recordId={undefined}/>);
 fireEvent.click(await screen.findByRole("button",{name:"核实原操作"}));
 await waitFor(()=>expect(f.replace).toHaveBeenCalled());
 const query=new URL(f.replace.mock.lastCall![0],"https://fixture.test").searchParams;
 expect(query.get("target")).toBe(id.base);expect(query.get("family")).toBe(familyId);expect(query.get("revision")).toBe("3");
 expect(f.context.pendingSupplyIntent).toBeNull();expect(f.save).not.toHaveBeenCalled();
});
