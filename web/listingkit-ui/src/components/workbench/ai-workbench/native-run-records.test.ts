import {expect,it,vi} from "vitest";
import {configurationRequest} from "@/lib/api/agent-configuration";
import {nativeRunDestination,readNativeRunPage} from "./native-run-records";
import type {CatalogEntry} from "@/lib/contracts/agent-configuration";
vi.mock("@/lib/api/agent-configuration",()=>({configurationRequest:vi.fn()}));
const scope={userId:"actor",organizationId:"org"},id="550e8400-e29b-41d4-a716-446655440001",time="2026-10-18T00:00:00Z";
it.each([
 ["product.title.agent","title-config-v1","acquisition","shein"],
 ["product.content.agent","content-config-v1","own_product","product"],
 ["listing.materials.agent","listing-materials-config-v1","listing-target-record","shein"],
 ["product.selection.agent","product-selection-config-v1",undefined,"product"],
 ["shop.operations-diagnosis.agent","shop-operations-diagnosis-config-v1",undefined,"business"],
 ["product.image.agent","image-config-v1","supply","product"],
 ["pod.design.agent","pod-artwork-config-v1","pod-artwork-v1","sds"],
] as const)("uses the original %s read contract",async(agentId,parameterSchema,contextKind,targetPlatform)=>{
 const image=parameterSchema==="image-config-v1"||parameterSchema==="pod-artwork-config-v1";
 const row=image?{runId:id,contextKind,contextId:id,targetPlatform,status:"completed",createdAt:time}:{runId:id,operationId:id,requestKey:id,productKey:contextKind?"product-key":"",...(contextKind?{contextKind}:{}),targetPlatform,phase:"running",revision:"1",startedAt:time};
 vi.mocked(configurationRequest).mockImplementation(async(_scope,_path,schema)=>schema.parse({items:[row],nextCursor:""}));
 const entry={agent:{agentId},name:agentId,parameterSchema} as CatalogEntry;
 const page=await readNativeRunPage(scope,entry,"",new AbortController().signal);
 expect(page.items).toHaveLength(1);expect(page.items[0].key).toBe(`${agentId}:${id}`);
 expect(page.items[0].state).toBe(image?"COMPLETED":"RUNNING");
 expect(page.items[0].contextKind).toBe(contextKind??(agentId==="product.selection.agent"?"selection":"diagnosis"));
});
it.each(["market","sds_template","amazon_data","collection","future_source"])("never invents an acquisition route for %s",kind=>{
 expect(nativeRunDestination(scope,"product.content.agent",kind,id,id,"product")).toEqual({href:"/workbench/agents/mine/product.content.agent",direct:false});
});
it("preserves the exact own-product actor/org/request instead of converting its identity",()=>{
 const destination=nativeRunDestination(scope,"product.content.agent","own_product",id,id,"product");
 const url=new URL(destination.href,"https://test.invalid");expect(url.pathname).toBe(`/workbench/data/mine/own/${id}/optimize`);
 expect(Object.fromEntries(url.searchParams)).toEqual({agent_actor:"actor",agent_org:"org",agent_key:id});
});
