import {expect,it} from "vitest";
import {completionDraftNavigation} from "./completion-draft-navigation";
const id="11111111-1111-4111-8111-111111111111";
it("accepts only optional fixed UUID navigation IDs",()=>{
 expect(completionDraftNavigation({})).toEqual({preparationId:undefined,itemId:undefined});
 expect(completionDraftNavigation({preparation:id,item:id})).toEqual({preparationId:id,itemId:id});
 for(const query of [{preparation:[id,id]},{item:[id]},{item:""},{preparation:"https://example.org"},{returnTo:"https://example.org"},{storeId:id}])expect(completionDraftNavigation(query)).toBeNull();
});
