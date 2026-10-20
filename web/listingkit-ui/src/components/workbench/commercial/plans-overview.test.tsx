import {cleanup,render,screen,within} from "@testing-library/react";
import {afterEach,expect,it} from "vitest";
import {PlansOverview} from "./plans-overview";
import {commercialOverviewFixture} from "@/test/fixtures/commercial-overview";
afterEach(cleanup);
it("counts allocated free data in the enterprise usable balance",()=>{
 const data=commercialOverviewFixture();
 if(data.resources.state!=="available")throw Error("fixture unavailable");
 const balance=data.resources.value.resources.find(r=>r.resource_type==="data_row")!;
 Object.assign(balance,{state:"recorded",available:"0",allocated:"100",reserved:"15",consumed:"0"});
 render(<PlansOverview data={data}/>);
 const label=screen.getByText("数据余额（条）");
 expect(within(label.closest("article")??label.parentElement!).getByText("100")).toBeInTheDocument();
});
