import {expect,it} from "vitest";
import {collectionPath,parseCollectionResponse} from "./product-collection";
const id="550e8400-e29b-41d4-a716-446655440002";
const item={id,batchId:id,revision:2,createdAt:"2026-10-14T00:00:00Z",source:{kind:"own",productKey:"own-test",publicationId:id,version:"3"}};
it("allows only exact read-only history routes with canonical version strings",()=>{
 expect(collectionPath("GET",["collections","items",id,"history"])).toBe("history");
 expect(collectionPath("GET",["collections","items",id,"history","9007199254740993"])).toBe("history-detail");
 for(const version of ["0","01","9223372036854775808","all"]){expect(collectionPath("GET",["collections","items",id,"history",version])).toBeNull()}
 expect(collectionPath("POST",["collections","items",id,"history"])).toBeNull();
});
it("binds history projections to exact item/version and strips internal evidence",()=>{
 const entry={version:"1",publicationId:id,savedAt:"2026-10-14T00:00:00Z",title:"旧资料",envelope:"private"};
 expect(parseCollectionResponse("history-detail" as never,{item,version:entry,product:{title:"旧资料"}},`${id}:1`)).toEqual({item,version:{version:"1",publicationId:id,savedAt:entry.savedAt,title:"旧资料"},product:{title:"旧资料"}});
 expect(parseCollectionResponse("history-detail" as never,{item,version:entry,product:{}},`${id}:3`)).toBeNull();
 expect(parseCollectionResponse("history-detail",{item,version:{...entry,version:"all"},product:{title:"旧资料"}},`${id}:1`)).toBeNull();
 expect(parseCollectionResponse("history",{item:{...item,source:{...item.source,version:"invalid"}},versions:[entry]},id)).toBeNull();
});
