import {z} from "zod";
const text=(n:number)=>z.string().refine(s=>new TextEncoder().encode(s).length<=n&&!/[\u0000-\u001f\u007f]/.test(s));
export const artworkParametersSchema=z.strictObject({schema:z.literal("pod-artwork-config-v1"),style:text(512),constraints:text(4096)});
