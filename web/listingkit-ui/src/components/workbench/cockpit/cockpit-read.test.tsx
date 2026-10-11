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
const totals={revenue:0,refunds:0,procurement:0,logistics:0,platform:0,advertising:0,other:0,netRevenue:0,netProfit:0,margin:null};
const matrix={rows:[{store:caps.stores[0],complete:false,state:"data_incomplete",totals,growth:null,gapCount:1,excludedCount:0,recordCount:0}],total:60,page:1,summary:totals,summaryComplete:false,states:{loss:10,break_even:10,recorded_complete:10,data_incomplete:30},capturedAt:rules.capturedAt};
const client=()=>new QueryClient({defaultOptions:{queries:{retry:false}}});
beforeEach(()=>{window.history.replaceState(null,"","/");context.effectiveOrganization={id:"org-a"};});
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
