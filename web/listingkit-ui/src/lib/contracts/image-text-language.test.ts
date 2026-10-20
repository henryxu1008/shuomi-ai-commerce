import {describe,expect,it} from "vitest";
import {imageSetRequestSchema} from "./product-image-set";
import {contentAgentStartRequestSchema} from "./product-agent";

describe("image text language consent",()=>{
 it("retains manual-only translation consent in the strict BFF request",()=>{
  const request={expectedModeRevision:"0",targetPlatform:"product",contentSelection:{targetLanguage:"en",title:true,sellingPoints:true},manualReviewOnly:true};
  expect(contentAgentStartRequestSchema.parse(request)).toMatchObject({manualReviewOnly:true});
 });
 it("accepts only a Review reference, never client-authored approved copy",()=>{
  const request={expectedModeRevision:"0",target:{Platform:"product"},sharedOriginalIds:["source-1"],text:{mode:"reviewed",review:{proposalId:"a3ce4c41-745a-47bf-a418-c940d3ba4b84",revision:"2"},taskIds:["detail"]}};
  expect(imageSetRequestSchema("prepare").safeParse(request).success).toBe(true);
  expect(imageSetRequestSchema("prepare").safeParse({...request,text:{...request.text,lines:["unverified"]}}).success).toBe(false);
  expect(imageSetRequestSchema("prepare").safeParse({...request,text:{mode:"none"}}).success).toBe(true);
 });
});
