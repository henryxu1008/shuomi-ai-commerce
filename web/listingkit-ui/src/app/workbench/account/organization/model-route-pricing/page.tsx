import {requireAccountUserId} from "@/lib/server/account-page-auth";
import {ModelRoutePricingPage} from "@/components/workbench/account/model-route-pricing";
export default async function Page(){const user=await requireAccountUserId("/workbench/account/organization/model-route-pricing");return <ModelRoutePricingPage expectedUserId={user}/>}
