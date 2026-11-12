"use client";

import {useRef,useState} from "react";
import {Button} from "@/components/ui/button";
import {AcquisitionPage} from "./acquisition-page";
import {DataServicesPage} from "../data-services/data-services-page";

type Platform="1688"|"amazon";
export function OnlineAcquisitionHub({has1688,hasAmazon,initial,plansEnabled=false}:{has1688:boolean;hasAmazon:boolean;initial:Platform;plansEnabled?:boolean}){
 const [active,setActive]=useState<Platform>(initial);
 const activeRef=useRef<Platform>(initial);
 const [visited,setVisited]=useState({"1688":initial==="1688",amazon:initial==="amazon"});
 function choose(platform:Platform){
  activeRef.current=platform;setVisited(previous=>({...previous,[platform]:true}));setActive(platform);
 }
 return <div>
  <nav aria-label="采集平台" className="flex flex-wrap gap-2 px-4 pt-4 sm:px-6">
   {has1688?<Button type="button" variant={active==="1688"?"default":"outline"} aria-pressed={active==="1688"} onClick={()=>choose("1688")}>1688</Button>:null}
   {hasAmazon?<Button type="button" variant={active==="amazon"?"default":"outline"} aria-pressed={active==="amazon"} onClick={()=>choose("amazon")}>Amazon</Button>:null}
  </nav>
  {has1688&&visited["1688"]?<div hidden={active!=="1688"}><AcquisitionPage plansEnabled={plansEnabled} shouldAutoOpen={()=>activeRef.current==="1688"}/></div>:null}
  {hasAmazon&&visited.amazon?<div hidden={active!=="amazon"}><DataServicesPage mode="amazon"/></div>:null}
 </div>;
}
