import Link from "next/link";
import { Card } from "@/components/ui/card";
import styles from "../data-services/data-services-overview.module.css";
import homeStyles from "./primary-introduction.module.css";

const introductions = {
  supply: {
    label: "我的供应链", title: "统一管理商品，按目标平台完成发布",
    subtitle: "所有来源商品先进入「我的数据」统一管理，再选择批次进入发布流程。",
    entries: [
      ["01 · 数据管理", "我的数据", "所有来源商品先进入「我的数据」，统一管理商品、来源与批次。", "下一步：选择批次 → 转入「待适配」", "商品批次管理", "/workbench/data/mine"],
      ["02 · 平台处理", "适配与补全", "转入「待适配」后，按目标平台要求检查商品；缺少必填内容时进入「待补全」。", "流程：待适配 → 待补全（按需）→ 已适配", "查看待适配商品", "/workbench/supply/mine/waiting"],
      ["03 · 发布与复用", "上传到平台", "进入「已适配」后，可直接上传；也可先用已准入智能体优化，审核后再上传。", "最终：已上传 → 可继续发布到其他平台", "完成商品发布", "/workbench/supply/mine/ready"],
    ],
    flowTitle: "完整流程", flowSubtitle: "我的数据 → 待适配 → 待补全（按需）→ 已适配",
    steps: [["我的数据", "统一管理商品与批次，选择批次进入下一步"], ["待适配", "按目标平台要求检查商品，判断资料是否完整"], ["待补全（按需）", "只有缺少平台必填内容时进入，补全后继续"], ["已适配", "商品已满足发布要求，可以选择发布方式"]],
    notes: [
      ["路径 A 直接上传", "已适配 → 开始上传 → 已上传", "商品内容已经合适时，直接上传到目标平台，不经过待审核。", "上传和同步需要正式平台渠道配置"],
      ["路径 B 智能体优化", "已适配 → 智能体优化 → 待审核 → 已上传", "需要优化标题或图片时，选择已准入的智能体，先优化并审核，再上传。", "模型执行与预算按当前企业配置开放"],
    ],
  },
  agents: {
    label: "智能市场", title: "按岗位启用智能体，打造属于自己的AI电商团队",
    subtitle: "根据选品、运营、设计、数据分析等业务岗位选择智能体，了解能力、配置要求与使用方式。",
    entries: [
      ["01 · 岗位能力", "智能体市场", "按岗位和业务场景查找智能体，了解能力、适用范围和使用方式。", "适合：选择标准化岗位能力", "浏览智能体市场", "/workbench/agents/market"],
      ["02 · 已启用能力", "我的智能体", "管理本企业智能体配置，查看启用状态和所需模型配置。", "适合：日常管理与配置查看", "管理我的智能体", "/workbench/agents/mine"],
      ["03 · 专属需求", "智能体定制", "标准智能体无法满足需求时，可提交专属岗位和业务能力定制需求。", "适合：特殊流程与专属业务能力", "提交定制需求", "/workbench/agents/custom"],
    ],
    flowTitle: "如何开始使用", flowSubtitle: "从选择岗位能力到参与业务，只需四步",
    steps: [["选择智能体", "按岗位或业务场景查找合适的智能体"], ["查看岗位能力", "了解适用范围、调用方式和所需数据"], ["配置智能体", "确认权益、费用与企业模型配置"], ["开始使用", "就绪能力可在硕米 Chat 或任务中心使用"]],
    notes: [
      ["使用位置", "启用后在哪里使用", "企业配置在“我的智能体”中管理。模型、计价和预算配置就绪后，才可在AI工作台执行对应能力。", "智能市场负责选择能力 · AI工作台负责执行任务"],
      ["服务说明", "能力、权益和费用均提前说明", "能力目录与配置不代表模型执行已经开放；具体准入、使用权益和费用以各能力页面为准。", "定制智能体由专员评估需求、交付周期和费用"],
    ],
  },
  tools: {
    label: "工具市场", title: "按业务场景启用工具，让智能体获得可靠执行能力",
    subtitle: "选择适合业务场景的工具，了解权限、适用范围和使用方式，再接入对应业务流程。",
    entries: [
      ["01 · 标准工具", "官方工具", "浏览平台官方维护的工具，了解能力、适用范围、权限和使用方式。", "适合：快速接入标准工具能力", "浏览官方工具", "/workbench/tools/official"],
      ["02 · 已启用工具", "我的工具", "管理本企业已启用的工具，查看权限与安装说明，进入现有业务入口。", "适合：日常管理与授权查看", "管理我的工具", "/workbench/tools/mine"],
      ["03 · 专属需求", "工具定制", "标准工具无法满足流程时，可提交专属工具与连接能力定制需求。", "适合：特殊流程与专属连接能力", "提交工具定制", "/workbench/tools/custom"],
    ],
    flowTitle: "如何开始使用", flowSubtitle: "从发现合适工具到接入业务流程，只需四步",
    steps: [["发现工具", "按业务场景筛选适合的官方工具"], ["查看工具能力", "了解权限、适用范围和调用方式"], ["启用并授权", "获准管理员确认权限与费用后启用"], ["开始使用", "按工具说明进入现有业务流程"]],
    notes: [
      ["使用位置", "启用后在哪里使用", "启用后进入“我的工具”，当前商品采集工具通过对应插件和1688采集入口使用。", "工具市场负责接入工具 · AI工作台负责执行任务"],
      ["服务说明", "权限、适用范围和费用均提前说明", "使用前明确查看权限、适用范围和费用；通用任务、智能体自动调用能力尚未开放。", "定制工具由专员评估需求、交付周期和费用"],
    ],
  },
  stores: {
    label: "店铺中心", title: "统一管理店铺、商品与订单，让经营链路更清晰",
    subtitle: "店铺中心负责维护店铺授权、查看商品与订单；经营分析和AI执行分别进入对应业务模块。",
    entries: [
      ["01 · 店铺资产", "我的店铺", "管理店铺授权、经营站点、店铺服务和连接状态。", "适合：绑定、授权和维护店铺", "管理我的店铺", "/workbench/stores"],
      ["02 · 商品资产", "店铺商品", "查看平台商品详情、刊登状态、库存信息和最近同步结果。", "适合：查看店铺内的商品资产", "查看店铺商品", "/workbench/store-products"],
      ["03 · 交易履约", "订单履约", "同步并查看订单详情、物流、包裹和异常；发货与售后到平台处理。", "适合：查看订单与物流履约", "查看订单履约", "/workbench/store-orders"],
    ],
    flowTitle: "店铺业务如何流转", flowSubtitle: "从绑定店铺到查看订单履约，只需四步",
    steps: [["绑定并授权店铺", "完成平台、站点和店铺信息配置"], ["同步店铺商品", "拉取在线商品、刊登与库存状态"], ["同步并查看订单", "统一查看订单详情、物流和异常"], ["到平台处理履约", "发货、售后与退款在平台完成"]],
    notes: [
      ["业务协同", "店铺业务如何协同", "我的店铺负责授权与服务，店铺商品负责商品查看，订单履约负责订单与物流事实查看。", "绑定店铺 → 同步商品 → 查看订单"],
      ["使用边界", "经营分析与AI执行统一分工", "经营数据与预警进入运营驾驶舱，商品优化和运营任务进入AI工作台。", "当前商品和订单只读 · 发货与售后到平台处理"],
    ],
  },
};

