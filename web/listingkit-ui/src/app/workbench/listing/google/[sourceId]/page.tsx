import { notFound } from "next/navigation";
import { collectionID } from "@/lib/contracts/product-collection";
import { GoogleUploadPage } from "@/components/workbench/listing/google-upload-page";

export default async function Page({ params, searchParams }: {
  params: Promise<{sourceId:string}>;
  searchParams: Promise<{plan?:string;preparation?:string}>;
}) {
  const {sourceId}=await params;
  const {plan,preparation}=await searchParams;
  if(process.env.NEXT_PUBLIC_GOOGLE_MERCHANT_ENABLED!=="true" || !collectionID.safeParse(sourceId).success ||
    (plan!==undefined&&!collectionID.safeParse(plan).success) ||
    (preparation!==undefined&&!collectionID.safeParse(preparation).success)) notFound();
  return <GoogleUploadPage sourceId={sourceId} initialPlan={plan} preparation={preparation}/>;
}
