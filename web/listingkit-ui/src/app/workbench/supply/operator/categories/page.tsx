import {connection} from "next/server";
import {notFound} from "next/navigation";
import {MarketCategoryManagement} from "@/components/workbench/supply-market/category-management";
import {requireAccountUserId} from "@/lib/server/account-page-auth";

export default async function Page() {
 await connection();
 if(process.env.LISTINGKIT_SUPPLY_MARKET_ENABLED!=="true")notFound();
 const expectedUserId=await requireAccountUserId("/workbench/supply/operator/categories");
 return <MarketCategoryManagement expectedUserId={expectedUserId}/>;
}
