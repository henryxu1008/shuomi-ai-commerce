import {handlePOD,rejectPOD} from "@/lib/server/pod-route";
export const GET=handlePOD;
export const POST=handlePOD;
export const PUT=rejectPOD;
export const DELETE=rejectPOD;
export const PATCH=rejectPOD;
export const dynamic="force-dynamic";
