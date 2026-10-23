import { expect,it } from "vitest";
import { loadAmazonMarker,saveAmazonMarker } from "./capture-marker";
it("isolates original key by user, enterprise and exact member",()=>{
 const scope={userId:"actor",organizationId:"org",memberId:"member"};const marker={...scope,key:"22222222-2222-4222-8222-22222222222b",source:"amazon" as const,site:"us" as const,sourceID:"B012345678",digest:"a".repeat(64)};
 expect(saveAmazonMarker(sessionStorage,marker)).toBe(true);expect(loadAmazonMarker(sessionStorage,scope,marker.key)).toEqual(marker);
 expect(loadAmazonMarker(sessionStorage,{...scope,memberId:"new-member"},marker.key)).toBeNull();
 expect(loadAmazonMarker(sessionStorage,{...scope,organizationId:"other"},marker.key)).toBeNull();
 expect(saveAmazonMarker({setItem(){throw new Error();}} as unknown as Storage,marker)).toBe(false);
});
