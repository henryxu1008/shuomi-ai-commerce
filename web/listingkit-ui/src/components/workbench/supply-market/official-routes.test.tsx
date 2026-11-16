import {afterEach,beforeEach,expect,it,vi} from "vitest";
import OfficialNewRoute from "@/app/workbench/supply/official/new/page";
import OwnNewRoute from "@/app/workbench/data/mine/own/new/page";
import OperatorRoute from "@/app/workbench/supply/operator/page";
import {SupplyApplicationForm,SupplyApplicationsPage} from "./applications";
import {OfficialOwnProductEntryPage,OwnProductEntryPage} from "../collections/own-product-entry-page";
vi.mock("next/server",()=>({connection:vi.fn()}));
vi.mock("next/navigation",()=>({notFound:()=>{throw new Error("NOT_FOUND")}}));
vi.mock("@/lib/server/account-page-auth",()=>({requireAccountUserId:async()=>"specialist"}));
beforeEach(()=>{vi.stubEnv("LISTINGKIT_SUPPLY_MARKET_ENABLED","true");vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED","true")});
afterEach(()=>vi.unstubAllEnvs());
const item="11111111-1111-4111-8111-111111111111";
it("passes only a valid exact item ID from the continuation URL to the existing form",async()=>{
 const page=await OfficialNewRoute({searchParams:Promise.resolve({item})});expect(page.type).toBe(SupplyApplicationForm);expect(page.props.initialItemId).toBe(item);
 const duplicate=await OfficialNewRoute({searchParams:Promise.resolve({item:[item,item]})});expect(duplicate.props.initialItemId).toBeUndefined();
});
it("keeps the normal own-product route and uses the permission-gated continuation only for official",async()=>{
 expect((await OwnNewRoute({searchParams:Promise.resolve({})})).type).toBe(OwnProductEntryPage);
 expect((await OwnNewRoute({searchParams:Promise.resolve({continuation:"official"})})).type).toBe(OfficialOwnProductEntryPage);
 expect((await OwnNewRoute({searchParams:Promise.resolve({continuation:"https://untrusted.example"})})).type).toBe(OwnProductEntryPage);
 vi.stubEnv("LISTINGKIT_SUPPLY_MARKET_ENABLED","false");
 await expect(OwnNewRoute({searchParams:Promise.resolve({continuation:"official"})})).rejects.toThrow("NOT_FOUND");
 expect((await OwnNewRoute({searchParams:Promise.resolve({})})).type).toBe(OwnProductEntryPage);
});
it("opens the existing operator route on official records while retaining server session bootstrap",async()=>{
 const page=await OperatorRoute({searchParams:Promise.resolve({kind:"official"})});expect(page.type).toBe(SupplyApplicationsPage);expect(page.props).toMatchObject({admin:true,expectedUserId:"specialist",initialKind:"official"});
});
