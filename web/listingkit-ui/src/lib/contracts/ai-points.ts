import {z} from "zod";
export const aiPointUnit=z.literal("ai-micropoint-v1");
const scale=BigInt(1000000),maximum=BigInt("9223372036854775807");
export const micropoints=z.string().regex(/^(0|[1-9][0-9]*)$/).refine(value=>{try{return BigInt(value)<=maximum}catch{return false}});
export const positiveMicropoints=micropoints.refine(value=>value!=="0");
export function parseDisplayPoints(value:string):bigint{if(!/^(0|[1-9][0-9]*)(\.[0-9]{1,6})?$/.test(value))throw new Error("INVALID_POINTS");const[a,b=""]=value.split("."),n=BigInt(a)*scale+BigInt(b.padEnd(6,"0"));if(n>maximum)throw new Error("INVALID_POINTS");return n}
export const displayPoints=z.string().refine(value=>{try{parseDisplayPoints(value);return true}catch{return false}});
export const signedDisplayPoints=z.string().refine(value=>{try{if(value==="-0"||value.startsWith("-")&&parseDisplayPoints(value.slice(1))===BigInt(0))return false;parseDisplayPoints(value.startsWith("-")?value.slice(1):value);return true}catch{return false}});
export function formatMicropoints(value:string|bigint):string{const n=typeof value==="bigint"?value:BigInt(micropoints.parse(value));if(n<BigInt(0)||n>maximum)throw new Error("INVALID_POINTS");const whole=n/scale,tail=(n%scale).toString().padStart(6,"0").replace(/0+$/g,"");return whole.toString()+(tail?"."+tail:"")}
export function sumMicropoints(values:readonly string[]):bigint{return values.reduce((sum,value)=>sum+BigInt(micropoints.parse(value)),BigInt(0))}
