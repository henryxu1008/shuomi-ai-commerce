// This is the same trusted origin used by the existing configuration BFF.
// Availability advertises a read endpoint; the catalog still authorizes each run source.
export function configuredAgentConfigurationOrigin():string|null{
 try{
  const url=new URL(process.env.LISTINGKIT_SERVICE_API_BASE??"");
  return ["http:","https:"].includes(url.protocol)&&!url.username&&!url.password&&!url.search&&!url.hash&&["/api/v1","/api/v1/"].includes(url.pathname)?url.origin:null;
 }catch{return null;}
}
