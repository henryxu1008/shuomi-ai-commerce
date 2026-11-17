export const resourceDefinitions = [
  {
    type: "store_renewal_period",
    name: "店铺服务",
    unit: "期",
    detail: "1 期为 30 天；购买后需分配给成员，并在已连接店铺中开通或续费。",
    tone: "blue",
  },
  {
    type: "ai_point",
    name: "AI 点数",
    unit: "点",
    detail: "企业共用预付余额；成员按管理员设置的月度上限使用。",
    tone: "teal",
  },
  {
    type: "data_row",
    name: "数据资源",
    unit: "条",
    detail: "按采集成功入库的商品结果扣减；成员额度由管理员分配。",
    tone: "purple",
  },
] as const;

export const storagePurchaseId = "enterprise-storage";
export const walletBalanceId = "wallet-balance";
export const walletPath = "/workbench/plans/top-up";

export function resourcePurchaseHref(id: (typeof resourceDefinitions)[number]["type"] | typeof storagePurchaseId | typeof walletBalanceId) {
  return `${walletPath}#${id}`;
}
