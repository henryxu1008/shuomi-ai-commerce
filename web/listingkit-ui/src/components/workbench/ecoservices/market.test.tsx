import {cleanup, fireEvent, render, screen, waitFor} from "@testing-library/react";
import {QueryClient, QueryClientProvider} from "@tanstack/react-query";
import {afterEach, beforeEach, expect, it, vi} from "vitest";
import type {ReactNode} from "react";
import {ecoRequest, EcoservicesError, type EcoListing} from "@/lib/api/ecoservices";
import {EcoservicesMarket} from "./market";

const context = vi.hoisted(() => ({user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.ecoservices.read","workbench.ecoservices.purchase"],isLoading:false,isSwitching:false,error:null,blockingError:null,registerOrganizationSwitchGuard:vi.fn(()=>()=>undefined)}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>context}));
vi.mock("@/components/workbench/resources/resource-dialog",()=>({ResourceDialog:({title,children}:{title:string;children:ReactNode})=><div role="dialog" aria-label={title}>{children}</div>}));
vi.mock("@/lib/api/ecoservices",async original=>({...await original<typeof import("@/lib/api/ecoservices")>(),ecoRequest:vi.fn()}));
let client: QueryClient;
beforeEach(()=>{
  localStorage.clear();
  context.permissions=["workbench.ecoservices.read","workbench.ecoservices.purchase"];
  client=new QueryClient({defaultOptions:{queries:{retry:false},mutations:{retry:false}}});
});
afterEach(()=>{cleanup();client.clear();vi.resetAllMocks();localStorage.clear();});
function mount(nonPaymentOnly=false){render(<QueryClientProvider client={client}><EcoservicesMarket nonPaymentOnly={nonPaymentOnly}/></QueryClientProvider>);}
const empty={listings:[],total:"0",counts:{COMPANY_REGISTRATION:0,TRADEMARK_REGISTRATION:0,STORE_OPENING:0,STORE_OPERATION:0}};

it("keeps four informational categories for an empty catalog without invented availability",async()=>{
  vi.mocked(ecoRequest).mockResolvedValue(empty);mount();
  expect(await screen.findByRole("heading",{name:"公司注册"})).toBeVisible();
  for(const name of ["商标注册","店铺开通服务","店铺代运营"])expect(screen.getByRole("heading",{name})).toBeVisible();
  expect(screen.getAllByRole("button",{name:"暂无已发布服务"})).toHaveLength(4);
  for(const button of screen.getAllByRole("button",{name:"暂无已发布服务"}))expect(button).toBeDisabled();
  expect(screen.getByRole("button",{name:"全部服务 0"})).toHaveAttribute("aria-pressed","true");
  expect(screen.queryByRole("button",{name:/查看详情/})).not.toBeInTheDocument();
  expect(screen.queryByText(/¥/)).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button",{name:"店铺服务 0"}));
  await waitFor(()=>expect(screen.getAllByRole("button",{name:"暂无已发布服务"})).toHaveLength(2));
  expect(screen.queryByRole("heading",{name:"公司注册"})).not.toBeInTheDocument();
  expect(vi.mocked(ecoRequest).mock.calls.every(call=>!call[3]?.method)).toBe(true);
});

it("keeps published listing facts and disables purchase in nonpayment mode",async()=>{
  const listing:EcoListing={id:"4841d296-ef14-4c16-8d25-a7667e534feb",providerName:"实际服务商",category:"COMPANY_REGISTRATION",title:"实际公司注册方案",description:"实际交付范围",items:["实际事项"],regions:["中国"],platforms:[],priceMinor:"12345",deliveryDays:7,state:"PUBLISHED",version:"1"};
  vi.mocked(ecoRequest).mockResolvedValue({...empty,listings:[listing],total:"1",counts:{...empty.counts,COMPANY_REGISTRATION:1}});mount(true);
  expect(await screen.findByRole("heading",{name:listing.title})).toBeVisible();
  expect(screen.getByText("实际服务商 · ¥123.45起")).toBeVisible();
  expect(screen.queryByText("服务类型介绍")).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button",{name:/查看详情/}));
  expect(screen.getByRole("dialog",{name:listing.title})).toBeVisible();
  expect(screen.getByRole("button",{name:"提交需求"})).toBeDisabled();
});

it("does not present informational cards as a successful catalog read on errors or denial",async()=>{
  vi.mocked(ecoRequest).mockRejectedValue(new EcoservicesError("ECOSERVICES_UNAVAILABLE",503));mount();
  await screen.findByRole("alert");
  expect(screen.queryByRole("heading",{name:"公司注册"})).not.toBeInTheDocument();
  cleanup();client.clear();vi.mocked(ecoRequest).mockClear();context.permissions=[];mount();
  expect(screen.getByRole("heading",{name:"当前身份没有服务市场读取权限"})).toBeVisible();
  expect(vi.mocked(ecoRequest)).not.toHaveBeenCalled();
});
