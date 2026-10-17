import {EcoservicesMine} from "@/components/workbench/ecoservices/mine";
import {ConsoleState} from "@/components/workbench/console/console-page";
import {isEcoservicesAvailable,isEcoservicesNonPaymentOnly} from "@/lib/server/ecoservices-availability";
export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){const query=await searchParams;return isEcoservicesAvailable()?<EcoservicesMine initialSide={query.side==="provider"?"provider":"buyer"} nonPaymentOnly={isEcoservicesNonPaymentOnly()}/>:<ConsoleState kind="unavailable" title="生态服务尚未开放"/>;}
