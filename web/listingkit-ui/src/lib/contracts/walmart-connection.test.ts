import {describe,it,expect} from "vitest";
import {officialConnectionBeginSchema,officialConnectionCompleteSchema,officialConnectionViewSchema} from "./store-connection";
describe("Walmart US OAuth boundary",()=>{
 it("only admits the official US authorization URL and exact callback",()=>{
  const begin={authorizationMethod:"oauth_callback",attemptId:"00000000-0000-4000-8000-000000000001",authorizationUrl:`https://login.account.wal-mart.com/authorize?responseType=code&clientId=client&redirectUri=${encodeURIComponent("https://app.example/callback")}&clientType=seller&nonce=${"n".repeat(43)}&state=${"s".repeat(43)}`,expiresAt:"2026-10-16T00:00:00Z"};
  expect(officialConnectionBeginSchema.safeParse(begin).success).toBe(true);
  expect(officialConnectionBeginSchema.safeParse({...begin,authorizationUrl:begin.authorizationUrl.replace("wal-mart.com","evil.example")}).success).toBe(false);
  expect(officialConnectionCompleteSchema.safeParse({attemptId:begin.attemptId,appId:"client",state:"state",authCode:"code",sellerId:"seller",type:"auth"}).success).toBe(true);
  expect(officialConnectionCompleteSchema.safeParse({attemptId:begin.attemptId,appId:"client",merchantConfirmation:"seller",authCode:"code"}).success).toBe(false);
 });
 it("cannot project connected before a verified merchant binding",()=>{
  expect(officialConnectionViewSchema.safeParse({platform:"walmart",authorizationMethod:"oauth_callback",appId:"client",appRevision:"r1",attemptId:"00000000-0000-4000-8000-000000000001",state:"merchant_confirmation_required",connectionStatus:"disconnected",version:1,observedAt:null,walmart:{environment:"production",market:"US",partnerId:"seller",merchantName:"Seller"}}).success).toBe(true);
 });
});
