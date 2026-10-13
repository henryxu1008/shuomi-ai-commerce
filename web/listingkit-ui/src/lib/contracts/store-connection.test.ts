import { expect, it } from "vitest";
import { officialApplicationListSchema, officialConnectionBeginSchema, officialConnectionViewSchema } from "./store-connection";

it("keeps TEMU application identities distinct and requires the correct mode set",()=>{
 const application=(identityPartition:"CN"|"US"|"EU"|"GLOBAL")=>({platform:"temu",identityPartition,appId:"same-app",revision:`v1:${identityPartition}`,authorizationMethod:"manual_token",supportedModes:identityPartition==="CN"?6:1});
 const applications=[application("CN"),application("US"),application("EU"),application("GLOBAL")];
 expect(officialApplicationListSchema.safeParse({applications}).success).toBe(true);
 expect(officialApplicationListSchema.safeParse({applications:[...applications,application("CN")]}).success).toBe(false);
 expect(officialApplicationListSchema.safeParse({applications:[{...application("CN"),supportedModes:2}]}).success).toBe(false);
});

it("manual input has no authorization redirect and connection projections contain no token",()=>{
 const attemptId="22222222-2222-4222-8222-22222222222b";
 expect(officialConnectionBeginSchema.safeParse({attemptId,authorizationMethod:"manual_token",authorizationUrl:"",expiresAt:"2026-10-12T02:00:00Z"}).success).toBe(true);
 expect(officialConnectionBeginSchema.safeParse({attemptId,authorizationMethod:"manual_token",authorizationUrl:"https://attacker.example",expiresAt:"2026-10-12T02:00:00Z"}).success).toBe(false);
 const view={platform:"temu",appId:"test-app",appRevision:"v1",identityPartition:"CN",authorizationMethod:"manual_token",mode:"fully_managed",merchantIdentityHash:"a".repeat(64),credentialExpiresAt:"2026-10-12T02:00:00Z",attemptId,state:"verified",connectionStatus:"connected",version:1,observedAt:"2026-10-12T01:00:00Z"};
 expect(officialConnectionViewSchema.safeParse(view).success).toBe(true);
 expect(officialConnectionViewSchema.safeParse({...view,accessToken:"secret"}).success).toBe(false);
 expect(officialConnectionViewSchema.safeParse({...view,identityPartition:"US"}).success).toBe(false);
});
