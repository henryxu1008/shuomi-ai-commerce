import Link from "next/link";
import { Button } from "@/components/ui/button";
import type { Stage, SupplyStagePage } from "@/lib/contracts/supply-chain";
import { listingStages, supplyStageHref, type SupplyStageContext } from "@/lib/workbench/supply-navigation";

export function ListingStageTabs({ current, context, counts }: {
  current: Stage;
  context: SupplyStageContext;
  counts?: SupplyStagePage["counts"];
}) {
  return (
    <nav aria-label="商品刊登阶段" className="mb-4 flex flex-wrap gap-2 border-b border-border pb-3">
      {listingStages.map(({ id, label }) => {
        const active = current === id;
        const count = counts && (id === "waiting" ? counts.waiting + counts.ready : counts[id]);
        return (
          <Button key={id} asChild variant={active ? "default" : "ghost"}>
            <Link href={supplyStageHref(id, context)} prefetch={false} aria-current={active ? "page" : undefined}>
              {label}{count === undefined ? "" : " (" + count + ")"}
            </Link>
          </Button>
        );
      })}
    </nav>
  );
}
