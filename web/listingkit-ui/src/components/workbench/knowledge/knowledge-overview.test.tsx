import {cleanup,render,screen,within} from "@testing-library/react";
import {afterEach,expect,it} from "vitest";
import {KnowledgeOverview} from "./knowledge-overview";

afterEach(cleanup);
it.each([true,false])("shows the shared navigation on the knowledge home when enterprise availability is %s", enterpriseAvailable=>{
 render(<KnowledgeOverview enterpriseAvailable={enterpriseAvailable}/>);
 const nav=screen.getByRole("navigation",{name:"知识库分类"});
 expect(within(nav).getByRole("link",{name:"知识库首页"})).toHaveAttribute("aria-current","page");
 expect(within(nav).getByRole("link",{name:"切换到官方知识库"})).toHaveAttribute("href","/workbench/ai/knowledge/official");
 if(enterpriseAvailable)expect(within(nav).getByRole("link",{name:"切换到我的知识库"})).toHaveAttribute("href","/workbench/ai/knowledge/mine");
 else expect(within(nav).getByText("我的知识库暂未开放")).toHaveAttribute("aria-disabled","true");
});
