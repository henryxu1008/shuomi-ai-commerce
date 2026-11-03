import {afterEach,expect,it,vi} from "vitest";
import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {ImagePlanRoutePicker} from "./image-plan-route-picker";
const scope={userId:"actor",organizationId:"org",kind:"acquisition" as const,contextId:"11111111-1111-4111-8111-111111111111"};
const route={BindingID:"retired-plan-binding",RevisionID:"original-revision"};
const response={contextKind:scope.kind,contextId:scope.contextId,options:[{unitVersion:"ai-micropoint-v1",model:"synthetic-image",quality:"auto",resolution:"1024x1024",points:"1250000"}]};
afterEach(()=>{cleanup();vi.unstubAllGlobals();document.documentElement.className=""});
// RETIRE per-plan VIP selection; EXTRACT original binding/uncertain-request safety.
it.each(["light","dark"])("uses the enterprise model read-only with an exact reserve bound in %s",async theme=>{
 document.documentElement.className=theme;const change=vi.fn(),validity=vi.fn();vi.stubGlobal("fetch",vi.fn(async()=>Response.json(response)));
 render(<ImagePlanRoutePicker scope={scope} value={null} disabled={false} onChange={change} onValidity={validity}/>);
 const model=await screen.findByRole("textbox",{name:"企业图片模型"});await waitFor(()=>expect(model).toHaveValue("synthetic-image"));expect(model).toHaveAttribute("readonly");expect(screen.queryByRole("combobox")).not.toBeInTheDocument();expect(screen.getByText(/每张预留上界 1.25 点/)).toBeVisible();expect(change).not.toHaveBeenCalled();expect(validity).toHaveBeenLastCalledWith(true);
});
it("preserves an old exact binding without silently adopting the enterprise default",async()=>{
 vi.stubGlobal("fetch",vi.fn(async()=>Response.json(response)));const change=vi.fn(),validity=vi.fn();
 render(<ImagePlanRoutePicker scope={scope} value={route} disabled={false} onChange={change} onValidity={validity}/>);
 const reset=await screen.findByRole("button",{name:"新计划使用企业模型"});await waitFor(()=>expect(reset).toBeEnabled());expect(validity).toHaveBeenLastCalledWith(false);expect(change).not.toHaveBeenCalled();expect(screen.getByRole("status")).toHaveTextContent("original-revision");fireEvent.click(reset);expect(change).toHaveBeenCalledExactlyOnceWith(null);
});
it("cannot replace the original selection or refresh during an uncertain outcome",async()=>{
 vi.stubGlobal("fetch",vi.fn(async()=>{throw Error("unavailable")}));const change=vi.fn();
 render(<ImagePlanRoutePicker scope={scope} value={route} disabled onChange={change} onValidity={vi.fn()}/>);
 await screen.findByText(/企业图片模型读取失败/);expect(screen.getByRole("button",{name:"新计划使用企业模型"})).toBeDisabled();expect(screen.getByRole("button",{name:"重新读取图片模型"})).toBeDisabled();expect(change).not.toHaveBeenCalled();
});
it("rejects a per-plan route option or a unitless price as current enterprise readiness",async()=>{
 const validity=vi.fn();vi.stubGlobal("fetch",vi.fn(async()=>Response.json({...response,options:[{...response.options[0],route}]})));
 const view=render(<ImagePlanRoutePicker scope={scope} value={null} disabled={false} onChange={vi.fn()} onValidity={validity}/>);await waitFor(()=>expect(validity).toHaveBeenLastCalledWith(false));expect(screen.getByRole("textbox")).toHaveValue("尚未取得可用的企业模型");view.unmount();
 vi.stubGlobal("fetch",vi.fn(async()=>Response.json({...response,options:[{...response.options[0],unitVersion:undefined}]})));render(<ImagePlanRoutePicker scope={scope} value={null} disabled={false} onChange={vi.fn()} onValidity={validity}/>);await screen.findByText(/企业图片模型读取失败/);expect(validity).toHaveBeenLastCalledWith(false);
});
