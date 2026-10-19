import {handleModelRoutePricing,rejectModelRoutePricing} from "@/lib/server/model-route-pricing-route";
export const GET=handleModelRoutePricing;
export const POST=handleModelRoutePricing;
export const PUT=rejectModelRoutePricing;
export const PATCH=rejectModelRoutePricing;
export const DELETE=rejectModelRoutePricing;
export const HEAD=rejectModelRoutePricing;
export const OPTIONS=rejectModelRoutePricing;
