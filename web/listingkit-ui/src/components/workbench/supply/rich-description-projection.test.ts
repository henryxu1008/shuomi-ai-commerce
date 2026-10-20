import {expect,it} from "vitest";
import {initialTikTokDraft,carryTikTokCompletion} from "./tiktok-draft-fields";
import {initialTemuDraft} from "./temu-draft-fields";
import {initialSupplyDraft} from "./draft-fields";
import type {SupplySourceDetail} from "@/lib/contracts/supply-chain";
it("projects rich source and candidate descriptions into current platform text fields",()=>{
 const source={title:"Title",description:"<h2>Details</h2><p><strong>Cotton</strong><br>Care</p>",description_format:"html-v1" as const};
 const expected="Details\nCotton\nCare";
 const tiktok=initialTikTokDraft(source,"LOCAL_REPLICATION");expect(tiktok.description).toBe(expected);
 expect(carryTikTokCompletion(tiktok,{id:"saved",revision:2,content:{title:"Candidate",description:source.description,descriptionFormat:"html-v1"}}).draft.description).toBe(expected);
 expect(initialTemuDraft(source,"local").local?.goodsBasic.goodsDesc).toBe(expected);
 expect(initialSupplyDraft({product:source} as SupplySourceDetail).product.multi_language_desc_list?.[0]?.name).toBe(expected);
 expect(source.description_format).toBe("html-v1");
});
