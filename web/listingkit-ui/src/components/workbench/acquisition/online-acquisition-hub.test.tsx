import {useState} from "react";
import {cleanup,fireEvent,render,screen} from "@testing-library/react";
import {afterEach,expect,it,vi} from "vitest";
import {OnlineAcquisitionHub} from "./online-acquisition-hub";

const calls=vi.hoisted(()=>({acquisition:vi.fn(),amazon:vi.fn()}));
vi.mock("./acquisition-page",()=>({AcquisitionPage:()=>{calls.acquisition();const [value,setValue]=useState("");return <label>1688 草稿<input aria-label="1688 草稿" value={value} onChange={e=>setValue(e.target.value)}/></label>}}));
vi.mock("../data-services/data-services-page",()=>({DataServicesPage:()=>{calls.amazon();const [value,setValue]=useState("");return <label>Amazon 草稿<input aria-label="Amazon 草稿" value={value} onChange={e=>setValue(e.target.value)}/></label>}}));
afterEach(()=>{cleanup();vi.clearAllMocks()});

it("switches within one page while retaining each existing flow and its draft",()=>{
 render(<OnlineAcquisitionHub has1688 hasAmazon initial="1688"/>);
 expect(screen.getByRole("navigation",{name:"采集平台"})).toBeVisible();
 expect(screen.getByRole("button",{name:"1688"})).toHaveAttribute("aria-pressed","true");
 expect(calls.amazon).not.toHaveBeenCalled();
 fireEvent.change(screen.getByLabelText("1688 草稿"),{target:{value:"待采集的1688链接"}});
 fireEvent.click(screen.getByRole("button",{name:"Amazon"}));
 expect(screen.getByRole("button",{name:"Amazon"})).toHaveAttribute("aria-pressed","true");
 fireEvent.change(screen.getByLabelText("Amazon 草稿"),{target:{value:"待采集的ASIN"}});
 fireEvent.click(screen.getByRole("button",{name:"1688"}));
 expect(screen.getByLabelText("1688 草稿")).toHaveValue("待采集的1688链接");
 fireEvent.click(screen.getByRole("button",{name:"Amazon"}));
 expect(screen.getByLabelText("Amazon 草稿")).toHaveValue("待采集的ASIN");
});

it.each([
 {has1688:true,hasAmazon:false,initial:"1688" as const,visible:"1688",hidden:"Amazon"},
 {has1688:false,hasAmazon:true,initial:"amazon" as const,visible:"Amazon",hidden:"1688"},
])("does not expose an unavailable platform",({visible,hidden,...props})=>{
 render(<OnlineAcquisitionHub {...props}/>);
 expect(screen.getByRole("button",{name:visible})).toBeEnabled();
 expect(screen.queryByRole("button",{name:hidden})).not.toBeInTheDocument();
});
