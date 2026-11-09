import {cleanup,render,screen} from "@testing-library/react";
import {afterEach,expect,it,vi} from "vitest";
import {ColorSplitLinks} from "./color-split-panel";
import type {SupplySourceDetail} from "@/lib/contracts/supply-chain";
const state=vi.hoisted(()=>({list:vi.fn()}));
vi.mock("next/navigation",()=>({useRouter:()=>({replace:vi.fn()})}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),listColorSplitFamilies:state.list}));
afterEach(cleanup);
it("reopens the fixed parent and candidate so the family retains its original store context",async()=>{
 const source="11111111-1111-4111-8111-111111111111",parent="22222222-2222-4222-8222-222222222222",family="33333333-3333-4333-8333-333333333333";
 state.list.mockResolvedValue({items:[{familyId:family,records:[{input:{colorPartition:{baseRecordId:parent,completionDraft:{revision:1}}}},{input:{}}]}]});
 render(<ColorSplitLinks scope={{organizationId:"org",userId:"actor"}} storeId={source} detail={{source:{id:source,preparationId:source,collectionItemId:source}} as SupplySourceDetail}/>);
 const link=await screen.findByRole("link",{name:"本地拆分草稿 2 份"});
 const query=new URL(link.getAttribute("href")!,"http://fixture.test").searchParams;
 expect(query.get("target")).toBe(parent);expect(query.get("revision")).toBe("1");expect(query.get("family")).toBe(family);
});
