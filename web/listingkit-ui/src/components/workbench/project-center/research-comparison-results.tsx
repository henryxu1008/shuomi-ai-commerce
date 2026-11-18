"use client";

import Image from "next/image";
import Link from "next/link";
import { useState } from "react";
import type { ResearchCandidate } from "@/lib/contracts/market-research";
import type { ResearchInspection } from "@/lib/contracts/research-analysis";
import { FactTable, factLabel } from "../agents/product-selection-flow";
import styles from "./research-comparison.module.css";

const amountFormat = new Intl.NumberFormat("en-US", {
  minimumFractionDigits: 2,
  maximumFractionDigits: 8,
});

export function researchQuote(amount: number, currency: string) {
  return `${currency} ${amountFormat.format(amount)}`;
}

export function amazonMarket(site: string) {
  return site === "us" ? "Amazon 美国站" : `Amazon · ${site.toUpperCase()}`;
}

const missingLabels: Record<string, string> = {
  availability: "库存状态",
  price: "页面报价",
  currency: "币种",
  description: "商品描述",
  brand: "品牌",
  attributes: "属性",
  variants: "商品规格",
  rating: "评分",
  reviewCount: "评论数",
  review_count: "评论数",
  omitted_fields: "其他未展示的资料",
};

export function researchMissingLabel(path: string) {
  const field = path.replace(/^observation\./, "");
  const label = factLabel(field);
  return missingLabels[field] ?? (label !== field ? label : "其他商品资料");
}

export function AmazonResearchThumbnail({ url, alt }: { url?: string; alt: string }) {
  const [failedUrl, setFailedUrl] = useState<string>();
  const missing = !url || failedUrl === url;
  return (
    <span className={styles.thumbnail}>
      {missing ? (
        <span>{url ? "主图暂不可用" : "暂无主图"}</span>
      ) : (
        <Image
          unoptimized
          loading="lazy"
          src={`/api/image-proxy?url=${encodeURIComponent(url)}`}
          alt={alt}
          width={56}
          height={56}
          onError={() => setFailedUrl(url)}
        />
      )}
    </span>
  );
}

export function ResearchComparisonResults({ inspection, originals }: {
  inspection: ResearchInspection;
  originals: ResearchCandidate[];
}) {
  const { candidates, summary } = inspection;
  return (
    <div className={styles.comparison}>
      <div className={styles.overview} role="status">
        <p><strong>本次比较 {summary.selected} 件</strong>{summary.unpriced ? <span> · {summary.unpriced} 件缺少可比报价</span> : null}</p>
        {summary.prices.map(({ currency, basis, count, minimum, maximum }) => (
          <p key={currency + "|" + basis}>
            <strong>
              {researchQuote(minimum, currency)}{maximum !== minimum ? `–${amountFormat.format(maximum)}` : ""}
            </strong>
            <span> · {count} 件{basis === "synthetic_listing_offer" ? "示例报价" : "页面报价"}</span>
          </p>
        ))}
      </div>
      <ul className={styles.products} aria-label="所选商品报价">
        {candidates.map(candidate => {
          const { ref, observation, facts, missing } = candidate;
          const original = originals.find(value => value.id === candidate.research.CandidateID)?.observation;
          const amazon = original && !("shein" in original) ? original : undefined;
          const savedTitle = facts.find(fact => fact.path === "title")?.value;
          const title = amazon?.title ?? (typeof savedTitle === "string" ? savedTitle : "未命名商品");
          const belongsToQuoteGroup = summary.prices.some(range =>
            range.currency === observation.currency && range.basis === observation.priceBasis);
          const price = belongsToQuoteGroup && observation.price != null && observation.currency
            ? researchQuote(observation.price, observation.currency) : null;
          return (
            <li key={ref.ItemID} className={styles.product}>
              <AmazonResearchThumbnail url={amazon?.item?.thumbnailUrl} alt={`${title} 主图`} />
              <div className={styles.identity}>
                <strong>{title}</strong>
                <span>{amazonMarket(observation.site)} · {observation.asin}</span>
              </div>
              <div className={styles.quote}>
                {price && <span>{observation.priceBasis === "synthetic_listing_offer" ? "示例报价" : "页面报价"}</span>}
                {price ? <strong>{price}</strong> : <span>暂无可比报价</span>}
                {missing.includes("observation.availability") && <span>库存未知</span>}
              </div>
              <Link href={`/workbench/data/mine?item=${ref.ItemID}`} prefetch={false}>查看已存资料</Link>
            </li>
          );
        })}
      </ul>
      <details className={styles.details}>
        <summary>展开商品资料与缺项</summary>
        <FactTable inspection={{ products: candidates.map(candidate => ({
          ref: candidate.ref,
          sourceKind: "amazon_data",
          facts: candidate.facts.filter(fact =>
            !fact.path.startsWith("observation.") && fact.path !== "review" && fact.path !== "warnings"),
          missing: [...new Set(candidate.missing.map(researchMissingLabel))],
        })) }} />
      </details>
    </div>
  );
}
