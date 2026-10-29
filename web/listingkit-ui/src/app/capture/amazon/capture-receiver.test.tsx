import { act,cleanup,fireEvent,render,screen,waitFor } from "@testing-library/react";
import { beforeEach,afterEach,expect,it,vi } from "vitest";
import { readFileSync } from "node:fs";
import { AmazonCaptureReceiver } from "./capture-receiver";
import { AmazonCaptureError } from "@/lib/api/amazon-browser-capture";
const mocks=vi.hoisted(()=>({context:{} as Record<string,unknown>,request:vi.fn(),message:vi.fn(),payload:vi.fn(),retry:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>mocks.context}));
vi.mock("@/lib/api/amazon-browser-capture",async()=>({...await vi.importActual<object>("@/lib/api/amazon-browser-capture"),amazonCaptureRequest:mocks.request}));
const source=JSON.parse(readFileSync("../../internal/product/sourcing/testdata/amazon-capture-v2.json","utf8"));
const receipt=JSON.parse(readFileSync("../../internal/product/sourcing/testdata/amazon-capture-result-v1.json","utf8"));
const fresh=(member="amazon-member")=>({user:{id:"amazon-actor"},effectiveOrganizationId:"amazon-org",effectiveMemberId:member});
beforeEach(()=>{
 vi.clearAllMocks();sessionStorage.clear();window.history.replaceState(null,"",`/capture/amazon#extensionId=${"a".repeat(32)}&handoffId=11111111-1111-4111-8111-11111111111a&idempotencyKey=${receipt.key}`);
 mocks.context={user:{id:"amazon-actor"},effectiveOrganization:{id:"amazon-org",name:"Test enterprise"},effectiveMemberId:"amazon-member",retry:mocks.retry};mocks.retry.mockResolvedValue(fresh());mocks.request.mockResolvedValue(receipt);mocks.payload.mockResolvedValue(source);
 let phase="ready";
 mocks.message.mockImplementation(async m=>{
  if(m.type==="capture.send")phase="sending";
  if(m.type==="capture.status")phase=m.phase;
  if(m.type==="capture.retry")phase="bound";
  if(m.type==="capture.read")return {version:2,type:"capture.draft",phase,canSubmit:phase==="ready",idempotencyKey:receipt.key,payload:await mocks.payload()};
  return {version:2,type:"capture.metadata",phase,idempotencyKey:receipt.key};
 });
 vi.stubGlobal("chrome",{runtime:{sendMessage:(_id:string,m:unknown,cb:(v:unknown)=>void)=>{void mocks.message(m).then(cb,()=>cb({version:2,code:"DRAFT_CHANGED"}));}}});
});
afterEach(()=>{cleanup();vi.restoreAllMocks();vi.unstubAllGlobals();});
const loaded=()=>screen.findByRole("button",{name:"确认保存"});
const submit=()=>fireEvent.click(screen.getByRole("button",{name:"确认保存"}));
it("shows new and used quotes without dispatch before consent",async()=>{
 const input=structuredClone(source);input.evidence.variants[0].price={amount:"35.98",currency:"USD",condition:"new",alternative:{amount:"31.99",currency:null,condition:"used",conditionNote:"Very Good"}};mocks.payload.mockResolvedValue(input);
 render(<AmazonCaptureReceiver/>);await loaded();const row=screen.getByText(/35\.98/);expect(row).toHaveTextContent("新品");expect(row).toHaveTextContent("31.99");expect(row).toHaveTextContent("二手");expect(row).toHaveTextContent("Very Good");expect(mocks.request).not.toHaveBeenCalled();
});
it("persists exact bytes and scope before the first POST, then reload only verifies the original key",async()=>{
 const mounted=render(<AmazonCaptureReceiver/>);await loaded();expect(mocks.request).not.toHaveBeenCalled();submit();await screen.findByText(/已保存：Cotton bag/);
 const sent=mocks.request.mock.calls[0];expect(sent[2]).toContain('"captureVersion":2');expect(mocks.message.mock.calls.map(c=>c[0].type)).toEqual(["capture.meta","capture.read","capture.bind","capture.send","capture.status"]);expect(mocks.message.mock.calls[2][0].body).toBe(sent[2]);expect(mocks.message.mock.calls[2][0].scope).toEqual({userId:"amazon-actor",organizationId:"amazon-org",memberId:"amazon-member"});
 mounted.unmount();mocks.request.mockClear();render(<AmazonCaptureReceiver/>);fireEvent.click(await screen.findByRole("button",{name:"检查原操作"}));await waitFor(()=>expect(mocks.request).toHaveBeenCalledOnce());expect(mocks.request.mock.calls[0][2]).toBeUndefined();
});
it("does not require a second body owner in sessionStorage",async()=>{
 vi.spyOn(Storage.prototype,"setItem").mockImplementation(()=>{throw Error("blocked");});render(<AmazonCaptureReceiver/>);await loaded();submit();await screen.findByText(/已保存：Cotton bag/);expect(mocks.request).toHaveBeenCalledOnce();
});
it("refuses POST if the durable freeze cannot commit",async()=>{
 render(<AmazonCaptureReceiver/>);await loaded();mocks.message.mockImplementation(async()=>({version:2,code:"DRAFT_STORAGE_FULL"}));submit();await screen.findByText(/保存未发送：DRAFT_STORAGE_FULL/);expect(mocks.request).not.toHaveBeenCalled();
});
it.each(["capture.meta","capture.read","capture.bind","capture.send"])("guides original-draft recovery after %s expires without dispatching or resubmitting",async stage=>{
 const message=mocks.message.getMockImplementation()!;
 if(stage==="capture.meta"||stage==="capture.read")mocks.message.mockImplementation(async m=>m.type===stage?{version:2,code:"CAPTURE_EXPIRED"}:message(m));
 render(<AmazonCaptureReceiver/>);
 if(stage==="capture.bind"||stage==="capture.send"){
  await loaded();mocks.message.mockImplementation(async m=>m.type===stage?{version:2,code:"CAPTURE_EXPIRED"}:message(m));submit();
 }
 expect(await screen.findByRole("status")).toBeInTheDocument();
 await waitFor(()=>expect(screen.getByRole("status")).toHaveTextContent("交接已过期（CAPTURE_EXPIRED）"));
 expect(screen.getByRole("status")).toHaveTextContent("本机保全资料");expect(screen.getByRole("status")).toHaveTextContent("恢复原操作");
 expect(screen.getByRole("status")).not.toHaveTextContent("OUTCOME_UNKNOWN");
 expect(screen.queryByRole("button",{name:"确认保存"})).toBeNull();expect(screen.queryByRole("button",{name:"用原键重试同一份资料"})).toBeNull();
 expect(mocks.request).not.toHaveBeenCalled();expect(mocks.message.mock.calls.filter(c=>c[0].type==="capture.status")).toHaveLength(0);
});
it("requires fresh consent after explicit recovery with a new nonce and the same key and body",async()=>{
 const message=mocks.message.getMockImplementation()!;
 const mounted=render(<AmazonCaptureReceiver/>);await loaded();
 mocks.message.mockImplementation(async m=>m.type==="capture.bind"?{version:2,code:"CAPTURE_EXPIRED"}:message(m));submit();
 await waitFor(()=>expect(screen.getByRole("status")).toHaveTextContent("CAPTURE_EXPIRED"));
 const original=mocks.message.mock.calls.find(c=>c[0].type==="capture.bind")![0];
 mounted.unmount();mocks.message.mockImplementation(message);
 const nonce="33333333-3333-4333-8333-33333333333c";
 window.history.replaceState(null,"",`/capture/amazon#extensionId=${"a".repeat(32)}&handoffId=${nonce}&idempotencyKey=${receipt.key}`);
 render(<AmazonCaptureReceiver/>);await loaded();expect(mocks.request).not.toHaveBeenCalled();submit();await screen.findByText(/已保存：Cotton bag/);
 const restored=mocks.message.mock.calls.filter(c=>c[0].type==="capture.bind").at(-1)![0];
 expect(restored).toEqual({...original,handoffId:nonce});expect(mocks.request).toHaveBeenCalledOnce();expect(mocks.request.mock.calls[0][1]).toBe(receipt.key);expect(mocks.request.mock.calls[0][2]).toBe(original.body);
});
it("keeps the saved result when the handoff expires while reporting the receipt",async()=>{
 const message=mocks.message.getMockImplementation()!;
 mocks.message.mockImplementation(async m=>m.type==="capture.status"?{version:2,code:"CAPTURE_EXPIRED"}:message(m));
 render(<AmazonCaptureReceiver/>);await loaded();submit();await screen.findByText(/已保存：Cotton bag/);
 expect(mocks.request).toHaveBeenCalledOnce();expect(screen.getByRole("status")).not.toHaveTextContent("交接已过期");
 expect(screen.queryByRole("button",{name:"确认保存"})).toBeNull();
});
it("unknown response only allows a manual same-key same-body retry after explicit original-key 404",async()=>{
 mocks.request.mockRejectedValueOnce(new AmazonCaptureError("OUTCOME_UNKNOWN"));render(<AmazonCaptureReceiver/>);await loaded();submit();await screen.findByText(/保存结果待核实/);expect(mocks.request).toHaveBeenCalledOnce();
 mocks.request.mockRejectedValueOnce(new AmazonCaptureError("CAPTURE_NOT_FOUND"));fireEvent.click(screen.getByRole("button",{name:"检查原操作"}));await screen.findByRole("button",{name:"用原键重试同一份资料"});expect(mocks.request).toHaveBeenCalledTimes(2);expect(mocks.request.mock.calls[1][2]).toBeUndefined();
 fireEvent.click(screen.getByRole("button",{name:"用原键重试同一份资料"}));await screen.findByText(/已保存：Cotton bag/);expect(mocks.request).toHaveBeenCalledTimes(3);expect(mocks.request.mock.calls[2].slice(0,3)).toEqual(mocks.request.mock.calls[0].slice(0,3));expect(mocks.message.mock.calls.filter(c=>c[0].type==="capture.retry")).toHaveLength(1);
});
it("retains unknown-outcome semantics when the retry capability has expired after original-key verification",async()=>{
 mocks.request.mockRejectedValueOnce(new AmazonCaptureError("OUTCOME_UNKNOWN"));render(<AmazonCaptureReceiver/>);await loaded();submit();await screen.findByText(/保存结果待核实/);
 const message=mocks.message.getMockImplementation()!;
 mocks.message.mockImplementation(async m=>m.type==="capture.armRetry"?{version:2,code:"CAPTURE_EXPIRED"}:message(m));
 mocks.request.mockRejectedValueOnce(new AmazonCaptureError("CAPTURE_NOT_FOUND"));fireEvent.click(screen.getByRole("button",{name:"检查原操作"}));
 await waitFor(()=>expect(screen.getByRole("status")).toHaveTextContent("CAPTURE_EXPIRED"));
 expect(screen.getByRole("status")).toHaveTextContent("原请求仍可能完成");expect(screen.getByRole("status")).toHaveTextContent("恢复原操作");
 expect(screen.queryByRole("button",{name:"用原键重试同一份资料"})).toBeNull();expect(mocks.request).toHaveBeenCalledTimes(2);expect(mocks.request.mock.calls[1][2]).toBeUndefined();
});
it("does not offer resend for an original key without a recoverable local draft",async()=>{
 window.history.replaceState(null,"",`/capture/amazon#operationKey=${receipt.key}`);render(<AmazonCaptureReceiver/>);fireEvent.click(await screen.findByRole("button",{name:"检查原操作"}));await screen.findByText(/没有原恢复标记/);expect(mocks.request).not.toHaveBeenCalled();expect(screen.queryByRole("button",{name:"用原键重试同一份资料"})).toBeNull();
});
it("does not reuse a bound intent for a replaced member",async()=>{
 render(<AmazonCaptureReceiver/>);await loaded();mocks.retry.mockResolvedValue(fresh("new-member"));submit();await screen.findByText(/IDENTITY_CONTEXT_CHANGED/);expect(mocks.request).not.toHaveBeenCalled();
});
it("hides old source data and late success when switching identity",async()=>{
 let finish!:(v:typeof receipt)=>void;mocks.request.mockReturnValue(new Promise(r=>{finish=r;}));const view=render(<AmazonCaptureReceiver/>);await loaded();submit();await waitFor(()=>expect(mocks.request).toHaveBeenCalledOnce());mocks.context={...mocks.context,isSwitching:true};view.rerender(<AmazonCaptureReceiver/>);await act(async()=>finish(receipt));expect(screen.queryByText(/已保存：Cotton bag/)).toBeNull();expect(mocks.request.mock.calls[0][3].aborted).toBe(true);
});
