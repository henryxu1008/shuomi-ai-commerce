import {render,screen,within} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {expect,it,vi} from "vitest";
import {WorkbenchViewNav} from "./workbench-view-nav";

it("keeps route links and current state accessible",()=>{
 render(<WorkbenchViewNav label="会话视图"><a href="/new" aria-current="page">新建会话</a><a href="/recent">最近会话</a></WorkbenchViewNav>);
 const nav=screen.getByRole("navigation",{name:"会话视图"});
 expect(within(nav).getByRole("link",{name:"新建会话"})).toHaveAttribute("aria-current","page");
 expect(within(nav).getByRole("link",{name:"最近会话"})).toHaveAttribute("href","/recent");
});
it("keeps local view buttons keyboard operable and disabled items inactive",async()=>{
 const user=userEvent.setup(),change=vi.fn();render(<WorkbenchViewNav label="项目内容" group><button type="button" aria-pressed="true">概览</button><button type="button" onClick={change}>任务</button><button type="button" disabled onClick={change}>不可用</button></WorkbenchViewNav>);
 const group=screen.getByRole("group",{name:"项目内容"});expect(within(group).getByRole("button",{name:"概览"})).toHaveAttribute("aria-pressed","true");
 await user.tab();await user.tab();await user.keyboard("{Enter}");expect(change).toHaveBeenCalledTimes(1);expect(within(group).getByRole("button",{name:"不可用"})).toBeDisabled();
});
