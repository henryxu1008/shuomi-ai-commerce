import {connection} from "next/server";
import {notFound} from "next/navigation";
import {collectionID} from "@/lib/contracts/product-collection";
import {parseSkillLaunchRecord} from "@/lib/contracts/skill-launch";
import {OwnProductImagesPage} from "@/components/workbench/collections/own-product-images-page";
export default async function Page({params,searchParams}:{params:Promise<{itemId:string}>;searchParams:Promise<Record<string,string|string[]|undefined>>}){
 await connection();const {itemId}=await params,query=await searchParams;
 if(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true"||process.env.LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED!=="true")notFound();
 const item=collectionID.safeParse(itemId),source=collectionID.safeParse(query.source),preparation=collectionID.safeParse(query.preparation);
 if(!item.success||!source.success||!preparation.success)notFound();
 return <OwnProductImagesPage itemId={item.data} sourceId={source.data} preparationId={preparation.data} launch={parseSkillLaunchRecord(query)}/>;
}
