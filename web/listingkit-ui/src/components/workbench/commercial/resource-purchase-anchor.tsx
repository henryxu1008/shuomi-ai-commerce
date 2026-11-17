"use client";

import { useEffect, useRef, type ReactNode } from "react";
import styles from "./commercial.module.css";

export function ResourcePurchaseAnchor({ id, children }: { id: string; children: ReactNode }) {
  const anchor = useRef<HTMLDivElement>(null);
  useEffect(() => {
    function followFragment() {
      if (window.location.hash !== `#${id}`) return;
      anchor.current?.scrollIntoView();
      anchor.current?.focus({ preventScroll: true });
    }
    // On a direct link or client navigation, the catalog can load after the fragment is resolved.
    followFragment();
    window.addEventListener("hashchange", followFragment);
    return () => window.removeEventListener("hashchange", followFragment);
  }, [id]);
  return <div className={styles.resourceAnchor} id={id} tabIndex={-1} ref={anchor}>{children}</div>;
}
