"use client";
import {Button} from "@/components/ui/button";
import {catalogEntrySchema} from "@/lib/contracts/agent-configuration";
import type {CollectionItem} from "@/lib/contracts/product-collection";
import type {ProjectScope} from "@/lib/api/project-center";
import {useConfigurationRead,configurationError} from "../agents/agent-configuration-hooks";
import {ProductSelectionFlow} from "../agents/product-selection-flow";
import {useState} from "react";
export function ResearchComparison({scope,items}:{scope:ProjectScope;items:CollectionItem[]}){
 const [nonce,setNonce]=useState(0);const catalog=useConfigurationRead(scope,"product.selection.agent",catalogEntrySchema,nonce);
 if(catalog.error)return <p role="status">{configurationError(catalog.error)} <Button variant="outline" onClick={()=>setNonce(v=>v+1)}>重新读取对比能力</Button></p>;
 if(!catalog.data)return <p role="status">正在读取原选品能力…</p>;
 return <ProductSelectionFlow scope={scope} entry={catalog.data} nonce={nonce} initialItems={items}/>;
}
