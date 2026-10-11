import { describe,expect,it } from "vitest";
import { metadataCommandSchema,catalogFiltersSchema,profileSchema } from "./store-center";
describe("Store Center contracts",()=>{
 it("keeps an unset threshold distinct from zero and requires explicit versions",()=>{
  const storeId="00000000-0000-4000-8000-000000000001";
  const base={action:"set_profile",name:"",groupId:null,groupVersion:0,stores:[{storeId,profileVersion:0}],responsibleMemberId:null,lowStockThreshold:null};
  expect(metadataCommandSchema.parse(base).lowStockThreshold).toBeNull();
  expect(metadataCommandSchema.parse({...base,lowStockThreshold:0}).lowStockThreshold).toBe(0);
  expect(metadataCommandSchema.safeParse({...base,stores:[{storeId}]}).success).toBe(false);
  expect(metadataCommandSchema.safeParse({...base,lowStockThreshold:-1}).success).toBe(false);
  expect(metadataCommandSchema.safeParse({...base,organizationId:"foreign"}).success).toBe(false);
  expect(profileSchema.safeParse({storeId,groupId:null,responsibleMemberId:null,lowStockThreshold:null,profileVersion:0,updatedAt:null}).success).toBe(true);
 });
 it("accepts literal search and rejects unknown predicates",()=>{
  expect(catalogFiltersSchema.parse({page:2,pageSize:20,q:"店 100%_"}).q).toBe("店 100%_");
  expect(catalogFiltersSchema.safeParse({page:1,pageSize:20,attention:"active"}).success).toBe(false);
 });
});
