// Public directory information checked against the linked official sources.
// Coverage is a confirmed subset, not a live inventory or connection receipt.
export const catalogRegions = {
 CN:"中国",US:"美国",CA:"加拿大",GB:"英国",DE:"德国",FR:"法国",CZ:"捷克",ES:"西班牙",JP:"日本",AU:"澳大利亚",MX:"墨西哥",SA:"沙特阿拉伯",
} as const;
export type CatalogRegion = keyof typeof catalogRegions;
export type WarehouseKind = "domestic"|"overseas";
type WarehouseCoverage = {kind:WarehouseKind;region:CatalogRegion};
export type CatalogEntry = {
 id:string;name:string;domain:string;website:string;logo:string;description:string;
 apiNote:string;coverageSource:string;warehouses:readonly WarehouseCoverage[];
};
const overseas=(...regions:CatalogRegion[]):WarehouseCoverage[]=>regions.map(region=>({kind:"overseas",region}));
export const catalogDirectory:readonly CatalogEntry[] = [
 {
  id:"gigacloud",name:"大健云仓",domain:"gigacloudtech.cn",website:"https://www.gigacloudtech.cn/",logo:"/brands/catalogs/gigacloud.png",
  description:"面向家具、家居及大件商品的跨境 B2B 货盘。",
  apiNote:"平台提供 EDI / API 服务，接入条件需与平台确认。",
  coverageSource:"https://www.gigacloudtech.com/about",
  warehouses:overseas("US","CA","GB","DE","JP"),
 },
 {
  id:"doba",name:"Doba",domain:"doba.com",website:"https://www.doba.com/",logo:"/brands/catalogs/doba.svg",
  description:"提供多品类分销货源与美国仓商品。",
  apiNote:"平台提供 Retailer API，需按平台要求申请权限。",
  coverageSource:"https://www.doba.com/high-quality-suppliers",
  warehouses:overseas("US"),
 },
 {
  id:"saleyee",name:"SaleYee",domain:"saleyee.com",website:"https://www.saleyee.com/",logo:"/brands/catalogs/saleyee.png",
  description:"赛盈分销，提供海外仓货源与一件代发服务。",
  apiNote:"平台支持自研 ERP 的 API 对接，需联系平台确认。",
  coverageSource:"https://www.saleyee.com/",
  warehouses:overseas("US","GB","DE","FR","CZ","ES","CA","JP","AU","MX","SA"),
 },
 {
  id:"cj",name:"CJdropshipping",domain:"cjdropshipping.com",website:"https://cjdropshipping.com/",logo:"/brands/catalogs/cj.svg",
  description:"覆盖中国仓与海外仓的选品、采购及代发平台。",
  apiNote:"平台提供开放 API，需使用平台账号申请密钥。",
  coverageSource:"https://elites.cjdropshipping.com/article-detail?id=5747",
  warehouses:[{kind:"domestic",region:"CN"},...overseas("US","GB","DE","ES")],
 },
];

export function catalogWarehouseMatches(warehouse:WarehouseCoverage,kind:WarehouseKind|"",region:CatalogRegion|""){
 return (!kind||warehouse.kind===kind)&&(!region||warehouse.region===region);
}
