import {expect,it} from "vitest";
import {imageSetRequestSchema} from "./product-image-set";
import {skillPackageSchema} from "./skill";

it("accepts compatible skill metadata and only a bounded exact selection for image preparation",()=>{
 expect(skillPackageSchema.safeParse({name:"image-method",description:"Method",digest:"a".repeat(64),files:null,target:"product-image",displayName:"标准商品套图"}).success).toBe(true);
 const skill={skillId:"2bb819bd-af13-4e68-95e0-1cc86130fb1f",version:1,digest:"a".repeat(64)};
 const input={expectedModeRevision:"0",target:{Platform:"product"},skill};
 expect(imageSetRequestSchema("prepare").safeParse(input).success).toBe(true);
 expect(imageSetRequestSchema("prepare").safeParse({...input,skill:{...skill,ownerKind:"PLATFORM"}}).success).toBe(false);
});
