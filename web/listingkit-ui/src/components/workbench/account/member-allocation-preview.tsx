"use client";

import { useQuery } from "@tanstack/react-query";
import { Card } from "@/components/ui/card";
import { getMemberAIPointLimits } from "@/lib/api/member-ai-point-limits";
import { enterpriseRoleKey } from "@/lib/api/enterprise-role-schema";
import { getMemberResources, MemberResourceError } from "@/lib/api/member-resources";
import { name, roles } from "../resources/member-resource-view";
import styles from "./account.module.css";
import overviewStyles from "../console/primary-overview.module.css";

const quantity = new Intl.NumberFormat("zh-CN");
const formatQuantity = (value: string, unit: string) => `${quantity.format(BigInt(value))} ${unit}`;

export function MemberAllocationPreview({ userId, organizationId, scope }: {
  userId: string;
  organizationId: string;
  scope: string;
}) {
  const identity = { expectedUserId: userId, expectedOrganizationId: organizationId };
  const directory = useQuery({
    queryKey: ["account-overview", userId, organizationId, "member-allocation", scope],
    queryFn: ({ signal }) => getMemberResources(identity, signal),
    gcTime: 0,
    staleTime: 0,
    retry: false,
  });
  const members = !directory.isFetching && !directory.isError && directory.data?.organizationId === organizationId
    ? directory.data.members
    : undefined;
  const points = useQuery({
    queryKey: ["account-overview", userId, organizationId, "member-allocation-points", scope],
    queryFn: ({ signal }) => getMemberAIPointLimits(identity, signal),
    gcTime: 0,
    staleTime: 0,
    retry: false,
    enabled: Boolean(members?.length),
  });
  const limits = !points.isFetching && !points.isError && points.data?.organizationId === organizationId
    ? new Map(points.data.members.map(member => [member.memberId, member]))
    : undefined;

  return <Card role="region" aria-label="成员资源分配" className={overviewStyles.panel}>
    <h2>成员资源分配</h2>
    {directory.isPending || directory.isFetching ? <p className={styles.note} role="status">正在读取成员资源…</p>
      : !members ? <p className={styles.note} role="status">{directory.error instanceof MemberResourceError && directory.error.status === 403
        ? "当前无权查看成员资源"
        : "暂时无法读取成员资源"}</p>
      : members.length === 0 ? <p className={styles.note} role="status">暂无可查看的成员资源</p>
      : <div className={styles.allocationPreview} role="region" aria-label="成员资源分配表格" tabIndex={0}>
        <table aria-label="成员资源分配">
          <thead><tr><th scope="col">成员／角色</th><th scope="col">已授权店铺</th><th scope="col">可用续费期数</th><th scope="col">本月剩余 AI 消费额度</th><th scope="col">可用数据条数</th></tr></thead>
          <tbody>{members.slice(0, 5).map(member => {
            const limit = limits?.get(member.memberId);
            const labels = member.roles.map(role =>
              Object.hasOwn(roles, role) ? roles[role] : enterpriseRoleKey.safeParse(role).success ? "自定义角色" : "角色名称暂未提供",
            );
            return <tr key={member.memberId}>
              <th scope="row">{name(member)}{labels.length > 0 ? <small>{[...new Set(labels)].join("、")}</small> : null}</th>
              <td>{member.storeCount === null ? "暂不可用" : formatQuantity(member.storeCount, "家")}</td>
              <td>{member.periods.recorded ? formatQuantity(member.periods.free, "期") : "暂无记录"}</td>
              <td>{limit ? (limit.configured ? `${limit.remaining} 点` : "未设置") : points.isPending || points.isFetching ? "正在读取" : "暂不可用"}</td>
              <td>{member.dataRows.recorded ? formatQuantity(member.dataRows.free, "条") : "暂无记录"}</td>
            </tr>;
          })}</tbody>
        </table>
      </div>}
  </Card>;
}
