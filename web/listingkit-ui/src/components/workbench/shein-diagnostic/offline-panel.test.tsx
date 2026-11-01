import {cleanup,render as renderRaw,screen,waitFor} from "@testing-library/react";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import type {ReactNode} from "react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {OfflineDiagnosticPanel} from "./offline-panel";
import {OfflineRecordError,writeOfflineIntent,readOfflineIntent} from "@/lib/api/offline-shein-records";
import {diagnosticFixture} from "@/test/fixtures/shein-diagnostic";
const clients:QueryClient[]=[];
function render(node:ReactNode){const client=new QueryClient();clients.push(client);return renderRaw(node,{wrapper:({children})=><QueryClientProvider client={client}>{children}</QueryClientProvider>})}
const state=vi.hoisted(()=>({request:vi.fn(),context:{} as Record<string,unknown>}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>state.context}));
vi.mock("@/lib/api/offline-shein-records",async original=>({...await original<object>(),offlineRequest:state.request}));
vi.mock("@/lib/api/store-center",async original=>({...await original<object>(),storeCenterRequest:vi.fn(async()=>({items:[{id:"550e8400-e29b-41d4-a716-446655440001",name:"Synthetic SHEIN"}],pagination:{total:1}}))}));
const id="550e8400-e29b-41d4-a716-446655440001",scope={userId:"actor-a",organizationId:"org-a"};
const input={product_key:"p",snapshot_version:"1",store_id:id,country:"US" as const,language:"en" as const,action:"save_draft" as const};
const item={id,batchId:id,source:{productKey:"p",version:"1",publicationId:id,kind:"own" as const},revision:1,createdAt:"2026-01-01T00:00:00Z"};
const diagnostic=diagnosticFixture("save_draft");
const saved={origin:"saved",recordId:id,operationId:id,input,sourceVersion:`sha256:${"b".repeat(64)}`,createdAt:"2026-01-01T00:00:00Z",diagnostic:{...diagnostic,not_evaluated:diagnostic.not_evaluated.filter(v=>v!=="approved_asset_provenance_and_consent")}};
beforeEach(()=>{sessionStorage.clear();state.request.mockReset();state.context={user:{id:"actor-a"},effectiveOrganization:{id:"org-a"},permissions:["listingkit.admin.read","listingkit.admin.write","workbench.collection.read","workbench.store.read","workbench.report.read","workbench.report.manage"],roles:["platform_admin"]}});
afterEach(()=>{cleanup();for(const c of clients)c.clear();clients.length=0;});
it("keeps the original intent after an unknown outcome and a 404 lookup",async()=>{
 state.request.mockRejectedValueOnce(new OfflineRecordError("OUTCOME_UNKNOWN")).mockRejectedValueOnce(new OfflineRecordError("NOT_FOUND"));render(<OfflineDiagnosticPanel item={item}/>);
 await screen.findByRole("option",{name:"Synthetic SHEIN"});await userEvent.selectOptions(screen.getByLabelText("离线诊断店铺"),id);await userEvent.click(screen.getByRole("button",{name:"生成离线诊断"}));
 await screen.findByRole("button",{name:"查询原请求"});const original=readOfflineIntent(scope);expect(original).not.toBeNull();
 await userEvent.click(screen.getByRole("button",{name:"查询原请求"}));await screen.findByRole("alert");expect(state.request.mock.calls[1][1]).toBe(`/commands/${original!.key}`);expect(readOfflineIntent(scope)?.key).toBe(original!.key);expect(screen.getByRole("button",{name:"生成离线诊断"})).toBeDisabled();
});
it("finds a committed original result without creating another record",async()=>{
 writeOfflineIntent(scope,{key:id,collectionItemId:id,input});state.request.mockResolvedValue(saved);render(<OfflineDiagnosticPanel item={item}/>);
 await userEvent.click(await screen.findByRole("button",{name:"查询原请求"}));await screen.findByText("以下为保存时的诊断结果。");expect(state.request).toHaveBeenCalledTimes(1);expect(state.request.mock.calls[0][1]).toBe(`/commands/${id}`);expect(readOfflineIntent(scope)).toBeNull();expect(screen.getByRole("link",{name:"前往我的报告保存此结果"})).toHaveAttribute("href",`/workbench/ai/reports?sourceKind=SHEIN_RECORD&sourceId=${id}`);
});
it("allows a new explicit operation after a determinate not-ready rejection",async()=>{
 state.request.mockRejectedValue(new OfflineRecordError("NOT_READY"));render(<OfflineDiagnosticPanel item={item}/>);await screen.findByRole("option",{name:"Synthetic SHEIN"});await userEvent.selectOptions(screen.getByLabelText("离线诊断店铺"),id);await userEvent.click(screen.getByRole("button",{name:"生成离线诊断"}));await waitFor(()=>expect(readOfflineIntent(scope)).toBeNull());expect(screen.getByRole("button",{name:"生成离线诊断"})).toBeEnabled();
});
it("revoked write permission hides generation",async()=>{
 state.context.permissions=["listingkit.admin.read"];render(<OfflineDiagnosticPanel item={item}/>);expect(screen.queryByRole("button",{name:"生成离线诊断"})).not.toBeInTheDocument();expect(state.request).not.toHaveBeenCalled();
});

it("ignores late recovery after changing enterprise and retains the original intent",async()=>{
 let finish!:(value:typeof saved)=>void;
 writeOfflineIntent(scope,{key:id,collectionItemId:id,input});state.request.mockImplementation(()=>new Promise(resolve=>{finish=resolve}));
 const view=render(<OfflineDiagnosticPanel item={item}/>);await userEvent.click(await screen.findByRole("button",{name:"查询原请求"}));
 state.context={...state.context,effectiveOrganization:{id:"org-b"}};view.rerender(<OfflineDiagnosticPanel item={item}/>);
 finish(saved);await waitFor(()=>expect(screen.queryByText("以下为保存时的诊断结果。")).not.toBeInTheDocument());expect(readOfflineIntent(scope)?.key).toBe(id);expect(readOfflineIntent({...scope,organizationId:"org-b"})).toBeNull();
});
