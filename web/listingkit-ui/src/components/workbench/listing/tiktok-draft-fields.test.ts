import {it,expect} from "vitest";
import {initialTikTokDraft,carryTikTokCompletion} from "./tiktok-draft-fields";
it("uses approved text as a candidate and leaves merchant prices and inventory unset",()=>{
 const draft=initialTikTokDraft({title:"Source title",description:"Source description",variants:[{sku:"provider-sku",price:{amount:99},stock:42}]},"GLOBAL_PUBLISHING");
 expect(draft.title).toBe("Source title");expect(draft.selection.categoryId).toBe("");expect(draft.skus[0]).toMatchObject({sellerSku:"",salePrice:"",globalBasePrice:"",inventory:[],globalInventory:[]});expect(draft.mainImages).toEqual([]);
 const chosen=carryTikTokCompletion(draft,{id:"record",revision:8,content:{title:"Reviewed English title",description:"Reviewed description"}});
 expect(chosen.selection).toEqual({recordId:"record",revision:8});expect(chosen.draft.title).toBe("Reviewed English title");expect(draft.title).toBe("Source title");
});
