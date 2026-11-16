import {useState} from "react";
import {Button} from "@/components/ui/button";
import {Input} from "@/components/ui/input";
import {Select} from "@/components/ui/select";
import {marketFiltersSchema,type MarketFilters} from "@/lib/contracts/supply-market";
import styles from "./market-filters.module.css";
import {isShippingLocation, SupplyLocationFields, useShippingLocations} from "./supply-location-fields";

export type MarketCriteria=MarketFilters&{keyword:string};
const empty=()=>({keyword:"",province:"",city:"",minPrice:"",maxPrice:"",priceCurrency:"CNY",maxMOQ:"",maxLeadDays:""});
export function MarketFiltersForm({subject,onApply}:{subject:string;onApply:(criteria:MarketCriteria)=>void}){
 const locations=useShippingLocations(subject);
 const [draft,setDraft]=useState(empty),[error,setError]=useState("");
 const update=(key:keyof typeof draft,value:string)=>{setDraft(previous=>({...previous,[key]:value}));setError("")};
 return <form className={styles.form} onSubmit={e=>{
  e.preventDefault();
  if((draft.province||draft.city)&&!isShippingLocation(locations.data?.country,draft,false)){setError("请重新选择发货地，或清除筛选");return}
  if(draft.minPrice!==""&&draft.maxPrice!==""&&Number(draft.minPrice)>Number(draft.maxPrice)){setError("最低价格不能高于最高价格");return}
  const filters:Record<string,string|number>={};
  if(draft.province)filters.province=draft.province;
  if(draft.city)filters.city=draft.city;
  for(const key of ["minPrice","maxPrice","maxMOQ","maxLeadDays"] as const){if(draft[key]!=="")filters[key]=Number(draft[key])}
  if(draft.minPrice!==""||draft.maxPrice!=="")filters.priceCurrency=draft.priceCurrency;
  const parsed=marketFiltersSchema.safeParse(filters);
  if(!parsed.success){setError("请检查价格、起订量和供货周期");return}
  setError("");onApply({...parsed.data,keyword:draft.keyword.trim()});
 }}>
  <div className={styles.search}><Input aria-label="商品关键词" maxLength={80} value={draft.keyword} onChange={e=>update("keyword",e.target.value)} placeholder="搜索商品名称"/><Button type="submit">搜索</Button><Button type="button" variant="ghost" onClick={()=>{setDraft(empty());setError("");onApply({keyword:""})}}>清除筛选</Button></div>
  <div className={styles.fields}>
   <SupplyLocationFields locations={locations} province={draft.province} city={draft.city} required={false} filter onChange={location=>{setDraft(previous=>({...previous,...location}));setError("")}}/>
   <fieldset className={styles.price}><legend>价格</legend><div className={styles.priceInputs}>
    <Select aria-label="价格币种" value={draft.priceCurrency} onChange={e=>update("priceCurrency",e.target.value)}>{[["CNY","人民币"],["USD","美元"],["EUR","欧元"],["GBP","英镑"],["JPY","日元"]].map(([code,name])=><option key={code} value={code}>{name}</option>)}</Select>
    <Input aria-label="最低价格" type="number" min={0} step="any" value={draft.minPrice} onChange={e=>update("minPrice",e.target.value)} placeholder="最低"/><span aria-hidden="true">—</span><Input aria-label="最高价格" type="number" min={0} step="any" value={draft.maxPrice} onChange={e=>update("maxPrice",e.target.value)} placeholder="最高"/>
   </div></fieldset>
   <label>起订量不超过（件）<Input type="number" min={1} max={1e9} value={draft.maxMOQ} onChange={e=>update("maxMOQ",e.target.value)} placeholder="不限"/></label>
   <label>供货周期不超过（天）<Input type="number" min={0} max={3650} value={draft.maxLeadDays} onChange={e=>update("maxLeadDays",e.target.value)} placeholder="不限"/></label>
  </div>
  {error?<p role="alert" className={styles.error}>{error}</p>:null}
 </form>;
}
