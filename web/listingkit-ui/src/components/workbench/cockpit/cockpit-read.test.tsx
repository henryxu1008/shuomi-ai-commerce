import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { CockpitPage } from "./cockpit-page";
const id = "123e4567-e89b-42d3-a456-426614174000";
const context = vi.hoisted(() => ({ user:{id:"actor-a"},effectiveOrganization:{id:"org-a"},operationsCockpitAvailable:true,permissions:["workbench.cockpit.alerts.read"],isLoading:false,isSwitching:false,selectionRequired:false,error:null,blockingError:null,retry:vi.fn(),registerOrganizationSwitchGuard:()=>()=>{} }));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>context}));
vi.mock("next/link",()=>({default:({href,children,...props}:React.AnchorHTMLAttributes<HTMLAnchorElement>)=><a href={href} {...props}>{children}</a>}));
const caps={access:{goalsRead:true,goalsCreate:true,goalsManage:false,storesRead:true,factsWrite:false,alertsRead:true,adviceRead:true},today:"2026-10-10",stores:[{id,name:"真实店铺",platform:"shein",region:"US",status:"active"}]};
const rules={rules:[],total:61,summary:{total:61,critical:1,attention:25,data:35},page:1,capturedAt:"2026-10-10T01:00:00Z",goalUnavailable:false,observations:{state:"unavailable",exceptional:0,complete:false,sources:[],latest:[],actionPath:"/workbench/store-orders"}};
const period = { startDate: "2026-09-01", endDate: "2026-09-30" };
const goalRule = {
 id: "goal_assessment:example:1:2026-09-29",
 kind: "goal_assessment", level: "attention", title: "利润目标需关注", reason: "尚未达到利润进度阈值",
 advice: "核查目标和收入成本", actionPath: "/workbench/overview/goals", actionLabel: "查看目标依据",
 source: "manual", capturedAt: rules.capturedAt, period: { ...period, endDate: "2026-09-29" },
};
const storeRule = {
 ...goalRule, id: "data_missing:" + id, kind: "data_missing", level: "data", title: "店铺经营数据缺失",
 actionPath: "/workbench/overview/stores?" + new URLSearchParams({ storeId: id, ...period }), storeId: id, period,
};
const visibleRules = { ...rules, rules: [goalRule, storeRule], total: 2, summary: { total: 2, critical: 0, attention: 1, data: 1 } };
const totals={revenue:0,refunds:0,procurement:0,logistics:0,platform:0,advertising:0,other:0,netRevenue:0,netProfit:0,margin:null};
const matrix={rows:[{store:caps.stores[0],complete:false,state:"data_incomplete",totals,growth:null,gapCount:1,excludedCount:0,recordCount:0}],total:60,page:1,summary:totals,summaryComplete:false,states:{loss:10,break_even:10,recorded_complete:10,data_incomplete:30},capturedAt:rules.capturedAt};
const client=()=>new QueryClient({defaultOptions:{queries:{retry:false}}});
beforeEach(()=>{window.history.replaceState(null,"","/");context.effectiveOrganization={id:"org-a"};});
it("preserves every linked store in the matrix and never opens an arbitrary first-store detail",async()=>{
 const other="123e4567-e89b-42d3-a456-426614174001";
 window.history.replaceState(null,"",`/?storeId=${id}&storeId=${other}&startDate=2026-09-01&endDate=2026-09-30`);
 context.permissions=["workbench.cockpit.stores.read"];
 const requests:string[]=[];
 vi.stubGlobal("fetch",vi.fn(async(url:string)=>{requests.push(url);return Response.json(url.endsWith("capabilities")?caps:matrix);}));
 render(<QueryClientProvider client={client()}><CockpitPage mode="stores"/></QueryClientProvider>);
 await screen.findByLabelText("选择真实店铺");
 const matrixURL=requests.find(url=>url.includes("stores?"))!;
 expect(new URL(matrixURL,"https://app.test").searchParams.getAll("storeId")).toEqual([id,other]);
 expect(requests.some(url=>url.includes(`stores/${id}`))).toBe(false);
 fireEvent.click(screen.getByRole("button",{name:"重置"}));
 await waitFor(()=>expect(new URL(requests.at(-1)!,"https://app.test").searchParams.getAll("storeId")).toEqual([]));
});
afterEach(()=>{cleanup();vi.unstubAllGlobals();});

