import { connection } from "next/server";
import { notFound } from "next/navigation";
import Link from "next/link";
import { PrimaryIntroduction } from "@/components/workbench/console/primary-introduction";
import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";
export default async function Page({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){
 await connection();
 const full=process.env.LISTINGKIT_SUPPLY_CHAIN_ENABLED==="true";
 const preparation=process.env.LISTINGKIT_ACQUISITION_PLANS_ENABLED==="true";
 const completion=process.env.LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED==="true";
 const collections=process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED==="true";
 const params=await searchParams;
 const read=(name:string)=>{const value=params[name];if(value===undefined)return undefined;if(typeof value!=="string"||!isAcquisitionUUID(value))notFound();return value};
 const selected=new URLSearchParams();
 for(const name of ["preparation","store","operation"]){const value=read(name);if(value)selected.set(name,value);}
 const suffix=selected.size?`?${selected}`:"";
 return <><PrimaryIntroduction kind="supply" available={[collections,collections&&(full||preparation||completion),collections&&full]}
  entryHrefs={["/workbench/data/mine",`/workbench/supply/mine/waiting${suffix}`,`/workbench/supply/mine/ready${suffix}`]}
  notice={full?undefined:completion?"当前可保存、编辑未绑定店铺的待补全草稿；平台适配和上传在官方应用与渠道配置后开放。":"当前可查看我的数据与已转入批次；平台适配、智能体优化和上传需要正式官方应用及渠道配置。"}/>{collections&&completion?<p className="mt-5 text-sm"><Link className="text-primary underline" href={`/workbench/supply/mine/missing${suffix}`} prefetch={false}>查看待补全草稿</Link></p>:null}</>;
}
