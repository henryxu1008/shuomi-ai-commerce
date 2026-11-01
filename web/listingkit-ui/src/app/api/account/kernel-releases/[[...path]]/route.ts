import { handleKernelReleases, rejectKernelReleases } from "@/lib/server/kernel-releases-route";
export const GET = handleKernelReleases;
export const PUT = handleKernelReleases;
export const POST = rejectKernelReleases;
export const PATCH = rejectKernelReleases;
export const DELETE = rejectKernelReleases;
export const HEAD = rejectKernelReleases;
export const OPTIONS = rejectKernelReleases;
