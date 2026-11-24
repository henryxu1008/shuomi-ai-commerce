import {notFound} from "next/navigation";
import {collectionID} from "@/lib/contracts/product-collection";
import {MetaUploadPage} from "@/components/workbench/listing/meta-upload-page";

export default async function Page({params,searchParams}:{
  params:Promise<{sourceId:string}>;
  searchParams:Promise<{plan?:string;preparation?:string}>;
}){
  const {sourceId}=await params;
  const {plan,preparation}=await searchParams;
  if(process.env.NEXT_PUBLIC_META_CATALOG_SIMULATION!=="true"||!collectionID.safeParse(sourceId).success||
    (plan!==undefined&&!collectionID.safeParse(plan).success)||(preparation!==undefined&&!collectionID.safeParse(preparation).success))notFound();
  return <MetaUploadPage sourceId={sourceId} initialPlan={plan} preparation={preparation}/>;
}
