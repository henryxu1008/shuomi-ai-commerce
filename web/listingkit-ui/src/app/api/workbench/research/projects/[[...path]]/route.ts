import {handleMarketResearch,rejectMarketResearch} from "@/lib/server/market-research-route";
export const runtime="nodejs";
export const dynamic="force-dynamic";
export const GET=handleMarketResearch;
export const POST=handleMarketResearch;
export const PATCH=rejectMarketResearch;
export const PUT=rejectMarketResearch;
export const DELETE=rejectMarketResearch;
