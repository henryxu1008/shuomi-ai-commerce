import {requireAccountUserId} from "@/lib/server/account-page-auth";
import {DataRowPricingPage} from "@/components/workbench/account/data-row-pricing";
export default async function Page(){const user=await requireAccountUserId("/workbench/account/organization/data-row-pricing");return <DataRowPricingPage expectedUserId={user}/>}
