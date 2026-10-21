import {afterEach,expect,it,vi} from "vitest";
import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {ImagePlanRoutePicker} from "./image-plan-route-picker";
const scope={userId:"actor",organizationId:"org",kind:"acquisition" as const,contextId:"11111111-1111-4111-8111-111111111111"};
const route={BindingID:"image-vip",RevisionID:"route-vip"};
const response={contextKind:scope.kind,contextId:scope.contextId,options:[{model:"qwen-image-3.0-pro",quality:"auto",resolution:"1024x1024",points:1},{route,model:"gpt-image-2-vip",quality:"medium",resolution:"1024x1024",points:2}]};
afterEach(()=>{cleanup();vi.unstubAllGlobals();document.documentElement.className=""});
it.each(["light","dark"])("defaults to enterprise route and chooses the exact published VIP in %s",async theme=>{
 document.documentElement.className=theme;const change=vi.fn(),validity=vi.fn();vi.stubGlobal("fetch",vi.fn(async()=>Response.json(response)));
 const view=render(<ImagePlanRoutePicker scope={scope} value={null} disabled={false} onChange={change} onValidity={validity}/>);
 const picker=await screen.findByRole("combobox",{name:"本次计划使用的图片模型"});
 await screen.findByRole("option",{name:"GPT Image VIP · 2 点/张"});expect(picker).toHaveValue("default");
 fireEvent.change(picker,{target:{value:JSON.stringify(route)}});expect(change).toHaveBeenLastCalledWith(route);
 view.rerender(<ImagePlanRoutePicker scope={scope} value={route} disabled={false} onChange={change} onValidity={validity}/>);
 expect(await screen.findByText(/gpt-image-2-vip · medium · 原始输出 1024×1024/)).toBeVisible();expect(validity).toHaveBeenLastCalledWith(true);
});
it("keeps a restored unavailable exact version and blocks it instead of changing model",async()=>{
 vi.stubGlobal("fetch",vi.fn(async()=>Response.json({...response,options:[response.options[0]]})));const change=vi.fn(),validity=vi.fn();
 render(<ImagePlanRoutePicker scope={scope} value={route} disabled={false} onChange={change} onValidity={validity}/>);
 await waitFor(()=>expect(validity).toHaveBeenLastCalledWith(false));expect(change).not.toHaveBeenCalled();expect(screen.getByRole("combobox")).toHaveValue(JSON.stringify(route));
});
it("preserves selection and forbids editing while the original outcome is unknown",async()=>{
 vi.stubGlobal("fetch",vi.fn(async()=>{throw Error("unavailable")}));const change=vi.fn();
 render(<ImagePlanRoutePicker scope={scope} value={route} disabled onChange={change} onValidity={vi.fn()}/>);
 await screen.findByText(/图片模型选项读取失败/);expect(screen.getByRole("combobox")).toBeDisabled();expect(screen.getByRole("button")).toBeDisabled();expect(change).not.toHaveBeenCalled();
});
