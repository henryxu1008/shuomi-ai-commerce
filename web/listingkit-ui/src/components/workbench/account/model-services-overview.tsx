import type { z } from "zod";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { modelChannelLabel, type ModelOption, type routeCurrent } from "@/lib/contracts/model-route-pricing";
import type { ModelServiceCategory } from "@/lib/contracts/model-services-navigation";
import styles from "./model-route-pricing.module.css";

const labels = {
  text: "硕米 Chat · 规划业务任务",
  multimodal: "图片分析模型",
  "image-generation": "图片生成模型",
};
const categoryOrder = ["text", "multimodal", "image-generation"] as const;
const priceReasons = new Set(["OFFICIAL_PRICE_MISSING", "PRICE_OR_FX_NOT_APPLICABLE", "PRICE_APPLICABILITY_MISSING"]);

type ChatStatus = "checking" | "unknown" | "closed" | "AVAILABLE" | "NEEDS_CONFIGURATION" | "UNAVAILABLE";
type Props = {
  current?: z.infer<typeof routeCurrent>;
  checking: boolean;
  chat: ChatStatus;
  credentials: boolean;
  platform: boolean;
  disabled: boolean;
  onSelectModel: (category: ModelServiceCategory) => void;
  onCredentials: (model: ModelOption) => void;
  onPrices: (category: ModelServiceCategory) => void;
  onRefresh: () => void;
};

function modelStatus(model: ModelOption | undefined, status: Props["current"], checking: boolean) {
  if (checking) return "正在检查";
  if (status?.status === "UNCONFIGURED") return "尚未发布模型";
  if (!model) return status?.revision?.payload.sharedModels?.schemaVersion===2 && status.preview ? "尚未选择" : "模型配置暂时无法确认";
  if (model.ready) return "模型配置符合条件";
  if (model.reason === "OFFICIAL_PRICE_MISSING") return "缺少适用参考价";
  if (model.reason && priceReasons.has(model.reason)) return "参考价或汇率暂不适用";
  if (model.reason === "CREDENTIAL_NOT_ADMITTED") return "渠道暂不可用，请核对渠道配置";
  if (model.reason === "PROTOCOL_AND_COST_BOUND_MISSING") return "图片生成服务尚未配置";
  return "模型配置待处理";
}

const chatLabels: Record<ChatStatus, string> = {
  checking: "正在检查",
  unknown: "Chat 规划状态暂时无法确认",
  closed: "当前企业暂未开放 Chat",
  AVAILABLE: "企业已具备 Chat 规划条件",
  NEEDS_CONFIGURATION: "Chat 规划需要检查配置",
  UNAVAILABLE: "Chat 规划暂不可用",
};

export function ModelServicesOverview({ current, checking, chat, credentials, platform, disabled, onSelectModel, onCredentials, onPrices, onRefresh }: Props) {
  const rows = categoryOrder.map(category => {
    const model = current?.preview?.models.find(option => option.category === category);
    const qualification = modelStatus(model, current, checking);
    const ready = category === "text" ? chat === "AVAILABLE" : qualification === "模型配置符合条件";
    return { category, model, qualification, ready };
  }).sort((first, second) => Number(first.ready) - Number(second.ready));

  return <Card className={styles.panel}>
    <div className={styles.sectionHeader}>
      <div>
        <h2>使用概览</h2>
        <p>查看当前企业的 Chat 规划和商品图片模型设置。实际操作时仍会检查权限、资料和点数。</p>
      </div>
      <Button variant="outline" disabled={checking} onClick={onRefresh}>重新读取状态</Button>
    </div>
    <div className={styles.modelGrid}>
      {rows.map(({ category, model, qualification, ready }) => {
        const credentialProblem = model?.reason === "CREDENTIAL_NOT_ADMITTED";
        const priceProblem = model?.reason && priceReasons.has(model.reason);
        const status = category === "text" ? chatLabels[chat] : qualification;
        const action = current?.status === "UNCONFIGURED" ? "去选模型" : "查看模型";
        return <section className={styles.modelCard} aria-label={labels[category]} key={category}>
          <h3 data-model-service={category} tabIndex={-1}>{labels[category]}</h3>
          <span className={styles.overviewStatus} data-ready={ready} role="status">{status}</span>
          {category === "text" && qualification !== status && <p>企业文本模型 · {qualification}</p>}
          {model && model.model !== "UNKNOWN" && <p>{model.model} · {modelChannelLabel(model.channel, model.channelName)}</p>}
          {credentialProblem && !credentials && <p>请联系本企业渠道管理员。</p>}
          {priceProblem && !platform && <p>请联系平台维护人员检查参考价与汇率。</p>}
          {!checking && (credentialProblem && credentials && model
            ? <Button variant="outline" disabled={disabled} onClick={() => onCredentials(model)}>查看渠道配置</Button>
            : priceProblem && platform
              ? <Button variant="outline" disabled={disabled} onClick={() => onPrices(category)}>查看参考价格与汇率</Button>
              : <Button variant="outline" disabled={disabled} onClick={() => onSelectModel(category)}>{action}</Button>)}
        </section>;
      })}
    </div>
  </Card>;
}
