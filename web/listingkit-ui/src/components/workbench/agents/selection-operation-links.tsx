import Link from "next/link";
import { selectionOperationLinks } from "./agent-responsibilities";

export function SelectionOperationLinks({className, agentIds, configuration = true}: {
  className?: string;
  agentIds?: readonly string[];
  configuration?: boolean;
}) {
  return <nav aria-label={configuration ? "智能选品操作配置" : "智能选品业务操作"} className={className}>
    {selectionOperationLinks.filter(operation => !agentIds || agentIds.includes(operation.agentId)).map(operation =>
      <Link key={operation.agentId} href={operation.href} prefetch={false}>{operation.label}{configuration ? "配置与历史" : ""}</Link>)}
  </nav>;
}
