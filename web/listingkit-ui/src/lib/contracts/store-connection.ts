import { z } from "zod";
export const officialConnectionViewSchema = z
	.object({
		platform:z.enum(["shein","temu","tiktokshop"]),
		identityPartition:z.enum(["CN","US","EU","GLOBAL"]).optional(),
		authorizationMethod:z.enum(["callback","manual_token","oauth_callback"]).optional(),
		tiktok:z.object({market:z.literal("US"),sellerType:z.literal("CROSS_BORDER"),shopId:z.string().min(1).max(128).optional(),grantGeneration:z.number().int().positive().max(Number.MAX_SAFE_INTEGER).optional(),grantRevision:z.number().int().positive().max(Number.MAX_SAFE_INTEGER).optional(),bindingRevision:z.number().int().positive().max(Number.MAX_SAFE_INTEGER).optional()}).strict().optional(),
		mode:z.enum(["fully_managed","semi_managed","local"]).optional(),
		merchantIdentityHash:z.string().regex(/^[a-f0-9]{64}$/).optional(),
		regionId:z.number().int().min(1).max(Number.MAX_SAFE_INTEGER).optional(),
		credentialExpiresAt:z.string().datetime({precision:null}).optional(),
		appId:z.string().min(1).max(128).optional(),
		appRevision:z.string().min(1).max(200).optional(),
		attemptId: z.union([z.uuid(), z.literal("")]),
    state: z.enum([
      "disconnected",
      "awaiting_consent",
      "exchange_dispatched",
      "credential_received",
      "verified",
      "failed",
      "unconfigured", "created", "bound", "rejected", "unknown",
    ]),
    connectionStatus: z.enum([
      "connected",
      "disconnected",
      "expired",
      "unavailable",
    ]),
    version: z.number().int().min(0).max(Number.MAX_SAFE_INTEGER),
    observedAt: z.string().datetime({ precision: null }).nullable(),
  })
  .strict().refine(view=>{
    if(view.platform==="tiktokshop")return !!view.tiktok && !view.identityPartition && !view.mode && !view.regionId && !view.merchantIdentityHash && (!view.authorizationMethod || view.authorizationMethod==="oauth_callback") && (view.connectionStatus!=="connected" || view.state==="bound" && !!view.appId && !!view.appRevision && !!view.credentialExpiresAt && !!view.tiktok.shopId && !!view.tiktok.grantGeneration && !!view.tiktok.grantRevision && !!view.tiktok.bindingRevision);
    if(view.tiktok || ["unconfigured","created","bound","rejected","unknown"].includes(view.state))return false;
    if(view.platform==="shein")return !view.identityPartition && (!view.authorizationMethod||view.authorizationMethod==="callback") && !view.mode && !view.merchantIdentityHash && !view.regionId;
    if(!view.appId)return view.connectionStatus!=="connected";
    if(!view.identityPartition||view.authorizationMethod!=="manual_token")return false;
    if(view.connectionStatus!=="connected")return true;
    return !!view.merchantIdentityHash && !!view.credentialExpiresAt && (view.identityPartition==="CN" ? (view.mode==="fully_managed"||view.mode==="semi_managed") && !view.regionId : view.mode==="local" && !!view.regionId);
  });
export const officialConnectionBeginSchema = z
  .object({
		authorizationMethod:z.enum(["callback","manual_token","oauth_callback"]),
    attemptId: z.uuid(),
    authorizationUrl: z
      .string()
      .max(4096),
    expiresAt: z.string().datetime({ precision: null }),
  })
  .strict().refine((value)=>{
      if(value.authorizationMethod==="manual_token")return value.authorizationUrl==="";
      if(value.authorizationMethod==="oauth_callback"){
        try{const url=new URL(value.authorizationUrl);return url.origin==="https://services.us.tiktokshop.com" && url.pathname==="/open/authorize" && !url.username && !url.password && !url.hash && [...url.searchParams.keys()].every(key=>["service_id","state"].includes(key)&&url.searchParams.getAll(key).length===1) && !!url.searchParams.get("service_id") && url.searchParams.get("state")?.length===43;}catch{return false;}
      }
      const raw=value.authorizationUrl;
        try {
          const url = new URL(raw);
          return (
            url.origin === "https://openapi-sem.sheincorp.com" &&
            !url.username &&
            !url.password &&
            url.pathname === "/" &&
            !url.search &&
            url.hash.startsWith("#/empower?") &&
            new URLSearchParams(url.hash.slice(10)).get("state")?.length === 43
          );
        } catch {
          return false;
        }
  });
export const officialConnectionCompleteSchema = z
  .object({
    attemptId: z.uuid(),
    appId: z.string().min(1).max(200),
    state: z.string().min(1).max(128),
    tempToken: z.string().min(1).max(4096).optional(),
    authCode:z.string().max(2048).optional(),
  })
  .strict().refine(value=>(value.tempToken!==undefined)!==(value.authCode!==undefined));
export const officialConnectionQuerySchema = z
  .object({ attemptId: z.uuid() })
	.strict();
const officialApplicationChoicesSchema=z.array(z.object({
	platform:z.enum(["shein","temu","tiktokshop"]),identityPartition:z.enum(["CN","US","EU","GLOBAL"]).optional(),
	authorizationMethod:z.enum(["callback","manual_token","oauth_callback"]),supportedModes:z.union([z.literal(1),z.literal(6)]).optional(),
	appId:z.string().min(1).max(128),revision:z.string().min(1).max(200),
	type:z.enum(["self_operated","semi_managed","fully_managed"]).optional(),
}).strict().refine(value=>value.platform==="tiktokshop" ? value.authorizationMethod==="oauth_callback" && !value.type && !value.identityPartition && !value.supportedModes : value.platform==="shein" ? value.authorizationMethod==="callback" && !!value.type && !value.identityPartition && !value.supportedModes : value.authorizationMethod==="manual_token" && !value.type && !!value.identityPartition && value.supportedModes===(value.identityPartition==="CN"?6:1))).max(17).refine(values=>new Set(values.map(v=>JSON.stringify([v.platform,v.identityPartition??"",v.appId]))).size===values.length);
export const officialConnectionStartSchema=z.object({appId:z.string().min(1).max(128),identityPartition:z.enum(["CN","US","EU","GLOBAL"]).optional()}).strict();
export const officialConnectionInstallSchema=z.object({attemptId:z.uuid(),accessToken:z.string().min(1).max(4096).refine(value=>value.trim()===value&&!/[\x00\r\n]/.test(value))}).strict();
export const officialApplicationListSchema=z.object({applications:officialApplicationChoicesSchema}).strict();
export type OfficialConnectionCallback = z.infer<
  typeof officialConnectionCompleteSchema
>;
