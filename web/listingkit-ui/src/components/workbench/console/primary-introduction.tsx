import Link from "next/link";
import type { ReactNode } from "react";
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
  skills: {
    label: "技能库", title: "把业务经验沉淀为可复用的技能",
    subtitle: "查看官方模板，整理个人方法，沉淀团队技能，在支持的业务操作中按需使用。",
    entries: [
      ["标准方法", "官方模板", "查看已发布启用的模板与使用说明，\n按许可复制，也可直接自建技能。", "", "浏览官方模板", "/workbench/agents/skills/official"],
      ["个人沉淀", "我的技能", "保存个人业务方法与参考材料，\n管理版本，需要时提交团队审核。", "", "管理我的技能", "/workbench/agents/skills/mine"],
      ["企业复用", "团队技能", "维护当前企业可复用的业务方法，\n按权限审批个人提交和管理准入。", "", "管理团队技能", "/workbench/agents/skills/team"],
    ],
    featuresTitle: "让方法可以积累，也能被团队复用",
    featuresSubtitle: "把反复使用的方法整理下来，按版本迭代，在个人与团队之间有序复用",
    features: [["方法可复用", "把反复使用的业务方法保存下来。"], ["材料可组织", "为方法补充参考文本与模板。"], ["版本可管理", "更新时保存新版本，保留原有内容。"], ["团队可沉淀", "指定版本经审批后生成企业副本。"]],
    flowTitle: "如何开始使用", flowSubtitle: "从找到方法到业务中使用，按需要逐步完成",
    steps: [["选择来源", "浏览官方模板，或新建自己的技能"], ["完善方法", "编写方法与参考材料，并检查格式"], ["保存并启用", "保存版本，发布并启用后按需选择"], ["业务中使用", "在支持的业务操作中选用技能"]],
    notes: [
      ["个人与团队", "个人与团队如何复用", "个人技能归属本人；本人同意并提交指定版本后，\n由企业维护者审核并生成独立的团队副本。", ""],
      ["执行边界", "技能如何参与执行", "启用后在支持的业务操作中选择技能；执行继续\n使用现有智能体的权限、预算与审核。", ""],
    ],
  },
  tools: {
    label: "工具市场", title: "连接工具能力，让业务流程更高效",
    subtitle: "发现官方工具、管理已启用能力，或申请专属定制服务，让工具按需参与实际业务。",
    entries: [
      ["01", "官方工具", "查看平台维护的工具，了解功能、适用范围和使用要求。", "", "浏览官方工具", "/workbench/tools/official"],
      ["02", "我的工具", "集中管理已启用工具，查看授权与配置，并进入使用。", "", "管理我的工具", "/workbench/tools/mine"],
      ["03", "工具定制", "提交特殊采集、系统连接或自动化工具需求。", "", "了解定制服务", "/workbench/tools/custom"],
    ],
    flowTitle: "标准工具使用流程", flowSubtitle: "从了解工具到正式调用，按实际权限与配置要求完成",
    steps: [["发现工具", "按业务需要浏览适合的工具"], ["了解能力", "确认适用范围、权限及费用"], ["启用与配置", "完成必要授权和使用配置"], ["按需使用", "在支持的业务流程中调用"]],
    notes: [
      ["使用与管理", "工具启用后在哪里使用", "启用后可在「我的工具」管理和配置。", "当前商品采集通过插件或1688采集入口使用。"],
      ["权限与计费", "了解授权要求与服务费用", "工具权限、额度与费用以服务规则为准。", "定制服务需评估报价；实际交付及开放另行核实。"],
    ],
  },
  orders: {
    label: "订单管理", title: "统一管理订单、采购与发货，让履约流程更清晰",
    subtitle: "围绕原订单核对需求、采购收货、库存装箱和发货交接，让每一步都有原单据可查。",
    entries: [
      ["原单处理", "订单处理", "查看原单商品、数量与时限，关联内部履约需求，按当前条件继续办理。", "", "查看订单与需求", "/workbench/store-orders/fulfill?step=source"],
      ["采购交付", "采购与收货", "按订单需求生成采购单，登记外部单号，跟进分批到货与合格实收。", "", "进入采购与收货", "/workbench/supply/mine/procurement"],
      ["仓库履约", "装箱与发货", "核对可用库存和预留，准备包裹，再逐笔确认平台发货操作。", "", "进入装箱与发货", "/workbench/store-orders/fulfill?step=stock"],
      ["物流核对", "交接与物流", "记录实际交接，分别查看平台回执、发货进度与物流结果。", "", "查看交接与物流", "/workbench/store-orders/fulfill?step=logistics"],
    ],
    entryLabels: [["原单核对", "履约需求"], ["内部采购", "分批收货"], ["库存预留", "包裹装箱"], ["实际交接", "平台回执"]],
    flowTitle: "订单业务如何流转", flowSubtitle: "从原单关联到采购、发货与交接，按业务步骤持续衔接",
    steps: [["核对原单与需求", "确认原单类型、商品数量和时限，关联履约需求。"], ["采购并登记收货", "按缺货需求采购，分批登记合格实收、拒收和短缺。"], ["预留库存并装箱", "核对可用数量，按原需求准备包裹与装箱单。"], ["核对并确认发货", "检查发货资料和原操作内容，单独确认平台操作。"], ["记录交接与物流", "记录实际交接，跟进物流及平台实收、入库差异。"]],
    notes: [
      ["业务协同", "订单、采购与仓库如何协同", "采购围绕原需求补货，仓库按合格收货预留库存，装箱后核对发货和实际交接。", "订单需求 · 采购收货 · 发货交接"],
      ["使用边界", "平台状态与实际交接分别核对", "内部移交不代表平台签收；供应商下单和付款仍在供应商平台完成。", "案例仅供合成试用 · 真实平台发货待联调"],
    ],
    notePoints: [
      [["订单处理", "原单与内部履约需求"], ["采购与收货", "采购单、实收与交付进度"], ["发货与物流", "包裹、回执与实际交接"]],
      [["平台订单", "状态、时限与收货观察"], ["企业库存", "以实际收货和交接为准"], ["操作确认", "逐笔核对平台操作内容"]],
    ],
  },
  stores: {
    label: "店铺中心", title: "统一管理店铺、商品与订单，让经营链路更清晰",
    subtitle: "店铺中心负责维护店铺授权、查看商品与订单；经营分析和AI执行分别进入对应业务模块。",
    entries: [
      ["01 · 店铺资产", "我的店铺", "管理店铺授权、经营站点、店铺服务和连接状态。", "适合：绑定、授权和维护店铺", "管理我的店铺", "/workbench/stores"],
      ["02 · 商品资产", "店铺商品", "查看平台商品详情、刊登状态、库存信息和最近同步结果。", "适合：查看店铺内的商品资产", "查看店铺商品", "/workbench/store-products"],
      ["03 · 平台记录", "平台订单记录", "同步并查看订单详情、物流、包裹和异常。", "适合：查看原平台订单与物流记录", "查看平台订单记录", "/workbench/store-orders/records"],
    ],
    flowTitle: "店铺业务如何流转", flowSubtitle: "从绑定店铺到查看订单履约，只需四步",
    steps: [["绑定并授权店铺", "完成平台、站点和店铺信息配置"], ["同步店铺商品", "拉取在线商品、刊登与库存状态"], ["同步并查看订单", "统一查看订单详情、物流和异常"], ["到平台处理履约", "发货、售后与退款在平台完成"]],
    notes: [
      ["业务协同", "店铺业务如何协同", "我的店铺负责授权与服务，店铺商品负责商品查看，订单履约负责订单与物流事实查看。", "绑定店铺 → 同步商品 → 查看订单"],
      ["使用边界", "经营分析与AI执行统一分工", "经营数据与预警进入运营驾驶舱，商品优化和运营任务进入AI工作台。", "当前商品和订单只读 · 发货与售后到平台处理"],
    ],
  },
};

