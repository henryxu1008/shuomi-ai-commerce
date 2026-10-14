import { describe, expect, it } from "vitest";
import { collectionCommandSchema,parseCollectionResponse } from "./product-collection";

describe("data-service original collection references", () => {
  const id = "0511e1d2-b555-4970-b323-3b629a901901";
  const createdAt = "2026-10-09T12:00:00Z";
  it("binds a revision to the exact original tuple and rejects fields from other commands",()=>{
    const command={action:"revise_product",itemId:id,expectedRevision:3,expectedSourceVersion:"9007199254740993",expectedPublicationId:id,product:{title:"新版",description:"",images:[]}};
    expect(collectionCommandSchema.safeParse(command).success).toBe(true);
    for(const value of ["01","0","+1","9223372036854775808",1])expect(collectionCommandSchema.safeParse({...command,expectedSourceVersion:value}).success).toBe(false);
    expect(collectionCommandSchema.safeParse({...command,batchId:id}).success).toBe(false);
  });
  for (const kind of ["amazon_data", "custom_dataset"]) {
    it(`reads ${kind} without masquerading as another source`, () => {
      expect(parseCollectionResponse("batches", { items: [{ id, name: "Data delivery", kind, revision: 1, count: 1, supplyTransferSupported:false, createdAt }], total: 1 })).toMatchObject({items:[{kind,supplyTransferSupported:false}],total:1});
      expect(parseCollectionResponse("items", { items: [{ id, batchId: id, revision: 1, createdAt, source: { productKey: "fixture", publicationId: id, version: "1", operationId: id, kind } }], total: 1 })).not.toBeNull();
    });
  }
  it("keeps unknown producer kinds rejected", () => {
    expect(parseCollectionResponse("batches", { items: [{ id, name: "Unknown", kind: "legacy_amazon", revision: 1, count: 1, supplyTransferSupported:false, createdAt }], total: 1 })).toBeNull();
  });
  it("rejects batches missing the current source eligibility projection",()=>{
    expect(parseCollectionResponse("batches",{items:[{id,name:"Incomplete",kind:"manual",revision:1,count:1,createdAt}],total:1})).toBeNull();
  });
});
