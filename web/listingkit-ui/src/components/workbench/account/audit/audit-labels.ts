import type { AccountAuditPage } from "@/lib/api/account-audit";

const operationLabels: Record<AccountAuditPage["items"][number]["operation"], string> = {
  register: "登记源账号",
  enable: "启用源账号",
  disable: "停用源账号",
  allocate_member_resource: "分配成员资源",
  reclaim_member_resource: "回收成员资源",
  set_member_ai_point_limit: "设置成员 AI 月度上限",
  update: "更新账户资料",
  invite: "邀请成员",
  role: "更新成员角色",
  remove: "移除成员",
  consume: "扣除图片 AI 点数",
  observe: "记录模型用量",
};

export function auditOperationLabel(item: AccountAuditPage["items"][number]) {
  return operationLabels[item.operation];
}
