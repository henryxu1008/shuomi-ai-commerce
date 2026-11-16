import {cleanup, render, screen, within} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach, beforeEach, expect, it, vi} from "vitest";
import {useMarketCommands} from "./shared";
import {SupplyCatalogsPage} from "./catalogs";
it.each([undefined,"apply"] as const)("provides a normal connection history entry on %s",view=>{
 page(view);expect(screen.getByRole("link",{name:"查看对接进度与记录"})).toHaveAttribute("href","/workbench/supply/catalogs/apply?view=progress");
});

const context=vi.hoisted(()=>({user:{id:"member-a"},effectiveOrganization:{id:"org-a"},isLoading:false,isSwitching:false,error:null,blockingError:null}));
const commands=vi.hoisted(()=>({execute:vi.fn(),saved:null as {recordId:string}|null,notice:null as string|null,locked:false,busy:false}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>context}));
vi.mock("./shared",async importOriginal=>({...await importOriginal<typeof import("./shared")>(),useMarketCommands:vi.fn(()=>commands)}));
afterEach(cleanup);
beforeEach(()=>{vi.clearAllMocks();context.isLoading=false;context.effectiveOrganization.id="org-a";commands.saved=null;commands.notice=null;commands.locked=false;commands.busy=false;
 Object.defineProperty(HTMLDialogElement.prototype,"showModal",{configurable:true,value:function(this:HTMLDialogElement){this.open=true}});
 Object.defineProperty(HTMLDialogElement.prototype,"close",{configurable:true,value:function(this:HTMLDialogElement){this.open=false}});
});
function page(view?:"selection"|"apply",acquisitionAvailable=false,sdsAvailable=false){return render(<SupplyCatalogsPage view={view} acquisitionAvailable={acquisitionAvailable} sdsAvailable={sdsAvailable}/>)}
it("shows the four branded catalog entries and honest API connection status",()=>{
 page();
 expect(screen.getByRole("button",{name:"API 对接"})).toHaveAttribute("aria-pressed","true");
 for(const name of ["大健云仓","Doba","SaleYee","CJdropshipping"]){
  const card=within(screen.getByRole("article",{name}));
  expect(card.getByRole("img",{name:`${name} Logo`})).toHaveAttribute("src",expect.stringContaining("/brands/catalogs/"));
  expect(card.getByText("未接入")).toBeVisible();
  expect(card.getByRole("link",{name:"访问官网"})).toHaveAttribute("target","_blank");
 }
 expect(screen.queryByRole("button",{name:/连接|立即采集/})).not.toBeInTheDocument();
});
it("filters the directory by domestic warehouse and country on the same coverage entry",async()=>{
 page();const user=userEvent.setup();
 await user.selectOptions(screen.getByRole("combobox",{name:"仓库类型"}),"domestic");
 expect(screen.getAllByRole("article")).toHaveLength(1);expect(screen.getByRole("article",{name:"CJdropshipping"})).toBeVisible();
 expect(within(screen.getByRole("combobox",{name:"仓库区域"})).queryByRole("option",{name:"美国"})).not.toBeInTheDocument();
 await user.selectOptions(screen.getByRole("combobox",{name:"仓库区域"}),"CN");
 await user.selectOptions(screen.getByRole("combobox",{name:"仓库类型"}),"overseas");
 expect(screen.getByRole("combobox",{name:"仓库区域"})).toHaveValue("");
 await user.selectOptions(screen.getByRole("combobox",{name:"仓库区域"}),"DE");
 expect(screen.getAllByRole("article")).toHaveLength(3);expect(screen.queryByRole("article",{name:"Doba"})).not.toBeInTheDocument();
});
it("retains warehouse filters when changing to browser plugin mode without claiming plugin support",async()=>{
 page();const user=userEvent.setup();
 await user.selectOptions(screen.getByRole("combobox",{name:"仓库区域"}),"JP");
 await user.click(screen.getByRole("button",{name:"浏览器插件对接"}));
 expect(screen.getByRole("combobox",{name:"仓库区域"})).toHaveValue("JP");
 expect(screen.getAllByRole("article")).toHaveLength(2);
 expect(screen.getAllByText("插件暂未支持")).toHaveLength(2);
 expect(screen.queryByText("未接入")).not.toBeInTheDocument();
 await user.click(screen.getByRole("button",{name:"清除筛选"}));
 expect(screen.getAllByRole("article")).toHaveLength(4);
});
it("removes the selection entry and defers the application form until the dialog opens",()=>{
  page();
  expect(screen.getByRole("heading",{name:"货盘集成",level:1})).toBeVisible();
  expect(screen.queryByRole("link",{name:"进入货盘选品"})).not.toBeInTheDocument();
  expect(screen.getByRole("button",{name:"申请对接货盘"})).toBeVisible();
  expect(screen.queryByRole("button",{name:"提交对接申请"})).not.toBeInTheDocument();
  expect(useMarketCommands).not.toHaveBeenCalled();
});
it("opens the shared application form in place, submits the existing command and retains directory filters after close",async()=>{
 page();const user=userEvent.setup();
 await user.selectOptions(screen.getByRole("combobox",{name:"仓库区域"}),"DE");
 const trigger=screen.getByRole("button",{name:"申请对接货盘"});await user.click(trigger);
 const dialog=within(screen.getByRole("dialog",{name:"申请对接货盘"}));
 expect(dialog.getByLabelText("货盘名称")).toHaveFocus();
 for(const [label,value] of [["货盘名称","Fixture Catalog"],["货盘网站（HTTPS）","https://catalog.example.test"],["联系人","Fixture contact"],["联系电话","12345"],["供应品类","Home"]])await user.type(dialog.getByLabelText(label),value);
 await user.click(dialog.getByRole("button",{name:"提交对接申请"}));
 expect(commands.execute).toHaveBeenCalledExactlyOnceWith({action:"submit_connection",connection:{name:"Fixture Catalog",website:"https://catalog.example.test",contact:"Fixture contact",telephone:"12345",categories:"Home"}});
 await user.click(dialog.getByRole("button",{name:"取消"}));
 expect(screen.queryByRole("dialog")).not.toBeInTheDocument();expect(trigger).toHaveFocus();
 expect(screen.getByRole("combobox",{name:"仓库区域"})).toHaveValue("DE");
});
it("keeps pending submission controls locked inside the dialog",async()=>{
 commands.busy=true;commands.locked=true;page();const user=userEvent.setup();
 await user.click(screen.getByRole("button",{name:"申请对接货盘"}));
 const dialog=within(screen.getByRole("dialog",{name:"申请对接货盘"}));
 expect(dialog.getByRole("button",{name:"关闭"})).toBeDisabled();expect(dialog.getByRole("button",{name:"取消"})).toBeDisabled();expect(dialog.getByRole("button",{name:"提交对接申请"})).toBeDisabled();
});
it("shows the original unknown-result notice without allowing another submission",async()=>{
 commands.locked=true;commands.notice="原操作待确认，请先核实。";
 page();await userEvent.setup().click(screen.getByRole("button",{name:"申请对接货盘"}));
 const dialog=within(screen.getByRole("dialog",{name:"申请对接货盘"}));
 expect(dialog.getByText("原操作待确认，请先核实。")).toBeVisible();
 expect(dialog.getByRole("button",{name:"提交对接申请"})).toBeDisabled();expect(commands.execute).not.toHaveBeenCalled();
});
it("keeps the exact saved record link in the dialog and closes without a second submission",async()=>{
 commands.saved={recordId:"11111111-1111-4111-8111-111111111111"};page();const user=userEvent.setup();
 await user.click(screen.getByRole("button",{name:"申请对接货盘"}));const dialog=within(screen.getByRole("dialog",{name:"申请对接货盘"}));
 expect(dialog.getByRole("link",{name:"查看对接进展"})).toHaveAttribute("href","/workbench/supply/applications/11111111-1111-4111-8111-111111111111");
 expect(dialog.getByRole("button",{name:"提交对接申请"})).toBeDisabled();await user.click(dialog.getByRole("button",{name:"完成"}));expect(commands.execute).not.toHaveBeenCalled();expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
});
it.each([[false,false],[true,false],[false,true],[true,true]])("keeps selection links dependent on actual acquisition=%s / SDS=%s",(acquisitionAvailable,sdsAvailable)=>{
  page("selection",acquisitionAvailable,sdsAvailable);
  expect(screen.getByRole("heading",{name:"货盘选品",level:1})).toBeVisible();
  expect(within(screen.getByRole("navigation",{name:"面包屑"})).getByText("货盘选品")).toBeVisible();
  if(acquisitionAvailable)expect(screen.getByRole("link",{name:"进入 1688 选品与采集"})).toHaveAttribute("href","/workbench/supply/acquisition");
  else expect(screen.queryByRole("link",{name:"进入 1688 选品与采集"})).not.toBeInTheDocument();
  if(sdsAvailable)expect(screen.getByRole("link",{name:"查看模板与定制成品"})).toHaveAttribute("href","/workbench/supply/catalogs/sds");
  else expect(screen.getByText("当前实例未配置 SDS 定制能力")).toBeVisible();
  expect(screen.queryByRole("button",{name:"提交对接申请"})).not.toBeInTheDocument();
  expect(useMarketCommands).not.toHaveBeenCalled();
});
it("keeps the original member-scoped connection command and form fields on the apply leaf",async()=>{
  page("apply");
  expect(screen.getByRole("heading",{name:"申请对接",level:1})).toBeVisible();
  const breadcrumb=within(screen.getByRole("navigation",{name:"面包屑"}));
  expect(breadcrumb.getByRole("link",{name:"货盘集成"})).toHaveAttribute("href","/workbench/supply/catalogs");
  expect(breadcrumb.getByText("申请对接")).toBeVisible();
  expect(useMarketCommands).toHaveBeenCalledWith({userId:"member-a",organizationId:"org-a"});
  const user=userEvent.setup();
  await user.type(screen.getByLabelText("货盘名称"),"Fixture Catalog");
  await user.type(screen.getByLabelText("货盘网站（HTTPS）"),"https://catalog.example.test");
  await user.type(screen.getByLabelText("联系人"),"Fixture contact");
  await user.type(screen.getByLabelText("联系电话"),"12345");
  await user.type(screen.getByLabelText("供应品类"),"Home");
  await user.click(screen.getByRole("button",{name:"提交对接申请"}));
  expect(commands.execute).toHaveBeenCalledExactlyOnceWith({action:"submit_connection",connection:{name:"Fixture Catalog",website:"https://catalog.example.test",contact:"Fixture contact",telephone:"12345",categories:"Home"}});
});
it("keeps the existing pending lock and saved record link",()=>{
  commands.locked=true; commands.saved={recordId:"11111111-1111-4111-8111-111111111111"};
  page("apply");
  expect(screen.getByRole("button",{name:"提交对接申请"})).toBeDisabled();
  expect(screen.getByRole("link",{name:"查看对接进展"})).toHaveAttribute("href","/workbench/supply/applications/11111111-1111-4111-8111-111111111111");
});
it("waits for the existing identity boundary before mounting the application consumer",()=>{
  context.isLoading=true; page("apply");
  expect(screen.getByText("正在确认当前身份")).toBeVisible();
  expect(useMarketCommands).not.toHaveBeenCalled();
});
