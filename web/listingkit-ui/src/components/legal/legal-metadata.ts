import type { Metadata } from "next";

import { socialPreviewImagePath } from "@/lib/site-metadata";

export function createSumiLegalMetadata(title: string, description: string): Metadata {
  const fullTitle = `${title} | 硕米智能引擎`;

  return {
    title: fullTitle,
    description,
    openGraph: {
      title: fullTitle,
      description,
      siteName: "硕米智能引擎",
      type: "website",
      images: [socialPreviewImagePath],
    },
    twitter: {
      card: "summary_large_image",
      title: fullTitle,
      description,
      images: [socialPreviewImagePath],
    },
  };
}
