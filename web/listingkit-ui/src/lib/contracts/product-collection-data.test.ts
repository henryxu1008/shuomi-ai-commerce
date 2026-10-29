import { describe, expect, it } from "vitest";
import { collectionCommandSchema,collectionDetailSchema,parseCollectionResponse,parseSourcePriceFact } from "./product-collection";
it('preserves new and used quote evidence and rejects nested duplicate fields',()=>{
 const fact='{"amount":"35.98","currency":"USD","condition":"new","alternative":{"amount":"31.99","currency":null,"condition":"used","conditionNote":"Very Good"}}';
 expect(parseSourcePriceFact(fact)?.alternative?.conditionNote).toBe('Very Good');
 expect(parseSourcePriceFact(fact.replace('"condition":"used"','"condition":"used","condition":"used"'))).toBeNull();
});

describe("data-service original collection references", () => {
  const id = "0511e1d2-b555-4970-b323-3b629a901901";
  const createdAt = "2026-10-09T12:00:00Z";
  it("does not render malformed or duplicate stored variant price facts as valid amounts",()=>{
    for(const raw of ['{"amount":"1","amount":"2","currency":null}','{"amount":"1","currency":null,"stock":2}','{"amount":"1","currency":"$"}','{"amount":"1","currency":null,"minQuantity":"0"}'])expect(parseSourcePriceFact(raw)).toBeNull();
    expect(parseSourcePriceFact('{"amount":"27.00000001","currency":null}')).toEqual({amount:"27.00000001",currency:null});
  });
  it("preserves fixed-version source quotes and refuses mismatched or guessed fields",()=>{
    const input={item:{id,batchId:id,source:{productKey:"product",publicationId:id,version:"1",kind:"acquisition"},revision:1,createdAt},product:{title:"known"},sourceQuotes:{state:"available",publicationId:id,version:"1",prices:[{amount:"27.00000001",currency:null,minQuantity:null}]}};
    expect(collectionDetailSchema.parse(input)).toHaveProperty("sourceQuotes",input.sourceQuotes);
    for(const quotes of [{...input.sourceQuotes,version:"2"},{...input.sourceQuotes,publicationId:"foreign"},{...input.sourceQuotes,prices:[{amount:"27",currency:"$",minQuantity:null}]},{...input.sourceQuotes,prices:[{amount:"27",currency:null}]},{state:"unavailable",prices:input.sourceQuotes.prices}])expect(collectionDetailSchema.safeParse({...input,sourceQuotes:quotes}).success).toBe(false);
    expect(collectionDetailSchema.parse({...input,sourceQuotes:{state:"unavailable"}}).product.title).toBe("known");
  });
  it("preserves bounded persisted lineage metadata as source data",()=>{
    const detail=collectionDetailSchema.parse({item:{id,batchId:id,source:{productKey:"product",publicationId:id,version:"1",kind:"acquisition",operationId:id},revision:1,createdAt},product:{sources:[{type:"crawler",platform:"1688",source_id:"981645030344",source_version:"capture-v1",snapshot_id:id,checksum:"a".repeat(64),captured_at:createdAt,metadata:{untrusted:"not projected"}}],warnings:[{code:"missing",field:"price",message:"价格缺失"}],review:{needs_review:true,reasons:["price"]}}});
    expect(detail.product.sources?.[0]).toMatchObject({source_version:"capture-v1",snapshot_id:id,checksum:"a".repeat(64),captured_at:createdAt});
    expect(detail.product.sources?.[0]).toHaveProperty("metadata",{untrusted:"not projected"});
    expect(detail.product.review?.needs_review).toBe(true);
    expect(detail.product.warnings?.[0]?.message).toBe("价格缺失");
  });
  it("keeps the Amazon observation and parent relationship in original JSON readback",()=>{
    const source={type:"crawler",platform:"amazon",source_id:"B012345678",detail:"B012345678",source_run_id:"amazon-browser-capture:original",request_id:"request",notes:["selected evidence"],metadata:{observation:JSON.stringify({scope:"partial_family",parentASIN:"B012345679"}),site:"us"}};
    const input={item:{id,batchId:id,source:{productKey:"amazon-private",publicationId:id,version:"1",kind:"amazon_data",operationId:id},revision:1,createdAt},product:{sources:[source]}};
    const detail=collectionDetailSchema.parse(input);
    expect(detail.product.sources?.[0]).toEqual(source);
    expect(JSON.parse(JSON.stringify(detail)).product.sources[0].metadata.observation).toBe(source.metadata.observation);
  });
  it("bounds source metadata without interpreting it as command authority",()=>{
    const input=(metadata:Record<string,string>)=>({item:{id,batchId:id,source:{productKey:"product",publicationId:id,version:"1",kind:"amazon_data"},revision:1,createdAt},product:{sources:[{metadata}]}});
    expect(collectionDetailSchema.safeParse(input({observation:"not a command",value:"x".repeat(8192)})).success).toBe(true);
    expect(collectionDetailSchema.safeParse(input({value:"x".repeat(8193)})).success).toBe(false);
    expect(collectionDetailSchema.safeParse(input(Object.fromEntries(Array.from({length:257},(_,index)=>[String(index),"value"])))).success).toBe(false);
  });
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
