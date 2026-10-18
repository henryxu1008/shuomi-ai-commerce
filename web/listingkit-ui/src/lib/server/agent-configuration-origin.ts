// This is the same trusted origin used by the existing configuration BFF.
// The BFF origin alone does not assert that the optional module is mounted.
export function configuredAgentConfigurationOrigin():string|null{
 try{
  const url=new URL(process.env.LISTINGKIT_SERVICE_API_BASE??"");
  return ["http:","https:"].includes(url.protocol)&&!url.username&&!url.password&&!url.search&&!url.hash&&["/api/v1","/api/v1/"].includes(url.pathname)?url.origin:null;
 }catch{return null;}
}

// Deployment serving assertion, independent of Chat and text execution admission.
export function isAgentConfigurationAvailable():boolean{
 return process.env.LISTINGKIT_AGENT_CONFIGURATION_ENABLED==="true"&&configuredAgentConfigurationOrigin()!==null;
}