it.each(["alerts","advice"] as const)("%s shows full summary and sends bounded search while resetting pagination",async mode=>{
 context.permissions=[`workbench.cockpit.${mode}.read`];
 const requests:string[]=[];
 vi.stubGlobal("fetch",vi.fn(async(url:string)=>{requests.push(url);return Response.json(url.endsWith("capabilities")?caps:rules);}));
 render(<QueryClientProvider client={client()}><CockpitPage mode={mode}/></QueryClientProvider>);
 for(const [label,count] of [["当前规则","61"],["目标异常","1"],["需关注","25"],["数据提醒","35"]]){
  const card=(await screen.findByText(label,{selector:"span"})).closest<HTMLElement>("[data-stat]")!;
  expect(within(card).getByText(count,{selector:"strong"})).toBeVisible();
 }
 fireEvent.click(screen.getByRole("button",{name:"下一页"}));
 await waitFor(()=>expect(requests.at(-1)).toContain("page=2"));
 fireEvent.change(screen.getByLabelText("搜索规则或店铺"),{target:{value:"真实"}});
 await waitFor(()=>{const q=new URL(requests.at(-1)!,"https://app.test").searchParams;expect(q.get("search")).toBe("真实");expect(q.get("page")).toBe("1");});
});

it.each([goalRule, storeRule])("opens the original $kind alert with its query period", async rule => {
 context.permissions = ["workbench.cockpit.alerts.read", "workbench.cockpit.advice.read"];
 window.history.replaceState(null, "", "/workbench/overview/advice?" + new URLSearchParams(period));
 const fetch = vi.fn(async (url: string) => Response.json(url.endsWith("capabilities") ? caps : visibleRules));
 vi.stubGlobal("fetch", fetch);
 const queryClient = client();
 const renderMode = (mode: "advice" | "alerts") =>
  <QueryClientProvider client={queryClient}><CockpitPage mode={mode}/></QueryClientProvider>;
 const view = render(renderMode("advice"));
 fireEvent.click(await screen.findByRole("button", { name: new RegExp(rule.title) }));
 const link = screen.getByRole("link", { name: "查看原预警" });
 const expected = { ...period, kind: rule.kind, level: rule.level, ruleId: rule.id, ...("storeId" in rule ? { storeId: rule.storeId } : {}) };
 const target = new URL(link.getAttribute("href")!, window.location.origin);
 expect(target.pathname).toBe("/workbench/overview/alerts");
 expect(Object.fromEntries(target.searchParams)).toEqual(expected);

 window.history.replaceState(null, "", target.pathname + target.search);
 view.rerender(renderMode("alerts"));
 expect(await screen.findByRole("heading", { name: rule.title })).toBeVisible();
 const other = rule === goalRule ? storeRule : goalRule;
 expect(screen.queryByRole("heading", { name: other.title })).not.toBeInTheDocument();
 const alertRequest = fetch.mock.calls.find(([url]) => url.includes("/alerts?"))![0];
 const { ruleId, ...filters } = expected;
 expect(Object.fromEntries(new URL(alertRequest, window.location.origin).searchParams)).toMatchObject(filters);
 expect(new URL(alertRequest, window.location.origin).searchParams.has("ruleId")).toBe(false);
 expect(ruleId).toBe(rule.id);
});

it("shows no return link when the current member lacks alert read access", async () => {
 context.permissions = ["workbench.cockpit.advice.read"];
 const adviceOnly = { ...caps, access: { ...caps.access, alertsRead: false } };
 vi.stubGlobal("fetch", vi.fn(async (url: string) => Response.json(url.endsWith("capabilities") ? adviceOnly : visibleRules)));
 render(<QueryClientProvider client={client()}><CockpitPage mode="advice"/></QueryClientProvider>);
 expect(await screen.findByRole("heading", { name: goalRule.title })).toBeVisible();
 expect(screen.queryByRole("link", { name: "查看原预警" })).not.toBeInTheDocument();
});

