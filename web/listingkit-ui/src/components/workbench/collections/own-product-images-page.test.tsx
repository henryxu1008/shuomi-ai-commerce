import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {cleanup,render,screen,waitFor} from "@testing-library/react";
import {OwnProductImagesPage} from "./own-product-images-page";
const f=vi.hoisted(()=>({context:{user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.supply.read","listingkit.image_agent.read","listingkit.image_agent.write"]},read:vi.fn(),panel:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>f.context}));
vi.mock("@/lib/api/supply-chain",()=>({readSupplySource:f.read}));
vi.mock("../acquisition/product-image-set-panel",()=>({ProductImageSetPanel:(props:unknown)=>{f.panel(props);return <p>原图片面板</p>}}));
const props={itemId:"item",sourceId:"source",preparationId:"preparation",launch:{userId:"actor",organizationId:"org",target:"product-image" as const,selection:{skillId:"11111111-1111-4111-8111-111111111111",version:2,digest:"a".repeat(64)}}};
beforeEach(()=>{vi.clearAllMocks();f.context.effectiveOrganization.id="org";f.context.permissions=["workbench.supply.read","listingkit.image_agent.read","listingkit.image_agent.write"];f.read.mockResolvedValue({source:{id:"source",preparationId:"preparation",collectionItemId:"item",source:{kind:"own",version:"1"}},product:{title:"原商品"}})});
afterEach(cleanup);
it("opens the same supply image consumer with exact version after source identity validation",async()=>{
 render(<OwnProductImagesPage {...props}/>);await screen.findByText("原图片面板");expect(f.panel).toHaveBeenLastCalledWith({kind:"supply",contextId:"source",translationItemId:"item",initialSkill:props.launch.selection});expect(f.read).toHaveBeenCalledWith({userId:"actor",organizationId:"org"},"source",expect.any(AbortSignal));
});
it.each(["id","preparationId","collectionItemId","kind"])("does not expose image operations for mismatched %s",async field=>{
 f.read.mockResolvedValue({source:{id:field==="id"?"other":"source",preparationId:field==="preparationId"?"other":"preparation",collectionItemId:field==="collectionItemId"?"other":"item",source:{kind:field==="kind"?"acquisition":"own"}},product:{title:"foreign"}});
 render(<OwnProductImagesPage {...props}/>);await screen.findByText("商品图片来源无法读取或与所选商品不符");expect(f.panel).not.toHaveBeenCalled();
});
it("rejects enterprise drift before any source read and discards late old-enterprise display",async()=>{
 let resolve!:(value:unknown)=>void;f.read.mockImplementationOnce(()=>new Promise(r=>{resolve=r}));const view=render(<OwnProductImagesPage {...props}/>);f.context.effectiveOrganization.id="other";view.rerender(<OwnProductImagesPage {...props}/>);resolve({source:{id:"source",preparationId:"preparation",collectionItemId:"item",source:{kind:"own",version:"1"}},product:{title:"旧企业"}});await waitFor(()=>expect(screen.getByText("技能入口已失效或身份、企业已变化")).toBeVisible());expect(f.panel).not.toHaveBeenCalled();expect(screen.queryByText("旧企业")).not.toBeInTheDocument();
});
