import { act,cleanup,fireEvent,render,screen,waitFor } from "@testing-library/react";
import { afterEach,beforeEach,expect,it,vi } from "vitest";
import { browserCaptureFixture } from "@/lib/contracts/browser-capture.fixture";
import { WorkbenchContextError } from "@/lib/api/workbench-context";
const mocks=vi.hoisted(()=>({context:{} as Record<string,unknown>,capture:vi.fn(),read:vi.fn(),payload:vi.fn(),retry:vi.fn(),message:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>mocks.context}));
vi.mock("@/lib/api/browser-capture",()=>({capture1688:mocks.capture,readBrowserCaptureByKey:mocks.read,readBrowserCapture:mocks.read}));
import { CaptureReceiver } from "./capture-receiver";
const key="22222222-2222-4222-8222-22222222222b",op="11111111-1111-4111-8111-11111111111a";
const href=`/capture/1688#extensionId=${"a".repeat(32)}&handoffId=${op}&idempotencyKey=${key}`;
const receipt={schemaVersion:1,operationId:op,outcome:"published",replayed:false,productKey:"crawler:1688:981645030344",publicationId:"publication",catalogVersion:"1",warnings:[],missingFacts:[]};
const fresh=(org="org-B",member="member")=>({user:{id:"actor"},effectiveOrganizationId:org,effectiveMemberId:member,selectionRequired:false});
beforeEach(()=>{
 vi.clearAllMocks();window.history.replaceState(null,"",href);
 mocks.context={user:{id:"actor"},effectiveOrganization:{id:"org-B",name:"Enterprise B"},effectiveMemberId:"member",retry:mocks.retry};
 mocks.retry.mockResolvedValue(fresh());mocks.payload.mockResolvedValue(browserCaptureFixture());mocks.capture.mockResolvedValue(receipt);mocks.read.mockResolvedValue(receipt);
 mocks.message.mockImplementation(async(m)=>{
  if(m.type==="capture.read")return {version:2,type:"capture.draft",idempotencyKey:key,phase:"ready",canSubmit:true,payload:await mocks.payload()};
  return {version:2,type:"capture.metadata",idempotencyKey:key,phase:m.type==="capture.send"?"sending":"ready"};
 });
 vi.stubGlobal("chrome",{runtime:{sendMessage:(_id:string,m:unknown,callback:(v:unknown)=>void)=>{void mocks.message(m).then(callback,()=>callback({version:2,code:"DRAFT_CHANGED"}));}}});
});
afterEach(()=>{cleanup();vi.unstubAllGlobals();vi.restoreAllMocks();});
const loaded=async()=>{await screen.findByText("Browser fixture");};
const submit=()=>fireEvent.click(screen.getByRole("button",{name:"Confirm and submit"}));
it("verifies context before reading the body, then freezes and persists send intent before the only POST",async()=>{
 mocks.capture.mockImplementation(async(intent)=>{expect(mocks.message.mock.calls.map(c=>c[0].type)).toEqual(["capture.meta","capture.read","capture.bind","capture.send"]);expect(window.location.hash).toBe(`#operationKey=${key}`);expect(intent.key).toBe(key);return receipt;});
 render(<CaptureReceiver/>);await loaded();expect(mocks.retry).toHaveBeenCalledOnce();expect(mocks.capture).not.toHaveBeenCalled();submit();await screen.findByText("Published version 1");expect(mocks.capture).toHaveBeenCalledOnce();
 expect(mocks.message.mock.calls.find(c=>c[0].type==="capture.bind")![0].body).toBe(mocks.capture.mock.calls[0][0].body);
 expect(document.querySelector("[data-batch-submit-result]")).toHaveAttribute("data-batch-submit-result","published");
});
it("does not read any source content until a trusted member context is ready",async()=>{
 mocks.context.isLoading=true;const view=render(<CaptureReceiver/>);await act(async()=>{});expect(mocks.message).not.toHaveBeenCalled();expect(screen.queryByText("Browser fixture")).toBeNull();
 mocks.context={...mocks.context,isLoading:false};view.rerender(<CaptureReceiver/>);await loaded();
});
it("does not POST when the durable intent cannot be retained",async()=>{
 render(<CaptureReceiver/>);await loaded();mocks.message.mockImplementation(async(m)=>m.type==="capture.bind"?{version:2,code:"DRAFT_STORAGE_FULL"}:{version:2,type:"capture.metadata",idempotencyKey:key,phase:"ready"});
 submit();await screen.findByText(/DRAFT_STORAGE_FULL/);expect(mocks.capture).not.toHaveBeenCalled();
});
it.each(["capture.meta","capture.read","capture.bind","capture.send"])("guides original-draft recovery after %s expires without dispatching or resubmitting",async stage=>{
 const message=mocks.message.getMockImplementation()!;
 if(stage==="capture.meta"||stage==="capture.read")mocks.message.mockImplementation(async m=>m.type===stage?{version:2,code:"CAPTURE_EXPIRED"}:message(m));
 render(<CaptureReceiver/>);
 if(stage==="capture.bind"||stage==="capture.send"){
  await loaded();mocks.message.mockImplementation(async m=>m.type===stage?{version:2,code:"CAPTURE_EXPIRED"}:message(m));submit();
 }
 await waitFor(()=>expect(screen.getByRole("status")).toHaveTextContent("交接已过期（CAPTURE_EXPIRED）"));
 expect(screen.getByRole("status")).toHaveTextContent("本机保全资料");expect(screen.getByRole("status")).toHaveTextContent("恢复原操作");
 expect(screen.getByRole("status")).not.toHaveTextContent("OUTCOME_UNKNOWN");
 expect(screen.queryByRole("button",{name:"Confirm and submit"})).toBeNull();expect(mocks.capture).not.toHaveBeenCalled();
 expect(mocks.message.mock.calls.filter(c=>c[0].type==="capture.status")).toHaveLength(0);
});
it("requires fresh consent after explicit recovery with a new nonce and the same key and body",async()=>{
 const message=mocks.message.getMockImplementation()!;
 const mounted=render(<CaptureReceiver/>);await loaded();
 mocks.message.mockImplementation(async m=>m.type==="capture.bind"?{version:2,code:"CAPTURE_EXPIRED"}:message(m));submit();
 await waitFor(()=>expect(screen.getByRole("status")).toHaveTextContent("CAPTURE_EXPIRED"));
 const original=mocks.message.mock.calls.find(c=>c[0].type==="capture.bind")![0];
 mounted.unmount();mocks.message.mockImplementation(message);
 const nonce="33333333-3333-4333-8333-33333333333c";
 window.history.replaceState(null,"",`/capture/1688#extensionId=${"a".repeat(32)}&handoffId=${nonce}&idempotencyKey=${key}`);
 render(<CaptureReceiver/>);await loaded();expect(mocks.capture).not.toHaveBeenCalled();submit();await screen.findByText("Published version 1");
 const restored=mocks.message.mock.calls.filter(c=>c[0].type==="capture.bind").at(-1)![0];
 expect(restored).toEqual({...original,handoffId:nonce});expect(mocks.capture).toHaveBeenCalledOnce();expect(mocks.capture.mock.calls[0][0]).toMatchObject({key,body:original.body});
});
it("keeps the saved result when the handoff expires while reporting the receipt",async()=>{
 const message=mocks.message.getMockImplementation()!;
 mocks.message.mockImplementation(async m=>m.type==="capture.status"?{version:2,code:"CAPTURE_EXPIRED"}:message(m));
 render(<CaptureReceiver/>);await loaded();submit();await screen.findByText("Published version 1");
 expect(mocks.capture).toHaveBeenCalledOnce();expect(screen.getByRole("status")).not.toHaveTextContent("交接已过期");
 expect(screen.queryByRole("button",{name:"Confirm and submit"})).toBeNull();
});
it("reload only verifies the original key after an explicit click",async()=>{
 window.history.replaceState(null,"",`/capture/1688#operationKey=${key}`);render(<CaptureReceiver/>);fireEvent.click(await screen.findByRole("button",{name:"Check original operation"}));await screen.findByText("Published version 1");expect(mocks.message).not.toHaveBeenCalled();expect(mocks.capture).not.toHaveBeenCalled();expect(mocks.read.mock.calls[0].slice(0,2)).toEqual([{userId:"actor",organizationId:"org-B"},key]);
});
it.each(["org","member","revoked"])("refuses a fresh %s drift before dispatch",async mode=>{
 render(<CaptureReceiver/>);await loaded();mocks.retry.mockResolvedValue(mode==="revoked"?null:fresh(mode==="org"?"org-C":"org-B",mode==="member"?"other":"member"));
 submit();await screen.findByText(/IDENTITY_CONTEXT_CHANGED/);expect(mocks.capture).not.toHaveBeenCalled();
});
it.each(["ACQUISITION_CAPACITY","DEADLINE_EXCEEDED"])("retains the same local intent after %s pre-admission rejection",async code=>{
 mocks.capture.mockRejectedValue(new WorkbenchContextError(429,code,"",[]));render(<CaptureReceiver/>);await loaded();submit();await screen.findByText(code==="ACQUISITION_CAPACITY"?/capacity is full/:/before the deadline/);
 expect(screen.getByRole("button",{name:"Confirm and submit"})).toBeEnabled();expect(mocks.message).toHaveBeenCalledWith(expect.objectContaining({type:"capture.status",phase:"rejected",code}));
});
it("a lost response is unknown and manual verification never submits again",async()=>{
 mocks.capture.mockRejectedValue(new WorkbenchContextError(503,"OUTCOME_UNKNOWN","",[]));render(<CaptureReceiver/>);await loaded();submit();await screen.findByText(/Outcome is unknown/);fireEvent.click(screen.getByRole("button",{name:"Check original operation"}));await screen.findByText("Published version 1");expect(mocks.capture).toHaveBeenCalledOnce();expect(mocks.read).toHaveBeenCalledOnce();
});
it("context switching immediately hides source content and ignores a late saved receipt",async()=>{
 let finish!:(v:typeof receipt)=>void;mocks.capture.mockReturnValue(new Promise(r=>{finish=r;}));const view=render(<CaptureReceiver/>);await loaded();submit();await waitFor(()=>expect(mocks.capture).toHaveBeenCalledOnce());
 mocks.context={...mocks.context,isSwitching:true};view.rerender(<CaptureReceiver/>);expect(screen.queryByText("Browser fixture")).toBeNull();await act(async()=>finish(receipt));expect(screen.queryByText("Published version 1")).toBeNull();expect(mocks.capture.mock.calls[0][1].aborted).toBe(true);expect(document.querySelector("[data-batch-submit-result]")).toHaveAttribute("data-batch-submit-result","");
});
it("discards a late body after the effective enterprise changes",async()=>{
 let finish!:(v:unknown)=>void;mocks.payload.mockReturnValue(new Promise(r=>{finish=r;}));const view=render(<CaptureReceiver/>);await waitFor(()=>expect(mocks.payload).toHaveBeenCalledOnce());mocks.context={...mocks.context,isSwitching:true};view.rerender(<CaptureReceiver/>);await act(async()=>finish(browserCaptureFixture()));expect(screen.queryByText("Browser fixture")).toBeNull();expect(mocks.capture).not.toHaveBeenCalled();
});
it("renders source text literally and preserves the executor's one real confirm control",async()=>{
 const p=browserCaptureFixture();p.evidence.title='<img src=x onerror="alert(1)">';mocks.payload.mockResolvedValue(p);render(<CaptureReceiver/>);await screen.findByText(p.evidence.title!);expect(document.querySelector("img")).toBeNull();expect(document.querySelectorAll("[data-batch-confirm-submit]")).toHaveLength(1);expect(document.querySelector("[data-batch-scope-actor]")!.textContent).toBe("actor");
});
