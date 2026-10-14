import { expect,it } from "vitest";
import { officialConnectionBeginSchema,officialConnectionCompleteSchema,officialConnectionViewSchema,officialApplicationListSchema } from "./store-connection";
import { workbenchStoreCreateSchema } from "../validation/workbench-store";
const attemptId="22222222-2222-4222-8222-22222222222b";
it("supports only US cross-border TikTok Store identity",()=>{
 const store={name:"US Shop",platform:"tiktokshop",region:"US",externalStoreId:"USCODE"};
 expect(workbenchStoreCreateSchema.safeParse(store).success).toBe(true);
 expect(workbenchStoreCreateSchema.safeParse({...store,region:"UK"}).success).toBe(false);
 expect(workbenchStoreCreateSchema.safeParse({...store,externalStoreId:""}).success).toBe(false);
});
it("bounds TikTok authorization to the US official service URL",()=>{
 const begin={attemptId,authorizationMethod:"oauth_callback",authorizationUrl:`https://services.us.tiktokshop.com/open/authorize?service_id=service&state=${"a".repeat(43)}`,expiresAt:"2026-10-12T02:00:00Z"};
 expect(officialConnectionBeginSchema.safeParse(begin).success).toBe(true);
 for(const url of ["https://attacker.example/open/authorize","https://services.us.tiktokshop.com/other",`${begin.authorizationUrl}&state=duplicate`,`${begin.authorizationUrl}&app_secret=secret`])expect(officialConnectionBeginSchema.safeParse({...begin,authorizationUrl:url}).success).toBe(false);
});
it("uses an explicit authorization code variant and safe unconfigured projection",()=>{
 const code={attemptId,appId:"app",state:"state",authCode:"code"};
 expect(officialConnectionCompleteSchema.safeParse(code).success).toBe(true);
 expect(officialConnectionCompleteSchema.safeParse({...code,tempToken:"token"}).success).toBe(false);
 const view={platform:"tiktokshop",attemptId:"",state:"unconfigured",connectionStatus:"unavailable",version:0,observedAt:null,tiktok:{market:"US",sellerType:"CROSS_BORDER"}};
 expect(officialConnectionViewSchema.safeParse(view).success).toBe(true);
 expect(officialConnectionViewSchema.safeParse({...view,accessToken:"secret"}).success).toBe(false);
 expect(officialConnectionViewSchema.safeParse({...view,connectionStatus:"connected"}).success).toBe(false);
 expect(officialApplicationListSchema.safeParse({applications:[{platform:"tiktokshop",appId:"app",revision:"r1",authorizationMethod:"oauth_callback"}]}).success).toBe(true);
});
