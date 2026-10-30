import {afterEach,expect,it,vi} from "vitest";
import {nativeRunDestinationAvailability} from "./native-run-destination-availability";

afterEach(()=>vi.unstubAllEnvs());
it.each([
 [undefined,undefined,undefined,undefined,{ownProduct:false,acquisition:false,materials:false}],
 ["false","true","true","true",{ownProduct:false,acquisition:false,materials:false}],
 ["true","true","false","false",{ownProduct:true,acquisition:false,materials:false}],
 ["true","false","true","false",{ownProduct:false,acquisition:true,materials:false}],
 ["true","false","false","true",{ownProduct:false,acquisition:false,materials:true}],
 ["true","true","true","true",{ownProduct:true,acquisition:true,materials:true}],
] as const)("matches original target page/panel serving flags (%s/%s/%s/%s)",(agent,own,acquisition,materials,expected)=>{
 vi.stubEnv("LISTINGKIT_PRODUCT_AGENT_ENABLED",agent);vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED",own);
 vi.stubEnv("LISTINGKIT_PRODUCT_ACQUISITION_ENABLED",acquisition);vi.stubEnv("LISTINGKIT_SUPPLY_CHAIN_ENABLED",materials);
 expect(nativeRunDestinationAvailability()).toEqual(expected);
});
