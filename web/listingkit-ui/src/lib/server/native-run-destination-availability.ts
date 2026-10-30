import type {NativeRunDestinations} from "@/lib/agent-run-destination";
import {isProductAcquisitionAvailable} from "./product-acquisition-availability";

// Match the existing route guards and the Agent panel's serving flag.
// These flags never grant permissions or gate reading the original run history.
export function nativeRunDestinationAvailability():NativeRunDestinations{
 const agent=process.env.LISTINGKIT_PRODUCT_AGENT_ENABLED==="true";
 return {
  ownProduct:agent&&process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED==="true",
  acquisition:agent&&isProductAcquisitionAvailable(),
  materials:agent&&process.env.LISTINGKIT_SUPPLY_CHAIN_ENABLED==="true",
 };
}
