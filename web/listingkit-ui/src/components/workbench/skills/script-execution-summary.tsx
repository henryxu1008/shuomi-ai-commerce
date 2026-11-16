import type {z} from "zod";
import type {scriptReceiptSchema} from "@/lib/contracts/skill";

type ScriptReceipt = z.infer<typeof scriptReceiptSchema>;

const outcomes = { succeeded: "执行成功", failed: "执行失败", outcome_unknown: "执行结果未确认", not_dispatched: "未派发执行" };
const cleanup = { completed: "临时沙箱已清理", pending: "临时沙箱清理待确认", not_needed: "无须清理临时沙箱" };
const reasons: Record<string, string> = {
    profile_unavailable: "执行环境尚未开放", invalid_package: "保存的脚本包不符合当前合同",
    control_unavailable: "执行环境连接异常，请联系管理员检查连接、凭据或证书",
    capacity_deadline: "等待执行容量超时", creation_pending: "此前创建结果待确认",
    capacity_unavailable: "执行容量暂不可用", deadline: "剩余运行时间不足",
    credential_unavailable: "执行凭据无法准备", create_unconfirmed: "沙箱创建结果待确认",
    instance_mismatch: "执行实例与批准版本不一致", execd_auth_unavailable: "执行控制入口认证未通过核验",
    setup_failed: "执行目录准备失败", upload_failed: "脚本或输入上传未完成",
    script_failed: "脚本明确退出失败", execution_unconfirmed: "脚本终态未确认",
    result_unconfirmed: "结果文件尚未确认", invalid_result: "结果文件格式不符合约定",
    dispatch_unconfirmed: "执行派发结果待确认",
};

export function ScriptExecutionDetails({ receipt }: { receipt: ScriptReceipt }) {
    return <><p>{cleanup[receipt.cleanup]}</p>{receipt.outcome !== "succeeded" && <p>{reasons[receipt.reason] ?? "本次执行未产生可采用的结果"}</p>}</>;
}

const titleOutcomes = {
    succeeded: "技能处理完成",
    failed: "技能处理失败",
    outcome_unknown: "技能处理结果待确认",
    not_dispatched: "技能未执行",
};

export function ScriptExecutionSummary({ receipt, before, after, titleNextStep }: {
    receipt?: ScriptReceipt;
    before?: string;
    after?: string;
    titleNextStep?: string;
}) {
    if (!receipt) return null;
    const succeeded = receipt.outcome === "succeeded";
    if (titleNextStep !== undefined) {
        const nextStep = succeeded ? titleNextStep : receipt.reason === "control_unavailable"
            ? "标题建议暂不能采用。请联系管理员检查技能服务。"
            : receipt.outcome === "outcome_unknown"
                ? "确认前暂不能采用。请重新查看本次结果。"
                : "本次没有可采用的技能处理结果。标题建议暂不能采用。";
        return <section aria-label="标题技能处理结果" className="space-y-4">
            <div role="status" className="space-y-1">
                <h3 className="font-medium">{titleOutcomes[receipt.outcome]}</h3>
                <p className="text-sm text-muted-foreground">{nextStep}</p>
            </div>
            {(before || (succeeded && after)) && <dl className={succeeded && before && after ? "grid gap-3 sm:grid-cols-2" : "grid gap-3"}>
                {before && <div className="min-w-0 space-y-1 rounded-lg border border-border p-3">
                    <dt className="text-sm text-muted-foreground">{succeeded ? "技能处理前" : "已生成的标题建议"}</dt>
                    <dd className="whitespace-pre-wrap [overflow-wrap:anywhere]">{before}</dd>
                </div>}
                {succeeded && after && <div className="min-w-0 space-y-1 rounded-lg border border-border bg-muted p-3">
                    <dt className="text-sm text-muted-foreground">技能处理后</dt>
                    <dd className="whitespace-pre-wrap font-medium [overflow-wrap:anywhere]">{after}</dd>
                </div>}
            </dl>}
        </section>;
    }
    return <section aria-label="脚本后处理结果" className="space-y-2 rounded-lg border border-border p-3 text-sm">
        <h4 className="font-medium">脚本后处理 · {outcomes[receipt.outcome]}</h4>
        {before && <p className="whitespace-pre-wrap break-words">处理前：{before}</p>}
        {succeeded && after && <p className="whitespace-pre-wrap break-words">处理后：{after}</p>}
        <ScriptExecutionDetails receipt={receipt}/>
        {!succeeded && <p>本次后处理已停止，不自动重试或跳过脚本，不能采用未确认结果。</p>}
    </section>;
}
