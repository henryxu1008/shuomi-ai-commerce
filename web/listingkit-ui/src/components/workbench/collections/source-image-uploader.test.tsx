// @vitest-environment jsdom
import {act,cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {CollectionAPIError} from "@/lib/api/product-collection";
import {SourceImageUploader} from "./source-image-uploader";

const state=vi.hoisted(()=>({read:vi.fn(),upload:vi.fn(),context:{registerOrganizationSwitchGuard:vi.fn(()=>()=>{})}}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>state.context}));
vi.mock("@/lib/api/product-collection",async()=>({...await vi.importActual("@/lib/api/product-collection"),readSourceImage:state.read,uploadSourceImage:state.upload}));
const intent={userId:"user",organizationId:"org",hash:"00".repeat(32),bytes:3};
const image={hash:intent.hash,bytes:3,url:"https://files.example.org/original.png",mediaType:"image/png",width:2,height:2};
beforeEach(()=>{
 localStorage.clear();localStorage.setItem("shuomi_source_media_intent",JSON.stringify(intent));
 state.read.mockReset().mockResolvedValue(image);state.upload.mockReset().mockResolvedValue(image);
 vi.stubGlobal("crypto",{subtle:{digest:vi.fn().mockResolvedValue(new Uint8Array(32).buffer)}});
});
afterEach(()=>{cleanup();localStorage.clear();vi.unstubAllGlobals();});
function originalFile(){const file=new File([new Uint8Array([1,2,3])],"original.png",{type:"image/png"});Object.defineProperty(file,"arrayBuffer",{value:async()=>new Uint8Array([1,2,3]).buffer});return file;}
it("keeps a verified file pending until the same Upload confirms the original operation",async()=>{
 const onImage=vi.fn(),onMedia=vi.fn(),onBlocked=vi.fn();
 render(<SourceImageUploader scope={intent} disabled={false} onImage={onImage} onMedia={onMedia} onBlocked={onBlocked}/>);
 await userEvent.click(screen.getByRole("button",{name:"核实原图片"}));
 await waitFor(()=>expect(state.read).toHaveBeenCalledOnce());
 expect(localStorage.getItem("shuomi_source_media_intent")).toBe(JSON.stringify(intent));
 expect(onImage).not.toHaveBeenCalled();expect(onMedia).not.toHaveBeenCalled();expect(state.upload).not.toHaveBeenCalled();
 expect(screen.getByLabelText("图片文件")).not.toBeDisabled();
 fireEvent.change(screen.getByLabelText("图片文件"),{target:{files:[originalFile()]}});
 await waitFor(()=>expect(screen.getByRole("button",{name:"重试原文件"})).not.toBeDisabled());
 await userEvent.click(screen.getByRole("button",{name:"重试原文件"}));
 await waitFor(()=>expect(onImage).toHaveBeenCalledExactlyOnceWith(image.url));
 expect(state.upload).toHaveBeenCalledWith(intent,expect.any(ArrayBuffer),expect.any(AbortSignal));
 expect(onMedia).toHaveBeenCalledExactlyOnceWith(image);expect(localStorage.getItem("shuomi_source_media_intent")).toBeNull();
 await waitFor(()=>expect(onBlocked).toHaveBeenLastCalledWith(false));
});
it("retains the original intent if same-file replay is still unknown",async()=>{
 const onImage=vi.fn();state.upload.mockRejectedValue(new CollectionAPIError("OUTCOME_UNKNOWN",503));
 render(<SourceImageUploader scope={intent} disabled={false} onImage={onImage} onBlocked={()=>{}}/>);
 await userEvent.click(screen.getByRole("button",{name:"核实原图片"}));
 await waitFor(()=>expect(state.read).toHaveBeenCalledOnce());
 fireEvent.change(screen.getByLabelText("图片文件"),{target:{files:[originalFile()]}});
 await waitFor(()=>expect(screen.getByRole("button",{name:"重试原文件"})).not.toBeDisabled());
 await userEvent.click(screen.getByRole("button",{name:"重试原文件"}));
 await waitFor(()=>expect(state.upload).toHaveBeenCalledOnce());
 expect(localStorage.getItem("shuomi_source_media_intent")).toBe(JSON.stringify(intent));expect(onImage).not.toHaveBeenCalled();
});

it("retains a late confirmation for recovery when write permission changes in flight",async()=>{
 localStorage.clear();let finish!:(value:typeof image)=>void;
 state.upload.mockReturnValue(new Promise(resolve=>{finish=resolve}));
 const onImage=vi.fn(),onBlocked=vi.fn();
 const view=render(<SourceImageUploader scope={intent} disabled={false} onImage={onImage} onBlocked={onBlocked}/>);
 fireEvent.change(screen.getByLabelText("图片文件"),{target:{files:[originalFile()]}});
 await waitFor(()=>expect(screen.getByRole("button",{name:"上传并添加图片链接"})).toBeEnabled());
 fireEvent.click(screen.getByRole("button",{name:"上传并添加图片链接"}));await waitFor(()=>expect(state.upload).toHaveBeenCalledOnce());
 view.rerender(<SourceImageUploader scope={intent} disabled onImage={onImage} onBlocked={onBlocked}/>);
 finish(image);await screen.findByText(/身份或权限已变化/);
 expect(onImage).not.toHaveBeenCalled();expect(localStorage.getItem("shuomi_source_media_intent")).not.toBeNull();
});

it("does not consume a late upload after navigation and retains its same-file intent",async()=>{
 localStorage.clear();let finish!:(value:typeof image)=>void;state.upload.mockReturnValue(new Promise(resolve=>{finish=resolve}));
 const onImage=vi.fn();const view=render(<SourceImageUploader scope={intent} disabled={false} onImage={onImage} onBlocked={()=>{}}/>);
 fireEvent.change(screen.getByLabelText("图片文件"),{target:{files:[originalFile()]}});
 await waitFor(()=>expect(screen.getByRole("button",{name:"上传并添加图片链接"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"上传并添加图片链接"}));
 await waitFor(()=>expect(state.upload).toHaveBeenCalledOnce());view.unmount();await act(async()=>finish(image));
 expect(onImage).not.toHaveBeenCalled();expect(localStorage.getItem("shuomi_source_media_intent")).not.toBeNull();
});
