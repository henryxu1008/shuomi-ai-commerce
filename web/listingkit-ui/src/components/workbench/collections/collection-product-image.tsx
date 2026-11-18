"use client";

import Image from "next/image";
import { useState } from "react";
import styles from "./product-grid.module.css";

export function CollectionProductImage({ url, alt }: { url?: string; alt: string }) {
  const [failedUrl, setFailedUrl] = useState<string>();

  return (
    <span className={styles.image}>
      {url && failedUrl !== url ? (
        <Image
          unoptimized
          loading="lazy"
          src={`/api/image-proxy?url=${encodeURIComponent(url)}`}
          alt={alt}
          width={156}
          height={156}
          onError={() => setFailedUrl(url)}
        />
      ) : (
        <span>{url ? "主图暂不可用" : "暂无主图"}</span>
      )}
    </span>
  );
}
