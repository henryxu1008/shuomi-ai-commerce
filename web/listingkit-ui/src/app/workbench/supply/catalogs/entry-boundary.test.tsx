import {afterEach, expect, it, vi} from "vitest";
import SelectionPage from "./selection/page";
import ApplyPage from "./apply/page";
const gates=vi.hoisted(()=>({
  connection:vi.fn(),
  notFound:vi.fn(()=>{throw new Error("NOT_FOUND")}),
  redirect:vi.fn((path:string)=>{throw new Error(`REDIRECT:${path}`)}),
  acquisition:vi.fn(()=>false),
}));
vi.mock("next/server",()=>({connection:gates.connection}));
vi.mock("next/navigation",()=>({notFound:gates.notFound,redirect:gates.redirect}));
vi.mock("@/lib/server/product-acquisition-availability",()=>({isProductAcquisitionAvailable:gates.acquisition}));
const originalMarket=process.env.LISTINGKIT_SUPPLY_MARKET_ENABLED, originalSds=process.env.LISTINGKIT_SDS_POD_ENABLED;
afterEach(()=>{vi.clearAllMocks();gates.acquisition.mockReturnValue(false);if(originalMarket===undefined)delete process.env.LISTINGKIT_SUPPLY_MARKET_ENABLED;else process.env.LISTINGKIT_SUPPLY_MARKET_ENABLED=originalMarket;if(originalSds===undefined)delete process.env.LISTINGKIT_SDS_POD_ENABLED;else process.env.LISTINGKIT_SDS_POD_ENABLED=originalSds});
it.each([SelectionPage,ApplyPage])("keeps each catalog leaf closed without its installed runtime (%#)",async Page=>{
  process.env.LISTINGKIT_SUPPLY_MARKET_ENABLED="false";
  await expect(Page()).rejects.toThrow("NOT_FOUND");
  expect(gates.connection).toHaveBeenCalledOnce();
  expect(gates.redirect).not.toHaveBeenCalled();
});
it("redirects the retired standalone application to the catalog when the market is installed",async()=>{
  process.env.LISTINGKIT_SUPPLY_MARKET_ENABLED="true";
  await expect(ApplyPage()).rejects.toThrow("REDIRECT:/workbench/supply/catalogs");
  expect(gates.notFound).not.toHaveBeenCalled();
});
it.each([[false,false],[true,false],[false,true],[true,true]])("passes actual configured capabilities to selection (%s / %s)",async(acquisition,sds)=>{
  process.env.LISTINGKIT_SUPPLY_MARKET_ENABLED="true"; process.env.LISTINGKIT_SDS_POD_ENABLED=String(sds);gates.acquisition.mockReturnValue(acquisition);
  const element=await SelectionPage();expect(element.props).toEqual({view:"selection",acquisitionAvailable:acquisition,sdsAvailable:sds});
});