it("reports that a linked alert is unavailable without opening another rule", async () => {
 context.permissions = ["workbench.cockpit.alerts.read"];
 window.history.replaceState(null, "", "/workbench/overview/alerts?ruleId=obsolete");
 vi.stubGlobal("fetch", vi.fn(async (url: string) => Response.json(url.endsWith("capabilities") ? caps : visibleRules)));
 render(<QueryClientProvider client={client()}><CockpitPage mode="alerts"/></QueryClientProvider>);
 expect(await screen.findByText("当前结果中没有这条原预警。")).toBeVisible();
 expect(screen.queryByRole("heading", { name: goalRule.title })).not.toBeInTheDocument();
 expect(screen.queryByRole("heading", { name: storeRule.title })).not.toBeInTheDocument();
});

it("matrix platform and site selection uses capabilities, resets page and comparison",async()=>{
 context.permissions=["workbench.cockpit.stores.read"];
 const requests:string[]=[];
 vi.stubGlobal("fetch",vi.fn(async(url:string)=>{requests.push(url);return Response.json(url.endsWith("capabilities")?caps:matrix);}));
 render(<QueryClientProvider client={client()}><CockpitPage mode="stores"/></QueryClientProvider>);
 fireEvent.click(await screen.findByLabelText("选择真实店铺"));
 fireEvent.click(screen.getByRole("button",{name:"下一页"}));
 await waitFor(()=>expect(requests.at(-1)).toContain("page=2"));
 fireEvent.change(screen.getByLabelText("平台"),{target:{value:"shein"}});
 fireEvent.change(screen.getByLabelText("站点"),{target:{value:"US"}});
 await waitFor(()=>{const q=new URL(requests.at(-1)!,"https://app.test").searchParams;expect(q.get("platform")).toBe("shein");expect(q.get("region")).toBe("US");expect(q.get("page")).toBe("1");});
 expect(screen.getByLabelText("选择真实店铺")).not.toBeChecked();
 expect(screen.getByRole("button",{name:"开始对比"})).toBeDisabled();
});

it("refresh failure and organization switch hide previous summary",async()=>{
 context.permissions=["workbench.cockpit.alerts.read"];
 let failed=false;
 vi.stubGlobal("fetch",vi.fn(async(url:string)=>url.endsWith("capabilities")?Response.json(caps):failed?Response.json({code:"UNAVAILABLE"},{status:503}):Response.json(rules)));
 const queryClient=client(),ui=()=> <QueryClientProvider client={queryClient}><CockpitPage mode="alerts"/></QueryClientProvider>;
 const view=render(ui());await screen.findByText("61",{selector:"strong"});
 failed=true;fireEvent.click(screen.getByRole("button",{name:"刷新数据"}));
 await waitFor(()=>expect(screen.queryByText("61",{selector:"strong"})).not.toBeInTheDocument());
 context.effectiveOrganization={id:"org-b"};view.rerender(ui());
 await waitFor(()=>expect(screen.queryByText("61",{selector:"strong"})).not.toBeInTheDocument());
});

it("unreadable existing goal is never presented as an unconfigured goal",async()=>{
 context.permissions=["workbench.cockpit.goals.read"];
 const unavailable={goal:null,evaluation:null,head:{goalId:id,revision:"2",scopeValid:false,canReconfigure:true},goalUnavailable:true,capturedAt:rules.capturedAt,basis:[]};
 vi.stubGlobal("fetch",vi.fn(async(url:string)=>Response.json(url.endsWith("capabilities")?caps:unavailable)));
 render(<QueryClientProvider client={client()}><CockpitPage mode="goals"/></QueryClientProvider>);
 expect(await screen.findByText("不可访问",{selector:"span"})).toBeVisible();
 expect(screen.getAllByText("无法读取",{selector:"strong"})).toHaveLength(3);
 expect(screen.queryByText("尚未设置")).not.toBeInTheDocument();
 expect(screen.queryByText("尚未选择经营范围")).not.toBeInTheDocument();
 expect(screen.getByText("版本记录不可读取，原目标仍保留。")).toBeVisible();
 expect(screen.getByRole("link",{name:"维护经营目标"})).toHaveAttribute("href","/workbench/overview/goals/settings");
});
