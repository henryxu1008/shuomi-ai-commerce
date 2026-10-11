"use client";
import {useQuery} from "@tanstack/react-query";
import {useWorkbenchContext} from "@/components/providers/workbench-context-provider";
import {catalogSchema,catalogSearch,storeCenterRequest,type CatalogFilters} from "@/lib/api/store-center";
export function useStoreCatalog(filters:CatalogFilters,valid=true){
 const context=useWorkbenchContext(),organizationId=context.effectiveOrganization?.id??"",userId=context.user?.id??"";
 return useQuery({queryKey:["store-catalog",organizationId,userId,context.permissions,filters],queryFn:({signal})=>storeCenterRequest(`catalog?${catalogSearch(filters)}`,{organizationId,userId},catalogSchema,{signal}),enabled:valid&&Boolean(organizationId && userId && !context.isSwitching),retry:false,gcTime:0});
}