export function PrimaryIntroduction({ kind, available = introductions[kind].entries.map(() => true), entryHrefs, notice, actions, children }: { kind: keyof typeof introductions; available?: readonly boolean[]; entryHrefs?: readonly string[]; notice?: string; actions?: ReactNode; children?: ReactNode }) {
  const data = introductions[kind], id = `primary-${kind}`;
  const isSkills = kind === "skills", isOrders = kind === "orders", compact = kind === "tools" || isSkills;
  const entryTones = isOrders ? ["blue", "teal", "purple", "amber"] : ["teal", "blue", "purple"];
  const stepTones = isOrders ? ["blue", "teal", "purple", "amber", "blue"] : ["teal", "blue", "green", "purple"];
  return <section className={`${styles.page} ${homeStyles.page} ${kind === "tools" ? homeStyles.tools : ""} ${isSkills ? homeStyles.skills : ""} ${isOrders ? homeStyles.orders : ""}`} aria-labelledby={id}>
    <header className={styles.header}><p className={styles.eyebrow}>{data.label}</p><h1 id={id}>{data.title}</h1><p className={styles.subtitle}>{data.subtitle}</p>{actions ? <div className={homeStyles.headerActions}>{actions}</div> : null}</header>
    {notice ? <p role="status" className={styles.subtitle}>{notice}</p> : null}
    <div className={`${styles.entries} ${compact ? homeStyles.entries : ""} ${isOrders ? homeStyles.orderEntries : ""}`}>{data.entries.map(([badge, title, description, audience, action, href], index) => {
      const content = <>{isOrders ? <div className={homeStyles.orderEntryHeading}><h2>{title}</h2><span className={styles.badge}>{badge}</span></div> : isSkills ? <><div className={homeStyles.entryHeading}><span className={homeStyles.entryNumber} aria-hidden="true">{String(index + 1).padStart(2, "0")}</span><h2>{title}</h2></div><span className={`${styles.badge} ${homeStyles.skillBadge}`}>{badge}</span></> : kind === "tools" ? <div className={homeStyles.entryHeading}><span className={styles.badge}>{badge}</span><h2>{title}</h2></div> : <><span className={styles.badge}>{badge}</span><h2>{title}</h2></>}<p className={styles.description}>{description}</p>{audience ? <p className={styles.audience}>{audience}</p> : null}{"entryLabels" in data ? <div className={homeStyles.entryLabels}>{data.entryLabels[index].map(label => <span key={label}>{label}</span>)}</div> : null}<span className={styles.action}>{available[index] ? `${action}${compact || isOrders ? "" : " →"}` : "暂未启用"}</span></>;
      return available[index] ? <Link key={href} href={entryHrefs?.[index] ?? href} prefetch={false} className={styles.entry} data-tone={entryTones[index]}>{content}</Link> : <Card key={href} className={styles.entry} data-tone={entryTones[index]} data-state="unavailable">{content}</Card>;
    })}</div>
    {"features" in data ? <section className={homeStyles.features} aria-labelledby={`${id}-features`}>
      <div className={homeStyles.featureHeading}><h2 id={`${id}-features`}>{data.featuresTitle}</h2><p>{data.featuresSubtitle}</p></div>
      <Card className={homeStyles.featurePanel}>{data.features.map(([title, description]) => <div key={title}><h3>{title}</h3><p>{description}</p></div>)}</Card>
    </section> : null}
    <section className={styles.flow} aria-labelledby={`${id}-flow`}><div className={styles.flowHeading}><h2 id={`${id}-flow`}>{data.flowTitle}</h2><p>{data.flowSubtitle}</p></div>
      <ol className={`${styles.steps} ${compact ? homeStyles.steps : ""} ${isOrders ? homeStyles.orderSteps : ""}`}>{data.steps.map(([title, description], index) => <li key={title}><Card className={styles.step} data-tone={stepTones[index]}><span className={styles.number}>{String(index + 1).padStart(2, "0")}</span><h3>{title}</h3><p>{description}</p></Card></li>)}</ol>
    </section>
    {children}
    <div className={`${styles.notes} ${compact ? homeStyles.notes : ""} ${isOrders ? homeStyles.orderNotes : ""}`}>{data.notes.map(([label, title, description, footer], index) => <Card key={label} className={`${styles.note} ${index === 0 && !isSkills ? styles.dataFlow : ""}`} data-tone={isOrders ? index === 0 ? "blue" : "teal" : index === 0 ? "teal" : "purple"}>{isOrders && "notePoints" in data ? <><div className={homeStyles.orderNoteCopy}><p className={styles.noteLabel}>{label}</p><h2>{title}</h2><p className={styles.description}>{description}</p><p className={styles.noteFooter}>{footer}</p></div><ul className={homeStyles.orderNotePoints}>{data.notePoints[index].map(([heading, detail]) => <li key={heading}><strong>{heading}</strong><span>{detail}</span></li>)}</ul></> : <>{!isSkills ? <p className={styles.noteLabel}>{label}</p> : null}<h2>{title}</h2><p className={styles.description}>{description}</p>{footer ? <p className={styles.noteFooter}>{footer}</p> : null}</>}</Card>)}</div>
  </section>;
}
