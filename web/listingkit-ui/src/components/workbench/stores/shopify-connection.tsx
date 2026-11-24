"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import type { MemberResourceScope } from "@/lib/api/member-resources";
import type { WorkbenchStore } from "@/lib/api/workbench-stores";

export function ShopifyConnection({store, scope, canWrite, onChanged}: {
  store: WorkbenchStore;
  scope: MemberResourceScope;
  canWrite: boolean;
  onChanged: () => Promise<unknown>;
}) {
  const [working, setWorking] = useState(false);
  const [message, setMessage] = useState("");
  const pending = useQuery({
    enabled: canWrite,
    queryKey: ["shopify-pairing", scope.expectedUserId, scope.expectedOrganizationId, store.id],
    queryFn: async ({signal}) => {
      const response = await fetch(`/api/shopify/pending?store=${store.id}`, {signal, cache: "no-store"});
      if (!response.ok) return "";
      const result = await response.json();
      return typeof result.domain === "string" ? result.domain : "";
    },
    retry: false,
    gcTime: 0,
  });
  const issue = typeof window === "undefined" ? null : new URLSearchParams(window.location.search).get("shopify");
  async function run(action: "begin" | "confirm") {
    if (working) return;
    setWorking(true);
    setMessage("");
    try {
      const response = await fetch(`/api/shopify/${action}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Expected-User-ID": scope.expectedUserId,
          "X-Expected-Organization-ID": scope.expectedOrganizationId,
        },
        body: JSON.stringify(action === "begin" ? {store: store.id, version: store.version} : {store: store.id}),
      });
      if (!response.ok) {
        setMessage(action === "begin" ? "暂时无法连接 Shopify，请稍后重试。" : "无法确认这次授权。请重新连接 Shopify。");
        return;
      }
      if (action === "begin") {
        const data = await response.json();
        if (typeof data.authorizationUrl !== "string") throw new Error("missing installation link");
        window.location.assign(data.authorizationUrl);
        return;
      }
      await onChanged();
      await pending.refetch();
      setMessage("Shopify 店铺已连接。");
    } catch {
      setMessage("暂时无法确认连接状态，请刷新后查看。");
    } finally {
      setWorking(false);
    }
  }
  return <div className="grid gap-3">
    <p>{store.connectionStatus === "connected" ? "Shopify 店铺已连接。" : "请由 Shopify 店主授权，然后回到这里确认店铺。"}</p>
    {issue === "owner" ? <p role="alert">请让具备商品管理权限的 Shopify 店主重新授权。</p> : null}
    {issue === "restart" ? <p role="alert">这次授权未完成，请重新连接 Shopify。</p> : null}
    {pending.data ? <p>请确认连接 Shopify 店铺 <strong>{pending.data}</strong>。</p> : null}
    {canWrite && store.recordStatus === "active" ? <div>
      <Button disabled={working || pending.isPending} onClick={() => void run(pending.data ? "confirm" : "begin")}>
        {pending.data ? "确认连接店铺" : store.connectionStatus === "connected" ? "重新连接 Shopify" : "连接 Shopify"}
      </Button>
    </div> : null}
    {message ? <p role="status">{message}</p> : null}
  </div>;
}
