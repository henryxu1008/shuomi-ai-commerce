import {connection} from "next/server";
import {notFound} from "next/navigation";
import {OwnProductEntryPage} from "@/components/workbench/collections/own-product-entry-page";

export default async function Page(){
 await connection();
 if(process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true")notFound();
 return <OwnProductEntryPage/>;
}
