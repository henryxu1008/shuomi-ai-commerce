import { beforeEach,afterEach,expect,it,vi } from "vitest";
import { render,screen,waitFor,cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { readFileSync } from "node:fs";
import { AmazonCaptureReceiver } from "./capture-receiver";
const mocks=vi.hoisted(()=>({context:{} as Record<string,unknown>,entry:vi.fn(),handoff:vi.fn(),request:vi.fn(),notify:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>mocks.context}));
vi.mock("../1688/capture-handoff",()=>({parseCaptureEntry:mocks.entry,readCaptureHandoff:mocks.handoff,notifyCaptureStatus:mocks.notify}));
vi.mock("@/lib/api/amazon-browser-capture",async()=>{const actual=await vi.importActual<object>("@/lib/api/amazon-browser-capture");return {...actual,amazonCaptureRequest:mocks.request};});
const source=JSON.parse(readFileSync("../../internal/product/sourcing/testdata/amazon-capture-v2.json","utf8"));
const receipt=JSON.parse(readFileSync("../../internal/product/sourcing/testdata/amazon-capture-result-v1.json","utf8"));
beforeEach(()=>{sessionStorage.clear();mocks.entry.mockReturnValue({kind:"handoff",extensionId:"a".repeat(32),handoffId:"11111111-1111-4111-8111-11111111111a",key:receipt.key});mocks.handoff.mockResolvedValue(source);mocks.request.mockResolvedValue(receipt);mocks.context={user:{id:"amazon-actor"},effectiveOrganization:{id:"amazon-org",name:"Test enterprise"},effectiveMemberId:"amazon-member",retry:vi.fn(async()=>({user:{id:"amazon-actor"},effectiveOrganizationId:"amazon-org",effectiveMemberId:"amazon-member"}))};});
afterEach(()=>{cleanup();vi.restoreAllMocks();vi.clearAllMocks();});
it("requires explicit consent and refresh only checks the retained original key",async()=>{
 const user=userEvent.setup();const mounted=render(<AmazonCaptureReceiver/>);await screen.findByRole("button",{name:"确认保存"});expect(mocks.request).not.toHaveBeenCalled();
 await user.click(screen.getByRole("button",{name:"确认保存"}));await waitFor(()=>expect(mocks.request).toHaveBeenCalledOnce());await screen.findByText(/已保存：Cotton bag/);
 expect(mocks.request.mock.calls[0][2]).toContain('"captureVersion":2');mounted.unmount();mocks.request.mockClear();mocks.entry.mockReturnValue({kind:"recovery",key:receipt.key});
 render(<AmazonCaptureReceiver/>);await screen.findByRole("button",{name:"检查原操作"});expect(mocks.request).not.toHaveBeenCalled();await user.click(screen.getByRole("button",{name:"检查原操作"}));await waitFor(()=>expect(mocks.request).toHaveBeenCalledOnce());expect(mocks.request.mock.calls[0][2]).toBeUndefined();
});
it("refuses dispatch when recovery marker cannot be retained",async()=>{
 vi.spyOn(Storage.prototype,"setItem").mockImplementation(()=>{throw new Error("fixture storage denied");});render(<AmazonCaptureReceiver/>);await userEvent.click(await screen.findByRole("button",{name:"确认保存"}));await waitFor(()=>expect(screen.getByRole("status").textContent).toContain("保存未发送"));expect(mocks.request).not.toHaveBeenCalled();
});
it("does not carry an original marker into a re-created member",async()=>{
 mocks.entry.mockReturnValue({kind:"recovery",key:receipt.key});mocks.context.effectiveMemberId="new-member";mocks.context.retry=vi.fn(async()=>({user:{id:"amazon-actor"},effectiveOrganizationId:"amazon-org",effectiveMemberId:"new-member"}));render(<AmazonCaptureReceiver/>);await userEvent.click(await screen.findByRole("button",{name:"检查原操作"}));await waitFor(()=>expect(screen.getByRole("status").textContent).toContain("没有原恢复标记"));expect(mocks.request).not.toHaveBeenCalled();
});
