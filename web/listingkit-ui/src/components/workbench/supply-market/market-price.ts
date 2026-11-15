export function formatMarketPrice(value:number,currency:string){
 try{return new Intl.NumberFormat("zh-CN",{style:"currency",currency,minimumFractionDigits:2,maximumFractionDigits:20}).format(value)}
 catch{return `${currency} ${new Intl.NumberFormat("zh-CN",{minimumFractionDigits:2,maximumFractionDigits:20}).format(value)}`}
}
