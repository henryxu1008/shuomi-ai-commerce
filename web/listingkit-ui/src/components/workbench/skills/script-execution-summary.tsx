import type {z} from "zod";
import type {scriptReceiptSchema} from "@/lib/contracts/skill";

export function ScriptExecutionSummary({receipt,before,after}:{receipt?:z.infer<typeof scriptReceiptSchema>;before?:string;after?:string}){
 if(!receipt)return null;
 const outcome={succeeded:"执行成功",failed:"执行失败",outcome_unknown:"执行结果未确认",not_dispatched:"未派发执行"}[receipt.outcome];
 const cleanup={completed:"临时沙箱已清理",pending:"临时沙箱清理待确认",not_needed:"无须清理临时沙箱"}[receipt.cleanup];
 const reason:Record<string,string>={profile_unavailable:"执行环境尚未开放",invalid_package:"保存的脚本包不符合当前合同",capacity_deadline:"等待执行容量超时",creation_pending:"此前创建结果待确认",capacity_unavailable:"执行容量暂不可用",deadline:"剩余运行时间不足",credential_unavailable:"执行凭据无法准备",create_unconfirmed:"沙箱创建结果待确认",instance_mismatch:"执行实例与批准版本不一致",execd_auth_unavailable:"执行控制入口认证未通过核验",setup_failed:"执行目录准备失败",upload_failed:"脚本或输入上传未完成",script_failed:"脚本明确退出失败",execution_unconfirmed:"脚本终态未确认",result_unconfirmed:"结果文件尚未确认",invalid_result:"结果文件格式不符合约定",dispatch_unconfirmed:"执行派发结果待确认"};
 return <section aria-label="脚本后处理结果" className="space-y-2 rounded-lg border border-border p-3 text-sm"><h4 className="font-medium">脚本后处理 · {outcome}</h4>{before&&<p className="whitespace-pre-wrap break-words">处理前：{before}</p>}{receipt.outcome==="succeeded"&&after&&<p className="whitespace-pre-wrap break-words">处理后：{after}</p>}<p>{cleanup}</p>{receipt.outcome!=="succeeded"&&<><p>{reason[receipt.reason]??"本次执行未产生可采用的结果"}</p><p>本次后处理已停止，不自动重试或跳过脚本，不能采用未确认结果。</p></>}</section>;
}
