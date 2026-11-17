"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { ArrowUpRight } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { getStorageOffers } from "@/lib/api/storage-capacity";
import { formatRetailMoney } from "@/lib/api/retail-prices";
import type { ResourceOffers } from "@/lib/api/resource-purchase";
import { resourceDefinitions, resourcePurchaseHref, storagePurchaseId } from "./resource-catalog-display";
import styles from "./commercial.module.css";

type Props = {
  offers: ResourceOffers;
  userId: string;
  organizationId: string;
  storageAvailable: boolean;
};

export function PlanOptionsView({ offers, userId, organizationId, storageAvailable }: Props) {
  const storagePrices = useQuery({
    queryKey: ["workbench", organizationId, "plan-storage-prices", userId],
    queryFn: ({ signal }) => getStorageOffers(userId, organizationId, signal),
    enabled: storageAvailable,
    gcTime: 0,
    retry: false,
  });

  return (
    <div className={styles.stack}>
      <Card className={styles.panel}>
        <p className={styles.eyebrow}>当前收费方案</p>
        <h2>基础方案 · 按需使用</h2>
        <p className={styles.observation}>基础方案无需订阅。店铺服务按期购买，AI 点数和数据资源按需补充。</p>
      </Card>
      <section className={styles.charges} aria-labelledby="plan-charges">
        <h2 id="plan-charges">收费构成</h2>
        <div className={styles.resourcePurchaseCards} data-storage={storageAvailable || undefined}>
          {resourceDefinitions.map(resource => {
            const prices = [...new Set(offers.items
              .filter(offer => offer.resource_type === resource.type)
              .map(offer => offer.settlement_mode === "ZERO_PRICE" ? "免费领取" : `${formatRetailMoney(offer.unit_price_minor)} / ${resource.unit}`))];
            return (
              <PlanPriceCard key={resource.type} href={resourcePurchaseHref(resource.type)}
                name={resource.name} prices={prices} detail={resource.detail} tone={resource.tone} />
            );
          })}
          {storageAvailable ? (
            <PlanPriceCard href={resourcePurchaseHref(storagePurchaseId)} name="企业存储" tone="orange"
              prices={storagePrices.isPending ? ["正在读取价格…"] : storagePrices.isError ? ["价格暂不可用"] :
                [...new Set(storagePrices.data?.items.map(offer => `${formatRetailMoney(offer.unit_price_minor)} / 扩容包`))]}
              detail="企业默认共享 5 GiB。每个扩容包增加 50 GiB，有效期 30 天。" />
          ) : null}
        </div>
      </section>
      <div className={styles.planDetails}>
        <Card className={styles.panel}>
          <h2>计费规则</h2>
          <ul className={styles.planRules}>
            <li>购买店铺期数后，请到已连接的店铺开通或续费。每家店铺单独计算有效期。</li>
            <li>AI 点数和数据资源分别计量，未用完的余额不按月清零。</li>
            <li>付费资源从企业钱包扣款；免费资源按实际可领取的数量确认。</li>
            {storageAvailable ? <li>扩容包到期不自动续费，也不会删除已保存的文件。</li> : null}
          </ul>
        </Card>
        <Card className={styles.panel}>
          <h2>相关管理</h2>
          <div className={styles.actions}>
            <Button asChild variant="outline"><Link href="/workbench/plans/usage" prefetch={false}>查看用量明细</Link></Button>
            <Button asChild variant="outline"><Link href="/workbench/plans/orders" prefetch={false}>账单与订单</Link></Button>
          </div>
        </Card>
      </div>
    </div>
  );
}

function PlanPriceCard({ href, name, prices, detail, tone }: {
  href: string;
  name: string;
  prices: string[];
  detail: string;
  tone: "blue" | "teal" | "purple" | "orange";
}) {
  return (
    <Link href={href} prefetch={false} scroll={false} className={`${styles.resourcePurchaseCard} ${styles.planPriceCard} ${styles[tone]}`}>
      <h3>{name}<ArrowUpRight size={16} aria-hidden="true" /></h3>
      {prices.length ? prices.map(price => <p key={price} className={styles.offerPrice}>{price}</p>) : <p>价格未配置</p>}
      <p>{detail}</p>
    </Link>
  );
}
