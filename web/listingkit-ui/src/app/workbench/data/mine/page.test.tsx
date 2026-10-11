import {beforeEach,expect,it,vi} from "vitest";
vi.mock("next/server",()=>({connection:vi.fn()}));
vi.mock("next/navigation",()=>({notFound:()=>{throw new Error("not found")}}));
vi.mock("@/components/workbench/collections/collection-page",()=>({CollectionPage:()=>null}));
import Page from "./page";
beforeEach(()=>{process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED="true"});
it("passes the existing item deep link to the scoped collection reader",async()=>{
 const id="550e8400-e29b-41d4-a716-446655440002";
 const result=await Page({searchParams:Promise.resolve({item:id})});
 expect(result.props.initialItemId).toBe(id);
});
it.each(["bad",["550e8400-e29b-41d4-a716-446655440002"]])("rejects an invalid or repeated item selector (%s)",async item=>{
 const result=await Page({searchParams:Promise.resolve({item})});
 expect(result.props.initialItemId).toBeUndefined();
});
