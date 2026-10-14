import {it,expect} from "vitest";
import {tiktokDraftInputSchema,tiktokEffectSchema} from "./tiktokshop-model";
import {targetInputSchema,sourceSelectionSchema} from "./supply-chain";

it("accepts explicit TikTok input and rejects secret or mixed platform fields",()=>{
 const draft={selection:{method:"LOCAL_REPLICATION",categoryId:"cat"},title:"Human title",description:"Human description",saveMode:"AS_DRAFT",mainImages:[],skus:[{sellerSku:"",salePrice:"",inventory:[]}],packageWeight:{unit:"KILOGRAM",value:""}};
 expect(tiktokDraftInputSchema.safeParse(draft).success).toBe(true);
 const input={sourceId:"11111111-1111-4111-8111-111111111111",storeId:"22222222-2222-4222-8222-222222222222",expectedRevision:0,effectiveVersion:"1",tiktok:draft};
 expect(targetInputSchema.safeParse(input).success).toBe(true);
 expect(targetInputSchema.safeParse({...input,shopCipher:"private"}).success).toBe(false);
 expect(targetInputSchema.safeParse({...input,temu:{}}).success).toBe(false);
 expect(sourceSelectionSchema.safeParse({itemId:input.sourceId,originalPublicationId:"publication",originalSnapshotVersion:"1",effectiveCatalogVersion:"1",targetPlatform:"tiktokshop"}).success).toBe(true);
});
it("keeps DRAFT asset identity and cannot label an uncorrelated response as live",()=>{
 const hash="a".repeat(64);
 const effect={draftHash:hash,publication:{state:"remote_draft",shopId:"shop",productId:"product",skus:[{localSkuId:"l",globalSkuId:"g"}],failureReason:"official_validation_failed_saved_as_draft",failureCount:1,fingerprint:hash}};
 expect(tiktokEffectSchema.safeParse(effect).success).toBe(true);
 expect(tiktokEffectSchema.safeParse({...effect,publication:{...effect.publication,state:"live"}}).success).toBe(false);
 expect(tiktokEffectSchema.safeParse({...effect,global:{id:"global",skus:[{id:"g",seller_sku:"sku"}],responseHash:hash}}).success).toBe(false);
});
