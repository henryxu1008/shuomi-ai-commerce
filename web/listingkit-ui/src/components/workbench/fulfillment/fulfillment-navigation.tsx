"use client";
import { cloneElement, type ComponentProps, type ReactElement } from "react";
import { useQuery } from "@tanstack/react-query";
import type { ConsoleNavigation } from "../console/console-navigation";
import {
  fulfillmentCapabilities,
  fulfillmentRequest,
  type FulfillmentScope,
} from "@/lib/api/fulfillment";
type Props = {
  enabled: boolean;
  scope: FulfillmentScope;
  permissions: string[];
  children: ReactElement<ComponentProps<typeof ConsoleNavigation>>;
};
export function FulfillmentNavigation(props: Props) {
  if (!props.enabled) return props.children;
  return <ScopedNavigation {...props} />;
}
function ScopedNavigation({ scope, permissions, children }: Props) {
  const capability = useQuery({
    queryKey: [
      "fulfillment",
      scope.organizationId,
      scope.userId,
      "entry",
      permissions,
    ],
    queryFn: ({ signal }) =>
      fulfillmentRequest(
        scope,
        "capabilities",
        fulfillmentCapabilities,
        signal,
      ),
    retry: false,
  });
  return cloneElement(children, {
    storeOrdersAvailable:
      children.props.storeOrdersAvailable ||
      capability.data?.available === true,
  });
}
