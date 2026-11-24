"use client";

import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { Button } from "@/components/ui/button";
import type { MemberResourceScope } from "@/lib/api/member-resources";
import {
  AMAZON_AUTHORIZATION_PATH,
  readAmazonConsentURL,
} from "@/lib/contracts/amazon-store-authorization";

const authorizationLabels = {
  not_connected: "尚未连接",
  authorization_saved: "已保存授权",
  reconnect_required: "需要重新授权",
} as const;
type AuthorizationStatus = keyof typeof authorizationLabels;

type Props = {
  storeId: string;
  storeVersion: number;
  scope: MemberResourceScope;
  canWrite: boolean;
};

export function StoreAmazonConnection({ storeId, storeVersion, scope, canWrite }: Props) {
  const pending = useRef<AbortController | null>(null);
  const [working, setWorking] = useState(false);
  const [failed, setFailed] = useState(false);
  const headers = {
    "X-Expected-User-ID": scope.expectedUserId,
    "X-Expected-Organization-ID": scope.expectedOrganizationId,
    "X-Store-ID": storeId,
  };
  const status = useQuery({
    queryKey: ["workbench", scope.expectedUserId, scope.expectedOrganizationId, "amazon-authorization", storeId],
    queryFn: async ({ signal }): Promise<AuthorizationStatus> => {
      const response = await fetch(`${AMAZON_AUTHORIZATION_PATH}/status`, {
        headers, signal, cache: "no-store", redirect: "error",
      });
      if (!response.ok) throw new Error("Amazon authorization status unavailable");
      const { status } = await response.json();
      if (typeof status !== "string" || !Object.hasOwn(authorizationLabels, status)) {
        throw new Error("Amazon authorization status unavailable");
      }
      return status as AuthorizationStatus;
    },
    retry: false,
    gcTime: 0,
    staleTime: 0,
  });
  useEffect(() => () => pending.current?.abort(), []);

  async function connect() {
    if (!canWrite || status.isError || !status.data || pending.current) return;
    const attempt = new AbortController();
    pending.current = attempt;
    setWorking(true);
    setFailed(false);
    try {
      const response = await fetch(`${AMAZON_AUTHORIZATION_PATH}/begin`, {
        method: "POST",
        headers: { ...headers, "If-Match": `"${storeVersion}"` },
        signal: attempt.signal, cache: "no-store", redirect: "error",
      });
      if (!response.ok) throw new Error("Amazon authorization unavailable");
      const { authorizationUrl } = await response.json();
      const target = readAmazonConsentURL(authorizationUrl);
      if (!target) throw new Error("Invalid Amazon consent address");
      if (!attempt.signal.aborted) window.location.assign(target);
    } catch {
      if (!attempt.signal.aborted) setFailed(true);
    } finally {
      if (!attempt.signal.aborted) {
        pending.current = null;
        setWorking(false);
      }
    }
  }

  return (
    <section className="mt-4 space-y-3" aria-label="Amazon 美国站授权">
      <h2 className="font-semibold">Amazon 美国站</h2>
      {status.isError ? (
        <p role="alert">暂时无法读取 Amazon 授权状态。<Button variant="ghost" onClick={() => void status.refetch()}>重试</Button></p>
      ) : (
        <p role="status">{status.data ? authorizationLabels[status.data] : "正在读取授权状态…"}</p>
      )}
      <p className="text-sm text-muted-foreground">请使用这家店铺的 Amazon 美国站卖家账号授权。</p>
      {canWrite && status.data && !status.isError ? (
        <Button disabled={working || status.isFetching} onClick={() => void connect()}>
          {working ? "正在前往 Amazon…" : status.data === "not_connected" ? "连接 Amazon" : "重新授权"}
        </Button>
      ) : null}
      {failed ? <p role="alert">无法连接 Amazon，请稍后重试。</p> : null}
    </section>
  );
}