export function PrimaryIntroduction({ kind, available = [true, true, true], entryHrefs, notice }: { kind: keyof typeof introductions; available?: readonly boolean[]; entryHrefs?: readonly string[]; notice?: string }) {
  const data = introductions[kind], id = `primary-${kind}`;
  return <section className={`${styles.page} ${homeStyles.page}`} aria-labelledby={id}>
    <header className={styles.header}><p className={styles.eyebrow}>{data.label}</p><h1 id={id}>{data.title}</h1><p className={styles.subtitle}>{data.subtitle}</p></header>
    {notice ? <p role="status" className={styles.subtitle}>{notice}</p> : null}
    <div className={styles.entries}>{data.entries.map(([badge, title, description, audience, action, href], index) => {
      const content = <><span className={styles.badge}>{badge}</span><h2>{title}</h2><p className={styles.description}>{description}</p><p className={styles.audience}>{audience}</p><span className={styles.action}>{available[index] ? `${action} →` : "暂未启用"}</span></>;
      return available[index] ? <Link key={href} href={entryHrefs?.[index] ?? href} prefetch={false} className={styles.entry} data-tone={["teal", "blue", "purple"][index]}>{content}</Link> : <Card key={href} className={styles.entry} data-tone={["teal", "blue", "purple"][index]} data-state="unavailable">{content}</Card>;
    })}</div>
    <section className={styles.flow} aria-labelledby={`${id}-flow`}><div className={styles.flowHeading}><h2 id={`${id}-flow`}>{data.flowTitle}</h2><p>{data.flowSubtitle}</p></div>
      <ol className={styles.steps}>{data.steps.map(([title, description], index) => <li key={title}><Card className={styles.step} data-tone={["teal", "blue", "green", "purple"][index]}><span className={styles.number}>{String(index + 1).padStart(2, "0")}</span><h3>{title}</h3><p>{description}</p></Card></li>)}</ol>
    </section>
    <div className={styles.notes}>{data.notes.map(([label, title, description, footer], index) => <Card key={label} className={`${styles.note} ${index === 0 ? styles.dataFlow : ""}`} data-tone={index === 0 ? "teal" : "purple"}><p className={styles.noteLabel}>{label}</p><h2>{title}</h2><p className={styles.description}>{description}</p><p className={styles.noteFooter}>{footer}</p></Card>)}</div>
  </section>;
}
