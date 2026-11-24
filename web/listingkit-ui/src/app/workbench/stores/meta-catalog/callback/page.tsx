import type {Metadata} from "next";
import {notFound} from "next/navigation";
import {MetaAuthorizationCallback} from "@/components/workbench/stores/meta-connection";

export const metadata:Metadata={referrer:"no-referrer"};

export default function Page(){
  if(process.env.NEXT_PUBLIC_META_CATALOG_SIMULATION!=="true")notFound();
  return <MetaAuthorizationCallback/>;
}
