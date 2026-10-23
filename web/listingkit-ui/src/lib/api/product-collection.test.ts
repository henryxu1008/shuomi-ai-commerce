import {afterEach,expect,it,vi} from "vitest";
import {readCollectionItem} from "./product-collection";

afterEach(()=>vi.unstubAllGlobals());
it("reads complete fixed-version detail and quotes across the browser response boundary",async()=>{
 const id="550e8400-e29b-41d4-a716-446655440000";
 const detail={item:{id,batchId:id,source:{productKey:"product",publicationId:id,version:"1",kind:"acquisition"},revision:1,createdAt:"2026-10-14T00:00:00Z"},product:{description:"x".repeat(1024*1024),attributes:Array.from({length:128},(_,i)=>({name:String(i),value:"x".repeat(8192)}))},sourceQuotes:{state:"available",publicationId:id,version:"1",prices:[]}};
 const fetcher=vi.fn().mockResolvedValueOnce(Response.json(detail));vi.stubGlobal("fetch",fetcher);
 expect(await readCollectionItem({userId:"actor",organizationId:"org"},id)).toEqual(detail);
 expect(fetcher.mock.calls[0]?.[1].method).toBe("GET");
 fetcher.mockResolvedValueOnce(Response.json({...detail,product:{description:"x".repeat(3*1024*1024)}}));
 await expect(readCollectionItem({userId:"actor",organizationId:"org"},id)).rejects.toMatchObject({code:"DEPENDENCY_UNAVAILABLE"});
});
