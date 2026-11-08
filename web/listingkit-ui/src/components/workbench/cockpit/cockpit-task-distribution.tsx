"use client";

import type { CSSProperties } from "react";
import { EChart } from "@/components/ui/chart";
import type { TaskCounts } from "@/lib/contracts/cockpit-metrics";
import styles from "./cockpit-overview.module.css";

const states = [
  ["RUNNING", "执行中", "#3da977"],
  ["WAITING_CONFIRMATION", "待确认", "#a28de0"],
  ["COMPLETED", "已完成", "#4b8be5"],
  ["ERROR", "异常", "#dd6368"],
  ["PAUSED", "已暂停", "#c9913b"],
  ["UNKNOWN", "状态未知", "#8993a2"],
] as const;

export function CockpitTaskDistribution({ counts, message }: { counts?: TaskCounts; message?: string }) {
  const total = counts && Object.values(counts).reduce((sum, value) => sum + value, 0);
  if (!counts) return <p className={styles.empty}>{message ?? "任务统计暂不可用"}</p>;
  return <div className={styles.distribution}>
    {total ? <EChart ariaLabel={`执行状态分布，共 ${total} 条任务`} className={styles.taskRing} option={{
      series: [{
        type: "pie",
        radius: ["62%", "92%"],
        label: { show: false },
        emphasis: { disabled: true },
        data: states.map(([key, name, color]) => ({ name, value: counts[key], itemStyle: { color } })),
      }],
    }} /> : <div className={styles.emptyRing} aria-hidden="true">0</div>}
    <ul className={styles.legend} aria-label="任务状态数量">
      {states.map(([key, label, color]) => <li key={key}>
        <span className={styles.dot} style={{ "--tone": color } as CSSProperties} aria-hidden="true" />
        <span>{label}</span><span>{counts[key]}</span>
      </li>)}
    </ul>
  </div>;
}
