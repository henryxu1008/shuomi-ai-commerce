import {expect,it} from "vitest";
import {analysisResultSchema,analysisRequestSchema,analysisPath,analysisMatches} from "./product-image-analysis";
const id="11111111-1111-4111-8111-111111111111",key="22222222-2222-4222-8222-222222222222",hash="a".repeat(64);
const prepared={contextKind:"acquisition",contextId:id,requestKey:key,catalogVersion:"9007199254740993",publicationId:"pub",unitVersion:"ai-micropoint-v1",status:"prepared",stopReason:"",runId:"",kind:"source",imageId:"image-1",modelId:"mini",preparationId:id,configurationDigest:hash,quoteDigest:hash,selectionDigest:hash,contentSha256:hash,maximumMicropoints:"9223372036854775807",actualMicropoints:"",width:10,height:10};
it("keeps exact original micropoints and rejects executable or unbound result claims",()=>{
 expect(analysisResultSchema.parse(prepared)).toEqual(prepared);
 expect(analysisResultSchema.safeParse({...prepared,maximumMicropoints:9007199254740993}).success).toBe(false);
 expect(analysisResultSchema.safeParse({...prepared,status:"completed"}).success).toBe(false);
 expect(analysisResultSchema.safeParse({...prepared,apply:{}}).success).toBe(false);
 expect(analysisMatches(prepared,{contextKind:"acquisition",contextId:id,requestKey:key})).toBe(true);
 expect(analysisMatches(prepared,{contextKind:"collection",contextId:id,requestKey:key})).toBe(false);
});
it("allows only existing single-image selection and exact explicit consent",()=>{
 const body={kind:"source",imageId:"image-1",catalogVersion:"1",publicationId:"pub",expectedModeRevision:"0"};
 expect(analysisRequestSchema("prepare").safeParse(body).success).toBe(true);
 for(const extra of [{url:"https://untrusted.test/x"},{modelId:"other"},{maximumMicropoints:"1"},{imageBytes:"blob"}])expect(analysisRequestSchema("prepare").safeParse({...body,...extra}).success).toBe(false);
 const confirm={confirmed:true,preparationId:id,configurationDigest:hash,quoteDigest:hash,selectionDigest:hash,contentSha256:hash,modelId:"mini",maximumMicropoints:"1"};
 expect(analysisRequestSchema("confirm").safeParse(confirm).success).toBe(true);
 expect(analysisRequestSchema("confirm").safeParse({...confirm,confirmed:false}).success).toBe(false);
 for(const root of [["sourcing","1688","acquisitions",id],["supply-preparations","sources",id],["collections","items",id]]){
  expect(analysisPath([...root,"images","analysis","by-key",key,"confirm"],"confirm")).not.toBeNull();
  expect(analysisPath([...root,"images","analysis","resume"],"confirm")).toBeNull();
 }
});
