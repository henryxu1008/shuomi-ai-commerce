"use client";

import Image from "next/image";
import { Select } from "@/components/ui/select";
import type { SupplySourceDetail } from "@/lib/contracts/supply-chain";

export const sourceImageRoles = { main: "主图", white_background: "白底图", gallery: "展示图", design: "设计图" };
export type SourceImageRole = keyof typeof sourceImageRoles;
export function SourceImageSelection({ images, selected, disabled, onChange, layout = "rows" }: {
  images: SupplySourceDetail["images"]; selected: Record<string, SourceImageRole>; disabled: boolean;
  layout?: "rows" | "grid";
  onChange: (value: Record<string, SourceImageRole>) => void;
}) {
  return <div className={layout === "grid" ? "grid gap-3 sm:grid-cols-3" : "space-y-3"}>{images.map((image, index) => <div key={image.id} className={layout === "grid" ? "space-y-2 rounded-lg border border-border p-3" : "flex flex-wrap items-center gap-4 rounded-2xl bg-muted p-4"}>
    <Image unoptimized src={`/api/image-proxy?url=${encodeURIComponent(image.url)}`} width={layout === "grid" ? 120 : 80} height={layout === "grid" ? 120 : 80} alt={`来源图 ${index + 1}`} className={layout === "grid" ? "h-28 w-full object-contain" : "h-20 w-20 rounded-lg object-contain"} />
    <label className={layout === "grid" ? "flex items-center gap-2 text-sm" : "flex min-w-36 flex-1 items-center gap-2 text-sm font-semibold"}><input type="checkbox" disabled={disabled} checked={!!selected[image.id]} onChange={event => {
      const next = { ...selected }; if (event.target.checked) next[image.id] = "gallery"; else delete next[image.id]; onChange(next);
    }} />使用来源图 {index + 1}</label>
    {selected[image.id] ? <Select className={layout === "grid" ? undefined : "ml-auto w-36"} aria-label={`来源图 ${index + 1} 用途`} disabled={disabled} value={selected[image.id]} onChange={event => onChange({ ...selected, [image.id]: event.target.value as SourceImageRole })}>
      {Object.entries(sourceImageRoles).map(([role, name]) => <option key={role} value={role}>{name}</option>)}
    </Select> : null}
  </div>)}</div>;
}
