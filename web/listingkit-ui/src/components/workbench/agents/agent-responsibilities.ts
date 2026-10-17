import type { CatalogEntry } from "@/lib/contracts/agent-configuration";

export const agentCategories = ["全部", "商品", "选品", "运营", "广告", "客服", "数据"] as const;
export type AgentCategory = (typeof agentCategories)[number];
type AgentId = CatalogEntry["agent"]["agentId"];
export const agentGroups: readonly {
  id: string;
  agentIds: readonly AgentId[];
  category: AgentCategory;
  name?: string;
  description?: string;
  summary: string;
}[] = [
  {
    id: "product-content", agentIds: ["product.content.agent", "product.title.agent"], category: "商品",
    name: "商品内容与本地化", description: "完善商品标题、描述、卖点与关键词，按目标语言处理内容。",
    summary: "标题优化 · 内容完善与本地化",
  },
  { id: "listing-materials", agentIds: ["listing.materials.agent"], category: "商品", summary: "平台上架资料 · 可检查草稿" },
  { id: "product-image", agentIds: ["product.image.agent"], category: "商品", summary: "主图与详情图 · 人工选择批准" },
  { id: "pod-design", agentIds: ["pod.design.agent"], category: "商品", summary: "原创图案 · 保存后布局与效果图" },
  { id: "product-selection", agentIds: ["product.selection.agent"], category: "选品", summary: "商品事实对比 · 有引用建议" },
  { id: "shop-diagnosis", agentIds: ["shop.operations-diagnosis.agent"], category: "运营", summary: "经营事实诊断 · 有引用建议" },
];

export function groupAgentEntries(entries: readonly CatalogEntry[]) {
  return agentGroups.flatMap(group => {
    const operations = group.agentIds.flatMap(id => entries.filter(entry => entry.agent.agentId === id));
    return operations.length ? [{...group, name:group.name ?? operations[0].name, description:group.description ?? operations[0].description, entries:operations}] : [];
  });
}

export function groupActivation(group: { agentIds: readonly AgentId[]; entries: readonly CatalogEntry[] }) {
  const enabled = group.entries.filter(entry => entry.agent.activation === "ENABLED").length;
  const disabled = group.entries.every(entry => entry.agent.activation === "DISABLED");
  return {active:enabled > 0, disabled, label:enabled === group.agentIds.length ? "已启用" : enabled ? "部分已启用" : disabled ? "已停用" : "尚未启用"};
}
