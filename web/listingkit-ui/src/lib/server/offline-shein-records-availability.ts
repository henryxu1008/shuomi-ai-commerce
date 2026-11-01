export function isOfflineSheinRecordsAvailable(){
 if(process.env.LISTINGKIT_SHEIN_OFFLINE_RECORDS_ENABLED!=="true"||process.env.LISTINGKIT_REPORT_CENTER_ENABLED!=="true"||process.env.LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED!=="true")return false;
 try{const base=new URL(process.env.LISTINGKIT_SERVICE_API_BASE??"");return ["http:","https:"].includes(base.protocol)&&!base.username&&!base.password&&!base.search&&!base.hash&&["/api/v1","/api/v1/"].includes(base.pathname);}catch{return false;}
}
